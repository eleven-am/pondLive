import type { Channel } from '../types/pondlive';
import { Logger } from './logger';
import { ATTR_SCRIPT } from './wire';

export const ATTR_PPROPS = 'data-pprops';
export const ATTR_POWN = 'data-pown';

export type ScriptDef = (el: Element, io: Channel<unknown, unknown, unknown>) => unknown;
export type ScriptErrorName = 'def' | 'on' | 'props' | 'cleanup' | 'queue' | 'undefined';
export type ScriptAction = 'm' | 'err';

const QUEUE_MAX = 32;
const ERROR_MAX_BYTES = 1024;
const ERROR_INTERVAL_MS = 1000;

const defs = new Map<string, ScriptDef>();
let sealed = false;

export function define(key: string, fn: ScriptDef): void {
    if (sealed) {
        throw new Error(`define(${key}) after start`);
    }
    defs.set(key, fn);
}

export function seal(): void {
    sealed = true;
}

export function lookup(key: string): ScriptDef | undefined {
    return defs.get(key);
}

export interface ScriptsConfig {
    send: (id: number, action: ScriptAction, payload: unknown) => void;
    lookup?: (key: string) => ScriptDef | undefined;
}

interface Instance {
    id: number;
    queue: unknown[];
    overflowed: boolean;
    gone: boolean;
    lastError: number;
    props: unknown;
    raw: string | undefined;
    io: Channel<unknown, unknown, unknown>;
    handler?: (msg: unknown) => void;
    propsHandler?: (props: unknown) => void;
    cleanup?: () => void;
}

const utf8 = new TextEncoder();
const scratch = new Uint8Array(ERROR_MAX_BYTES);

function cap(text: string): string {
    return text.slice(0, utf8.encodeInto(text, scratch).read);
}

function describe(e: unknown): string {
    return e instanceof Error ? `${e.name}: ${e.message}` : String(e);
}

function isThenable(v: unknown): v is PromiseLike<unknown> {
    return typeof (v as PromiseLike<unknown> | null)?.then === 'function';
}

function parseProps(attr: string | null): Record<string, unknown> {
    if (attr === null) return {};
    try {
        const parsed = JSON.parse(attr);
        return typeof parsed === 'object' && parsed !== null ? parsed : {};
    } catch (e) {
        Logger.warn('Scripts', 'invalid data-pprops', String(e));
        return {};
    }
}

function parseTokens(attr: string | null): Array<[token: string, key: string, id: number]> {
    const out: Array<[string, string, number]> = [];
    for (const token of (attr ?? '').split(' ')) {
        if (token === '') continue;
        const cut = token.lastIndexOf(':');
        const id = cut > 0 && /^[1-9]\d*$/.test(token.slice(cut + 1)) ? Number(token.slice(cut + 1)) : 0;
        if (id === 0) {
            Logger.warn('Scripts', 'invalid data-pscript token', token);
            continue;
        }
        out.push([token, token.slice(0, cut), id]);
    }
    return out;
}

export class Scripts {
    private readonly hosts = new Map<Element, Map<string, Instance>>();
    private readonly byId = new Map<number, Instance>();
    private readonly find: (key: string) => ScriptDef | undefined;

    constructor(private readonly config: ScriptsConfig) {
        this.find = config.lookup ?? lookup;
    }

    reconcile(el: Element): void {
        const wanted = parseTokens(el.getAttribute(ATTR_SCRIPT));
        const have = this.hosts.get(el);
        if (have) {
            const keep = new Set(wanted.map((w) => w[0]));
            for (const [token, inst] of have) {
                if (!keep.has(token)) {
                    have.delete(token);
                    this.release(inst);
                }
            }
        }
        if (wanted.length === 0) {
            this.hosts.delete(el);
            return;
        }
        const props = parseProps(el.getAttribute(ATTR_PPROPS));
        const next = have ?? new Map<string, Instance>();
        this.hosts.set(el, next);
        for (const [token, key, id] of wanted) {
            const value = props[String(id)];
            const inst = next.get(token);
            if (!inst) {
                this.attach(el, next, token, key, id, value);
                continue;
            }
            const raw = JSON.stringify(value);
            if (raw === inst.raw) continue;
            inst.raw = raw;
            inst.props = value;
            if (inst.propsHandler) this.guard(inst, 'props', () => inst.propsHandler!(value));
        }
    }

    releaseHost(el: Element): void {
        const have = this.hosts.get(el);
        if (!have) return;
        this.hosts.delete(el);
        for (const inst of have.values()) this.release(inst);
    }

    deliver(id: number, data: unknown): void {
        const inst = this.byId.get(id);
        if (!inst) return;
        if (inst.handler) {
            this.invoke(inst, data);
        } else if (inst.queue.length < QUEUE_MAX) {
            inst.queue.push(data);
        } else if (!inst.overflowed) {
            inst.overflowed = true;
            this.report(inst, 'queue', `more than ${QUEUE_MAX} messages before on()`);
        }
    }

    destroy(): void {
        for (const el of [...this.hosts.keys()]) this.releaseHost(el);
    }

    private attach(el: Element, host: Map<string, Instance>, token: string, key: string, id: number, props: unknown): void {
        const inst: Instance = {
            id,
            queue: [],
            overflowed: false,
            gone: false,
            lastError: -Infinity,
            props,
            raw: JSON.stringify(props),
            io: {
                on: (fn) => {
                    inst.handler = fn;
                    while (inst.queue.length > 0 && inst.handler === fn) {
                        this.invoke(inst, inst.queue.shift());
                    }
                    inst.overflowed = false;
                },
                send: (msg) => {
                    if (!inst.gone) this.config.send(id, 'm', msg);
                },
                get props() {
                    return inst.props;
                },
                onProps: (fn) => {
                    inst.propsHandler = fn;
                },
            },
        };
        host.set(token, inst);
        this.byId.set(id, inst);
        const def = this.find(key);
        if (!def) {
            this.report(inst, 'undefined', `no script defined for ${key}`);
            return;
        }
        try {
            const out = def(el, inst.io);
            if (isThenable(out)) {
                out.then(
                    (cleanup) => this.settle(inst, cleanup),
                    (e) => this.report(inst, 'def', e),
                );
            } else {
                this.settle(inst, out);
            }
        } catch (e) {
            this.report(inst, 'def', e);
        }
    }

    private settle(inst: Instance, cleanup: unknown): void {
        if (typeof cleanup !== 'function') return;
        inst.cleanup = cleanup as () => void;
        if (inst.gone) this.runCleanup(inst);
    }

    private release(inst: Instance): void {
        inst.gone = true;
        inst.queue.length = 0;
        if (this.byId.get(inst.id) === inst) this.byId.delete(inst.id);
        this.runCleanup(inst);
    }

    private runCleanup(inst: Instance): void {
        const cleanup = inst.cleanup;
        if (!cleanup) return;
        inst.cleanup = undefined;
        this.guard(inst, 'cleanup', cleanup);
    }

    private invoke(inst: Instance, data: unknown): void {
        this.guard(inst, 'on', () => inst.handler!(data));
    }

    private guard(inst: Instance, name: ScriptErrorName, fn: () => void): void {
        try {
            fn();
        } catch (e) {
            this.report(inst, name, e);
        }
    }

    private report(inst: Instance, name: ScriptErrorName, error: unknown): void {
        const m = cap(typeof error === 'string' ? error : describe(error));
        Logger.error('Scripts', `${name} failed`, { id: inst.id, error: m });
        const now = Date.now();
        if (now - inst.lastError < ERROR_INTERVAL_MS) return;
        inst.lastError = now;
        this.config.send(inst.id, 'err', { n: name, m });
    }
}

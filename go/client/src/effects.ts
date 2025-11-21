import {DOMRequest, DOMResponse} from './protocol';
import {Logger} from './logger';
import {
    DOMActionEffect,
    CookieEffect,
    Effect,
    EffectExecutorConfig,
    RefResolver,
    DOMResponseCallback
} from './types';

export class EffectExecutor {
    private readonly sessionId: string;
    private readonly resolveRef: RefResolver;
    private readonly onDOMResponse: DOMResponseCallback;

    constructor(config: EffectExecutorConfig) {
        this.sessionId = config.sessionId;
        this.resolveRef = config.resolveRef;
        this.onDOMResponse = config.onDOMResponse;
    }

    execute(effects: Effect[]): void {
        if (!effects || effects.length === 0) return;

        for (const effect of effects) {
            this.executeOne(effect);
        }
    }

    handleDOMRequest(req: DOMRequest): void {
        const el = this.resolveRef(req.ref);

        if (!el) {
            this.sendDOMResponse(req.id, undefined, undefined, `ref not found: ${req.ref}`);
            return;
        }

        try {
            if (req.props && req.props.length > 0) {
                const values = this.readProps(el, req.props);
                this.sendDOMResponse(req.id, values, undefined, undefined);
            } else if (req.method) {
                const result = this.callMethod(el, req.method, req.args ?? []);
                this.sendDOMResponse(req.id, undefined, result, undefined);
            } else {
                this.sendDOMResponse(req.id, undefined, undefined, 'no props or method specified');
            }
        } catch (e) {
            const error = e instanceof Error ? e.message : String(e);
            this.sendDOMResponse(req.id, undefined, undefined, error);
        }
    }

    private executeOne(effect: Effect): void {
        switch (effect.type) {
            case 'dom':
                this.executeDOMAction(effect);
                break;
            case 'cookies':
                this.executeCookieSync(effect);
                break;
        }
    }

    private executeDOMAction(effect: DOMActionEffect): void {
        const el = this.resolveRef(effect.ref);
        if (!el) {
            Logger.warn('Effects', 'Ref not found', effect.ref);
            return;
        }

        try {
            switch (effect.kind) {
                case 'dom.call':
                    this.executeCall(el, effect);
                    break;
                case 'dom.set':
                    this.executeSet(el, effect);
                    break;
                case 'dom.toggle':
                    this.executeToggle(el, effect);
                    break;
                case 'dom.class':
                    this.executeClass(el, effect);
                    break;
                case 'dom.scroll':
                    this.executeScroll(el, effect);
                    break;
                default:
                    Logger.warn('Effects', 'Unknown kind', effect.kind);
            }
        } catch (e) {
            Logger.error('Effects', 'Execution failed', e);
        }
    }

    private executeCall(el: Element, effect: DOMActionEffect): void {
        if (!effect.method) return;

        const method = (el as unknown as Record<string, unknown>)[effect.method];
        if (typeof method === 'function') {
            method.apply(el, effect.args ?? []);
        } else {
            Logger.warn('Effects', 'Method not found', effect.method);
        }
    }

    private executeSet(el: Element, effect: DOMActionEffect): void {
        if (!effect.prop) return;
        (el as unknown as Record<string, unknown>)[effect.prop] = effect.value;
    }

    private executeToggle(el: Element, effect: DOMActionEffect): void {
        if (!effect.prop) return;
        const current = (el as unknown as Record<string, unknown>)[effect.prop];
        (el as unknown as Record<string, unknown>)[effect.prop] = !current;
    }

    private executeClass(el: Element, effect: DOMActionEffect): void {
        if (!effect.class) return;

        if (effect.on === true) {
            el.classList.add(effect.class);
        } else if (effect.on === false) {
            el.classList.remove(effect.class);
        } else {
            el.classList.toggle(effect.class);
        }
    }

    private executeScroll(el: Element, effect: DOMActionEffect): void {
        if (!el.scrollIntoView) return;

        const opts: ScrollIntoViewOptions = {};
        if (effect.behavior) opts.behavior = effect.behavior;
        if (effect.block) opts.block = effect.block;
        if (effect.inline) opts.inline = effect.inline;
        el.scrollIntoView(opts);
    }

    private executeCookieSync(effect: CookieEffect): void {
        const url = `${effect.endpoint}?sid=${encodeURIComponent(effect.sid)}&token=${encodeURIComponent(effect.token)}`;
        fetch(url, {
            method: effect.method ?? 'GET',
            credentials: 'include'
        }).catch(e => {
            Logger.error('Effects', 'Cookie sync failed', e);
        });
    }

    private readProps(el: Element, props: string[]): Record<string, unknown> {
        const values: Record<string, unknown> = {};
        for (const prop of props) {
            values[prop] = this.resolveProp(el, prop);
        }
        return values;
    }

    private resolveProp(el: Element, path: string): unknown {
        const segments = path.split('.');
        let current: unknown = el;

        for (const segment of segments) {
            if (current == null) return undefined;
            current = (current as Record<string, unknown>)[segment];
        }

        return this.serializeValue(current);
    }

    private callMethod(el: Element, method: string, args: unknown[]): unknown {
        const fn = (el as unknown as Record<string, unknown>)[method];
        if (typeof fn !== 'function') {
            throw new Error(`method not found: ${method}`);
        }
        const result = fn.apply(el, args);
        return this.serializeValue(result);
    }

    private serializeValue(value: unknown): unknown {
        if (value === null || value === undefined) return null;

        const type = typeof value;
        if (type === 'string' || type === 'number' || type === 'boolean') return value;

        if (Array.isArray(value)) {
            return value.map(v => this.serializeValue(v));
        }

        if (value instanceof Date) return value.toISOString();
        if (value instanceof DOMTokenList) return Array.from(value);
        if (value instanceof Node) return undefined;

        try {
            return JSON.parse(JSON.stringify(value));
        } catch {
            return undefined;
        }
    }

    private sendDOMResponse(id: string, values?: Record<string, unknown>, result?: unknown, error?: string): void {
        const response: DOMResponse = {
            t: 'dom_res',
            sid: this.sessionId,
            id,
        };

        if (values !== undefined) response.values = values;
        if (result !== undefined) response.result = result;
        if (error !== undefined) response.error = error;

        this.onDOMResponse(response);
    }
}

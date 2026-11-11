import { Applier } from '../../src/applier';
import { domEvent } from '../../src/fields';
import type { Edit } from '../../src/protocol';

export interface Step {
    edits: Edit[];
    expected: string;
    live: string;
    reset: string;
    keep: number[];
    fresh: boolean;
}

export interface Case {
    name: string;
    ssr: string;
    steps: Step[];
}

export interface FixtureFile {
    scenario: string;
    description: string;
    cases: Case[];
}

export interface Failure {
    scenario: string;
    case: string;
    step: number;
    check: string;
    detail: string;
}

interface Recorded {
    id: number;
    event: string;
}

const VOID = new Set(['area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input', 'link', 'meta', 'param', 'source', 'track', 'wbr']);
const MARKER = /^(t|r|\/r):(\d+)$/;

function escapeText(value: string): string {
    return value.replace(/&/g, '&amp;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&#34;');
}

function escapeAttr(value: string): string {
    return value.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

export function canonical(node: Node, strip: boolean): string {
    switch (node.nodeType) {
        case Node.TEXT_NODE: {
            const data = node.nodeValue ?? '';
            if (data === '') return '';
            const parent = node.parentNode as Element | null;
            const tag = parent?.localName;
            return tag === 'script' || tag === 'style' ? data : escapeText(data);
        }
        case Node.COMMENT_NODE: {
            const data = node.nodeValue ?? '';
            if (strip && MARKER.test(data)) return '';
            return `<!--${data}-->`;
        }
        case Node.ELEMENT_NODE: {
            const el = node as Element;
            const attrs = Array.from(el.attributes)
                .filter((a) => !(strip && a.name === 'data-pi'))
                .sort((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0));
            let out = `<${el.localName}`;
            for (const a of attrs) out += ` ${a.name}="${escapeAttr(a.value)}"`;
            out += '>';
            if (VOID.has(el.localName)) return out;
            for (let c = el.firstChild; c; c = c.nextSibling) out += canonical(c, strip);
            return `${out}</${el.localName}>`;
        }
        default:
            return '';
    }
}

type Kind = 'el' | 't' | 'r';

function scan(root: Node): Map<number, { kind: Kind; node: Node }> {
    const out = new Map<number, { kind: Kind; node: Node }>();
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_COMMENT);
    let n: Node | null = walker.currentNode;
    while (n) {
        if (n.nodeType === Node.ELEMENT_NODE) {
            const v = (n as Element).getAttribute('data-pi');
            if (v !== null) out.set(Number(v), { kind: 'el', node: n });
            if ((n as Element).hasAttribute('data-praw')) {
                n = nextSkippingChildren(walker, n);
                continue;
            }
        } else {
            const m = MARKER.exec(n.nodeValue ?? '');
            if (m && m[1] === 't') out.set(Number(m[2]), { kind: 't', node: n });
            else if (m && m[1] === 'r') out.set(Number(m[2]), { kind: 'r', node: n });
        }
        n = walker.nextNode();
    }
    return out;
}

function nextSkippingChildren(walker: TreeWalker, from: Node): Node | null {
    let n: Node | null = from;
    while (n && n !== walker.root) {
        if (n.nextSibling) {
            walker.currentNode = n.nextSibling;
            if (n.nextSibling.nodeType === Node.ELEMENT_NODE || n.nextSibling.nodeType === Node.COMMENT_NODE) return n.nextSibling;
            return walker.nextNode();
        }
        n = n.parentNode;
    }
    return null;
}

function mount(html: string): { applier: Applier; events: Recorded[] } {
    const parsed = new DOMParser().parseFromString(html, 'text/html');
    document.replaceChild(document.importNode(parsed.documentElement, true), document.documentElement);
    const events: Recorded[] = [];
    const applier = new Applier(document.documentElement, {
        onEvent: (id, event) => {
            events.push({ id, event });
        },
        onScript: () => {},
        onScriptCleanup: () => {},
    });
    applier.index();
    return { applier, events };
}

function message(e: unknown): string {
    return e instanceof Error ? `${e.name}: ${e.message}` : String(e);
}

function checkIndex(applier: Applier, reset: string): string | null {
    const expected = scan(new DOMParser().parseFromString(reset, 'text/html').documentElement);
    const live = scan(document.documentElement);
    const want = [...expected.entries()].map(([id, v]) => `${id}:${v.kind}`).sort();
    const got = [...live.entries()].map(([id, v]) => `${id}:${v.kind}`).sort();
    if (want.join(',') !== got.join(',')) return `index ids differ from the reset document\nlive:  ${got.join(',')}\nreset: ${want.join(',')}`;
    for (const [id, v] of live) {
        const node = applier.node(id);
        if (!node) return `id ${id} (${v.kind}) is not indexed`;
        if (v.kind === 'el' && node !== v.node) return `id ${id} indexes a different element than the one carrying data-pi`;
        if (v.kind === 'r' && node !== v.node) return `id ${id} does not index its region start marker`;
        if (v.kind === 't' && (node.nodeType !== Node.TEXT_NODE || node.previousSibling !== v.node)) return `id ${id} does not index the text after its marker`;
    }
    return null;
}

interface Spec {
    e: string;
    pv?: number;
    db?: number;
    th?: number;
    o?: number;
}

interface Dirty {
    id: number;
    prop: 'value' | 'checked';
    want: string | boolean;
}

function dirty(applier: Applier, edits: Edit[]): Dirty[] {
    const last = new Map<string, Dirty>();
    for (const e of edits) {
        const id = e[1] as number;
        const el = applier.element(id);
        if (!el) continue;
        if (e[0] === 'a' && e[2] === 'value' && (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement)) {
            el.value = 'typed by the user';
            last.set(`${id}:value`, { id, prop: 'value', want: e[3] as string });
        } else if (e[0] === 'a' && e[2] === 'checked' && el instanceof HTMLInputElement) {
            el.checked = false;
            last.set(`${id}:checked`, { id, prop: 'checked', want: true });
        } else if (e[0] === 'ra' && e[2] === 'checked' && el instanceof HTMLInputElement) {
            el.checked = true;
            last.set(`${id}:checked`, { id, prop: 'checked', want: false });
        }
    }
    return [...last.values()];
}

function checkDirty(applier: Applier, pending: Dirty[]): string | null {
    for (const d of pending) {
        const el = applier.element(d.id) as HTMLInputElement | undefined;
        if (el && el[d.prop] !== d.want) return `the ${d.prop} property of element ${d.id} is ${String(el[d.prop])}, want ${String(d.want)} after the edit`;
    }
    return null;
}

function checkHandlers(events: Recorded[]): string | null {
    for (const el of Array.from(document.querySelectorAll('[data-pon]'))) {
        const id = Number(el.getAttribute('data-pi'));
        let specs: Spec[];
        try {
            specs = JSON.parse(el.getAttribute('data-pon') ?? '');
        } catch (e) {
            return `data-pon on element ${id} is not JSON: ${message(e)}`;
        }
        if (!Number.isInteger(id)) return `element with data-pon has no data-pi: ${el.outerHTML}`;
        for (const spec of specs) {
            if (spec.db || spec.th || spec.o) continue;
            events.length = 0;
            const event = new Event(domEvent(el, spec.e), { bubbles: true, cancelable: true });
            el.dispatchEvent(event);
            const hit = events.some((r) => r.id === id && r.event === spec.e);
            if (!hit) return `a ${spec.e} event on element ${id} did not reach the server callback`;
            if (spec.pv && !event.defaultPrevented) return `a ${spec.e} listener on element ${id} did not prevent the default`;
        }
    }
    events.length = 0;
    return null;
}

export function runEditCase(scenario: string, c: Case): Failure[] {
    const failures: Failure[] = [];
    const fail = (step: number, check: string, detail: string) => failures.push({ scenario, case: c.name, step, check, detail });
    const { applier, events } = mount(c.ssr);
    for (let i = 0; i < c.steps.length; i++) {
        const step = c.steps[i];
        const before = new Map<number, Node | undefined>();
        for (const id of step.keep) before.set(id, applier.node(id));
        const pending = dirty(applier, step.edits);
        try {
            applier.apply(step.edits);
        } catch (e) {
            fail(i, 'apply', `${message(e)} edits=${JSON.stringify(step.edits)}`);
            return failures;
        }
        const live = canonical(document.documentElement, false);
        if (live !== step.live) fail(i, 'live', `the client DOM differs from the Go model\nclient: ${live}\ngo:     ${step.live}`);
        const normalized = canonical(document.documentElement, true);
        if (normalized !== step.expected) fail(i, 'expected', `the client DOM differs from the fresh render\nclient: ${normalized}\nfresh:  ${step.expected}`);
        for (const id of step.keep) {
            const was = before.get(id);
            if (!was) fail(i, 'keep', `id ${id} did not resolve before the step`);
            else if (applier.node(id) !== was) fail(i, 'keep', `id ${id} resolves to a different node after the step`);
        }
        const form = checkDirty(applier, pending);
        if (form) fail(i, 'form', form);
        const handlers = checkHandlers(events);
        if (handlers) fail(i, 'handlers', handlers);
        const index = checkIndex(applier, step.reset);
        if (index) fail(i, 'index', index);
        if (failures.length > 0) return failures;
    }
    applier.destroy();
    return failures;
}

export function runMorphCase(scenario: string, c: Case): Failure[] {
    const failures: Failure[] = [];
    const fail = (step: number, check: string, detail: string) => failures.push({ scenario, case: c.name, step, check, detail });
    const { applier } = mount(c.ssr);
    for (let i = 0; i < c.steps.length; i++) {
        const step = c.steps[i];
        const before = scan(document.documentElement);
        try {
            applier.morph(step.reset);
        } catch (e) {
            fail(i, 'morph', message(e));
            return failures;
        }
        const live = canonical(document.documentElement, false);
        if (live !== step.live) fail(i, 'morph-live', `the morphed DOM differs from the reset document\nclient: ${live}\ngo:     ${step.live}`);
        for (const id of step.keep) {
            const was = before.get(id);
            if (was?.kind === 'el' && applier.node(id) !== was.node) fail(i, 'morph-identity', `element ${id} was replaced by the morph`);
        }
        const index = checkIndex(applier, step.reset);
        if (index) fail(i, 'morph-index', index);
        if (failures.length > 0) return failures;
    }
    applier.destroy();
    return failures;
}

export function runFile(file: FixtureFile): { failures: Failure[]; cases: number; steps: number; edits: number } {
    const failures: Failure[] = [];
    let steps = 0;
    let edits = 0;
    for (const c of file.cases) {
        failures.push(...runEditCase(file.scenario, c));
        failures.push(...runMorphCase(file.scenario, c));
        steps += c.steps.length;
        for (const s of c.steps) edits += s.edits.length;
    }
    return { failures, cases: file.cases.length, steps, edits };
}

export function describeFailures(failures: Failure[], limit = 5): string {
    return failures
        .slice(0, limit)
        .map((f) => `${f.scenario}/${f.case} step ${f.step} [${f.check}]: ${f.detail}`)
        .join('\n\n');
}

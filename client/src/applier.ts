import { domEvent, readFields } from './fields';
import { limit } from './rate';
import morphdom from 'morphdom';
import { Edit, ScriptMeta } from './protocol';
import { ATTR_HANDLERS, ATTR_ID, ATTR_RAW, ATTR_SCRIPT, ATTR_SELECTION, HandlerSpec, MARKER, ScriptSpec, SIGNATURES } from './wire';
import { Logger } from './logger';

export type EditFailure = 'unresolved' | 'apply-error';

export class EditError extends Error {
    constructor(readonly reason: EditFailure, readonly edit: unknown, detail: string, options?: { cause?: unknown }) {
        super(`${reason}:${Array.isArray(edit) ? String(edit[0]) : typeof edit} ${detail}`, options);
        this.name = 'EditError';
    }
}

export interface ApplierCallbacks {
    onEvent: (id: number, event: string, data: Record<string, unknown>) => void;
    onScript: (meta: ScriptMeta, el: Element) => void;
    onScriptCleanup: (scriptId: string) => void;
}

const MAX_ID = 4294967295;
const SHOW_ELEMENT_AND_COMMENT = 129;
const SVG_NS = 'http://www.w3.org/2000/svg';
const MATH_NS = 'http://www.w3.org/1998/Math/MathML';
const FOLLOWING = 4;

function isUint(value: unknown, min: number): boolean {
    return typeof value === 'number' && Number.isInteger(value) && value >= min && value <= MAX_ID;
}

export class Applier {
    private readonly nodes = new Map<number, Node>();
    private readonly ends = new Map<number, Comment>();
    private readonly bound = new WeakSet<Element>();
    private readonly synth = new WeakMap<Element, string>();
    private readonly scripts = new Map<Element, string>();
    private pending: Element[] = [];

    constructor(private readonly root: Element, private readonly callbacks: ApplierCallbacks) {}

    index(): void {
        this.nodes.clear();
        this.ends.clear();
        this.scan(this.root, false);
        this.flush();
    }

    node(id: number): Node | undefined {
        return this.nodes.get(id);
    }

    element(id: number): Element | undefined {
        const node = this.nodes.get(id);
        return node?.nodeType === Node.ELEMENT_NODE ? (node as Element) : undefined;
    }

    destroy(): void {
        for (const scriptId of this.scripts.values()) {
            this.callbacks.onScriptCleanup(scriptId);
        }
        this.scripts.clear();
    }

    apply(edits: Edit[]): void {
        for (const edit of edits) {
            try {
                this.validate(edit);
                this.run(edit);
            } catch (e) {
                this.pending = [];
                if (e instanceof EditError) throw e;
                throw new EditError('apply-error', edit, e instanceof Error ? e.name : typeof e, { cause: e });
            }
        }
    }

    morph(html: string): void {
        const root = this.root;
        const next = new DOMParser().parseFromString(html, 'text/html').documentElement;
        const scrollX = window.scrollX;
        const scrollY = window.scrollY;

        const preservedValues = new Map<Element, { value: string; start: number | null; end: number | null }>();
        const priorSelects = new Map<HTMLSelectElement, { value: string; explicit: boolean }>();

        morphdom(root, next, {
            getNodeKey: (node) => (node.nodeType === Node.ELEMENT_NODE ? ((node as Element).getAttribute(ATTR_ID) ?? undefined) : undefined),
            onBeforeElUpdated: (from, to) => {
                if (from instanceof HTMLInputElement && to instanceof HTMLInputElement) {
                    if (!to.hasAttribute('value') && from.value !== from.defaultValue && !['checkbox', 'radio', 'file'].includes(from.type)) {
                        let start: number | null = null;
                        let end: number | null = null;
                        try {
                            start = from.selectionStart;
                            end = from.selectionEnd;
                        } catch {
                        }
                        preservedValues.set(from, { value: from.value, start, end });
                    }
                } else if (from instanceof HTMLTextAreaElement && to instanceof HTMLTextAreaElement) {
                    if (to.value === '' && from.value !== from.defaultValue) {
                        preservedValues.set(from, { value: from.value, start: from.selectionStart, end: from.selectionEnd });
                    }
                } else if (from instanceof HTMLSelectElement && !from.multiple) {
                    priorSelects.set(from, { value: from.value, explicit: to.querySelector('option[selected]') !== null });
                }
                return true;
            },
        });

        for (const [el, saved] of preservedValues) {
            const field = el as HTMLInputElement | HTMLTextAreaElement;
            if (field.value !== saved.value) field.value = saved.value;
            if (saved.start !== null && saved.end !== null && document.activeElement === field) {
                try {
                    field.setSelectionRange(saved.start, saved.end);
                } catch {
                }
            }
        }
        for (const [select, prior] of priorSelects) {
            this.settleSelect(select, prior.value, prior.explicit);
        }
        for (const select of Array.from(root.querySelectorAll('select'))) {
            if (!priorSelects.has(select)) {
                this.settleSelect(select, null, Array.from(select.options).some((o) => o.hasAttribute('selected')));
            }
        }

        for (const [el, scriptId] of this.scripts) {
            if (!el.isConnected) {
                this.scripts.delete(el);
                this.callbacks.onScriptCleanup(scriptId);
            }
        }

        this.index();

        if (window.scrollX !== scrollX || window.scrollY !== scrollY) {
            window.scrollTo(scrollX, scrollY);
        }
    }

    private settleSelect(select: HTMLSelectElement, prior: string | null, explicit: boolean): void {
        if (select.multiple || explicit) return;
        const options = Array.from(select.options);
        for (const option of options) {
            option.removeAttribute('selected');
        }
        const chosen = prior === null ? undefined : options.find((o) => o.value === prior);
        if (chosen) {
            select.selectedIndex = chosen.index;
        } else if (options.length > 0) {
            select.selectedIndex = 0;
        }
    }

    private validate(edit: unknown): void {
        if (!Array.isArray(edit) || typeof edit[0] !== 'string') {
            throw new EditError('apply-error', edit, 'edit is not a tuple');
        }
        const sig = Object.prototype.hasOwnProperty.call(SIGNATURES, edit[0]) ? (SIGNATURES as Record<string, string>)[edit[0]] : undefined;
        if (sig === undefined) {
            throw new EditError('apply-error', edit, 'unknown op');
        }
        if (edit.length !== sig.length + 1) {
            throw new EditError('apply-error', edit, `expected ${sig.length} arguments`);
        }
        for (let i = 0; i < sig.length; i++) {
            const arg = edit[i + 1];
            const ok = sig[i] === 's' ? typeof arg === 'string' : isUint(arg, sig[i] === 'i' ? 1 : 0);
            if (!ok) {
                throw new EditError('apply-error', edit, `argument ${i + 1} is invalid`);
            }
        }
    }

    private run(edit: Edit): void {
        switch (edit[0]) {
            case 't': {
                const node = this.require(edit[1], edit);
                if (node.nodeType === Node.TEXT_NODE) {
                    node.nodeValue = edit[2];
                } else if (node.nodeType === Node.ELEMENT_NODE) {
                    node.textContent = edit[2];
                } else {
                    throw new EditError('unresolved', edit, 'target is not a text slot or element');
                }
                break;
            }
            case 'a': {
                const el = this.requireElement(edit[1], edit);
                const [, , name, value] = edit;
                el.setAttribute(name, value);
                if (name === 'src' || name === 'srcset') this.synth.delete(el);
                if (el instanceof HTMLSelectElement && (name === ATTR_SELECTION || name === 'value')) {
                    const wanted = new Set(name === 'value' ? [value] : (JSON.parse(value) as string[]));
                    for (const option of Array.from(el.options)) {
                        option.selected = wanted.has(option.value);
                        option.toggleAttribute('selected', option.selected);
                    }
                } else if (name === 'value' && /^(input|textarea|select)$/.test(el.localName)) {
                    if (el instanceof HTMLTextAreaElement) el.defaultValue = value;
                    const field = el as HTMLInputElement;
                    if (field.value !== value) field.value = value;
                } else if (name === 'checked' && el instanceof HTMLInputElement) {
                    if (!el.checked) el.checked = true;
                } else if (name === 'selected' && el instanceof HTMLOptionElement) {
                    if (!el.selected) el.selected = true;
                }
                break;
            }
            case 'ra': {
                const el = this.requireElement(edit[1], edit);
                const name = edit[2];
                el.removeAttribute(name);
                if (name === 'src' || name === 'srcset') this.synth.delete(el);
                if (name === 'checked' && el instanceof HTMLInputElement) {
                    if (el.checked) el.checked = false;
                } else if (name === 'selected' && el instanceof HTMLOptionElement) {
                    if (el.selected) el.selected = false;
                }
                break;
            }
            case 'st': {
                const style = (this.requireElement(edit[1], edit) as HTMLElement).style;
                if (!style) {
                    throw new EditError('unresolved', edit, 'target has no style');
                }
                if (edit[3] === '') {
                    style.removeProperty(edit[2]);
                } else {
                    style.setProperty(edit[2], edit[3]);
                }
                break;
            }
            case 'h': {
                const el = this.requireElement(edit[1], edit);
                if (!el.hasAttribute(ATTR_RAW)) {
                    throw new EditError('unresolved', edit, 'target is not a raw host');
                }
                el.innerHTML = edit[2];
                break;
            }
            case 'ins':
                this.insert(edit[1], edit[2], edit[3], edit);
                break;
            case 'mv':
                this.move(edit[1], edit[2], edit[3], edit);
                break;
            case 'rm':
                this.remove(this.unit(edit[1], edit));
                break;
            case 'clr': {
                const [start, end] = this.region(edit[1], edit);
                const doomed: Node[] = [];
                for (let n = start.nextSibling; n && n !== end; n = n.nextSibling) {
                    doomed.push(n);
                }
                this.remove(doomed);
                break;
            }
        }
    }

    private require(id: number, edit: Edit): Node {
        const node = this.nodes.get(id);
        if (!node) {
            throw new EditError('unresolved', edit, `id ${id} does not resolve`);
        }
        return node;
    }

    private requireElement(id: number, edit: Edit): Element {
        const node = this.require(id, edit);
        if (node.nodeType !== Node.ELEMENT_NODE) {
            throw new EditError('unresolved', edit, `id ${id} is not an element`);
        }
        return node as Element;
    }

    private region(id: number, edit: Edit): [Comment, Comment] {
        const start = this.nodes.get(id);
        const end = this.ends.get(id);
        if (!start || start.nodeType !== Node.COMMENT_NODE || !end || start.parentNode !== end.parentNode) {
            throw new EditError('unresolved', edit, `id ${id} is not a region`);
        }
        return [start as Comment, end];
    }

    private unit(id: number, edit: Edit): Node[] {
        const start = this.require(id, edit);
        if (start.nodeType === Node.ELEMENT_NODE) return [start];
        const [, end] = this.region(id, edit);
        const nodes: Node[] = [];
        for (let n: Node | null = start; n; n = n.nextSibling) {
            nodes.push(n);
            if (n === end) return nodes;
        }
        throw new EditError('unresolved', edit, `region ${id} is not closed`);
    }

    private anchor(region: number, ref: number, edit: Edit): Node {
        const [start, end] = this.region(region, edit);
        if (ref === 0) return end;
        const before = this.unit(ref, edit)[0];
        this.requireInside(start, end, before, ref, edit);
        return before;
    }

    private requireInside(start: Comment, end: Comment, node: Node, id: number, edit: Edit): void {
        if (
            node.parentNode !== start.parentNode ||
            !(start.compareDocumentPosition(node) & FOLLOWING) ||
            !(node.compareDocumentPosition(end) & FOLLOWING)
        ) {
            throw new EditError('unresolved', edit, `unit ${id} is not inside its region`);
        }
    }

    private insert(region: number, ref: number, html: string, edit: Edit): void {
        const before = this.anchor(region, ref, edit);
        const parent = before.parentNode!;
        const fragment = this.parse(html, parent);
        this.scan(fragment, true);
        parent.insertBefore(fragment, before);
        this.flush();
    }

    private move(unit: number, region: number, ref: number, edit: Edit): void {
        const nodes = this.unit(unit, edit);
        const [start, end] = this.region(region, edit);
        this.requireInside(start, end, nodes[0], unit, edit);
        const before = this.anchor(region, ref, edit);
        if (nodes.includes(before)) {
            throw new EditError('unresolved', edit, 'unit cannot move before itself');
        }
        if (nodes.length === 1) {
            before.parentNode!.insertBefore(nodes[0], before);
            return;
        }
        const fragment = document.createDocumentFragment();
        fragment.append(...nodes);
        before.parentNode!.insertBefore(fragment, before);
    }

    private remove(nodes: Node[]): void {
        for (const node of nodes) {
            this.walk(node, (n) => this.forget(n));
            node.parentNode?.removeChild(node);
        }
    }

    private parse(html: string, parent: Node): DocumentFragment {
        const ns = (parent as Element).namespaceURI;
        const wrap = ns === SVG_NS && (parent as Element).localName !== 'foreignObject' ? 'svg' : ns === MATH_NS ? 'math' : '';
        const template = document.createElement('template');
        template.innerHTML = wrap ? `<${wrap}>${html}</${wrap}>` : html;
        const fragment = template.content;
        if (wrap) {
            fragment.replaceChildren(...Array.from(fragment.firstChild!.childNodes));
        }
        for (const old of Array.from(fragment.querySelectorAll('script'))) {
            if (old.closest(`[${ATTR_RAW}]`)) continue;
            const live = document.createElementNS(old.namespaceURI, 'script');
            for (const attr of Array.from(old.attributes)) {
                live.setAttribute(attr.name, attr.value);
            }
            live.textContent = old.textContent;
            old.replaceWith(live);
        }
        return fragment;
    }

    private walk(scope: Node, fn: (node: Node) => void): void {
        const walker = document.createTreeWalker(scope, SHOW_ELEMENT_AND_COMMENT);
        const visible = scope.nodeType === Node.ELEMENT_NODE || scope.nodeType === Node.COMMENT_NODE;
        let n: Node | null = visible ? scope : walker.nextNode();
        while (n) {
            fn(n);
            n = walker.nextNode();
        }
    }

    private scan(scope: Node, strict: boolean): void {
        this.walk(scope, (n) => this.visit(n, strict));
    }

    private fault(detail: string, strict: boolean): void {
        if (strict) {
            throw new EditError('apply-error', 'index', detail);
        }
        Logger.warn('Applier', detail);
    }

    private register(id: number, node: Node, strict: boolean): void {
        if (this.nodes.has(id)) {
            this.fault(`duplicate id ${id}`, strict);
            return;
        }
        this.nodes.set(id, node);
    }

    private visit(n: Node, strict: boolean): void {
        if (n.nodeType === Node.ELEMENT_NODE) {
            const el = n as Element;
            const pi = el.getAttribute(ATTR_ID);
            if (pi !== null) {
                if (/^[1-9]\d*$/.test(pi) && isUint(+pi, 1)) {
                    this.register(+pi, el, strict);
                } else {
                    this.fault(`invalid data-pi ${pi}`, strict);
                }
            }
            if (el.hasAttribute(ATTR_HANDLERS)) this.bind(el, strict);
            if (el.hasAttribute(ATTR_SCRIPT) && !this.scripts.has(el)) this.pending.push(el);
            return;
        }
        const match = MARKER.exec(n.nodeValue ?? '');
        if (!match) return;
        const id = +match[2];
        if (match[1] === 't') {
            let slot = n.nextSibling;
            if (!slot || slot.nodeType !== Node.TEXT_NODE) {
                slot = document.createTextNode('');
                n.parentNode!.insertBefore(slot, n.nextSibling);
            }
            this.register(id, slot, strict);
        } else if (match[1] === 'r') {
            this.register(id, n, strict);
        } else if (this.ends.has(id)) {
            this.fault(`duplicate region end ${id}`, strict);
        } else {
            this.ends.set(id, n as Comment);
        }
    }

    private forget(n: Node): void {
        if (n.nodeType === Node.ELEMENT_NODE) {
            const el = n as Element;
            const pi = el.getAttribute(ATTR_ID);
            if (pi !== null && this.nodes.get(+pi) === el) this.nodes.delete(+pi);
            const scriptId = this.scripts.get(el);
            if (scriptId !== undefined) {
                this.scripts.delete(el);
                this.callbacks.onScriptCleanup(scriptId);
            }
            return;
        }
        const match = MARKER.exec(n.nodeValue ?? '');
        if (!match) return;
        const id = +match[2];
        if (match[1] === '/r') {
            this.ends.delete(id);
        } else if (match[1] === 't' || this.nodes.get(id) === n) {
            this.nodes.delete(id);
        }
    }

    private flush(): void {
        const queue = this.pending;
        this.pending = [];
        for (const el of queue) {
            if (this.scripts.has(el)) continue;
            let spec: ScriptSpec;
            try {
                spec = JSON.parse(el.getAttribute(ATTR_SCRIPT)!);
                if (typeof spec.i !== 'string' || typeof spec.s !== 'string') throw new TypeError('invalid script spec');
            } catch (e) {
                Logger.warn('Applier', 'invalid data-pscript', String(e));
                continue;
            }
            this.scripts.set(el, spec.i);
            this.callbacks.onScript({ scriptId: spec.i, script: spec.s }, el);
        }
    }

    private bind(el: Element, strict: boolean): void {
        if (this.bound.has(el)) return;
        const id = Number(el.getAttribute(ATTR_ID));
        let specs: HandlerSpec[];
        try {
            specs = JSON.parse(el.getAttribute(ATTR_HANDLERS)!);
            if (!isUint(id, 1) || !Array.isArray(specs) || specs.some((s) => typeof s?.e !== 'string')) {
                throw new TypeError('invalid handler spec');
            }
        } catch (e) {
            this.fault(`invalid data-pon: ${String(e)}`, strict);
            return;
        }
        this.bound.add(el);
        for (const spec of specs) {
            this.listen(el, id, spec);
        }
    }

    private listen(el: Element, id: number, spec: HandlerSpec): void {
        const send = limit(spec.db, spec.th, (data: Record<string, unknown>) => this.callbacks.onEvent(id, spec.e, data));
        const once = new AbortController();
        el.addEventListener(
            domEvent(el, spec.e),
            (e: Event) => {
                const marked = this.synth.get(el);
                if (marked !== undefined && marked === (el as HTMLImageElement).currentSrc) {
                    this.synth.delete(el);
                    return;
                }
                if (spec.k && !spec.k.includes((e as KeyboardEvent).key)) return;
                if (spec.o) once.abort();
                if (spec.pv && e.cancelable) e.preventDefault();
                if (spec.sp) e.stopPropagation();
                if (spec.cp) (el as any).setPointerCapture((e as PointerEvent).pointerId);
                send(spec.p ? readFields(e, spec.p) : {});
            },
            { capture: !!spec.c, passive: !!spec.ps, signal: once.signal },
        );
        if (spec.e === 'load' && el instanceof HTMLImageElement && el.complete && el.naturalWidth) {
            queueMicrotask(() => {
                el.dispatchEvent(new Event('load'));
                this.synth.set(el, (el as HTMLImageElement).currentSrc);
            });
        }
    }
}

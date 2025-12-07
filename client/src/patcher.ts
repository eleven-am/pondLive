import { Patch, HandlerMeta, ScriptMeta } from './protocol';

export interface StructuredNode {
    tag?: string;
    text?: string;
    comment?: string;
    attrs?: Record<string, string[]>;
    style?: Record<string, string>;
    children?: StructuredNode[];
    handlers?: HandlerMeta[];
    script?: ScriptMeta;
    refId?: string;
    unsafeHTML?: string;
    key?: string;
}

export type EventCallback = (handlerId: string, data: Record<string, unknown>) => void;
export type RefCallback = (refId: string, el: Element) => void;
export type RefDeleteCallback = (refId: string) => void;
export type ScriptCallback = (meta: ScriptMeta, el: Element) => void;
export type ScriptCleanupCallback = (scriptId: string) => void;

export interface PatcherCallbacks {
    onEvent: EventCallback;
    onRef: RefCallback;
    onRefDelete: RefDeleteCallback;
    onScript: ScriptCallback;
    onScriptCleanup: ScriptCleanupCallback;
}

interface HandlerState {
    listener: (e: Event) => void;
    cleanup?: () => void;
}

export class Patcher {
    private readonly root: Node;
    private callbacks: PatcherCallbacks;
    private handlerStore = new WeakMap<Element, Map<string, HandlerState>>();
    private scriptStore = new WeakMap<Element, string>();
    private keyedElements = new Map<string, Element>();

    constructor(root: Node, callbacks: PatcherCallbacks) {
        this.root = root;
        this.callbacks = callbacks;
    }

    apply(patches: Patch[]): void {
        const sorted = [...patches].sort((a, b) => a.seq - b.seq);
        for (const patch of sorted) {
            this.applyPatch(patch);
        }
    }

    private applyPatch(patch: Patch): void {
        const node = this.resolvePath(patch.path);
        if (!node) {
            return;
        }

        switch (patch.op) {
            case 'setText':
                this.setText(node, patch.value as string);
                break;
            case 'setComment':
                this.setComment(node, patch.value as string);
                break;
            case 'setAttr':
                this.setAttr(node as Element, patch.value as Record<string, string[]>);
                break;
            case 'delAttr':
                this.delAttr(node as Element, patch.name!);
                break;
            case 'setStyle':
                this.setStyle(node as HTMLElement, patch.value as Record<string, string>);
                break;
            case 'delStyle':
                this.delStyle(node as HTMLElement, patch.name!);
                break;
            case 'setStyleDecl':
                this.setStyleDecl(node as HTMLStyleElement, patch.selector!, patch.name!, patch.value as string);
                break;
            case 'delStyleDecl':
                this.delStyleDecl(node as HTMLStyleElement, patch.selector!, patch.name!);
                break;
            case 'setHandlers':
                this.setHandlers(node as Element, patch.value as HandlerMeta[]);
                break;
            case 'setScript':
                this.setScript(node as Element, patch.value as ScriptMeta);
                break;
            case 'delScript':
                this.delScript(node as Element);
                break;
            case 'setRef':
                this.callbacks.onRef(patch.value as string, node as Element);
                break;
            case 'delRef':
                this.callbacks.onRefDelete(patch.value as string);
                break;
            case 'replaceNode':
                this.replaceNode(node, patch.value as StructuredNode);
                break;
            case 'addChild':
                this.addChild(node, patch.index!, patch.value as StructuredNode, patch.path);
                break;
            case 'delChild':
                this.delChild(node, patch.index!);
                break;
            case 'moveChild':
                this.moveChild(node, patch.value as { fromIndex: number; newIdx: number; key?: string }, patch.path);
                break;
        }
    }

    private resolvePath(path: number[] | null): Node | null {
        let node: Node | null = this.root;
        if (path) {
            for (const index of path) {
                if (!node) return null;
                node = node.childNodes[index] ?? null;
            }
        }
        return node;
    }

    private setText(node: Node, text: string): void {
        node.textContent = text;
    }

    private setComment(node: Node, text: string): void {
        node.textContent = text;
    }

    private setAttr(el: Element, attrs: Record<string, string[]>): void {
        for (const [name, values] of Object.entries(attrs)) {
            if (name === 'class') {
                el.className = values.join(' ');
            } else if (name === 'value' && el instanceof HTMLInputElement) {
                el.value = values[0] ?? '';
            } else if (name === 'checked' && el instanceof HTMLInputElement) {
                el.checked = values.length > 0 && values[0] !== 'false';
            } else if (name === 'selected' && el instanceof HTMLOptionElement) {
                el.selected = values.length > 0 && values[0] !== 'false';
            } else if (values.length === 0) {
                el.setAttribute(name, '');
            } else {
                el.setAttribute(name, values.join(' '));
            }
        }
    }

    private delAttr(el: Element, name: string): void {
        if (name === 'value' && el instanceof HTMLInputElement) {
            el.value = '';
        } else if (name === 'checked' && el instanceof HTMLInputElement) {
            el.checked = false;
        } else if (name === 'selected' && el instanceof HTMLOptionElement) {
            el.selected = false;
        } else {
            el.removeAttribute(name);
        }
    }

    private setStyle(el: HTMLElement, styles: Record<string, string>): void {
        for (const [prop, value] of Object.entries(styles)) {
            el.style.setProperty(prop, value);
        }
    }

    private delStyle(el: HTMLElement, prop: string): void {
        el.style.removeProperty(prop);
    }

    private setStyleDecl(styleEl: HTMLStyleElement, selector: string, prop: string, value: string): void {
        const sheet = styleEl.sheet;
        if (!sheet) return;

        const rule = this.findOrCreateRule(sheet, selector);
        if (rule) {
            rule.style.setProperty(prop, value);
        }
    }

    private delStyleDecl(styleEl: HTMLStyleElement, selector: string, prop: string): void {
        const sheet = styleEl.sheet;
        if (!sheet) return;

        const rule = this.findRule(sheet, selector);
        if (rule) {
            rule.style.removeProperty(prop);
        }
    }

    private findRule(sheet: CSSStyleSheet, selector: string): CSSStyleRule | null {
        for (let i = 0; i < sheet.cssRules.length; i++) {
            const rule = sheet.cssRules[i];
            if (rule instanceof CSSStyleRule && rule.selectorText === selector) {
                return rule;
            }
        }
        return null;
    }

    private findOrCreateRule(sheet: CSSStyleSheet, selector: string): CSSStyleRule | null {
        let rule = this.findRule(sheet, selector);
        if (!rule) {
            const index = sheet.insertRule(`${selector} {}`, sheet.cssRules.length);
            rule = sheet.cssRules[index] as CSSStyleRule;
        }
        return rule;
    }

    private setHandlers(el: Element, handlers: HandlerMeta[]): void {
        const oldHandlers = this.handlerStore.get(el);
        if (oldHandlers) {
            oldHandlers.forEach((state) => {
                if (state.cleanup) state.cleanup();
            });
        }

        const newHandlers = new Map<string, HandlerState>();

        for (const meta of handlers) {
            const state = this.createHandler(el, meta);
            newHandlers.set(meta.event, state);
        }

        this.handlerStore.set(el, newHandlers);
    }

    private createHandler(el: Element, meta: HandlerMeta): HandlerState {
        let timeoutId: ReturnType<typeof setTimeout> | null = null;
        let lastCall = 0;

        const invoke = (e: Event) => {
            if (meta.prevent && e.cancelable) {
                e.preventDefault();
            }
            if (meta.stop) {
                e.stopPropagation();
            }

            const data = this.extractEventData(e, meta.props ?? []);
            this.callbacks.onEvent(meta.handler, data);
        };

        let handler: (e: Event) => void;

        if (meta.debounce && meta.debounce > 0) {
            handler = (e: Event) => {
                if (meta.prevent && e.cancelable) e.preventDefault();
                if (meta.stop) e.stopPropagation();

                if (timeoutId) clearTimeout(timeoutId);
                timeoutId = setTimeout(() => {
                    const data = this.extractEventData(e, meta.props ?? []);
                    this.callbacks.onEvent(meta.handler, data);
                }, meta.debounce);
            };
        } else if (meta.throttle && meta.throttle > 0) {
            handler = (e: Event) => {
                if (meta.prevent && e.cancelable) e.preventDefault();
                if (meta.stop) e.stopPropagation();

                const now = Date.now();
                if (now - lastCall >= meta.throttle!) {
                    lastCall = now;
                    const data = this.extractEventData(e, meta.props ?? []);
                    this.callbacks.onEvent(meta.handler, data);
                }
            };
        } else {
            handler = invoke;
        }

        const options: AddEventListenerOptions = {};
        if (meta.passive) options.passive = true;
        if (meta.once) options.once = true;
        if (meta.capture) options.capture = true;

        el.addEventListener(meta.event, handler, options);

        return {
            listener: handler,
            cleanup: () => {
                el.removeEventListener(meta.event, handler, options);
                if (timeoutId) clearTimeout(timeoutId);
            },
        };
    }

    private extractEventData(e: Event, props: string[]): Record<string, unknown> {
        const result: Record<string, unknown> = {};
        for (const prop of props) {
            const value = this.resolveProp(e, prop);
            if (value !== undefined) {
                result[prop] = value;
            }
        }
        return result;
    }

    private resolveProp(e: Event, path: string): unknown {
        const segments = path.split('.').map(s => s.trim()).filter(Boolean);
        if (segments.length === 0) return undefined;

        const root = segments.shift()!;
        let current: unknown;

        switch (root) {
            case 'event':
                current = e;
                break;
            case 'target':
                current = e.target;
                break;
            case 'currentTarget':
                current = e.currentTarget;
                break;
            default:
                current = (e as unknown as Record<string, unknown>)[root];
        }

        for (const segment of segments) {
            if (current == null) return undefined;
            try {
                current = (current as Record<string, unknown>)[segment];
            } catch {
                return undefined;
            }
        }

        return this.serializeValue(current);
    }

    private serializeValue(value: unknown): unknown {
        if (value === null || value === undefined) return null;

        const type = typeof value;
        if (type === 'string' || type === 'number' || type === 'boolean') return value;

        if (Array.isArray(value)) {
            const mapped = value.map(v => this.serializeValue(v)).filter(v => v !== undefined);
            return mapped.length > 0 ? mapped : null;
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

    private setScript(el: Element, meta: ScriptMeta): void {
        this.delScript(el);
        this.scriptStore.set(el, meta.scriptId);
        this.callbacks.onScript(meta, el);
    }

    private delScript(el: Element): void {
        const scriptId = this.scriptStore.get(el);
        if (scriptId) {
            this.scriptStore.delete(el);
            this.callbacks.onScriptCleanup(scriptId);
        }
    }

    private replaceNode(oldNode: Node, newNodeData: StructuredNode): void {
        const newNode = this.createNode(newNodeData);
        if (newNode && oldNode.parentNode) {
            this.cleanupTree(oldNode);
            oldNode.parentNode.replaceChild(newNode, oldNode);
        }
    }

    private addChild(parent: Node, index: number, nodeData: StructuredNode, parentPath: number[]): void {
        const newNode = this.createNode(nodeData);
        if (!newNode) return;

        if (nodeData.key && newNode instanceof Element) {
            const keyId = `${parentPath.join(',')}-${nodeData.key}`;
            this.keyedElements.set(keyId, newNode);
        }

        const refChild = parent.childNodes[index] ?? null;
        parent.insertBefore(newNode, refChild);
    }

    private delChild(parent: Node, index: number): void {
        const child = parent.childNodes[index];
        if (child) {
            this.cleanupTree(child);
            parent.removeChild(child);
        }
    }

    private moveChild(parent: Node, move: { fromIndex: number; newIdx: number; key?: string }, parentPath: number[]): void {
        let child: ChildNode | null = null;

        if (move.key) {
            const keyId = `${parentPath.join(',')}-${move.key}`;
            child = this.keyedElements.get(keyId) ?? null;
        }

        if (!child) {
            child = parent.childNodes[move.fromIndex] ?? null;
        }

        if (!child) return;

        parent.removeChild(child);
        const refChild = parent.childNodes[move.newIdx] ?? null;
        parent.insertBefore(child, refChild);
    }

    private cleanupTree(node: Node): void {
        if (node.nodeType === Node.ELEMENT_NODE) {
            const el = node as Element;

            const handlers = this.handlerStore.get(el);
            if (handlers) {
                handlers.forEach((state) => {
                    if (state.cleanup) state.cleanup();
                });
                this.handlerStore.delete(el);
            }

            const scriptId = this.scriptStore.get(el);
            if (scriptId) {
                this.scriptStore.delete(el);
                this.callbacks.onScriptCleanup(scriptId);
            }

            for (let i = 0; i < node.childNodes.length; i++) {
                this.cleanupTree(node.childNodes[i]);
            }
        }
    }

    private createNode(data: StructuredNode): Node | null {
        if (data.text !== undefined) {
            return document.createTextNode(data.text);
        }

        if (data.comment !== undefined) {
            return document.createComment(data.comment);
        }

        if (!data.tag) return null;

        const el = document.createElement(data.tag);

        if (data.attrs) {
            this.setAttr(el, data.attrs);
        }

        if (data.style) {
            this.setStyle(el as HTMLElement, data.style);
        }

        if (data.handlers && data.handlers.length > 0) {
            this.setHandlers(el, data.handlers);
        }

        if (data.script) {
            this.setScript(el, data.script);
        }

        if (data.refId) {
            this.callbacks.onRef(data.refId, el);
        }

        if (data.unsafeHTML) {
            el.innerHTML = data.unsafeHTML;
        } else if (data.children) {
            for (const child of data.children) {
                const childNode = this.createNode(child);
                if (childNode) {
                    el.appendChild(childNode);
                }
            }
        }

        return el;
    }
}

import {HandlerMeta, Patch, PatcherCallbacks, RouterMeta, ScriptMeta, StructuredNode, UploadMeta} from './types';
import {Logger} from './logger';

export class Patcher {
    private readonly root: Node;
    private callbacks: PatcherCallbacks;
    private handlerStore = new WeakMap<Element, Map<string, (e: Event) => void>>();
    private routerStore = new WeakMap<Element, (e: Event) => void>();
    private uploadStore = new WeakMap<HTMLInputElement, (e: Event) => void>();
    private scriptStore = new WeakMap<Element, string>();

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
            Logger.warn('Patcher', 'Could not resolve path', patch.path);
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
            case 'setRouter':
                this.setRouter(node as Element, patch.value as RouterMeta);
                break;
            case 'delRouter':
                this.delRouter(node as Element);
                break;
            case 'setUpload':
                this.setUpload(node as HTMLInputElement, patch.value as UploadMeta);
                break;
            case 'delUpload':
                this.delUpload(node as HTMLInputElement);
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
                this.addChild(node, patch.index!, patch.value as StructuredNode);
                break;
            case 'delChild':
                this.delChild(node, patch.index!);
                break;
            case 'moveChild':
                this.moveChild(node, patch.value as { fromIndex: number; newIdx: number });
                break;
        }
    }

    private resolvePath(path: number[]): Node | null {
        let node: Node | null = this.root;
        for (const index of path) {
            if (!node) return null;
            node = node.childNodes[index] ?? null;
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
        Logger.info('Patcher', 'setAttr', el, attrs);
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
        Logger.info('Patcher', 'delAttr', el, name);
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
        Logger.info('Patcher', 'setStyle', el, styles);
        for (const [prop, value] of Object.entries(styles)) {
            el.style.setProperty(prop, value);
        }
    }

    private delStyle(el: HTMLElement, prop: string): void {
        Logger.info('Patcher', 'delStyle', el, prop);
        el.style.removeProperty(prop);
    }

    private setStyleDecl(styleEl: HTMLStyleElement, selector: string, prop: string, value: string): void {
        Logger.info('Patcher', 'setStyleDecl', styleEl, selector, prop, value);
        const sheet = styleEl.sheet;
        if (!sheet) return;

        const rule = this.findOrCreateRule(sheet, selector);
        if (rule) {
            rule.style.setProperty(prop, value);
        }
    }

    private delStyleDecl(styleEl: HTMLStyleElement, selector: string, prop: string): void {
        Logger.info('Patcher', 'delStyleDecl', styleEl, selector, prop);
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
        Logger.info('Patcher', 'setHandlers', el, handlers);
        const oldHandlers = this.handlerStore.get(el);
        if (oldHandlers) {
            for (const [event, listener] of oldHandlers) {
                el.removeEventListener(event, listener);
            }
        }

        const newHandlers = new Map<string, (e: Event) => void>();
        for (const meta of handlers) {
            const listen = meta.listen ?? [];
            const listener = (e: Event) => {
                if (!listen.includes('allowDefault') && e.cancelable) {
                    e.preventDefault();
                }

                const data = this.extractEventData(e, meta.props ?? []);
                this.callbacks.onEvent(meta.event, meta.handler, data);

                if (!listen.includes('bubble')) {
                    e.stopPropagation();
                }
            };
            el.addEventListener(meta.event, listener);
            newHandlers.set(meta.event, listener);
        }
        this.handlerStore.set(el, newHandlers);
    }

    private extractEventData(e: Event, props: string[], el?: Element): Record<string, unknown> {
        Logger.info('Patcher', 'extractEventData', e, props, el);
        return Object.fromEntries(props.map(prop => [prop, this.resolveProp(e, prop, el)]).filter(([_, value]) => value !== undefined));
    }

    private resolveProp(e: Event, path: string, el?: Element): unknown {
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
            case 'element':
            case 'ref':
                current = el ?? (e.currentTarget instanceof Element ? e.currentTarget : null);
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

    private setRouter(el: Element, meta: RouterMeta): void {
        Logger.info('Patcher', 'setRouter', el, meta);
        this.delRouter(el);

        const listener = (e: Event) => {
            e.preventDefault();
            this.callbacks.onRouter(meta);
        };
        el.addEventListener('click', listener);
        this.routerStore.set(el, listener);
    }

    private delRouter(el: Element): void {
        Logger.info('Patcher', 'delRouter', el);
        const listener = this.routerStore.get(el);
        if (listener) {
            el.removeEventListener('click', listener);
            this.routerStore.delete(el);
        }
    }

    private setUpload(el: HTMLInputElement, meta: UploadMeta): void {
        Logger.info('Patcher', 'setUpload', el, meta);
        this.delUpload(el);

        if (meta.multiple) {
            el.multiple = true;
        }
        if (meta.accept && meta.accept.length > 0) {
            el.accept = meta.accept.join(',');
        }

        const listener = () => {
            if (el.files && el.files.length > 0) {
                this.callbacks.onUpload(meta, el.files);
            }
        };
        el.addEventListener('change', listener);
        this.uploadStore.set(el, listener);
    }

    private delUpload(el: HTMLInputElement): void {
        Logger.info('Patcher', 'delUpload', el);
        const listener = this.uploadStore.get(el);
        if (listener) {
            el.removeEventListener('change', listener);
            this.uploadStore.delete(el);
        }
        el.multiple = false;
        el.accept = '';
    }

    private setScript(el: Element, meta: ScriptMeta): void {
        Logger.info('Patcher', 'setScript', el, meta);
        this.delScript(el);

        this.scriptStore.set(el, meta.scriptId);
        this.callbacks.onScript(meta, el);
    }

    private delScript(el: Element): void {
        Logger.info('Patcher', 'delScript', el);
        const scriptId = this.scriptStore.get(el);
        if (scriptId) {
            this.scriptStore.delete(el);
            this.callbacks.onScriptCleanup(scriptId);
        }
    }

    private replaceNode(oldNode: Node, newNodeData: StructuredNode): void {
        Logger.info('Patcher', 'replaceNode', oldNode, newNodeData);
        const newNode = this.createNode(newNodeData);
        if (newNode && oldNode.parentNode) {
            this.cleanupScriptsInTree(oldNode);
            oldNode.parentNode.replaceChild(newNode, oldNode);
        }
    }

    private addChild(parent: Node, index: number, nodeData: StructuredNode): void {
        Logger.info('Patcher', 'addChild', parent, index, nodeData);
        const newNode = this.createNode(nodeData);
        if (!newNode) return;

        const refChild = parent.childNodes[index] ?? null;
        parent.insertBefore(newNode, refChild);
    }

    private delChild(parent: Node, index: number): void {
        Logger.info('Patcher', 'delChild', parent, index);
        const child = parent.childNodes[index];
        if (child) {
            this.cleanupScriptsInTree(child);
            parent.removeChild(child);
        }
    }

    private cleanupScriptsInTree(node: Node): void {
        if (node.nodeType === Node.ELEMENT_NODE) {
            const el = node as Element;
            const scriptId = this.scriptStore.get(el);
            if (scriptId) {
                this.scriptStore.delete(el);
                this.callbacks.onScriptCleanup(scriptId);
            }

            for (let i = 0; i < node.childNodes.length; i++) {
                this.cleanupScriptsInTree(node.childNodes[i]);
            }
        }
    }

    private moveChild(parent: Node, move: { fromIndex: number; newIdx: number }): void {
        Logger.info('Patcher', 'moveChild', parent, move);
        const child = parent.childNodes[move.fromIndex];
        if (!child) return;

        parent.removeChild(child);
        const refChild = parent.childNodes[move.newIdx] ?? null;
        parent.insertBefore(child, refChild);
    }

    private createNode(data: StructuredNode): Node | null {
        Logger.info('Patcher', 'createNode', data);
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

        if (data.router) {
            this.setRouter(el, data.router);
        }

        if (data.upload && el instanceof HTMLInputElement) {
            this.setUpload(el, data.upload);
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

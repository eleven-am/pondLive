import { ClientNode, Patch, StructuredNode } from './types';
import { Logger } from './logger';
import { hydrate } from './vdom';
import { Router } from './router';
import { UploadManager } from './uploads';

import { EventManager } from './events';

export class Patcher {
    constructor(
        private root: ClientNode,
        private events: EventManager,
        private router: Router,
        private uploads: UploadManager,
        private refs: Map<string, ClientNode>
    ) { }

    apply(patch: Patch) {
        const target = this.traverse(patch.path);
        if (!target) {
            Logger.warn('Patcher', 'Target not found for path', patch.path);
            return;
        }

        Logger.debug('Patcher', 'Apply', {
            op: patch.op,
            path: patch.path,
            index: patch.index,
            name: patch.name,
            selector: patch.selector,
        });

        switch (patch.op) {
            case 'setText':
                this.setText(target, patch.value);
                break;
            case 'setAttr':
                this.setAttr(target, patch.value);
                break;
            case 'delAttr':
                this.delAttr(target, patch.name!);
                break;
            case 'setStyleDecl':
                this.setStyleDecl(target, patch.selector!, patch.name!, patch.value);
                break;
            case 'delStyleDecl':
                this.delStyleDecl(target, patch.selector!, patch.name!);
                break;
            case 'replaceNode':
                this.replaceNode(target, patch.value, patch.path);
                break;
            case 'addChild':
                this.addChild(target, patch.value, patch.index!);
                break;
            case 'delChild':
                this.delChild(target, patch.index!, (patch.value as any)?.key);
                break;
            case 'moveChild':
                this.moveChild(target, patch.value);
                break;
            case 'setRef':
                this.setRef(target, patch.value);
                break;
            case 'delRef':
                this.delRef(target);
                break;
            case 'setComment':
                this.setComment(target, patch.value);
                break;
            case 'setStyle':
                this.setStyle(target, patch.value);
                break;
            case 'delStyle':
                this.delStyle(target, patch.name!);
                break;
            case 'setHandlers':
                this.setHandlers(target, patch.value);
                break;
            case 'setRouter':
                this.setRouter(target, patch.value);
                break;
            case 'delRouter':
                this.delRouter(target);
                break;
            case 'setUpload':
                this.setUpload(target, patch.value);
                break;
            case 'delUpload':
                this.delUpload(target);
                break;
            case 'setComponent':
                target.componentId = patch.value;
                break;
            default:
                Logger.warn('Patcher', 'Unsupported op', patch.op);
        }
    }

    private traverse(path: number[]): ClientNode | null {
        let current = this.root;
        for (const idx of path) {
            if (!current.children || !current.children[idx]) {
                Logger.warn('Patcher', 'Traverse missing child', { path, failedIndex: idx, currentTag: current.tag, childrenLength: current.children?.length });
                return null;
            }
            current = current.children[idx];
        }
        Logger.debug('Patcher', 'Traverse resolved', {
            path,
            tag: current.tag,
            componentId: current.componentId,
            key: current.key,
            hasChildren: !!current.children?.length,
        });
        return current;
    }

    private setText(node: ClientNode, text: string) {
        if (node.el) {
            node.el.textContent = text;
        }
        node.text = text;
    }

    private setAttr(node: ClientNode, attrs: Record<string, string[]>) {
        if (node.el && node.el instanceof Element) {
            for (const [name, tokens] of Object.entries(attrs)) {
                node.el.setAttribute(name, tokens.join(' '));
            }
        }
        if (!node.attrs) node.attrs = {};
        Object.assign(node.attrs, attrs);
    }

    private delAttr(node: ClientNode, name: string) {
        if (node.el && node.el instanceof Element) {
            node.el.removeAttribute(name);
        }
        if (node.attrs) delete node.attrs[name];
    }

    private replaceNode(oldNode: ClientNode, newJson: StructuredNode, path: number[]) {
        Logger.debug('Patcher', 'replaceNode start', { path });
        const oldDoms = this.collectDomNodes(oldNode);
        Logger.debug('Patcher', 'replaceNode collected DOM nodes', { count: oldDoms.length });

        if (oldDoms.length === 0) {
            Logger.warn('Patcher', 'Cannot replace node with no DOM elements', oldNode);
            return;
        }

        const firstDom = oldDoms[0];
        if (!firstDom.parentNode) {
            Logger.warn('Patcher', 'Cannot replace node without parent', oldNode);
            return;
        }

        const newDom = this.render(newJson);
        firstDom.parentNode.replaceChild(newDom, firstDom);

        for (let i = 1; i < oldDoms.length; i++) {
            const node = oldDoms[i];
            if (node.parentNode) node.parentNode.removeChild(node);
        }

        this.events.detach(oldNode);
        this.router.detach(oldNode);
        this.uploads.unbind(oldNode);
        this.detachRefsRecursively(oldNode);

        const parentPath = path.slice(0, -1);
        const childIdx = path[path.length - 1];
        const parent = this.traverse(parentPath);

        if (parent && parent.children) {
            const newNode = hydrate(newJson, newDom, this.refs);
            parent.children[childIdx] = newNode;
            this.events.attach(newNode);
            this.router.attach(newNode);
        }
    }

    private collectDomNodes(node: ClientNode): Node[] {
        if (node.el) {
            return [node.el];
        }
        const nodes: Node[] = [];
        if (node.children) {
            for (const child of node.children) {
                const childNodes = this.collectDomNodes(child);
                for (const n of childNodes) {
                    nodes.push(n);
                }
            }
        }
        return nodes;
    }

    private render(json: StructuredNode): Node {
        if (json.text !== undefined) {
            return document.createTextNode(json.text);
        }
        if (json.tag) {
            const el = document.createElement(json.tag);
            if (json.attrs) {
                for (const [k, v] of Object.entries(json.attrs)) {
                    el.setAttribute(k, v.join(' '));
                }
            }
            if (json.style && el instanceof HTMLElement) {
                for (const [name, value] of Object.entries(json.style)) {
                    el.style.setProperty(name, value);
                }
            }
            if (json.styles && el instanceof HTMLStyleElement) {
                el.textContent = this.buildStyleContent(json.styles);
            }
            if (json.children) {
                for (const child of json.children) {
                    el.appendChild(this.render(child));
                }
            }
            return el;
        }
        if (json.children && json.children.length > 0) {
            const fragment = document.createDocumentFragment();
            for (const child of json.children) {
                fragment.appendChild(this.render(child));
            }
            return fragment;
        }
        return document.createComment(json.comment || '');
    }

    private addChild(parent: ClientNode, childJson: StructuredNode, index: number) {
        if (!parent.el || !parent.el.childNodes) {
            Logger.warn('Patcher', 'Cannot add child to non-element', parent);
            return;
        }

        const newDom = this.render(childJson);
        const newClientNode = hydrate(childJson, newDom, this.refs);

        if (!parent.children) parent.children = [];
        const safeIndex = Math.max(0, Math.min(index, parent.children.length));

        if (safeIndex >= parent.children.length) {
            parent.el.appendChild(newDom);
            parent.children.push(newClientNode);
        } else {
            let referenceNode: Node | null = null;
            for (let i = safeIndex; i < parent.children.length; i++) {
                if (parent.children[i].el) {
                    referenceNode = parent.children[i].el;
                    break;
                }
            }

            if (referenceNode) {
                parent.el.insertBefore(newDom, referenceNode);
            } else {
                parent.el.appendChild(newDom);
            }
            parent.children.splice(safeIndex, 0, newClientNode);
        }

        this.events.attach(newClientNode);
        this.router.attach(newClientNode);
    }

    private delChild(parent: ClientNode, index: number, key?: string) {
        if (!parent.children || !parent.children[index]) {
            if (key && parent.children) {
                const idxByKey = parent.children.findIndex((c) => c && c.key === key);
                if (idxByKey >= 0) {
                    index = idxByKey;
                }
            }
            if (!parent.children || !parent.children[index]) {
                Logger.warn('Patcher', 'Cannot delete missing child', {
                    index,
                    childrenLength: parent.children?.length,
                    parentTag: parent.tag,
                    parentComponentId: parent.componentId
                });
                return;
            }
        }

        const child = parent.children[index];
        this.removeDomNodes(child);
        parent.children.splice(index, 1);

        this.events.detach(child);
        this.router.detach(child);
        this.uploads.unbind(child);
        this.detachRefsRecursively(child);
    }

    private removeDomNodes(node: ClientNode) {
        const domNodes = this.collectDomNodes(node);
        for (const domNode of domNodes) {
            if (domNode.parentNode) {
                domNode.parentNode.removeChild(domNode);
            }
        }
    }

    private moveChild(parent: ClientNode, value: any) {
        if (!parent.children || parent.children.length === 0) {
            Logger.warn('Patcher', 'Cannot move child in empty parent', { value });
            return;
        }

        let fromIdx = -1;
        const toIdx = Math.max(0, Math.min(value.newIdx, parent.children.length - 1));

        // Prefer resolving by key when available to avoid stale indexes.
        if (value.key) {
            const key = value.key as string;
            const found = parent.children.findIndex((c) => c && c.key === key);
            if (found >= 0) {
                fromIdx = found;
            }
        }

        if (fromIdx < 0 || fromIdx >= parent.children.length) {
            Logger.warn('Patcher', 'Cannot move missing child', { fromIdx, value });
            return;
        }

        const child = parent.children[fromIdx];

        parent.children.splice(fromIdx, 1);
        const insertIdx = Math.max(0, Math.min(toIdx, parent.children.length));
        parent.children.splice(insertIdx, 0, child);

        if (!parent.el || !child.el) return;

        let referenceNode: Node | null = null;
        for (let i = insertIdx + 1; i < parent.children.length; i++) {
            if (parent.children[i].el) {
                referenceNode = parent.children[i].el;
                break;
            }
        }

        if (referenceNode) {
            parent.el.insertBefore(child.el, referenceNode);
        } else {
            parent.el.appendChild(child.el);
        }
    }

    private setRef(node: ClientNode, refId: string) {
        node.refId = refId;
        this.refs.set(refId, node);
    }

    private delRef(node: ClientNode) {
        if (node.refId) {
            this.refs.delete(node.refId);
            delete node.refId;
        }
    }

    private setComment(node: ClientNode, comment: string) {
        if (node.el) {
            node.el.textContent = comment;
        }
        node.comment = comment;
    }

    private setStyle(node: ClientNode, styles: Record<string, string>) {
        if (node.el && node.el instanceof HTMLElement) {
            for (const [name, value] of Object.entries(styles)) {
                node.el.style.setProperty(name, value);
            }
        }
        if (!node.style) node.style = {};
        Object.assign(node.style, styles);
    }

    private setStyleDecl(node: ClientNode, selector: string, name: string, value: string) {
        if (node.el && node.el instanceof HTMLStyleElement && node.el.sheet) {
            const sheet = node.el.sheet;

            for (let i = 0; i < sheet.cssRules.length; i++) {
                const rule = sheet.cssRules[i] as CSSStyleRule;
                if (rule.selectorText === selector) {
                    rule.style.setProperty(name, value);
                    return;
                }
            }

            const idx = sheet.cssRules.length;
            try {
                sheet.insertRule(`${selector} { ${name}: ${value}; }`, idx);
            } catch (e) {
                Logger.warn('Patcher', 'Failed to insert rule', { selector, error: e });
            }
        }
    }

    private delStyleDecl(node: ClientNode, selector: string, name: string) {
        if (node.el && node.el instanceof HTMLStyleElement && node.el.sheet) {
            const sheet = node.el.sheet;
            for (let i = 0; i < sheet.cssRules.length; i++) {
                const rule = sheet.cssRules[i] as CSSStyleRule;
                if (rule.selectorText === selector) {
                    rule.style.removeProperty(name);
                    return;
                }
            }
        }
    }

    private delStyle(node: ClientNode, name: string) {
        if (node.el && node.el instanceof HTMLElement) {
            node.el.style.removeProperty(name);
        }
        if (node.style) delete node.style[name];
    }

    private setHandlers(node: ClientNode, handlers: any) {
        this.events.detach(node);
        node.handlers = handlers;
        this.events.attach(node);
    }

    private setRouter(node: ClientNode, router: any) {
        this.router.detach(node);
        node.router = router;
        this.router.attach(node);
    }

    private delRouter(node: ClientNode) {
        this.router.detach(node);
        delete node.router;
    }

    private setUpload(node: ClientNode, meta: any) {
        this.uploads.bind(node, meta);
    }

    private delUpload(node: ClientNode) {
        this.uploads.unbind(node);
    }

    private detachRefsRecursively(node: ClientNode) {
        this.uploads.unbind(node);
        if (node.refId) {
            this.refs.delete(node.refId);
        }
        if (node.children) {
            for (const child of node.children) {
                this.detachRefsRecursively(child);
            }
        }
    }

    private buildStyleContent(styles: Record<string, Record<string, string>>): string {
        const blocks: string[] = [];
        for (const [selector, props] of Object.entries(styles)) {
            const entries: string[] = [];
            for (const [name, value] of Object.entries(props)) {
                entries.push(`${name}: ${value};`);
            }
            if (entries.length > 0) {
                blocks.push(`${selector} { ${entries.join(' ')} }`);
            }
        }
        return blocks.join('\n');
    }
}

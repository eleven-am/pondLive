import { ClientNode, RouterMeta } from './types';
import { Logger } from './logger';

export class Router {
    private listeners = new WeakMap<Node, EventListener>();

    constructor(private channel: any, private sessionId: string) {
        window.addEventListener('popstate', (e) => this.onPopState(e));
    }

    attach(node: ClientNode) {
        if (!node || !node.router || !node.el) return;

        const el = node.el;
        if (this.listeners.has(el)) return;

        const listener = (e: Event) => {
            e.preventDefault();
            this.navigate(node.router!);
        };

        el.addEventListener('click', listener);
        this.listeners.set(el, listener);
    }

    detach(node: ClientNode) {
        if (!node || !node.el) return;

        const el = node.el;
        const listener = this.listeners.get(el);
        if (listener) {
            el.removeEventListener('click', listener);
            this.listeners.delete(el);
        }
    }

    navigate(meta: RouterMeta) {
        const path = meta.path ?? window.location.pathname;
        
        
        const query = meta.query !== undefined ? meta.query : window.location.search;
        const hash = meta.hash !== undefined ? meta.hash : window.location.hash;

        
        const cleanQuery = query.startsWith('?') ? query.substring(1) : query;
        const url = path + (cleanQuery ? '?' + cleanQuery : '') + (hash ? '#' + hash : '');

        if (meta.replace) {
            window.history.replaceState({}, '', url);
        } else {
            window.history.pushState({}, '', url);
        }

        this.sendNav('nav', path, cleanQuery, hash);
    }

    private onPopState(_e: PopStateEvent) {
        const path = window.location.pathname;
        const query = window.location.search;
        const hash = window.location.hash;

        this.sendNav('pop', path, query, hash);
    }

    private sendNav(type: 'nav' | 'pop', path: string, query: string, hash: string) {
        Logger.debug('Router', `Sending ${type}`, { path, query, hash });

        
        
        
        
        
        const q = query.startsWith('?') ? query.substring(1) : query;

        this.channel.sendMessage(type, {
            sid: this.sessionId,
            path: path,
            q: q,
            hash: hash
        });
    }
}

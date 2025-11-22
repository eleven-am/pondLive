import {RouterMeta, NavCallback} from './types';

export class Router {
    private readonly onNav: NavCallback;

    constructor(onNav: NavCallback) {
        this.onNav = onNav;
        window.addEventListener('popstate', () => this.handlePopState());
    }

    navigate(meta: RouterMeta): void {
        const path = meta.path;
        const query = meta.query !== undefined ? meta.query : window.location.search;
        const hash = meta.hash !== undefined ? meta.hash : window.location.hash;

        const cleanQuery = query.startsWith('?') ? query.substring(1) : query;
        const cleanHash = hash.startsWith('#') ? hash.substring(1) : hash;
        const url = path + (cleanQuery ? '?' + cleanQuery : '') + (cleanHash ? '#' + cleanHash : '');

        if (meta.replace) {
            window.history.replaceState({}, '', url);
        } else {
            window.history.pushState({}, '', url);
        }

        this.onNav('nav', path, cleanQuery, cleanHash);
    }

    private handlePopState(): void {
        const path = window.location.pathname;
        const query = window.location.search;
        const hash = window.location.hash;

        const cleanQuery = query.startsWith('?') ? query.substring(1) : query;
        const cleanHash = hash.startsWith('#') ? hash.substring(1) : hash;
        this.onNav('pop', path, cleanQuery, cleanHash);
    }

    destroy(): void {
        window.removeEventListener('popstate', () => this.handlePopState());
    }
}

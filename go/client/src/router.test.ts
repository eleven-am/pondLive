import { describe, it, expect, beforeEach, vi } from 'vitest';
import { Router } from './router';
import { NavCallback } from './types';

describe('Router', () => {
    let onNav: NavCallback;
    let router: Router;

    beforeEach(() => {
        onNav = vi.fn();
        window.history.replaceState({}, '', '/');
        router = new Router(onNav);
    });

    describe('navigate', () => {
        it('should update URL with pushState', () => {
            router.navigate({ pathValue: '/about' });

            expect(window.location.pathname).toBe('/about');
        });

        it('should call onNav with nav type', () => {
            router.navigate({ pathValue: '/contact' });

            expect(onNav).toHaveBeenCalledWith('nav', '/contact', '', '');
        });

        it('should handle query string', () => {
            router.navigate({ pathValue: '/search', query: 'q=test' });

            expect(window.location.pathname).toBe('/search');
            expect(window.location.search).toBe('?q=test');
            expect(onNav).toHaveBeenCalledWith('nav', '/search', 'q=test', '');
        });

        it('should handle query with leading ?', () => {
            router.navigate({ pathValue: '/search', query: '?q=test' });

            expect(window.location.search).toBe('?q=test');
            expect(onNav).toHaveBeenCalledWith('nav', '/search', 'q=test', '');
        });

        it('should handle hash', () => {
            router.navigate({ pathValue: '/docs', hash: '#section' });

            expect(window.location.pathname).toBe('/docs');
            expect(window.location.hash).toBe('#section');
            expect(onNav).toHaveBeenCalledWith('nav', '/docs', '', 'section');
        });

        it('should handle query and hash together', () => {
            router.navigate({ pathValue: '/page', query: 'foo=bar', hash: '#top' });

            expect(window.location.href).toContain('/page?foo=bar#top');
            expect(onNav).toHaveBeenCalledWith('nav', '/page', 'foo=bar', 'top');
        });

        it('should use replaceState when replace is true', () => {
            const replaceStateSpy = vi.spyOn(window.history, 'replaceState');

            router.navigate({ pathValue: '/replaced', replace: true });

            expect(replaceStateSpy).toHaveBeenCalled();
            expect(window.location.pathname).toBe('/replaced');
        });

        it('should use pushState when replace is false', () => {
            const pushStateSpy = vi.spyOn(window.history, 'pushState');

            router.navigate({ pathValue: '/pushed', replace: false });

            expect(pushStateSpy).toHaveBeenCalled();
        });

        it('should preserve current pathname when pathValue not provided', () => {
            window.history.replaceState({}, '', '/current');

            router.navigate({ pathValue: '/current', query: 'new=param' });

            expect(window.location.pathname).toBe('/current');
            expect(window.location.search).toBe('?new=param');
        });

        it('should preserve current query when query not provided', () => {
            window.history.replaceState({}, '', '/page?existing=value');

            router.navigate({ pathValue: '/page' });

            expect(window.location.search).toBe('?existing=value');
        });

        it('should preserve current hash when hash not provided', () => {
            window.history.replaceState({}, '', '/page#existing');

            router.navigate({ pathValue: '/page' });

            expect(window.location.hash).toBe('#existing');
        });
    });

    describe('popstate', () => {
        it('should call onNav with pop type on popstate', () => {
            window.history.pushState({}, '', '/first');
            window.history.pushState({}, '', '/second');

            window.dispatchEvent(new PopStateEvent('popstate'));

            expect(onNav).toHaveBeenCalledWith('pop', '/second', '', '');
        });

        it('should include query and hash in popstate', () => {
            window.history.replaceState({}, '', '/page?q=test#section');

            window.dispatchEvent(new PopStateEvent('popstate'));

            expect(onNav).toHaveBeenCalledWith('pop', '/page', 'q=test', 'section');
        });

        it('should strip leading ? from query in popstate', () => {
            window.history.replaceState({}, '', '/page?foo=bar');

            window.dispatchEvent(new PopStateEvent('popstate'));

            expect(onNav).toHaveBeenCalledWith('pop', '/page', 'foo=bar', '');
        });
    });
});

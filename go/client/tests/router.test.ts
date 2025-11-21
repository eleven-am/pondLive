import { describe, it, expect, beforeEach, vi } from 'vitest';
import { Router } from '../src/router';
import { ClientNode } from '../src/types';

describe('Router', () => {
    let channel: any;
    let router: Router;

    beforeEach(() => {
        channel = { sendMessage: vi.fn() };
        router = new Router(channel);

        // Mock window.history
        Object.defineProperty(window, 'location', {
            value: { pathname: '/', search: '', hash: '' },
            writable: true
        });
        window.history.pushState = vi.fn();
        window.history.replaceState = vi.fn();
    });

    it('attaches click listener to node with router meta', () => {
        const el = document.createElement('a');
        el.href = '/foo';
        const node: ClientNode = {
            tag: 'a',
            router: { path: '/foo' },
            el: el
        };

        router.attach(node);

        // Simulate click
        const event = new MouseEvent('click', { bubbles: true, cancelable: true });
        el.dispatchEvent(event);

        expect(event.defaultPrevented).toBe(true);
        expect(window.history.pushState).toHaveBeenCalled();
        expect(channel.sendMessage).toHaveBeenCalledWith('nav', expect.objectContaining({ path: '/foo' }));
    });

    it('ignores click if no router meta', () => {
        const el = document.createElement('a');
        const node: ClientNode = { tag: 'a', el: el };

        router.attach(node);

        const event = new MouseEvent('click', { bubbles: true, cancelable: true });
        el.dispatchEvent(event);

        expect(event.defaultPrevented).toBe(false);
    });

    it('handles popstate', () => {
        // Simulate popstate
        const event = new PopStateEvent('popstate');
        window.dispatchEvent(event);

        expect(channel.sendMessage).toHaveBeenCalledWith('pop', expect.anything());
    });
});

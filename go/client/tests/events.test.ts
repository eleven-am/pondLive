import { describe, it, expect, beforeEach, vi } from 'vitest';
import { EventManager } from '../src/events';
import { ClientNode } from '../src/types';

describe('EventManager', () => {
    let channel: any;
    let manager: EventManager;

    beforeEach(() => {
        channel = { sendMessage: vi.fn() };
        manager = new EventManager(channel, 'sid-1');
    });

    it('attaches event listener', () => {
        const el = document.createElement('button');
        const node: ClientNode = {
            tag: 'button',
            handlers: [{ event: 'click', handler: 'handleClick' }],
            el: el
        };

        manager.attach(node);

        const event = new MouseEvent('click');
        el.dispatchEvent(event);

        expect(channel.sendMessage).toHaveBeenCalledWith('evt', expect.objectContaining({
            hid: 'handleClick'
        }));
    });

    it('detaches event listener', () => {
        const el = document.createElement('button');
        const node: ClientNode = {
            tag: 'button',
            handlers: [{ event: 'click', handler: 'handleClick' }],
            el: el
        };

        manager.attach(node);
        manager.detach(node);

        const event = new MouseEvent('click');
        el.dispatchEvent(event);

        expect(channel.sendMessage).not.toHaveBeenCalled();
    });

    it('attaches multiple handlers for same event', () => {
        const el = document.createElement('button');
        const node: ClientNode = {
            tag: 'button',
            handlers: [
                { event: 'click', handler: 'h1' },
                { event: 'click', handler: 'h2' }
            ],
            el: el
        };

        manager.attach(node);

        const event = new MouseEvent('click');
        el.dispatchEvent(event);

        const calls = channel.sendMessage.mock.calls.filter((c: any[]) => c[0] === 'evt');
        const handlerIds = calls.map((c: any[]) => c[1].hid);
        expect(handlerIds).toContain('h1');
        expect(handlerIds).toContain('h2');
    });

    it('respects allowDefault flag', () => {
        const el = document.createElement('a');
        const node: ClientNode = {
            tag: 'a',
            handlers: [{ event: 'click', handler: 'h1', listen: ['allowDefault'] }],
            el: el
        };

        manager.attach(node);
        const event = new MouseEvent('click', { cancelable: true });
        el.dispatchEvent(event);

        expect(event.defaultPrevented).toBe(false);
    });
});

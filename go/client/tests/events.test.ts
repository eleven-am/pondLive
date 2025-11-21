import { describe, it, expect, beforeEach, vi } from 'vitest';
import { EventManager } from '../src/events';
import { ClientNode } from '../src/types';

describe('EventManager', () => {
    let channel: any;
    let manager: EventManager;

    beforeEach(() => {
        channel = { sendMessage: vi.fn() };
        manager = new EventManager(channel);
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
});

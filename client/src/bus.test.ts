import { describe, it, expect, vi, beforeEach } from 'vitest';
import { Bus } from './bus';

describe('Bus', () => {
    let bus: Bus;

    beforeEach(() => {
        bus = new Bus();
    });

    describe('subscribe', () => {
        it('should subscribe to topic and action', () => {
            const callback = vi.fn();
            bus.subscribe('frame', 'patch', callback);

            bus.publish('frame', 'patch', { seq: 1, patches: [] });

            expect(callback).toHaveBeenCalledWith({ seq: 1, patches: [] });
        });

        it('should return unsubscribe function', () => {
            const callback = vi.fn();
            const sub = bus.subscribe('frame', 'patch', callback);

            sub.unsubscribe();
            bus.publish('frame', 'patch', { seq: 1, patches: [] });

            expect(callback).not.toHaveBeenCalled();
        });

        it('should allow multiple subscribers', () => {
            const callback1 = vi.fn();
            const callback2 = vi.fn();

            bus.subscribe('frame', 'patch', callback1);
            bus.subscribe('frame', 'patch', callback2);

            bus.publish('frame', 'patch', { seq: 1, patches: [] });

            expect(callback1).toHaveBeenCalled();
            expect(callback2).toHaveBeenCalled();
        });
    });

    describe('upsert', () => {
        it('should create new subscription if none exists', () => {
            const callback = vi.fn();
            bus.upsert('frame', 'patch', callback);

            bus.publish('frame', 'patch', { seq: 1, patches: [] });

            expect(callback).toHaveBeenCalled();
        });

        it('should replace existing subscription', () => {
            const callback1 = vi.fn();
            const callback2 = vi.fn();

            bus.upsert('frame', 'patch', callback1);
            bus.upsert('frame', 'patch', callback2);

            bus.publish('frame', 'patch', { seq: 1, patches: [] });

            expect(callback1).not.toHaveBeenCalled();
            expect(callback2).toHaveBeenCalled();
        });
    });

    describe('subscribeAll', () => {
        it('should receive all published messages', () => {
            const callback = vi.fn();
            bus.subscribeAll(callback);

            bus.publish('frame', 'patch', { seq: 1, patches: [] });
            bus.publish('router', 'push', { path: '/new', query: '', hash: '', replace: false });

            expect(callback).toHaveBeenCalledTimes(2);
            expect(callback).toHaveBeenCalledWith('frame', 'patch', { seq: 1, patches: [] });
            expect(callback).toHaveBeenCalledWith('router', 'push', { path: '/new', query: '', hash: '', replace: false });
        });

        it('should return unsubscribe function', () => {
            const callback = vi.fn();
            const sub = bus.subscribeAll(callback);

            sub.unsubscribe();
            bus.publish('frame', 'patch', { seq: 1, patches: [] });

            expect(callback).not.toHaveBeenCalled();
        });
    });

    describe('publish', () => {
        it('should not throw when no subscribers', () => {
            expect(() => {
                bus.publish('frame', 'patch', { seq: 1, patches: [] });
            }).not.toThrow();
        });

        it('should swallow subscriber errors', () => {
            const badCallback = vi.fn(() => {
                throw new Error('Subscriber error');
            });
            const goodCallback = vi.fn();

            bus.subscribe('frame', 'patch', badCallback);
            bus.subscribe('frame', 'patch', goodCallback);

            expect(() => {
                bus.publish('frame', 'patch', { seq: 1, patches: [] });
            }).not.toThrow();

            expect(goodCallback).toHaveBeenCalled();
        });
    });

    describe('subscriberCount', () => {
        it('should return 0 when no subscribers', () => {
            expect(bus.subscriberCount('frame', 'patch')).toBe(0);
        });

        it('should return correct count', () => {
            bus.subscribe('frame', 'patch', vi.fn());
            bus.subscribe('frame', 'patch', vi.fn());

            expect(bus.subscriberCount('frame', 'patch')).toBe(2);
        });

        it('should decrease after unsubscribe', () => {
            const sub1 = bus.subscribe('frame', 'patch', vi.fn());
            bus.subscribe('frame', 'patch', vi.fn());

            sub1.unsubscribe();

            expect(bus.subscriberCount('frame', 'patch')).toBe(1);
        });
    });

    describe('subscribeScript', () => {
        it('should subscribe to script topic', () => {
            const callback = vi.fn();
            bus.subscribeScript('script-1', 'send', callback);

            bus.publishScript('script-1', 'send', { scriptId: 'script-1', event: 'test' });

            expect(callback).toHaveBeenCalledWith({ scriptId: 'script-1', event: 'test' });
        });
    });

    describe('publishScript', () => {
        it('should publish to script subscribers', () => {
            const callback = vi.fn();
            bus.subscribeScript('script-1', 'message', callback);

            bus.publishScript('script-1', 'message', { scriptId: 'script-1', event: 'update', data: { foo: 'bar' } });

            expect(callback).toHaveBeenCalledWith({ scriptId: 'script-1', event: 'update', data: { foo: 'bar' } });
        });

        it('should notify wildcard subscribers', () => {
            const callback = vi.fn();
            bus.subscribeAll(callback);

            bus.publishScript('script-1', 'send', { scriptId: 'script-1', event: 'test' });

            expect(callback).toHaveBeenCalledWith('script:script-1', 'send', { scriptId: 'script-1', event: 'test' });
        });
    });

    describe('publishHandler', () => {
        it('should publish to handler subscribers', () => {
            const callback = vi.fn();
            bus.subscribe('c0:h0' as any, 'invoke', callback);

            bus.publishHandler('c0:h0', { cseq: 1 });

            expect(callback).toHaveBeenCalledWith({ cseq: 1 });
        });

        it('should notify wildcard subscribers', () => {
            const callback = vi.fn();
            bus.subscribeAll(callback);

            bus.publishHandler('c0:h0', { cseq: 1, value: 'test' });

            expect(callback).toHaveBeenCalledWith('c0:h0', 'invoke', { cseq: 1, value: 'test' });
        });
    });

    describe('clear', () => {
        it('should remove all subscribers', () => {
            const callback = vi.fn();
            bus.subscribe('frame', 'patch', callback);
            bus.subscribeAll(callback);

            bus.clear();
            bus.publish('frame', 'patch', { seq: 1, patches: [] });

            expect(callback).not.toHaveBeenCalled();
        });
    });
});

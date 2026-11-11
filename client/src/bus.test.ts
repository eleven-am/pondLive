import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Bus } from './bus';
import { Logger } from './logger';

describe('Bus', () => {
    let bus: Bus;

    beforeEach(() => {
        bus = new Bus();
    });

    describe('subscribe', () => {
        it('should subscribe to topic and action', () => {
            const callback = vi.fn();
            bus.subscribe('frame', 'patch', callback);

            bus.publish('frame', 'patch', []);

            expect(callback).toHaveBeenCalledWith([]);
        });

        it('should return unsubscribe function', () => {
            const callback = vi.fn();
            const sub = bus.subscribe('frame', 'patch', callback);

            sub.unsubscribe();
            bus.publish('frame', 'patch', []);

            expect(callback).not.toHaveBeenCalled();
        });

        it('should allow multiple subscribers', () => {
            const callback1 = vi.fn();
            const callback2 = vi.fn();

            bus.subscribe('frame', 'patch', callback1);
            bus.subscribe('frame', 'patch', callback2);

            bus.publish('frame', 'patch', []);

            expect(callback1).toHaveBeenCalled();
            expect(callback2).toHaveBeenCalled();
        });
    });

    describe('upsert', () => {
        it('should create new subscription if none exists', () => {
            const callback = vi.fn();
            bus.upsert('frame', 'patch', callback);

            bus.publish('frame', 'patch', []);

            expect(callback).toHaveBeenCalled();
        });

        it('should replace existing subscription', () => {
            const callback1 = vi.fn();
            const callback2 = vi.fn();

            bus.upsert('frame', 'patch', callback1);
            bus.upsert('frame', 'patch', callback2);

            bus.publish('frame', 'patch', []);

            expect(callback1).not.toHaveBeenCalled();
            expect(callback2).toHaveBeenCalled();
        });
    });

    describe('subscribeAll', () => {
        it('should receive all published messages', () => {
            const callback = vi.fn();
            bus.subscribeAll(callback);

            bus.publish('frame', 'patch', []);
            bus.publish('router', 'push', { path: '/new', query: '', hash: '', replace: false });

            expect(callback).toHaveBeenCalledTimes(2);
            expect(callback).toHaveBeenCalledWith('frame', 'patch', []);
            expect(callback).toHaveBeenCalledWith('router', 'push', { path: '/new', query: '', hash: '', replace: false });
        });

        it('should return unsubscribe function', () => {
            const callback = vi.fn();
            const sub = bus.subscribeAll(callback);

            sub.unsubscribe();
            bus.publish('frame', 'patch', []);

            expect(callback).not.toHaveBeenCalled();
        });
    });

    describe('publish', () => {
        it('should not throw when no subscribers', () => {
            expect(() => {
                bus.publish('frame', 'patch', []);
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
                bus.publish('frame', 'patch', []);
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

    describe('clear', () => {
        it('should remove all subscribers', () => {
            const callback = vi.fn();
            bus.subscribe('frame', 'patch', callback);
            bus.subscribeAll(callback);

            bus.clear();
            bus.publish('frame', 'patch', []);

            expect(callback).not.toHaveBeenCalled();
        });
    });

    describe('subscriber failures', () => {
        let errorSpy: ReturnType<typeof vi.spyOn>;

        beforeEach(() => {
            Logger.configure({ enabled: true, level: 'error' });
            errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
        });

        afterEach(() => {
            errorSpy.mockRestore();
            Logger.configure({ enabled: false, level: 'info' });
        });

        const boom = () => {
            throw new Error('boom');
        };

        it('logs a throwing subscriber on publish and still runs the others', () => {
            const after = vi.fn();
            const wild = vi.fn();
            bus.subscribe('frame', 'patch', boom);
            bus.subscribe('frame', 'patch', after);
            bus.subscribeAll(wild);

            bus.publish('frame', 'patch', []);

            expect(after).toHaveBeenCalled();
            expect(wild).toHaveBeenCalled();
            expect(errorSpy).toHaveBeenCalledTimes(1);
            expect(errorSpy.mock.calls[0][1]).toBe('Subscriber failed');
            expect(errorSpy.mock.calls[0][2]).toMatchObject({ topic: 'frame', action: 'patch' });
        });

        it('logs a throwing wildcard subscriber on publish', () => {
            const after = vi.fn();
            bus.subscribeAll(boom);
            bus.subscribeAll(after);

            bus.publish('frame', 'patch', []);

            expect(after).toHaveBeenCalled();
            expect(errorSpy).toHaveBeenCalledTimes(1);
        });
    });
});

import { describe, it, expect, vi, beforeEach, Mock } from 'vitest';
import { Executor, ExecutorConfig } from './executor';
import { Bus } from './bus';
import { Transport } from './transport';

vi.mock('./transport');

describe('Executor', () => {
    let executor: Executor;
    let bus: Bus;
    let mockTransport: { send: Mock };
    let refs: Map<string, Element>;
    let resolveRef: (refId: string) => Element | undefined;

    beforeEach(() => {
        vi.clearAllMocks();

        bus = new Bus();
        refs = new Map();
        resolveRef = (refId) => refs.get(refId);

        mockTransport = {
            send: vi.fn(),
        };

        (Transport as unknown as Mock).mockImplementation(() => mockTransport);

        const config: ExecutorConfig = {
            bus,
            transport: mockTransport as unknown as Transport,
            resolveRef,
        };

        executor = new Executor(config);
    });

    describe('DOM operations', () => {
        describe('call', () => {
            it('should call method on element', () => {
                const el = document.createElement('input');
                el.focus = vi.fn();
                refs.set('myInput', el);

                bus.publish('dom', 'call', { ref: 'myInput', method: 'focus' });

                expect(el.focus).toHaveBeenCalled();
            });

            it('should call method with args', () => {
                const el = document.createElement('input');
                el.setSelectionRange = vi.fn();
                refs.set('myInput', el);

                bus.publish('dom', 'call', { ref: 'myInput', method: 'setSelectionRange', args: [0, 5] });

                expect(el.setSelectionRange).toHaveBeenCalledWith(0, 5);
            });

            it('should ignore missing refs', () => {
                expect(() => {
                    bus.publish('dom', 'call', { ref: 'missing', method: 'focus' });
                }).not.toThrow();
            });
        });

        describe('set', () => {
            it('should set property on element', () => {
                const el = document.createElement('input') as HTMLInputElement;
                refs.set('myInput', el);

                bus.publish('dom', 'set', { ref: 'myInput', prop: 'value', value: 'hello' });

                expect(el.value).toBe('hello');
            });

            it('should set boolean property', () => {
                const el = document.createElement('input') as HTMLInputElement;
                refs.set('myInput', el);

                bus.publish('dom', 'set', { ref: 'myInput', prop: 'disabled', value: true });

                expect(el.disabled).toBe(true);
            });

            it('should ignore missing refs', () => {
                expect(() => {
                    bus.publish('dom', 'set', { ref: 'missing', prop: 'value', value: 'test' });
                }).not.toThrow();
            });
        });

        describe('query', () => {
            it('should respond with values for selectors', () => {
                const el = document.createElement('input') as HTMLInputElement;
                el.value = 'test value';
                el.type = 'text';
                refs.set('myInput', el);

                bus.publish('dom', 'query', {
                    requestId: 'req-1',
                    ref: 'myInput',
                    selectors: ['value', 'type'],
                });

                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    values: { value: 'test value', type: 'text' },
                });
            });

            it('should respond with error for missing ref', () => {
                bus.publish('dom', 'query', {
                    requestId: 'req-1',
                    ref: 'missing',
                    selectors: ['value'],
                });

                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    error: 'ref not found: missing',
                });
            });

            it('should handle nested properties', () => {
                const el = document.createElement('div');
                el.dataset.id = '123';
                refs.set('myDiv', el);

                bus.publish('dom', 'query', {
                    requestId: 'req-1',
                    ref: 'myDiv',
                    selectors: ['dataset.id'],
                });

                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    values: { 'dataset.id': '123' },
                });
            });
        });

        describe('async', () => {
            it('should call async method and respond with result', async () => {
                const el = document.createElement('div');
                (el as any).asyncMethod = vi.fn().mockResolvedValue('result');
                refs.set('myDiv', el);

                bus.publish('dom', 'async', {
                    requestId: 'req-1',
                    ref: 'myDiv',
                    method: 'asyncMethod',
                });

                await vi.waitFor(() => {
                    expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                        requestId: 'req-1',
                        result: 'result',
                    });
                });
            });

            it('should respond with error for rejected promise', async () => {
                const el = document.createElement('div');
                (el as any).asyncMethod = vi.fn().mockRejectedValue(new Error('async error'));
                refs.set('myDiv', el);

                bus.publish('dom', 'async', {
                    requestId: 'req-1',
                    ref: 'myDiv',
                    method: 'asyncMethod',
                });

                await vi.waitFor(() => {
                    expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                        requestId: 'req-1',
                        error: 'async error',
                    });
                });
            });

            it('should respond with error for missing ref', () => {
                bus.publish('dom', 'async', {
                    requestId: 'req-1',
                    ref: 'missing',
                    method: 'asyncMethod',
                });

                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    error: 'ref not found: missing',
                });
            });

            it('should respond with error for missing method', () => {
                const el = document.createElement('div');
                refs.set('myDiv', el);

                bus.publish('dom', 'async', {
                    requestId: 'req-1',
                    ref: 'myDiv',
                    method: 'nonExistent',
                });

                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    error: 'method not found: nonExistent',
                });
            });
        });
    });

    describe('Router operations', () => {
        describe('push', () => {
            it('should push state with path', () => {
                const pushStateSpy = vi.spyOn(window.history, 'pushState');

                bus.publish('router', 'push', {
                    path: '/new-page',
                    query: '',
                    hash: '',
                    replace: false,
                });

                expect(pushStateSpy).toHaveBeenCalledWith({}, '', '/new-page');
                pushStateSpy.mockRestore();
            });

            it('should push state with query', () => {
                const pushStateSpy = vi.spyOn(window.history, 'pushState');

                bus.publish('router', 'push', {
                    path: '/search',
                    query: 'q=test',
                    hash: '',
                    replace: false,
                });

                expect(pushStateSpy).toHaveBeenCalledWith({}, '', '/search?q=test');
                pushStateSpy.mockRestore();
            });

            it('should push state with hash', () => {
                const pushStateSpy = vi.spyOn(window.history, 'pushState');

                bus.publish('router', 'push', {
                    path: '/page',
                    query: '',
                    hash: 'section',
                    replace: false,
                });

                expect(pushStateSpy).toHaveBeenCalledWith({}, '', '/page#section');
                pushStateSpy.mockRestore();
            });
        });

        describe('replace', () => {
            it('should replace state', () => {
                const replaceStateSpy = vi.spyOn(window.history, 'replaceState');

                bus.publish('router', 'replace', {
                    path: '/replaced',
                    query: '',
                    hash: '',
                    replace: true,
                });

                expect(replaceStateSpy).toHaveBeenCalledWith({}, '', '/replaced');
                replaceStateSpy.mockRestore();
            });
        });

        describe('back', () => {
            it('should call history.back', () => {
                const backSpy = vi.spyOn(window.history, 'back');

                bus.publish('router', 'back', undefined);

                expect(backSpy).toHaveBeenCalled();
                backSpy.mockRestore();
            });
        });

        describe('forward', () => {
            it('should call history.forward', () => {
                const forwardSpy = vi.spyOn(window.history, 'forward');

                bus.publish('router', 'forward', undefined);

                expect(forwardSpy).toHaveBeenCalled();
                forwardSpy.mockRestore();
            });
        });
    });

    describe('popstate listener', () => {
        it('should send popstate event on navigation', () => {
            window.dispatchEvent(new PopStateEvent('popstate'));

            expect(mockTransport.send).toHaveBeenCalledWith('router', 'popstate', {
                path: expect.any(String),
                query: expect.any(String),
                hash: expect.any(String),
            });
        });
    });

    describe('destroy', () => {
        it('should unsubscribe all subscriptions', () => {
            const callback = vi.fn();
            bus.subscribe('dom', 'call', callback);

            executor.destroy();

            bus.publish('dom', 'call', { ref: 'test', method: 'focus' });
        });

        it('should remove popstate listener', () => {
            executor.destroy();

            mockTransport.send.mockClear();
            window.dispatchEvent(new PopStateEvent('popstate'));

            expect(mockTransport.send).not.toHaveBeenCalled();
        });
    });
});

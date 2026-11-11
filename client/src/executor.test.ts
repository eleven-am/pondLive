import { describe, it, expect, vi, beforeEach, afterEach, Mock } from 'vitest';
import { Executor, ExecutorConfig } from './executor';
import { Bus } from './bus';
import { Transport } from './transport';

vi.mock('./transport');

describe('Executor', () => {
    let executor: Executor;
    let bus: Bus;
    let mockTransport: { send: Mock };
    let refs: Map<number, Element>;
    let resolveRef: (id: number) => Element | undefined;

    beforeEach(() => {
        vi.clearAllMocks();

        bus = new Bus();
        refs = new Map();
        resolveRef = (id) => refs.get(id);

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
                refs.set(7, el);

                bus.publish('dom', 'call', { k: 'InputRef', id: 7, method: 'focus' });

                expect(el.focus).toHaveBeenCalled();
            });

            it('should call method with args', () => {
                const el = document.createElement('input');
                el.setSelectionRange = vi.fn();
                refs.set(7, el);

                bus.publish('dom', 'call', { k: 'InputRef', id: 7, method: 'setSelectionRange', args: [0, 5] });

                expect(el.setSelectionRange).toHaveBeenCalledWith(0, 5);
            });

            it('should ignore missing refs', () => {
                expect(() => {
                    bus.publish('dom', 'call', { k: 'InputRef', id: 99, method: 'focus' });
                }).not.toThrow();
            });
        });

        describe('call allowlist', () => {
            it('rejects a method that is not on the ref type list', () => {
                const el = document.createElement('div');
                (el as any).remove = vi.fn();
                (el as any).submit = vi.fn();
                refs.set(7, el);

                bus.publish('dom', 'call', { k: 'ElementRef', id: 7, method: 'remove' });
                bus.publish('dom', 'call', { k: 'ElementRef', id: 7, method: 'submit' });
                bus.publish('dom', 'call', { k: 'FormRef', id: 7, method: 'remove' });
                bus.publish('dom', 'call', { k: '', id: 7, method: 'focus' });
                (el as any).focus = vi.fn();
                bus.publish('dom', 'call', { k: 'NopeRef', id: 7, method: 'focus' });

                expect((el as any).focus).not.toHaveBeenCalled();
                expect((el as any).remove).not.toHaveBeenCalled();
                expect((el as any).submit).not.toHaveBeenCalled();
            });

            it('accepts the base list on every type and the own list on its type', () => {
                const form = document.createElement('form');
                form.submit = vi.fn();
                form.focus = vi.fn();
                refs.set(7, form);

                bus.publish('dom', 'call', { k: 'FormRef', id: 7, method: 'submit' });
                bus.publish('dom', 'call', { k: 'FormRef', id: 7, method: 'focus' });

                expect(form.submit).toHaveBeenCalled();
                expect(form.focus).toHaveBeenCalled();
            });
        });

        describe('set', () => {
            it('should set a listed property', () => {
                const el = document.createElement('div');
                refs.set(7, el);

                bus.publish('dom', 'set', { k: 'ElementRef', id: 7, prop: 'scrollTop', value: 12 });

                expect(el.scrollTop).toBe(12);
            });

            it('should set a boolean property of its type', () => {
                const el = document.createElement('details');
                refs.set(7, el);

                bus.publish('dom', 'set', { k: 'DetailsRef', id: 7, prop: 'open', value: true });

                expect(el.open).toBe(true);
            });

            it('rejects a property that is not on the list', () => {
                const el = document.createElement('input');
                refs.set(7, el);

                bus.publish('dom', 'set', { k: 'InputRef', id: 7, prop: 'value', value: 'hello' });
                bus.publish('dom', 'set', { k: 'ElementRef', id: 7, prop: 'innerHTML', value: '<b>x</b>' });
                bus.publish('dom', 'set', { k: '', id: 7, prop: 'scrollTop', value: 3 });

                expect(el.value).toBe('');
                expect(el.innerHTML).toBe('');
            });

            it('should ignore missing refs', () => {
                expect(() => {
                    bus.publish('dom', 'set', { k: 'ElementRef', id: 99, prop: 'scrollTop', value: 1 });
                }).not.toThrow();
            });
        });

        describe('query', () => {
            it('should respond with values for listed selectors', () => {
                const el = document.createElement('input') as HTMLInputElement;
                el.value = 'abcdef';
                el.setSelectionRange(1, 4);
                refs.set(7, el);

                bus.publish('dom', 'query', {
                    k: 'InputRef',
                    requestId: 'req-1',
                    id: 7,
                    selectors: ['selectionStart', 'selectionEnd'],
                });

                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    values: { selectionStart: 1, selectionEnd: 4 },
                });
            });

            it('should refuse a selector that is not on the list', () => {
                const el = document.createElement('input');
                el.value = 'secret';
                refs.set(7, el);

                bus.publish('dom', 'query', { k: 'InputRef', requestId: 'req-1', id: 7, selectors: ['selectionStart', 'value'] });

                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    error: 'property not allowed',
                });
            });

            it('should respond with error for missing ref', () => {
                bus.publish('dom', 'query', {
                    k: 'InputRef',
                    requestId: 'req-1',
                    id: 99,
                    selectors: ['selectionStart'],
                });

                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    error: 'ref not found: 99',
                });
            });
        });

        describe('async', () => {
            it('should call a listed method and respond with its result', async () => {
                const el = document.createElement('canvas');
                el.toDataURL = vi.fn().mockReturnValue('data:image/png;base64,AA');
                refs.set(8, el);

                bus.publish('dom', 'async', { k: 'CanvasRef', requestId: 'req-1', id: 8, method: 'toDataURL' });

                await vi.waitFor(() => {
                    expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                        requestId: 'req-1',
                        result: 'data:image/png;base64,AA',
                    });
                });
            });

            it('should respond with error for a rejected promise', async () => {
                const el = document.createElement('div');
                el.matches = vi.fn().mockRejectedValue(new Error('async error')) as never;
                refs.set(8, el);

                bus.publish('dom', 'async', { k: 'ElementRef', requestId: 'req-1', id: 8, method: 'matches' });

                await vi.waitFor(() => {
                    expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                        requestId: 'req-1',
                        error: 'async error',
                    });
                });
            });

            it('should refuse a method that is not on the list', () => {
                const el = document.createElement('div');
                (el as any).asyncMethod = vi.fn();
                refs.set(8, el);

                bus.publish('dom', 'async', { k: 'ElementRef', requestId: 'req-1', id: 8, method: 'asyncMethod' });

                expect((el as any).asyncMethod).not.toHaveBeenCalled();
                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    error: 'method not allowed: asyncMethod',
                });
            });

            it('should respond with error for missing ref', () => {
                bus.publish('dom', 'async', { k: 'ElementRef', requestId: 'req-1', id: 99, method: 'matches' });

                expect(mockTransport.send).toHaveBeenCalledWith('dom', 'response', {
                    requestId: 'req-1',
                    error: 'ref not found: 99',
                });
            });
        });
    });

    describe('observe', () => {
        const values = () =>
            mockTransport.send.mock.calls.filter((c) => c[0] === 'dom' && c[1] === 'value').map((c) => c[2]);
        const observe = (kind: string, id: number, prop: string, extra: object = {}) =>
            bus.publish('dom', 'observe', { k: kind, id, prop, ms: -1, ...extra });

        it('sends the current value at once and again on each change, without a request id', () => {
            const el = document.createElement('video');
            Object.defineProperty(el, 'volume', { value: 0.5, writable: true, configurable: true });
            refs.set(7, el);

            observe('MediaRef', 7, 'volume');
            expect(values()).toEqual([{ id: 7, prop: 'volume', value: 0.5 }]);

            (el as any).volume = 0.2;
            el.dispatchEvent(new Event('volumechange'));
            expect(values().at(-1)).toEqual({ id: 7, prop: 'volume', value: 0.2 });
            expect(mockTransport.send.mock.calls.every((c) => c[1] !== 'response')).toBe(true);
        });

        it('does not send a value that did not change', () => {
            const el = document.createElement('video');
            refs.set(7, el);
            observe('MediaRef', 7, 'muted');
            el.dispatchEvent(new Event('volumechange'));
            el.dispatchEvent(new Event('volumechange'));
            expect(values()).toHaveLength(1);
        });

        it('stops listening on unobserve', () => {
            const el = document.createElement('video');
            refs.set(7, el);
            observe('MediaRef', 7, 'muted');
            bus.publish('dom', 'unobserve', { id: 7, prop: 'muted' });
            el.muted = true;
            el.dispatchEvent(new Event('volumechange'));
            expect(values()).toHaveLength(1);
        });

        it('replaces an earlier observer for the same element and property', () => {
            const el = document.createElement('video');
            refs.set(7, el);
            observe('MediaRef', 7, 'muted');
            observe('MediaRef', 7, 'muted');
            el.muted = true;
            el.dispatchEvent(new Event('volumechange'));
            expect(values().filter((v) => v.value === true)).toHaveLength(1);
        });

        it('refuses a property that is not on the ref type list', () => {
            const input = document.createElement('input');
            refs.set(7, input);
            observe('InputRef', 7, 'currentTime');
            observe('ElementRef', 7, 'value');
            observe('InputRef', 7, 'focused');
            observe('ElementRef', 7, 'muted');
            bus.publish('dom', 'observe', { k: '', id: 7, prop: 'focused', ms: -1 });
            expect(values().map((v) => v.prop)).toEqual(['focused']);
        });

        it('reads media state from the right events', () => {
            const el = document.createElement('video');
            refs.set(3, el);
            Object.defineProperty(el, 'paused', { value: true, configurable: true });
            observe('MediaRef', 3, 'paused');
            Object.defineProperty(el, 'paused', { value: false, configurable: true });
            el.dispatchEvent(new Event('play'));
            expect(values().map((v) => v.value)).toEqual([true, false]);
        });

        it('serializes a NaN duration as null', () => {
            const el = document.createElement('video');
            refs.set(3, el);
            Object.defineProperty(el, 'duration', { value: NaN, configurable: true });
            observe('MediaRef', 3, 'duration');
            const raw = JSON.parse(JSON.stringify(values()[0]));
            expect(raw.value).toBeNull();
        });

        it('reports focus changes', () => {
            const el = document.createElement('button');
            document.body.appendChild(el);
            refs.set(5, el);
            observe('ElementRef', 5, 'focused');
            el.focus();
            el.blur();
            expect(values().map((v) => v.value)).toEqual([false, true, false]);
            el.remove();
        });

        it('reads the scroll metrics on scroll', () => {
            const el = document.createElement('div');
            refs.set(5, el);
            vi.stubGlobal('requestAnimationFrame', (cb: () => void) => setTimeout(cb, 0));
            vi.useFakeTimers();
            try {
                observe('ElementRef', 5, 'scroll');
                el.scrollTop = 40;
                el.dispatchEvent(new Event('scroll'));
                el.dispatchEvent(new Event('scroll'));
                vi.runAllTimers();
                const sent = values();
                expect(sent[0].value.scrollTop).toBe(0);
                expect(sent.at(-1).value.scrollTop).toBe(40);
                expect(sent).toHaveLength(2);
            } finally {
                vi.useRealTimers();
                vi.unstubAllGlobals();
            }
        });

        it('throttles to one frame when the server asks for 0 and to the requested ms when given', () => {
            const el = document.createElement('video');
            refs.set(5, el);
            const frames: Array<() => void> = [];
            vi.stubGlobal('requestAnimationFrame', (cb: () => void) => frames.push(cb));
            let time = 0;
            Object.defineProperty(el, 'currentTime', { get: () => time, configurable: true });
            observe('MediaRef', 5, 'currentTime', { ms: 0 });
            for (let i = 1; i <= 50; i++) {
                time = i;
                el.dispatchEvent(new Event('timeupdate'));
            }
            expect(values().map((v) => v.value)).toEqual([0, 1]);
            expect(frames).toHaveLength(1);
            frames[0]();
            expect(values().at(-1).value).toBe(50);
            expect(values()).toHaveLength(3);

            vi.useFakeTimers();
            observe('MediaRef', 5, 'currentTime', { ms: 100 });
            time = 60;
            el.dispatchEvent(new Event('timeupdate'));
            time = 61;
            el.dispatchEvent(new Event('timeupdate'));
            expect(values().at(-1).value).toBe(60);
            vi.advanceTimersByTime(99);
            expect(values().at(-1).value).toBe(60);
            vi.advanceTimersByTime(2);
            expect(values().at(-1).value).toBe(61);
            vi.useRealTimers();
            vi.unstubAllGlobals();
        });

        it('observes size through a ResizeObserver and stops with unobserve', () => {
            const el = document.createElement('div');
            refs.set(5, el);
            const callbacks: Array<() => void> = [];
            const disconnect = vi.fn();
            vi.stubGlobal(
                'ResizeObserver',
                class {
                    constructor(cb: () => void) {
                        callbacks.push(cb);
                    }
                    observe() {}
                    disconnect = disconnect;
                }
            );
            vi.stubGlobal('requestAnimationFrame', (cb: () => void) => cb());
            observe('ElementRef', 5, 'rect');
            expect(values()[0].value).toHaveProperty('width');
            callbacks[0]();
            bus.publish('dom', 'unobserve', { id: 5, prop: 'rect' });
            expect(disconnect).toHaveBeenCalled();
            vi.unstubAllGlobals();
        });

        it('pushes a value changed by a server set, which fires no browser event', () => {
            const el = document.createElement('video');
            refs.set(7, el);
            observe('MediaRef', 7, 'muted');
            bus.publish('dom', 'set', { k: 'MediaRef', id: 7, prop: 'muted', value: true });
            expect(values().at(-1)).toEqual({ id: 7, prop: 'muted', value: true });
        });

        it('ignores unknown properties and missing elements', () => {
            expect(() => observe('MediaRef', 99, 'muted')).not.toThrow();
            refs.set(7, document.createElement('video'));
            observe('MediaRef', 7, 'nope');
            expect(values()).toHaveLength(0);
        });

        it('stops every observer on destroy', () => {
            const el = document.createElement('video');
            refs.set(7, el);
            observe('MediaRef', 7, 'muted');
            executor.destroy();
            el.muted = true;
            el.dispatchEvent(new Event('volumechange'));
            expect(values()).toHaveLength(1);
        });
    });

    describe('Router redirect', () => {
        let assign: Mock;

        beforeEach(() => {
            assign = vi.fn();
            vi.stubGlobal('location', { href: 'http://localhost/page', assign });
        });

        afterEach(() => {
            vi.unstubAllGlobals();
        });

        it('should assign a cross-origin https url', () => {
            bus.publish('router', 'redirect', { url: 'https://other.example/path?a=1' });
            expect(assign).toHaveBeenCalledWith('https://other.example/path?a=1');
        });

        it('should assign a relative url', () => {
            bus.publish('router', 'redirect', { url: '/login' });
            expect(assign).toHaveBeenCalledWith('/login');
        });

        it.each([
            'javascript:alert(1)',
            'JaVaScRiPt:alert(1)',
            'data:text/html,x',
            ' javascript:alert(1)',
            'java\tscript:alert(1)',
            '/\\evil.example',
            'vbscript:x',
            'file:///etc/passwd',
            '/a\u007fb',
            '/a\u00a0b',
            '/a\u2028b',
            '/a\u3000b',
            '/a\ufeffb',
        ])('should reject %j', (url) => {
            bus.publish('router', 'redirect', { url });
            expect(assign).not.toHaveBeenCalled();
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

            bus.publish('dom', 'call', { k: '', id: 5, method: 'focus' });
        });

        it('should remove popstate listener', () => {
            executor.destroy();

            mockTransport.send.mockClear();
            window.dispatchEvent(new PopStateEvent('popstate'));

            expect(mockTransport.send).not.toHaveBeenCalled();
        });
    });
});

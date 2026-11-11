import { describe, it, expect, beforeEach, vi, Mock } from 'vitest';
import { Runtime, RuntimeConfig } from './runtime';
import { Boot } from './protocol';
import { Transport, TransportConfig } from './transport';
import { Logger } from './logger';
import { define } from './scripts';

vi.mock('./transport');

function handlerBoot(): Boot {
    return {
        t: 'boot',
        tok: 'tok',
        ws: '/live',
        sid: 'test-session',
        ver: 1,
        seq: 1,
        location: { path: '/', query: {}, hash: '' },
    };
}

describe('Runtime', () => {
    let root: HTMLElement;
    let config: RuntimeConfig;
    let runtime: Runtime;
    let mockTransportInstance: {
        connect: Mock;
        disconnect: Mock;
        send: Mock;
        sendAck: Mock;
        sendResync: Mock;
        sendHandler: Mock;
        sendScript: Mock;
        onStateChange: Mock;
        sid: string;
    };
    let stateChangeHandler: (state: string) => void;
    let transportConfig: TransportConfig;

    beforeEach(() => {
        vi.clearAllMocks();
        Logger.configure({ enabled: false });

        mockTransportInstance = {
            connect: vi.fn(),
            disconnect: vi.fn(),
            send: vi.fn(),
            sendAck: vi.fn(),
            sendResync: vi.fn(),
            sendHandler: vi.fn(),
            sendScript: vi.fn(),
            onStateChange: vi.fn((handler) => {
                stateChangeHandler = handler;
                return () => {};
            }),
            sid: 'test-session',
        };

        (Transport as unknown as Mock).mockImplementation((cfg: TransportConfig) => {
            transportConfig = cfg;
            return mockTransportInstance;
        });

        root = document.createElement('div');
        document.body.appendChild(root);

        config = {
            root,
            sessionId: 'test-session',
            version: 1,
            seq: 0,
            endpoint: '/live',
            token: 'test-token',
            debug: false,
        };

        runtime = new Runtime(config);
    });

    describe('constructor', () => {
        it('should create runtime with config', () => {
            expect(runtime).toBeDefined();
        });

        it('should wire transport state change handler', () => {
            expect(mockTransportInstance.onStateChange).toHaveBeenCalled();
        });

        it('should set window.__POND_RUNTIME__', () => {
            expect(window.__POND_RUNTIME__).toBe(runtime);
        });
    });

    describe('connect', () => {
        it('should connect transport', () => {
            runtime.connect();
            expect(mockTransportInstance.connect).toHaveBeenCalled();
        });
    });

    describe('disconnect', () => {
        it('should disconnect transport', () => {
            runtime.disconnect();
            expect(mockTransportInstance.disconnect).toHaveBeenCalled();
        });
    });

    describe('connected', () => {
        it('should return false initially', () => {
            expect(runtime.connected()).toBe(false);
        });

        it('should return true when connected', () => {
            stateChangeHandler('connected');
            expect(runtime.connected()).toBe(true);
        });

        it('should return false when disconnected', () => {
            stateChangeHandler('connected');
            stateChangeHandler('disconnected');
            expect(runtime.connected()).toBe(false);
        });
    });

    describe('seq', () => {
        it('should return initial seq', () => {
            expect(runtime.seq).toBe(0);
        });

        it('should update after boot', () => {
            const boot: Boot = {
                t: 'boot',
                tok: 'tok',
                ws: '/live',
                sid: 'test-session',
                ver: 1,
                seq: 10,
                location: { path: '/', query: {}, hash: '' },
            };

            runtime.handleBoot(boot);
            expect(runtime.seq).toBe(10);
        });
    });

    describe('handleBoot', () => {
        it('should not ack on boot and should report the boot seq as applied', () => {
            const boot: Boot = {
                t: 'boot',
                tok: 'tok',
                ws: '/live',
                sid: 'test-session',
                ver: 1,
                seq: 5,
                location: { path: '/', query: {}, hash: '' },
            };

            runtime.handleBoot(boot);

            expect(mockTransportInstance.sendAck).not.toHaveBeenCalled();
            expect(transportConfig.applied()).toBe(5);
        });

        it('should index data-pi elements at boot so frames can address them', () => {
            root.innerHTML = '<span data-pi="1">old</span>';
            const child = root.firstElementChild!;

            runtime.handleBoot(handlerBoot());
            transportConfig.onMessage({ seq: 2, topic: 'frame', event: 'patch', data: [['t', 1, 'new']] });

            expect(child.textContent).toBe('new');
            expect(root.firstElementChild).toBe(child);
        });
    });

    describe('event handling', () => {
        it('should send events when handler is triggered', () => {
            root.innerHTML = '<button data-pi="3" data-pon=\'[{"e":"click"}]\'></button>';
            const button = root.querySelector('button')!;

            runtime.handleBoot(handlerBoot());
            mockTransportInstance.sendHandler.mockClear();

            button.click();

            expect(mockTransportInstance.sendHandler).toHaveBeenCalledWith(3, 'click', {});
        });

        it('should not add a cseq to the payload', () => {
            root.innerHTML = '<button data-pi="3" data-pon=\'[{"e":"click"}]\'></button>';
            const button = root.querySelector('button')!;
            runtime.handleBoot(handlerBoot());
            mockTransportInstance.sendHandler.mockClear();

            button.click();
            button.click();

            const calls = mockTransportInstance.sendHandler.mock.calls;
            expect(calls).toHaveLength(2);
            expect(calls[0][2]).not.toHaveProperty('cseq');
            expect(calls[1][2]).not.toHaveProperty('cseq');
        });

        it('should drop handler events while resyncing and resume after the reset', () => {
            root.innerHTML = '<button data-pi="3" data-pon=\'[{"e":"click"}]\'></button>';
            const button = root.querySelector('button')!;
            runtime.handleBoot(handlerBoot());
            mockTransportInstance.sendHandler.mockClear();
            stateChangeHandler('connected');

            transportConfig.onMessage({ seq: 2, topic: 'frame', event: 'patch', data: [] });
            expect(transportConfig.wantsReset()).toBe(false);
            transportConfig.onMessage({ seq: 2, topic: 'frame', event: 'patch', data: [] });
            transportConfig.onMessage({ seq: 9, topic: 'frame', event: 'patch', data: [] });

            expect(transportConfig.wantsReset()).toBe(true);
            expect(root.hasAttribute('data-pond-resyncing')).toBe(true);
            expect(mockTransportInstance.sendResync).toHaveBeenCalledWith(2, 'gap');

            button.click();
            expect(mockTransportInstance.sendHandler).not.toHaveBeenCalled();
        });
    });

    describe('message dispatch', () => {
        it('should apply frame patches in order and report them as applied', () => {
            root.innerHTML = '<span data-pi="1">old</span>';
            const child = root.firstElementChild!;
            runtime.handleBoot(handlerBoot());

            transportConfig.onMessage({
                seq: 2,
                topic: 'frame',
                event: 'patch',
                data: [['t', 1, 'new']],
            });

            expect(child.textContent).toBe('new');
            expect(transportConfig.applied()).toBe(2);
        });

        it('should not apply a duplicate frame twice', () => {
            root.innerHTML = '<span data-pi="1">old</span>';
            const child = root.firstElementChild!;
            runtime.handleBoot(handlerBoot());
            const frame = {
                seq: 2,
                topic: 'frame',
                event: 'patch',
                data: [['t', 1, 'new']],
            };

            transportConfig.onMessage(frame);
            child.textContent = 'changed';
            transportConfig.onMessage(frame);

            expect(child.textContent).toBe('changed');
        });
    });

    describe('strict apply and reset through the real applier', () => {
        it('an edit for an unknown id triggers a resync with reason unresolved and applies nothing after it', () => {
            root.innerHTML = '<span data-pi="1">old</span>';
            runtime.handleBoot(handlerBoot());
            stateChangeHandler('connected');

            transportConfig.onMessage({ seq: 2, topic: 'frame', event: 'patch', data: [['t', 1, 'first'], ['t', 99, 'x'], ['t', 1, 'never']] });

            expect(mockTransportInstance.sendResync).toHaveBeenCalledWith(1, 'unresolved');
            expect(transportConfig.wantsReset()).toBe(true);
            expect(root.firstElementChild!.textContent).toBe('first');
            expect(runtime.seq).toBe(1);
        });

        it('a malformed edit triggers a resync with reason apply-error', () => {
            root.innerHTML = '<span data-pi="1">old</span>';
            runtime.handleBoot(handlerBoot());
            stateChangeHandler('connected');

            transportConfig.onMessage({ seq: 2, topic: 'frame', event: 'patch', data: [['t', 1]] });

            expect(mockTransportInstance.sendResync).toHaveBeenCalledWith(1, 'apply-error');
        });

        it('a reset morphs by data-pi: the same nodes and listeners survive and the resync clears', () => {
            document.body.innerHTML = '<button data-pi="1" data-pon=\'[{"e":"click"}]\'>old</button>';
            const button = document.querySelector('button')!;
            runtime = new Runtime({ ...config, root: document.documentElement });
            runtime.handleBoot(handlerBoot());
            stateChangeHandler('connected');
            transportConfig.onMessage({ seq: 2, topic: 'frame', event: 'patch', data: [['t', 99, 'x']] });
            expect(document.documentElement.hasAttribute('data-pond-resyncing')).toBe(true);

            transportConfig.onMessage({
                seq: 3,
                topic: 'frame',
                event: 'reset',
                data: {
                    html: '<!DOCTYPE html><html><head></head><body><button data-pi="1" data-pon=\'[{"e":"click"}]\'>fresh</button></body></html>',
                    location: { path: '/', query: {}, hash: '' },
                    cause: 'client-request',
                },
            });

            expect(document.querySelector('button')).toBe(button);
            expect(button.textContent).toBe('fresh');
            expect(runtime.seq).toBe(3);
            expect(document.documentElement.hasAttribute('data-pond-resyncing')).toBe(false);
            mockTransportInstance.sendHandler.mockClear();
            button.click();
            expect(mockTransportInstance.sendHandler).toHaveBeenCalledTimes(1);
            document.body.innerHTML = '';
        });
    });

    describe('ref tracking', () => {
        it('should resolve DOM action ids against the indexed elements', () => {
            root.innerHTML = '<span data-pi="4"></span>';
            const span = root.firstElementChild!;
            const scrolled = vi.fn();
            span.scrollIntoView = scrolled;
            runtime.handleBoot(handlerBoot());

            transportConfig.onMessage({
                seq: 2,
                topic: 'dom',
                event: 'call',
                data: { k: 'ElementRef', id: 4, method: 'scrollIntoView', args: [] },
            });

            expect(scrolled).toHaveBeenCalled();
        });

        it('should ignore DOM actions for ids that are not indexed', () => {
            root.innerHTML = '<span data-pi="4"></span>';
            runtime.handleBoot(handlerBoot());

            transportConfig.onMessage({
                seq: 2,
                topic: 'dom',
                event: 'call',
                data: { k: 'ElementRef', id: 5, method: 'scrollIntoView', args: [] },
            });

            expect(() => root.firstElementChild!.hasAttribute('title')).not.toThrow();
        });
    });

    describe('observed values', () => {
        it('should start and stop observing through dom messages and send values on the dom topic', () => {
            root.innerHTML = '<video data-pi="4"></video>';
            const input = root.firstElementChild as HTMLVideoElement;
            input.muted = false;
            runtime.handleBoot(handlerBoot());
            const sent = () => mockTransportInstance.send.mock.calls.filter((c: unknown[]) => c[0] === 'dom' && c[1] === 'value');

            transportConfig.onMessage({ seq: 2, topic: 'dom', event: 'observe', data: { k: 'MediaRef', id: 4, prop: 'muted' } });
            expect(sent().at(-1)?.[2]).toEqual({ id: 4, prop: 'muted', value: false });

            input.muted = true;
            input.dispatchEvent(new Event('volumechange'));
            expect(sent().at(-1)?.[2].value).toBe(true);

            transportConfig.onMessage({ seq: 3, topic: 'dom', event: 'unobserve', data: { id: 4, prop: 'muted' } });
            const before = sent().length;
            input.muted = false;
            input.dispatchEvent(new Event('volumechange'));
            expect(sent()).toHaveLength(before);
        });
    });

    describe('script handling', () => {
        const insert = (seq: number, html: string) =>
            transportConfig.onMessage({ seq, topic: 'frame', event: 'patch', data: [['ins', 2, 0, html]] });
        const toScript = (seq: number, i: number, d: unknown, extra: Record<string, unknown> = {}) =>
            transportConfig.onMessage({ seq, topic: 's', event: 'm', data: { i, d, ...extra } });

        it('a message in the same batch as the ins reaches the handler', () => {
            const got: unknown[] = [];
            define('rt-batch', (_el, io) => {
                io.on((m) => got.push(m));
            });
            root.innerHTML = '<ul data-pi="1"><!--r:2--><!--/r:2--></ul>';
            runtime.handleBoot(handlerBoot());

            insert(2, '<li data-pi="3" data-pscript="rt-batch:41"></li>');
            toScript(3, 41, { n: 1 });

            expect(got).toEqual([{ n: 1 }]);
        });

        it('routing uses data.i only', () => {
            const a: unknown[] = [];
            const b: unknown[] = [];
            define('rt-a', (_el, io) => io.on((m) => a.push(m)));
            define('rt-b', (_el, io) => io.on((m) => b.push(m)));
            root.innerHTML = '<ul data-pi="1"><!--r:2--><!--/r:2--></ul>';
            runtime.handleBoot(handlerBoot());
            insert(2, '<li data-pi="3" data-pscript="rt-a:7 rt-b:8"></li>');

            toScript(3, 8, 'for-b', { scriptId: 'rt-a:7', id: 7, k: 'rt-a' });

            expect(a).toEqual([]);
            expect(b).toEqual(['for-b']);
        });

        it('ignores a script message without an integer instance id', () => {
            const got: unknown[] = [];
            define('rt-bad', (_el, io) => io.on((m) => got.push(m)));
            root.innerHTML = '<ul data-pi="1"><!--r:2--><!--/r:2--></ul>';
            runtime.handleBoot(handlerBoot());
            insert(2, '<li data-pi="3" data-pscript="rt-bad:9"></li>');

            transportConfig.onMessage({ seq: 3, topic: 's', event: 'm', data: { i: '9', d: 1 } });
            transportConfig.onMessage({ seq: 4, topic: 's', event: 'other', data: { i: 9, d: 1 } });

            expect(got).toEqual([]);
        });

        it('a script send goes out as an s event carrying the instance id', () => {
            define('rt-send', (_el, io) => io.send({ hello: 1 }));
            root.innerHTML = '<div data-pi="2" data-pscript="rt-send:5"></div>';

            runtime.handleBoot(handlerBoot());

            expect(mockTransportInstance.sendScript).toHaveBeenCalledWith(5, 'm', { hello: 1 });
        });

        it('disconnect cleans every running instance up', () => {
            const cleanup = vi.fn();
            define('rt-clean', () => cleanup);
            root.innerHTML = '<div data-pi="2" data-pscript="rt-clean:6"></div>';
            runtime.handleBoot(handlerBoot());

            runtime.disconnect();

            expect(cleanup).toHaveBeenCalledTimes(1);
        });

        it('joins with the bundle hash it was started with', () => {
            new Runtime({ ...config, bundle: 'abc123def456' });

            expect(transportConfig.bundle).toBe('abc123def456');
        });
    });

    describe('deploy skew', () => {
        it('a decline reloads exactly once', () => {
            vi.useFakeTimers();
            const reload = vi.fn();
            vi.stubGlobal('location', { ...window.location, reload });
            try {
                stateChangeHandler('declined');
                expect(reload).not.toHaveBeenCalled();
                vi.advanceTimersByTime(20000);
                expect(reload).toHaveBeenCalledTimes(1);
            } finally {
                vi.useRealTimers();
                vi.unstubAllGlobals();
            }
        });
    });
});

describe('Runtime with debug logging', () => {
    let root: HTMLElement;
    let mockTransportInstance: Record<string, Mock | string>;

    beforeEach(() => {
        vi.clearAllMocks();

        mockTransportInstance = {
            connect: vi.fn(),
            disconnect: vi.fn(),
            send: vi.fn(),
            sendAck: vi.fn(),
            sendResync: vi.fn(),
            sendHandler: vi.fn(),
            onStateChange: vi.fn(() => () => {}),
            sid: 'test-session',
        };

        (Transport as unknown as Mock).mockImplementation(() => mockTransportInstance);

        root = document.createElement('div');
        document.body.appendChild(root);
    });

    it('should configure logger with debug enabled', () => {
        const config: RuntimeConfig = {
            root,
            sessionId: 'test-session',
            version: 1,
            seq: 0,
            endpoint: '/live',
            token: 'test-token',
            debug: true,
        };

        new Runtime(config);
    });
});

import { describe, it, expect, beforeEach, vi, Mock } from 'vitest';
import { Runtime, RuntimeConfig } from './runtime';
import { Boot } from './protocol';
import { Transport } from './transport';
import { Logger } from './logger';

vi.mock('./transport');

describe('Runtime', () => {
    let root: HTMLElement;
    let config: RuntimeConfig;
    let runtime: Runtime;
    let mockTransportInstance: {
        connect: Mock;
        disconnect: Mock;
        send: Mock;
        sendAck: Mock;
        sendHandler: Mock;
        onStateChange: Mock;
        sid: string;
    };
    let stateChangeHandler: (state: string) => void;

    beforeEach(() => {
        vi.clearAllMocks();
        Logger.configure({ enabled: false });

        mockTransportInstance = {
            connect: vi.fn(),
            disconnect: vi.fn(),
            send: vi.fn(),
            sendAck: vi.fn(),
            sendHandler: vi.fn(),
            onStateChange: vi.fn((handler) => {
                stateChangeHandler = handler;
                return () => {};
            }),
            sid: 'test-session',
        };

        (Transport as unknown as Mock).mockImplementation(() => mockTransportInstance);

        root = document.createElement('div');
        document.body.appendChild(root);

        config = {
            root,
            sessionId: 'test-session',
            version: 1,
            seq: 0,
            endpoint: '/live',
            location: { path: '/', query: {}, hash: '' },
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
                sid: 'test-session',
                ver: 1,
                seq: 10,
                patch: [],
                location: { path: '/', query: {}, hash: '' },
            };

            runtime.handleBoot(boot);
            expect(runtime.seq).toBe(10);
        });
    });

    describe('handleBoot', () => {
        it('should send ack on boot', () => {
            const boot: Boot = {
                t: 'boot',
                sid: 'test-session',
                ver: 1,
                seq: 5,
                patch: [],
                location: { path: '/', query: {}, hash: '' },
            };

            runtime.handleBoot(boot);

            expect(mockTransportInstance.sendAck).toHaveBeenCalledWith(5);
        });

        it('should apply patches from boot', () => {
            const child = document.createElement('span');
            child.textContent = 'old';
            root.appendChild(child);

            const boot: Boot = {
                t: 'boot',
                sid: 'test-session',
                ver: 1,
                seq: 1,
                patch: [{ seq: 0, path: [0], op: 'setText', value: 'new' }],
                location: { path: '/', query: {}, hash: '' },
            };

            runtime.handleBoot(boot);

            expect(child.textContent).toBe('new');
        });
    });

    describe('handleMessage', () => {
        it('should handle server error', () => {
            const error = {
                t: 'error',
                sid: 'test-session',
                code: 'ERR001',
                message: 'Something went wrong',
            };

            expect(() => runtime.handleMessage(error)).not.toThrow();
        });

        it('should handle resume_ok', () => {
            const resumeOk = {
                t: 'resume_ok',
                from: 5,
                to: 10,
            };

            expect(() => runtime.handleMessage(resumeOk)).not.toThrow();
        });
    });

    describe('event handling', () => {
        it('should send events when handler is triggered', () => {
            const button = document.createElement('button');
            root.appendChild(button);

            const boot: Boot = {
                t: 'boot',
                sid: 'test-session',
                ver: 1,
                seq: 1,
                patch: [{
                    seq: 0,
                    path: [0],
                    op: 'setHandlers',
                    value: [{ event: 'click', handler: 'c0:h0' }]
                }],
                location: { path: '/', query: {}, hash: '' },
            };

            runtime.handleBoot(boot);
            mockTransportInstance.sendHandler.mockClear();

            button.click();

            expect(mockTransportInstance.sendHandler).toHaveBeenCalledWith('c0:h0', {
                cseq: 1,
            });
        });

        it('should increment cseq for each event', () => {
            const button = document.createElement('button');
            root.appendChild(button);

            const boot: Boot = {
                t: 'boot',
                sid: 'test-session',
                ver: 1,
                seq: 1,
                patch: [{
                    seq: 0,
                    path: [0],
                    op: 'setHandlers',
                    value: [{ event: 'click', handler: 'c0:h0' }]
                }],
                location: { path: '/', query: {}, hash: '' },
            };

            runtime.handleBoot(boot);
            mockTransportInstance.sendHandler.mockClear();

            button.click();
            button.click();
            button.click();

            const calls = mockTransportInstance.sendHandler.mock.calls;
            expect(calls[0][1].cseq).toBe(1);
            expect(calls[1][1].cseq).toBe(2);
            expect(calls[2][1].cseq).toBe(3);
        });
    });

    describe('ref tracking', () => {
        it('should track refs from patches', () => {
            const child = document.createElement('span');
            root.appendChild(child);

            const boot: Boot = {
                t: 'boot',
                sid: 'test-session',
                ver: 1,
                seq: 1,
                patch: [{ seq: 0, path: [0], op: 'setRef', value: 'myRef' }],
                location: { path: '/', query: {}, hash: '' },
            };

            runtime.handleBoot(boot);
        });

        it('should delete refs from patches', () => {
            const child = document.createElement('span');
            root.appendChild(child);

            const setRefBoot: Boot = {
                t: 'boot',
                sid: 'test-session',
                ver: 1,
                seq: 1,
                patch: [{ seq: 0, path: [0], op: 'setRef', value: 'myRef' }],
                location: { path: '/', query: {}, hash: '' },
            };
            runtime.handleBoot(setRefBoot);

            const delRefBoot: Boot = {
                t: 'boot',
                sid: 'test-session',
                ver: 1,
                seq: 2,
                patch: [{ seq: 0, path: [0], op: 'delRef', value: 'myRef' }],
                location: { path: '/', query: {}, hash: '' },
            };
            runtime.handleBoot(delRefBoot);
        });
    });

    describe('script handling', () => {
        it('should execute scripts from patches', async () => {
            const child = document.createElement('div');
            root.appendChild(child);

            const boot: Boot = {
                t: 'boot',
                sid: 'test-session',
                ver: 1,
                seq: 1,
                patch: [{
                    seq: 0,
                    path: [0],
                    op: 'setScript',
                    value: {
                        scriptId: 'script-1',
                        script: '(el, t) => { el.textContent = "script executed"; }'
                    }
                }],
                location: { path: '/', query: {}, hash: '' },
            };

            runtime.handleBoot(boot);

            await vi.waitFor(() => {
                expect(child.textContent).toBe('script executed');
            });
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
            location: { path: '/', query: {}, hash: '' },
            debug: true,
        };

        new Runtime(config);
    });
});

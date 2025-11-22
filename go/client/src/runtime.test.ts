import {describe, it, expect, beforeEach, vi, Mock} from 'vitest';
import {Runtime, RuntimeConfig} from './runtime';
import {Boot, Frame, Init, DOMRequest, ServerMessage} from './protocol';
import {Transport} from './transport';

vi.mock('./transport');
vi.mock('./router', () => ({
    Router: vi.fn().mockImplementation(() => ({
        navigate: vi.fn(),
        destroy: vi.fn()
    }))
}));

describe('Runtime', () => {
    let root: HTMLElement;
    let config: RuntimeConfig;
    let runtime: Runtime;
    let mockTransportInstance: {
        connect: Mock;
        disconnect: Mock;
        send: Mock;
        onMessage: Mock;
        onStateChange: Mock;
    };
    let messageHandler: (msg: ServerMessage) => void;

    beforeEach(() => {
        vi.clearAllMocks();

        mockTransportInstance = {
            connect: vi.fn(),
            disconnect: vi.fn(),
            send: vi.fn(),
            onMessage: vi.fn(),
            onStateChange: vi.fn()
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
            location: {path: '/', q: '', hash: ''},
            debug: false
        };

        runtime = new Runtime(config);
        messageHandler = mockTransportInstance.onMessage.mock.calls[0][0];
    });

    describe('constructor', () => {
        it('should create runtime with config', () => {
            expect(runtime).toBeDefined();
        });

        it('should wire transport message handler', () => {
            expect(mockTransportInstance.onMessage).toHaveBeenCalled();
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

    describe('handleMessage', () => {
        describe('boot', () => {
            it('should send ack on boot', () => {
                const boot: Boot = {
                    t: 'boot',
                    sid: 'test-session',
                    ver: 1,
                    seq: 5,
                    patch: [],
                    location: {path: '/', q: '', hash: ''}
                };

                messageHandler(boot);

                expect(mockTransportInstance.send).toHaveBeenCalledWith({
                    t: 'ack',
                    sid: 'test-session',
                    seq: 5
                });
            });
        });

        describe('init', () => {
            it('should send ack on init', () => {
                const init: Init = {
                    t: 'init',
                    sid: 'test-session',
                    ver: 2,
                    seq: 10,
                    location: {path: '/new', q: '', hash: ''}
                };

                messageHandler(init);

                expect(mockTransportInstance.send).toHaveBeenCalledWith({
                    t: 'ack',
                    sid: 'test-session',
                    seq: 10
                });
            });
        });

        describe('frame', () => {
            it('should send ack on frame', () => {
                const frame: Frame = {
                    t: 'frame',
                    sid: 'test-session',
                    seq: 15,
                    ver: 1,
                    patch: [],
                    effects: [],
                    metrics: {renderMs: 1, ops: 0}
                };

                messageHandler(frame);

                expect(mockTransportInstance.send).toHaveBeenCalledWith({
                    t: 'ack',
                    sid: 'test-session',
                    seq: 15
                });
            });

            it('should handle server push navigation', () => {
                const pushStateSpy = vi.spyOn(window.history, 'pushState');

                const frame: Frame = {
                    t: 'frame',
                    sid: 'test-session',
                    seq: 16,
                    ver: 1,
                    patch: [],
                    effects: [],
                    nav: {push: '/new-path'},
                    metrics: {renderMs: 1, ops: 0}
                };

                messageHandler(frame);

                expect(pushStateSpy).toHaveBeenCalledWith({}, '', '/new-path');
                pushStateSpy.mockRestore();
            });

            it('should handle server replace navigation', () => {
                const replaceStateSpy = vi.spyOn(window.history, 'replaceState');

                const frame: Frame = {
                    t: 'frame',
                    sid: 'test-session',
                    seq: 17,
                    ver: 1,
                    patch: [],
                    effects: [],
                    nav: {replace: '/replaced'},
                    metrics: {renderMs: 1, ops: 0}
                };

                messageHandler(frame);

                expect(replaceStateSpy).toHaveBeenCalledWith({}, '', '/replaced');
                replaceStateSpy.mockRestore();
            });
        });

        describe('resume_ok', () => {
            it('should handle resume_ok', () => {
                const resumeOk = {
                    t: 'resume_ok' as const,
                    sid: 'test-session',
                    from: 5,
                    to: 10
                };

                expect(() => messageHandler(resumeOk)).not.toThrow();
            });
        });

        describe('event_ack', () => {
            it('should handle event_ack', () => {
                const eventAck = {
                    t: 'evt_ack' as const,
                    sid: 'test-session',
                    cseq: 1
                };

                expect(() => messageHandler(eventAck)).not.toThrow();
            });
        });

        describe('error', () => {
            it('should handle server error', () => {
                const error = {
                    t: 'error' as const,
                    sid: 'test-session',
                    code: 'ERR001',
                    message: 'Something went wrong'
                };

                expect(() => messageHandler(error)).not.toThrow();
            });
        });

        describe('diagnostic', () => {
            it('should handle diagnostic', () => {
                const diagnostic = {
                    t: 'diagnostic' as const,
                    sid: 'test-session',
                    code: 'WARN001',
                    message: 'Warning message'
                };

                expect(() => messageHandler(diagnostic)).not.toThrow();
            });
        });

        describe('dom_req', () => {
            it('should handle dom request for missing ref', () => {
                const domReq: DOMRequest = {
                    t: 'dom_req',
                    id: 'req-1',
                    ref: 'nonexistent',
                    props: ['value']
                };

                messageHandler(domReq);

                expect(mockTransportInstance.send).toHaveBeenCalledWith({
                    t: 'dom_res',
                    sid: 'test-session',
                    id: 'req-1',
                    error: 'ref not found: nonexistent'
                });
            });
        });
    });
});

describe('Runtime with patcher integration', () => {
    let root: HTMLElement;
    let config: RuntimeConfig;
    let mockTransportInstance: {
        connect: Mock;
        disconnect: Mock;
        send: Mock;
        onMessage: Mock;
        onStateChange: Mock;
    };
    let messageHandler: (msg: ServerMessage) => void;

    beforeEach(() => {
        vi.clearAllMocks();

        mockTransportInstance = {
            connect: vi.fn(),
            disconnect: vi.fn(),
            send: vi.fn(),
            onMessage: vi.fn(),
            onStateChange: vi.fn()
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
            location: {path: '/', q: '', hash: ''},
            debug: false
        };

        new Runtime(config);
        messageHandler = mockTransportInstance.onMessage.mock.calls[0][0];
    });

    it('should apply patches from frame', () => {
        const textNode = document.createTextNode('old');
        root.appendChild(textNode);

        const frame: Frame = {
            t: 'frame',
            sid: 'test-session',
            seq: 1,
            ver: 1,
            patch: [{seq: 0, path: [0], op: 'setText', value: 'new'}],
            effects: [],
            metrics: {renderMs: 1, ops: 1}
        };

        messageHandler(frame);

        expect(textNode.textContent).toBe('new');
    });

    it('should track refs from patches', () => {
        const child = document.createElement('span');
        root.appendChild(child);

        const frame: Frame = {
            t: 'frame',
            sid: 'test-session',
            seq: 1,
            ver: 1,
            patch: [{seq: 0, path: [0], op: 'setRef', value: 'myRef'}],
            effects: [],
            metrics: {renderMs: 1, ops: 1}
        };

        messageHandler(frame);

        const domReq: DOMRequest = {
            t: 'dom_req',
            id: 'req-1',
            ref: 'myRef',
            props: ['tagName']
        };

        messageHandler(domReq);

        expect(mockTransportInstance.send).toHaveBeenLastCalledWith({
            t: 'dom_res',
            sid: 'test-session',
            id: 'req-1',
            values: {tagName: 'SPAN'}
        });
    });

    it('should delete refs from patches', () => {
        const child = document.createElement('span');
        root.appendChild(child);

        const setRefFrame: Frame = {
            t: 'frame',
            sid: 'test-session',
            seq: 1,
            ver: 1,
            patch: [{seq: 0, path: [0], op: 'setRef', value: 'myRef'}],
            effects: [],
            metrics: {renderMs: 1, ops: 1}
        };
        messageHandler(setRefFrame);

        const delRefFrame: Frame = {
            t: 'frame',
            sid: 'test-session',
            seq: 2,
            ver: 1,
            patch: [{seq: 0, path: [0], op: 'delRef', value: 'myRef'}],
            effects: [],
            metrics: {renderMs: 1, ops: 1}
        };
        messageHandler(delRefFrame);

        const domReq: DOMRequest = {
            t: 'dom_req',
            id: 'req-1',
            ref: 'myRef',
            props: ['tagName']
        };
        messageHandler(domReq);

        expect(mockTransportInstance.send).toHaveBeenLastCalledWith({
            t: 'dom_res',
            sid: 'test-session',
            id: 'req-1',
            error: 'ref not found: myRef'
        });
    });
});

describe('Runtime event handling', () => {
    let root: HTMLElement;
    let config: RuntimeConfig;
    let mockTransportInstance: {
        connect: Mock;
        disconnect: Mock;
        send: Mock;
        onMessage: Mock;
        onStateChange: Mock;
    };
    let messageHandler: (msg: ServerMessage) => void;

    beforeEach(() => {
        vi.clearAllMocks();

        mockTransportInstance = {
            connect: vi.fn(),
            disconnect: vi.fn(),
            send: vi.fn(),
            onMessage: vi.fn(),
            onStateChange: vi.fn()
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
            location: {path: '/', q: '', hash: ''},
            debug: false
        };

        new Runtime(config);
        messageHandler = mockTransportInstance.onMessage.mock.calls[0][0];
    });

    it('should send events when handler is triggered', () => {
        const button = document.createElement('button');
        root.appendChild(button);

        const frame: Frame = {
            t: 'frame',
            sid: 'test-session',
            seq: 1,
            ver: 1,
            patch: [{
                seq: 0,
                path: [0],
                op: 'setHandlers',
                value: [{event: 'click', handler: 'handleClick', props: []}]
            }],
            effects: [],
            metrics: {renderMs: 1, ops: 1}
        };

        messageHandler(frame);
        mockTransportInstance.send.mockClear();

        button.click();

        expect(mockTransportInstance.send).toHaveBeenCalledWith({
            t: 'evt',
            sid: 'test-session',
            hid: 'handleClick',
            cseq: 1,
            payload: {}
        });
    });

    it('should increment cseq for each event', () => {
        const button = document.createElement('button');
        root.appendChild(button);

        const frame: Frame = {
            t: 'frame',
            sid: 'test-session',
            seq: 1,
            ver: 1,
            patch: [{
                seq: 0,
                path: [0],
                op: 'setHandlers',
                value: [{event: 'click', handler: 'handleClick', props: []}]
            }],
            effects: [],
            metrics: {renderMs: 1, ops: 1}
        };

        messageHandler(frame);
        mockTransportInstance.send.mockClear();

        button.click();
        button.click();
        button.click();

        const calls = mockTransportInstance.send.mock.calls.filter(
            (call: unknown[]) => (call[0] as {t: string}).t === 'evt'
        );
        expect(calls[0][0].cseq).toBe(1);
        expect(calls[1][0].cseq).toBe(2);
        expect(calls[2][0].cseq).toBe(3);
    });
});

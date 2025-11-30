import { describe, it, expect, vi, beforeEach, Mock } from 'vitest';
import { Transport, TransportConfig } from './transport';
import { Bus } from './bus';
import { PondClient, ChannelState } from '@eleven-am/pondsocket-client';

vi.mock('@eleven-am/pondsocket-client', () => ({
    PondClient: vi.fn(),
    ChannelState: {
        JOINED: 'JOINED',
        STALLED: 'STALLED',
        CLOSED: 'CLOSED',
        JOINING: 'JOINING',
        IDLE: 'IDLE',
    },
}));

describe('Transport', () => {
    let transport: Transport;
    let bus: Bus;
    let mockChannel: {
        join: Mock;
        leave: Mock;
        sendMessage: Mock;
        onMessage: Mock;
        onChannelStateChange: Mock;
    };
    let mockClient: {
        connect: Mock;
        disconnect: Mock;
        createChannel: Mock;
    };
    let messageHandler: (event: string, payload: unknown) => void;
    let stateHandler: (state: ChannelState) => void;

    beforeEach(() => {
        vi.clearAllMocks();

        mockChannel = {
            join: vi.fn(),
            leave: vi.fn(),
            sendMessage: vi.fn(),
            onMessage: vi.fn((handler) => {
                messageHandler = handler;
            }),
            onChannelStateChange: vi.fn((handler) => {
                stateHandler = handler;
            }),
        };

        mockClient = {
            connect: vi.fn(),
            disconnect: vi.fn(),
            createChannel: vi.fn().mockReturnValue(mockChannel),
        };

        (PondClient as unknown as Mock).mockImplementation(() => mockClient);

        bus = new Bus();
        const config: TransportConfig = {
            endpoint: '/live',
            sessionId: 'test-session',
            version: 1,
            lastAck: 0,
            location: { path: '/', query: {}, hash: '' },
            bus,
        };

        transport = new Transport(config);
    });

    describe('constructor', () => {
        it('should create PondClient with endpoint', () => {
            expect(PondClient).toHaveBeenCalledWith('/live');
        });

        it('should create channel with session id and join payload', () => {
            expect(mockClient.createChannel).toHaveBeenCalledWith('live/test-session', {
                sid: 'test-session',
                ver: 1,
                ack: 0,
                loc: { path: '/', query: {}, hash: '' },
            });
        });

        it('should wire message handler', () => {
            expect(mockChannel.onMessage).toHaveBeenCalled();
        });

        it('should wire state change handler', () => {
            expect(mockChannel.onChannelStateChange).toHaveBeenCalled();
        });
    });

    describe('sid', () => {
        it('should return session id', () => {
            expect(transport.sid).toBe('test-session');
        });
    });

    describe('connectionState', () => {
        it('should return current state', () => {
            expect(transport.connectionState).toBe('disconnected');
        });
    });

    describe('connect', () => {
        it('should join channel and connect client', () => {
            transport.connect();

            expect(mockChannel.join).toHaveBeenCalled();
            expect(mockClient.connect).toHaveBeenCalled();
        });

        it('should set state to connecting', () => {
            transport.connect();

            expect(transport.connectionState).toBe('connecting');
        });
    });

    describe('disconnect', () => {
        it('should leave channel and disconnect client', () => {
            transport.disconnect();

            expect(mockChannel.leave).toHaveBeenCalled();
            expect(mockClient.disconnect).toHaveBeenCalled();
        });

        it('should set state to disconnected', () => {
            transport.connect();
            transport.disconnect();

            expect(transport.connectionState).toBe('disconnected');
        });
    });

    describe('onStateChange', () => {
        it('should notify listeners on state change', () => {
            const listener = vi.fn();
            transport.onStateChange(listener);

            transport.connect();

            expect(listener).toHaveBeenCalledWith('connecting');
        });

        it('should return unsubscribe function', () => {
            const listener = vi.fn();
            const unsub = transport.onStateChange(listener);

            unsub();
            transport.connect();

            expect(listener).not.toHaveBeenCalled();
        });

        it('should handle JOINED state', () => {
            const listener = vi.fn();
            transport.onStateChange(listener);

            stateHandler(ChannelState.JOINED);

            expect(listener).toHaveBeenCalledWith('connected');
        });

        it('should handle STALLED state', () => {
            const listener = vi.fn();
            transport.onStateChange(listener);

            stateHandler(ChannelState.STALLED);

            expect(listener).toHaveBeenCalledWith('stalled');
        });

        it('should handle CLOSED state', () => {
            const listener = vi.fn();
            transport.onStateChange(listener);

            stateHandler(ChannelState.CLOSED);

            expect(listener).toHaveBeenCalledWith('disconnected');
        });
    });

    describe('send', () => {
        it('should send event message', () => {
            transport.send('dom', 'response', { requestId: 'req-1', result: 'ok' });

            expect(mockChannel.sendMessage).toHaveBeenCalledWith('evt', {
                t: 'dom',
                sid: 'test-session',
                a: 'response',
                p: { requestId: 'req-1', result: 'ok' },
            });
        });
    });

    describe('sendAck', () => {
        it('should send ack message', () => {
            transport.sendAck(10);

            expect(mockChannel.sendMessage).toHaveBeenCalledWith('ack', {
                t: 'ack',
                sid: 'test-session',
                seq: 10,
            });
        });
    });

    describe('sendHandler', () => {
        it('should send handler event', () => {
            transport.sendHandler('c0:h0', { cseq: 1, value: 'test' });

            expect(mockChannel.sendMessage).toHaveBeenCalledWith('evt', {
                t: 'c0:h0',
                sid: 'test-session',
                a: 'invoke',
                p: { cseq: 1, value: 'test' },
            });
        });
    });

    describe('message handling', () => {
        it('should publish frame patch to bus', () => {
            const callback = vi.fn();
            bus.subscribe('frame', 'patch', callback);

            messageHandler('message', {
                seq: 1,
                topic: 'frame',
                event: 'patch',
                data: { seq: 1, patches: [] },
            });

            expect(callback).toHaveBeenCalledWith({ seq: 1, patches: [] });
        });

        it('should publish router push to bus', () => {
            const callback = vi.fn();
            bus.subscribe('router', 'push', callback);

            messageHandler('message', {
                seq: 1,
                topic: 'router',
                event: 'push',
                data: { path: '/new', query: '', hash: '', replace: false },
            });

            expect(callback).toHaveBeenCalledWith({ path: '/new', query: '', hash: '', replace: false });
        });

        it('should publish router replace to bus', () => {
            const callback = vi.fn();
            bus.subscribe('router', 'replace', callback);

            messageHandler('message', {
                seq: 1,
                topic: 'router',
                event: 'replace',
                data: { path: '/replaced', query: '', hash: '', replace: true },
            });

            expect(callback).toHaveBeenCalledWith({ path: '/replaced', query: '', hash: '', replace: true });
        });

        it('should publish dom call to bus', () => {
            const callback = vi.fn();
            bus.subscribe('dom', 'call', callback);

            messageHandler('message', {
                seq: 1,
                topic: 'dom',
                event: 'call',
                data: { ref: 'myRef', method: 'focus' },
            });

            expect(callback).toHaveBeenCalledWith({ ref: 'myRef', method: 'focus' });
        });

        it('should ignore invalid messages', () => {
            const callback = vi.fn();
            bus.subscribe('frame', 'patch', callback);

            messageHandler('message', { invalid: 'message' });

            expect(callback).not.toHaveBeenCalled();
        });

        it('should ignore unknown topics', () => {
            const callback = vi.fn();
            bus.subscribeAll(callback);

            messageHandler('message', {
                seq: 1,
                topic: 'unknown',
                event: 'action',
                data: {},
            });

            expect(callback).not.toHaveBeenCalled();
        });
    });
});

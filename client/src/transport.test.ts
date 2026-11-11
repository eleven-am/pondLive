import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Transport } from './transport';
import { Message } from './protocol';
import { Logger } from './logger';

class FakeWebSocket {
    static instances: FakeWebSocket[] = [];

    onopen: (() => void) | null = null;
    onmessage: ((event: { data: unknown }) => void) | null = null;
    onclose: ((event: { code: number }) => void) | null = null;
    sent: string[] = [];
    closed = false;
    readyState = 1;

    constructor(readonly url: string) {
        FakeWebSocket.instances.push(this);
    }

    send(data: string): void {
        this.sent.push(data);
    }

    close(): void {
        this.closed = true;
        this.onclose?.({ code: 1006 });
    }

    open(): void {
        this.onopen?.();
    }

    receive(frame: unknown): void {
        this.onmessage?.({ data: typeof frame === 'string' ? frame : JSON.stringify(frame) });
    }

    serverClose(code: number): void {
        this.onclose?.({ code });
    }

    frames(): Array<Record<string, unknown>> {
        return this.sent.map((s) => JSON.parse(s));
    }
}

function latest(): FakeWebSocket {
    return FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
}

describe('Transport', () => {
    let messages: Message[];
    let lastSeq: number;
    let resyncing: boolean;
    let transport: Transport;

    beforeEach(() => {
        vi.useFakeTimers();
        FakeWebSocket.instances = [];
        vi.stubGlobal('WebSocket', FakeWebSocket);
        messages = [];
        lastSeq = 0;
        resyncing = false;
        transport = new Transport({
            endpoint: '/live',
            token: 'tok-1',
            sessionId: 'test-session',
            version: 1,
            applied: () => lastSeq,
            wantsReset: () => resyncing,
            onMessage: (msg) => messages.push(msg),
        });
    });

    afterEach(() => {
        transport.disconnect();
        vi.unstubAllGlobals();
        vi.useRealTimers();
        Logger.configure({ enabled: false });
    });

    function joined(): FakeWebSocket {
        transport.connect();
        const ws = latest();
        ws.open();
        ws.receive({ t: 'joined', gen: 1, cs: 0 });
        return ws;
    }

    describe('identity and initial state', () => {
        it('exposes the session id', () => {
            expect(transport.sid).toBe('test-session');
        });

        it('starts disconnected without opening a socket', () => {
            expect(transport.connectionState).toBe('disconnected');
            expect(FakeWebSocket.instances).toHaveLength(0);
        });
    });

    describe('handshake', () => {
        it('opens ws on the page origin at the boot endpoint', () => {
            transport.connect();

            expect(latest().url).toBe('ws://localhost:3000/live');
            expect(transport.connectionState).toBe('connecting');
        });

        it('uses wss when the page is https', () => {
            vi.stubGlobal('window', { location: new URL('https://example.com/app') });

            transport.connect();

            expect(latest().url).toBe('wss://example.com/live');
        });

        it('sends exactly one join frame with sid, tok, ver, bs and resync on open', () => {
            lastSeq = 7;
            transport.connect();
            latest().open();

            expect(latest().frames()).toEqual([
                { t: 'join', sid: 'test-session', tok: 'tok-1', ver: 1, bs: 7, resync: false },
            ]);
            expect(transport.connectionState).toBe('connecting');
        });

        it('becomes connected on joined', () => {
            joined();

            expect(transport.connectionState).toBe('connected');
        });

        it('becomes declined on a declined frame, closes the socket and never reconnects', () => {
            transport.connect();
            const ws = latest();
            ws.open();
            ws.receive({ t: 'declined', code: 4404, reason: 'unknown session' });

            expect(transport.connectionState).toBe('declined');
            expect(ws.closed).toBe(true);
            expect(vi.getTimerCount()).toBe(0);
            vi.advanceTimersByTime(120000);
            expect(FakeWebSocket.instances).toHaveLength(1);
        });

        it('treats a handshake that never completes as a failed attempt', () => {
            transport.connect();
            const ws = latest();
            ws.open();

            vi.advanceTimersByTime(10000);

            expect(ws.closed).toBe(true);
            expect(transport.connectionState).toBe('stalled');
        });

        it('stays stalled and retries when the socket constructor throws', () => {
            vi.stubGlobal('WebSocket', function () {
                throw new Error('bad url');
            });

            transport.connect();

            expect(transport.connectionState).toBe('stalled');
            expect(vi.getTimerCount()).toBe(1);
        });
    });

    describe('close handling', () => {
        it.each([4400, 4401, 4404, 4408, 4409, 4499])('close %i is declined with no reconnect', (code) => {
            const ws = joined();

            ws.serverClose(code);

            expect(transport.connectionState).toBe('declined');
            expect(vi.getTimerCount()).toBe(0);
            vi.advanceTimersByTime(120000);
            expect(FakeWebSocket.instances).toHaveLength(1);
        });

        it.each([1000, 1001, 1006, 1013, 4500, 4399])('close %i stalls and reconnects', (code) => {
            const ws = joined();

            ws.serverClose(code);

            expect(transport.connectionState).toBe('stalled');
            vi.advanceTimersByTime(1000);
            expect(FakeWebSocket.instances).toHaveLength(2);
        });
    });

    describe('reconnect backoff', () => {
        function dropWithoutJoining(): void {
            latest().serverClose(1006);
        }

        it('doubles from 1s to a 30s cap using the full jitter ceiling', () => {
            vi.spyOn(Math, 'random').mockReturnValue(1);
            transport.connect();

            for (const ceiling of [1000, 2000, 4000, 8000, 16000, 30000, 30000]) {
                const before = FakeWebSocket.instances.length;
                dropWithoutJoining();
                vi.advanceTimersByTime(ceiling - 1);
                expect(FakeWebSocket.instances).toHaveLength(before);
                vi.advanceTimersByTime(1);
                expect(FakeWebSocket.instances).toHaveLength(before + 1);
            }
        });

        it('draws the delay uniformly from zero to the ceiling', () => {
            const random = vi.spyOn(Math, 'random').mockReturnValue(0.5);
            transport.connect();

            dropWithoutJoining();
            vi.advanceTimersByTime(499);
            expect(FakeWebSocket.instances).toHaveLength(1);
            vi.advanceTimersByTime(1);
            expect(FakeWebSocket.instances).toHaveLength(2);

            random.mockReturnValue(0);
            dropWithoutJoining();
            vi.advanceTimersByTime(0);
            expect(FakeWebSocket.instances).toHaveLength(3);
        });

        it('resets the backoff after a successful join', () => {
            vi.spyOn(Math, 'random').mockReturnValue(1);
            transport.connect();
            dropWithoutJoining();
            vi.advanceTimersByTime(1000);
            dropWithoutJoining();
            vi.advanceTimersByTime(2000);

            latest().open();
            latest().receive({ t: 'joined', gen: 1 });
            latest().serverClose(1006);

            const before = FakeWebSocket.instances.length;
            vi.advanceTimersByTime(999);
            expect(FakeWebSocket.instances).toHaveLength(before);
            vi.advanceTimersByTime(1);
            expect(FakeWebSocket.instances).toHaveLength(before + 1);
        });

        it('reports connecting while the retry socket opens', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            const ws = joined();
            ws.serverClose(1006);
            expect(transport.connectionState).toBe('stalled');

            vi.advanceTimersByTime(0);

            expect(transport.connectionState).toBe('connecting');
        });
    });

    describe('rejoin base seq', () => {
        it('sends the current applied seq, not the one from the first join', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            lastSeq = 3;
            const first = joined();
            expect(first.frames()[0].bs).toBe(3);

            lastSeq = 42;
            first.serverClose(1006);
            vi.advanceTimersByTime(0);
            latest().open();

            expect(latest().frames()[0]).toMatchObject({ t: 'join', bs: 42, resync: false });
        });

        it('asks for a reset when the sequencer is resyncing', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            const first = joined();
            resyncing = true;
            first.serverClose(1006);
            vi.advanceTimersByTime(0);
            latest().open();

            expect(latest().frames()[0]).toMatchObject({ t: 'join', resync: true });
        });
    });

    describe('token refresh', () => {
        it('rejoins with the token from the latest joined frame, not the boot token', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            transport.connect();
            latest().open();
            expect(latest().frames()[0].tok).toBe('tok-1');
            latest().receive({ t: 'joined', gen: 1, tok: 'tok-2' });

            latest().serverClose(1006);
            vi.advanceTimersByTime(0);
            latest().open();

            expect(latest().frames()[0].tok).toBe('tok-2');
        });

        it('rejoins with the token from a tok refresh frame and never delivers it as a message', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            transport.connect();
            latest().open();
            latest().receive({ t: 'joined', gen: 1, tok: 'tok-2' });
            latest().receive({ t: 'tok', tok: 'tok-3' });

            latest().serverClose(1006);
            vi.advanceTimersByTime(0);
            latest().open();

            expect(latest().frames()[0].tok).toBe('tok-3');
            expect(messages).toHaveLength(0);
        });

        it('keeps the previous token when joined carries none', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            const ws = joined();

            ws.serverClose(1006);
            vi.advanceTimersByTime(0);
            latest().open();

            expect(latest().frames()[0].tok).toBe('tok-1');
        });
    });

    describe('disconnect', () => {
        it('moves to disconnected and closes the socket', () => {
            const ws = joined();

            transport.disconnect();

            expect(ws.closed).toBe(true);
            expect(transport.connectionState).toBe('disconnected');
        });

        it('cancels a pending reconnect timer', () => {
            const ws = joined();
            ws.serverClose(1006);
            expect(vi.getTimerCount()).toBe(1);

            transport.disconnect();

            expect(vi.getTimerCount()).toBe(0);
            vi.advanceTimersByTime(120000);
            expect(FakeWebSocket.instances).toHaveLength(1);
            expect(transport.connectionState).toBe('disconnected');
        });

        it('cancels the handshake timer so a stale timeout cannot fire', () => {
            transport.connect();

            transport.disconnect();

            expect(vi.getTimerCount()).toBe(0);
        });

        it('ignores late events from the closed socket', () => {
            const ws = joined();
            transport.disconnect();

            ws.serverClose(1006);
            ws.receive({ seq: 1, topic: 'frame', event: 'patch', data: [] });

            expect(vi.getTimerCount()).toBe(0);
            expect(FakeWebSocket.instances).toHaveLength(1);
            expect(transport.connectionState).toBe('disconnected');
        });

        it('allows connect again after disconnect', () => {
            joined();
            transport.disconnect();

            transport.connect();

            expect(FakeWebSocket.instances).toHaveLength(2);
            expect(transport.connectionState).toBe('connecting');
        });
    });

    describe('connect idempotence', () => {
        it('keeps a single socket across repeated calls while connecting', () => {
            transport.connect();
            transport.connect();

            expect(FakeWebSocket.instances).toHaveLength(1);
        });

        it('keeps a single socket once joined', () => {
            joined();
            transport.connect();

            expect(FakeWebSocket.instances).toHaveLength(1);
        });

        it('does not open a second socket while a reconnect is pending', () => {
            const ws = joined();
            ws.serverClose(1006);

            transport.connect();
            vi.advanceTimersByTime(1000);

            expect(FakeWebSocket.instances).toHaveLength(2);
        });
    });

    describe('state listeners', () => {
        it('notifies listeners on change only', () => {
            const listener = vi.fn();
            transport.onStateChange(listener);

            transport.connect();
            latest().open();
            latest().receive({ t: 'joined', gen: 1 });
            latest().receive({ t: 'joined', gen: 1 });

            expect(listener.mock.calls.map((c) => c[0])).toEqual(['connecting', 'connected']);
        });

        it('stops notifying after unsubscribe', () => {
            const listener = vi.fn();
            const unsub = transport.onStateChange(listener);

            unsub();
            transport.connect();

            expect(listener).not.toHaveBeenCalled();
        });

        it('keeps notifying other listeners when one throws', () => {
            const second = vi.fn();
            transport.onStateChange(() => {
                throw new Error('boom');
            });
            transport.onStateChange(second);

            transport.connect();

            expect(second).toHaveBeenCalledWith('connecting');
        });
    });

    describe('outbound', () => {
        it('sends evt frames immediately once joined with a client seq', () => {
            const ws = joined();

            transport.send('dom', 'response', { requestId: 'req-1', result: 'ok' });

            expect(ws.frames()[1]).toEqual({
                t: 'dom',
                sid: 'test-session',
                a: 'response',
                p: { requestId: 'req-1', result: 'ok' },
                cs: 1,
            });
        });

        it('sends ack frames', () => {
            const ws = joined();

            transport.sendAck(10);

            expect(ws.frames()[1]).toEqual({ t: 'ack', sid: 'test-session', seq: 10 });
        });

        it('sends resync frames', () => {
            const ws = joined();

            transport.sendResync(4, 'unresolved');

            expect(ws.frames()[1]).toEqual({ t: 'resync', sid: 'test-session', seq: 4, reason: 'unresolved' });
        });

        it('sends handler frames', () => {
            const ws = joined();

            transport.sendHandler(3, 'click', { value: 'test' });

            expect(ws.frames()[1]).toEqual({
                t: 'h',
                sid: 'test-session',
                a: 'click',
                i: 3,
                p: { value: 'test' },
                cs: 1,
            });
        });

        it('sends script frames', () => {
            const ws = joined();

            transport.sendScript('s1', { scriptId: 's1', event: 'ready', data: 1 });

            expect(ws.frames()[1]).toEqual({
                t: 'script:s1',
                sid: 'test-session',
                a: 'message',
                p: { scriptId: 's1', event: 'ready', data: 1 },
                cs: 1,
            });
        });

        it('numbers events monotonically across topics', () => {
            const ws = joined();

            transport.sendHandler(3, 'click', {});
            transport.send('router', 'popstate', { path: '/', query: '', hash: '' });
            transport.sendScript('s1', { scriptId: 's1', event: 'e' });

            expect(ws.frames().slice(1).map((f) => f.cs)).toEqual([1, 2, 3]);
        });

        it('does not queue ack or resync frames while disconnected', () => {
            transport.sendAck(1);
            transport.sendResync(1, 'gap');
            transport.connect();
            const ws = latest();
            ws.open();
            transport.sendAck(2);
            ws.receive({ t: 'joined', gen: 1, cs: 0 });

            expect(ws.frames().map((f) => f.t)).toEqual(['join']);
        });

        it('queues events before joined and sends them in order right after joined', () => {
            transport.sendHandler(3, 'click', { n: 1 });
            transport.sendHandler(3, 'click', { n: 2 });
            transport.connect();
            const ws = latest();
            ws.open();
            transport.sendHandler(3, 'click', { n: 3 });
            expect(ws.sent).toHaveLength(1);

            ws.receive({ t: 'joined', gen: 1, cs: 0 });

            expect(ws.frames().slice(1).map((f) => [f.cs, (f.p as { n: number }).n])).toEqual([[1, 1], [2, 2], [3, 3]]);
        });

        it('keeps cs monotonic across reconnects', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            const first = joined();
            transport.sendHandler(3, 'click', {});
            first.receive({ t: 'evack', cs: 1 });
            first.serverClose(1006);
            transport.sendHandler(3, 'click', {});

            vi.advanceTimersByTime(0);
            latest().open();
            latest().receive({ t: 'joined', gen: 2, cs: 1 });
            transport.sendHandler(3, 'click', {});

            expect(latest().frames().slice(1).map((f) => f.cs)).toEqual([2, 3]);
        });

        it('trims on joined.cs and resends the rest in order', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            const first = joined();
            for (let i = 1; i <= 4; i++) transport.sendHandler(3, 'click', { n: i });
            first.serverClose(1006);

            vi.advanceTimersByTime(0);
            latest().open();
            latest().receive({ t: 'joined', gen: 2, cs: 2 });

            expect(latest().frames().slice(1).map((f) => f.cs)).toEqual([3, 4]);
        });

        it('resends everything the server never accepted', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            const first = joined();
            transport.sendHandler(3, 'click', {});
            transport.sendHandler(3, 'click', {});
            first.serverClose(1006);

            vi.advanceTimersByTime(0);
            latest().open();
            latest().receive({ t: 'joined', gen: 2, cs: 0 });

            expect(latest().frames().slice(1).map((f) => f.cs)).toEqual([1, 2]);
        });

        it('does not write to a socket that is closing and resends the event after reconnect', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            const first = joined();
            first.readyState = 2;
            transport.sendHandler(3, 'click', {});
            transport.sendAck(1);
            first.serverClose(1006);

            vi.advanceTimersByTime(0);
            latest().open();
            latest().receive({ t: 'joined', gen: 2, cs: 0 });

            expect(first.frames().map((f) => f.t)).toEqual(['join']);
            expect(latest().frames().slice(1).map((f) => f.cs)).toEqual([1]);
        });

        it('evack trims the buffer so a later reconnect does not resend confirmed events', () => {
            vi.spyOn(Math, 'random').mockReturnValue(0);
            const first = joined();
            for (let i = 1; i <= 3; i++) transport.sendHandler(3, 'click', {});
            first.receive({ t: 'evack', cs: 2 });
            first.serverClose(1006);

            vi.advanceTimersByTime(0);
            latest().open();
            latest().receive({ t: 'joined', gen: 2, cs: 0 });

            expect(latest().frames().slice(1).map((f) => f.cs)).toEqual([3]);
        });

        it('drops buffered events when declined', () => {
            transport.connect();
            transport.sendHandler(3, 'click', {});
            latest().open();
            latest().receive({ t: 'declined', code: 4401, reason: 'bad token' });

            transport.connect();
            latest().open();
            latest().receive({ t: 'joined', gen: 1, cs: 0 });

            expect(latest().frames().map((f) => f.t)).toEqual(['join']);
        });
    });

    describe('resend buffer bounds', () => {
        function flushedCS(): number[] {
            const ws = latest();
            ws.open();
            ws.receive({ t: 'joined', gen: 1, cs: 0 });
            return ws.frames().filter((f) => f.t !== 'join').map((f) => f.cs as number);
        }

        it('keeps at most 100 events and drops the oldest', () => {
            transport.connect();
            for (let i = 1; i <= 105; i++) {
                transport.sendHandler(3, 'click', {});
            }

            const sent = flushedCS();

            expect(sent).toHaveLength(100);
            expect(sent[0]).toBe(6);
            expect(sent[99]).toBe(105);
        });

        it('keeps at most 1 MiB and drops the oldest', () => {
            transport.connect();
            const chunk = 'x'.repeat(300 * 1024);
            for (let i = 1; i <= 5; i++) {
                transport.send('dom', 'response', { requestId: String(i), result: chunk });
            }
            const ws = latest();
            ws.open();
            ws.receive({ t: 'joined', gen: 1, cs: 0 });

            const ids = ws.frames().filter((f) => f.t === 'dom').map((f) => (f.p as { requestId: string }).requestId);

            expect(ids).toEqual(['3', '4', '5']);
        });

        it('drops a single message larger than the budget instead of stalling the buffer', () => {
            transport.connect();
            transport.sendHandler(3, 'click', {});
            transport.send('dom', 'response', { requestId: 'big', result: 'x'.repeat(2 * 1024 * 1024) });

            const ws = latest();
            ws.open();
            ws.receive({ t: 'joined', gen: 1, cs: 0 });

            expect(ws.frames().filter((f) => f.t === 'dom')).toHaveLength(0);
        });

        it('logs the overflow once however many events are dropped', () => {
            Logger.configure({ enabled: true, level: 'warn' });
            const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
            transport.connect();

            for (let i = 1; i <= 150; i++) {
                transport.sendHandler(3, 'click', {});
            }

            expect(warn).toHaveBeenCalledTimes(1);
        });
    });

    describe('inbound robustness', () => {
        it('ignores malformed JSON without throwing', () => {
            const ws = joined();

            expect(() => ws.receive('{not json')).not.toThrow();
            expect(() => ws.receive('')).not.toThrow();

            expect(messages).toHaveLength(0);
            expect(transport.connectionState).toBe('connected');
        });

        it('ignores non-text frames', () => {
            const ws = joined();

            expect(() => ws.onmessage?.({ data: new ArrayBuffer(4) })).not.toThrow();
            expect(transport.connectionState).toBe('connected');
        });

        it('ignores application frames that arrive before joined', () => {
            transport.connect();
            const ws = latest();
            ws.open();

            ws.receive({ seq: 1, topic: 'frame', event: 'patch', data: [] });

            expect(messages).toHaveLength(0);
        });

        it('ignores a declined frame without a numeric code', () => {
            const ws = joined();

            ws.receive({ t: 'declined' });

            expect(transport.connectionState).toBe('connected');
        });
    });

    describe('message handling', () => {
        it.each([
            ['frame', 'patch', []],
            ['frame', 'reset', { html: '<html></html>', meta: [], location: { path: '/', query: {}, hash: '' }, cause: 'rejoin-gap' }],
            ['router', 'push', { path: '/new', query: '', hash: '', replace: false }],
            ['dom', 'call', { ref: 'myRef', method: 'focus' }],
            ['script:s1', 'send', { scriptId: 's1', event: 'start' }],
        ])('hands %s/%s to onMessage untouched', (topic, event, data) => {
            const ws = joined();

            ws.receive({ seq: 1, topic, event, data });

            expect(messages).toEqual([{ seq: 1, topic, event, data }]);
        });

        it('ignores invalid messages', () => {
            const ws = joined();

            ws.receive({ invalid: 'message' });

            expect(messages).toHaveLength(0);
        });

        it('ignores unknown topics and the removed ack topic', () => {
            const ws = joined();

            ws.receive({ seq: 1, topic: 'unknown', event: 'action', data: {} });
            ws.receive({ seq: 2, topic: 'ack', event: 'ack', data: { seq: 1 } });

            expect(messages).toHaveLength(0);
        });

        it('does not treat evack as a message', () => {
            const ws = joined();

            ws.receive({ t: 'evack', cs: 1 });

            expect(messages).toHaveLength(0);
        });
    });
});

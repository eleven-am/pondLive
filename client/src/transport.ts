import {
    Topic,
    EventTopic,
    ActionFor,
    PayloadFor,
    ClientEvt,
    ClientAck,
    ClientResync,
    ResyncReason,
    Message,
    isMessage,
    JoinFrame,
    isServerFrame,
    HandlerEventPayload,
    SCRIPT_TOPIC,
} from './protocol';
import { STATIC_TOPICS } from './wire';
import {Logger} from "./logger";

const BACKOFF_BASE_MS = 1000;
const BACKOFF_CAP_MS = 30000;
const HANDSHAKE_TIMEOUT_MS = 10000;
const QUEUE_MAX_MESSAGES = 100;
const QUEUE_MAX_BYTES = 1024 * 1024;
const DECLINED_CLOSE_MIN = 4400;
const DECLINED_CLOSE_MAX = 4499;

export interface TransportConfig {
    endpoint: string;
    token: string;
    sessionId: string;
    version: number;
    bundle?: string;
    applied: () => number;
    wantsReset: () => boolean;
    onMessage: (msg: Message) => void;
}

export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'stalled' | 'declined';

interface UnconfirmedEvent {
    cs: number;
    data: string;
    bytes: number;
}

const utf8 = new TextEncoder();

export class Transport {
    private readonly endpoint: string;
    private token: string;
    private readonly version: number;
    private readonly bundle: string | undefined;
    private readonly applied: () => number;
    private readonly wantsReset: () => boolean;
    private readonly onMessage: (msg: Message) => void;
    private readonly sessionId: string;
    private state: ConnectionState = 'disconnected';
    private stateListeners: Array<(state: ConnectionState) => void> = [];
    private socket: WebSocket | null = null;
    private joined = false;
    private attempts = 0;
    private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    private handshakeTimer: ReturnType<typeof setTimeout> | null = null;
    private unconfirmed: UnconfirmedEvent[] = [];
    private unconfirmedBytes = 0;
    private overflowLogged = false;
    private nextCS = 0;

    constructor(config: TransportConfig) {
        this.endpoint = config.endpoint;
        this.token = config.token;
        this.sessionId = config.sessionId;
        this.version = config.version;
        this.bundle = config.bundle;
        this.applied = config.applied;
        this.wantsReset = config.wantsReset;
        this.onMessage = config.onMessage;
    }

    get sid(): string {
        return this.sessionId;
    }

    get connectionState(): ConnectionState {
        return this.state;
    }

    connect(): void {
        if (this.socket !== null || this.reconnectTimer !== null) {
            return;
        }
        this.open();
    }

    disconnect(): void {
        this.clearTimers();
        const socket = this.socket;
        this.socket = null;
        this.joined = false;
        this.attempts = 0;
        if (socket) {
            socket.close();
        }
        this.setState('disconnected');
    }

    onStateChange(listener: (state: ConnectionState) => void): () => void {
        this.stateListeners.push(listener);
        return () => {
            const idx = this.stateListeners.indexOf(listener);
            if (idx !== -1) {
                this.stateListeners.splice(idx, 1);
            }
        };
    }

    send<T extends Topic, A extends ActionFor<T>>(
        topic: T,
        action: A,
        payload: PayloadFor<T, A>
    ): void {
        this.sendEvent(topic, String(action), payload);
    }

    sendAck(seq: number): void {
        const ack: ClientAck = {
            t: 'ack',
            sid: this.sessionId,
            seq,
        };
        this.sendControl(ack);
    }

    sendResync(seq: number, reason: ResyncReason): void {
        const resync: ClientResync = {
            t: 'resync',
            sid: this.sessionId,
            seq,
            reason,
        };
        this.sendControl(resync);
    }

    sendHandler(id: number, event: string, payload: HandlerEventPayload): void {
        this.sendEvent('h', event, payload, id);
    }

    sendScript(id: number, action: 'm' | 'err', payload: unknown): void {
        this.sendEvent(SCRIPT_TOPIC, action, payload, id);
    }

    private handleMessage(payload: unknown): void {
        if (!isMessage(payload)) {
            return;
        }
        if (!this.isValidTopic(payload.topic)) {
            return;
        }
        this.onMessage(payload);
    }

    private isValidTopic(topic: string): boolean {
        return (STATIC_TOPICS as string[]).includes(topic) || topic === SCRIPT_TOPIC;
    }

    private open(): void {
        this.setState('connecting');
        let socket: WebSocket;
        try {
            socket = new WebSocket(this.url());
        } catch (err) {
            Logger.error('TRANSPORT', 'Failed to open socket', err);
            this.scheduleReconnect();
            return;
        }
        this.socket = socket;
        this.joined = false;
        this.handshakeTimer = setTimeout(() => {
            this.handshakeTimer = null;
            if (this.socket === socket && !this.joined) {
                socket.close();
            }
        }, HANDSHAKE_TIMEOUT_MS);

        socket.onopen = () => {
            if (this.socket !== socket) return;
            const join: JoinFrame & { b?: string } = {
                t: 'join',
                sid: this.sessionId,
                tok: this.token,
                ver: this.version,
                bs: this.applied(),
                resync: this.wantsReset(),
            };
            if (this.bundle !== undefined) join.b = this.bundle;
            socket.send(JSON.stringify(join));
        };
        socket.onmessage = (event: MessageEvent) => {
            if (this.socket !== socket) return;
            this.handleFrame(event.data);
        };
        socket.onclose = (event: CloseEvent) => {
            if (this.socket !== socket) return;
            this.handleClose(event.code);
        };
    }

    private url(): string {
        const url = new URL(this.endpoint, window.location.href);
        url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        return url.toString();
    }

    private handleFrame(raw: unknown): void {
        if (typeof raw !== 'string') {
            Logger.warn('TRANSPORT', 'Ignoring non-text frame');
            return;
        }
        let parsed: unknown;
        try {
            parsed = JSON.parse(raw);
        } catch (err) {
            Logger.warn('TRANSPORT', 'Ignoring malformed frame', err);
            return;
        }
        Logger.info('TRANSPORT', 'Transport received message:', parsed);

        if (isServerFrame(parsed)) {
            if (parsed.t === 'joined') {
                this.handleJoined(parsed.tok, parsed.cs);
            } else if (parsed.t === 'evack') {
                this.confirmThrough(parsed.cs);
            } else if (parsed.t === 'tok') {
                this.token = parsed.tok;
            } else {
                this.handleDeclined(parsed.code, parsed.reason);
            }
            return;
        }
        if (!this.joined) {
            Logger.warn('TRANSPORT', 'Ignoring frame received before joined');
            return;
        }
        this.handleMessage(parsed);
    }

    private handleJoined(token: string | undefined, cs: number): void {
        if (this.joined) return;
        if (typeof token === 'string' && token !== '') {
            this.token = token;
        }
        this.joined = true;
        this.attempts = 0;
        this.clearHandshakeTimer();
        this.setState('connected');
        if (typeof cs === 'number') {
            this.confirmThrough(cs);
        }
        this.resendUnconfirmed();
    }

    private handleDeclined(code: number, reason: string): void {
        Logger.warn('TRANSPORT', 'Join declined', code, reason);
        const socket = this.socket;
        this.socket = null;
        this.terminateDeclined();
        if (socket) {
            socket.close();
        }
    }

    private handleClose(code: number): void {
        this.socket = null;
        this.joined = false;
        this.clearHandshakeTimer();
        if (code >= DECLINED_CLOSE_MIN && code <= DECLINED_CLOSE_MAX) {
            this.terminateDeclined();
            return;
        }
        this.scheduleReconnect();
    }

    private terminateDeclined(): void {
        this.clearTimers();
        this.joined = false;
        this.clearUnconfirmed();
        this.setState('declined');
    }

    private scheduleReconnect(): void {
        this.setState('stalled');
        const ceiling = Math.min(BACKOFF_CAP_MS, BACKOFF_BASE_MS * 2 ** this.attempts);
        const delay = Math.random() * ceiling;
        this.attempts++;
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.open();
        }, delay);
    }

    private clearHandshakeTimer(): void {
        if (this.handshakeTimer !== null) {
            clearTimeout(this.handshakeTimer);
            this.handshakeTimer = null;
        }
    }

    private clearTimers(): void {
        this.clearHandshakeTimer();
        if (this.reconnectTimer !== null) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
    }

    private confirmThrough(cs: number): void {
        let drop = 0;
        while (drop < this.unconfirmed.length && this.unconfirmed[drop].cs <= cs) {
            this.unconfirmedBytes -= this.unconfirmed[drop].bytes;
            drop++;
        }
        if (drop > 0) {
            this.unconfirmed.splice(0, drop);
        }
    }

    private remember(event: UnconfirmedEvent): void {
        this.unconfirmed.push(event);
        this.unconfirmedBytes += event.bytes;
        while (this.unconfirmed.length > QUEUE_MAX_MESSAGES || this.unconfirmedBytes > QUEUE_MAX_BYTES) {
            const dropped = this.unconfirmed.shift();
            if (!dropped) break;
            this.unconfirmedBytes -= dropped.bytes;
            if (!this.overflowLogged) {
                this.overflowLogged = true;
                Logger.warn('TRANSPORT', 'Resend buffer full, dropping oldest events');
            }
        }
    }

    private resendUnconfirmed(): void {
        const socket = this.socket;
        if (!socket) return;
        for (const event of this.unconfirmed) {
            socket.send(event.data);
        }
    }

    private clearUnconfirmed(): void {
        this.unconfirmed = [];
        this.unconfirmedBytes = 0;
        this.overflowLogged = false;
    }

    private setState(state: ConnectionState): void {
        if (this.state === state) return;
        this.state = state;
        for (const listener of [...this.stateListeners]) {
            try {
                listener(state);
            } catch (err) {
                Logger.error('TRANSPORT', 'State listener failed', err);
            }
        }
    }

    private sendEvent(topic: EventTopic | typeof SCRIPT_TOPIC, action: string, payload: unknown, id?: number): void {
        const evt: ClientEvt = {
            t: topic as EventTopic,
            sid: this.sessionId,
            a: action,
            p: payload,
            cs: ++this.nextCS,
        };
        if (id !== undefined) evt.i = id;
        Logger.info('TRANSPORT', 'Transport sending event:', evt);
        const data = JSON.stringify(evt);
        this.remember({ cs: evt.cs, data, bytes: utf8.encode(data).length });
        if (this.isOpen()) {
            this.socket!.send(data);
        }
    }

    private isOpen(): boolean {
        return this.joined && !!this.socket && this.socket.readyState === 1;
    }

    private sendControl(frame: ClientAck | ClientResync): void {
        if (!this.isOpen()) return;
        Logger.info('TRANSPORT', 'Transport sending control:', frame);
        this.socket!.send(JSON.stringify(frame));
    }
}

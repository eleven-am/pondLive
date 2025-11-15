import {ChannelState, PondClient} from '@eleven-am/pondsocket-client';
import {BootLoader} from './boot';
import {TypedEventEmitter} from './emitter';
import {Logger} from './logger';
import type {
    BootPayload,
    ClientAckMessage,
    ClientEventMessage,
    UploadClientMessage,
    ClientNavMessage,
    ClientRecoverMessage,
    ConnectionState,
    DiagnosticMessage,
    DOMRequestMessage,
    DOMResponseMessage,
    ErrorMessage,
    FrameMessage,
    InitMessage,
    JoinMessage,
    LiveUIEventMap,
    LiveUIOptions,
    PubsubControlMessage,
    ResumeMessage,
    TemplateMessage,
    UploadControlMessage,
    UploadMeta,
} from './types';

export type ServerMessage =
  | InitMessage
  | FrameMessage
  | TemplateMessage
  | JoinMessage
  | ResumeMessage
  | ErrorMessage
  | DiagnosticMessage
  | PubsubControlMessage
  | UploadControlMessage
  | DOMRequestMessage;

type PondChannel = ReturnType<PondClient['createChannel']>;

type RuntimeEvents = {
  state: ConnectionState;
  connected: { sid: string; version: number };
  disconnected: void;
  error: { error: Error };
  message: ServerMessage;
  init: InitMessage;
  frame: FrameMessage;
  template: TemplateMessage;
  resume: ResumeMessage;
  join: JoinMessage;
  diagnostic: DiagnosticMessage;
  upload: UploadControlMessage;
  domreq: DOMRequestMessage;
};

export interface RuntimeOptions extends LiveUIOptions {
  reconnect?: boolean;
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
}

type ResolvedOptions = Required<
  Pick<RuntimeOptions, 'endpoint' | 'uploadEndpoint' | 'autoConnect' | 'debug'>
> & {
  reconnect: boolean;
  reconnectDelay: number;
  maxReconnectAttempts: number;
  boot?: BootPayload;
};

const DEFAULT_OPTIONS: ResolvedOptions = {
  endpoint: '/live',
  uploadEndpoint: '/pondlive/upload/',
  autoConnect: true,
  debug: false,
  reconnect: true,
  reconnectDelay: 1000,
  maxReconnectAttempts: 5,
};

export class LiveRuntime {
  private readonly options: ResolvedOptions;
  private readonly events = new TypedEventEmitter<RuntimeEvents>();
  private readonly bootLoader: BootLoader;

  private client: PondClient | null = null;
  private channel: PondChannel | null = null;
  private bootPayload: BootPayload | null = null;
  private connectPromise: Promise<void> | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts = 0;
  private disposed = false;
  private state: ConnectionState = { status: 'disconnected' };
  private lastAck = 0;
  private sessionId: string | null = null;
  private version = 0;

  constructor(options?: RuntimeOptions) {
    this.options = {
      ...DEFAULT_OPTIONS,
      ...(options ?? {}),
    };
    Logger.configure({ debug: this.options.debug });
    this.bootLoader = new BootLoader({ debug: this.options.debug });

    this.bootPayload = this.options.boot
      ? this.bootLoader.load(this.options.boot)
      : this.bootLoader.load();
    this.sessionId = this.bootPayload?.sid ?? null;
    this.version = this.bootPayload?.ver ?? 0;
    Logger.debug('[Runtime]', 'initialized', {
      sessionId: this.sessionId,
      version: this.version,
      hasBoot: Boolean(this.bootPayload),
    });

    if (this.options.autoConnect && this.bootPayload) {
      void this.connect();
    }
  }

  on<K extends keyof RuntimeEvents>(event: K, listener: (payload: RuntimeEvents[K]) => void): () => void {
    return this.events.on(event, listener);
  }

  once<K extends keyof RuntimeEvents>(event: K, listener: (payload: RuntimeEvents[K]) => void): () => void {
    return this.events.once(event, listener);
  }

  off<K extends keyof RuntimeEvents>(event: K, listener: (payload: RuntimeEvents[K]) => void): void {
    this.events.off(event, listener);
  }

  getState(): ConnectionState {
    return this.state;
  }

  getBootPayload(): BootPayload | null {
    return this.bootPayload;
  }

  getSessionId(): string | null {
    return this.sessionId ?? this.bootPayload?.sid ?? null;
  }

  getUploadEndpoint(): string {
    return this.bootPayload?.client?.upload ?? this.options.uploadEndpoint;
  }

  sendUploadMessage(payload: {
    id: string;
    op: 'change' | 'progress' | 'error' | 'cancelled';
    meta?: UploadMeta;
    loaded?: number;
    total?: number;
    error?: string;
  }): void {
    const sid = this.getSessionId();
    if (!this.channel || !sid || !payload.id) {
      return;
    }
    const message: UploadClientMessage = {
      t: 'upload',
      sid,
      id: payload.id,
      op: payload.op,
    };
    if (payload.meta) {
      message.meta = payload.meta;
    }
    if (typeof payload.loaded === 'number') {
      message.loaded = payload.loaded;
    }
    if (typeof payload.total === 'number') {
      message.total = payload.total;
    }
    if (payload.error) {
      message.error = payload.error;
    }
    Logger.debug('[Runtime]', '→ sending upload message', message);
    this.channel.sendMessage('upload', message);
  }

  sendDOMResponse(payload: {
    id: string;
    values?: Record<string, any>;
    result?: any;
    error?: string;
  }): void {
    const sid = this.getSessionId();
    if (!this.channel || !sid || !payload.id) {
      return;
    }
    const message: DOMResponseMessage = {
      t: 'domres',
      sid,
      id: payload.id,
    };
    if (payload.values) {
      message.values = payload.values;
    }
    if (payload.result !== undefined) {
      message.result = payload.result;
    }
    if (payload.error) {
      message.error = payload.error;
    }
    Logger.debug('[Runtime]', '→ sending domres', message);
    this.channel.sendMessage('domres', message);
  }

  async connect(): Promise<void> {
    if (this.disposed) {
      throw new Error('[LiveUI] runtime disposed');
    }
    Logger.debug('[Runtime]', 'connect requested', {
      hasBoot: Boolean(this.bootPayload),
      disposed: this.disposed,
      state: this.state.status,
    });
    if (this.channel && this.state.status === 'connected') {
      return;
    }
    if (this.connectPromise) {
      return this.connectPromise;
    }
    this.connectPromise = new Promise<void>((resolve, reject) => {
      try {
        const boot = this.bootPayload ?? this.bootLoader.load();
        if (!boot || typeof boot.sid !== 'string') {
          throw new Error('[LiveUI] missing boot payload; call load() before connecting');
        }
        this.bootPayload = boot;
        this.updateState({ status: 'connecting' });
        Logger.debug('[Runtime]', 'opening socket', { endpoint: this.options.endpoint });
        const client = new PondClient(this.options.endpoint);
        this.client = client;
        const joinPayload = this.buildJoinPayload(boot);
        Logger.debug('[Runtime]', 'joining channel', {
          sid: boot.sid,
          ack: joinPayload.ack,
          version: joinPayload.ver,
        });
        const channel = client.createChannel<LiveUIEventMap>(`live/${boot.sid}`, joinPayload);
        this.channel = channel;

        channel.onChannelStateChange((state) => {
          Logger.debug('[Runtime]', 'channel state changed', state);
          if (state === ChannelState.JOINED) {
            this.reconnectAttempts = 0;
            this.sessionId = boot.sid;
            this.version = boot.ver ?? 0;
            this.updateState({ status: 'connected', sessionId: boot.sid, version: this.version });
            this.events.emit('connected', { sid: boot.sid, version: this.version });
            Logger.debug('[Runtime]', 'session joined', { sid: boot.sid, version: this.version });
            resolve();
            this.connectPromise = null;
          }
        });

        channel.onMessage((_event, payload) => {
          Logger.debug('[Runtime]', '← received message', payload);
          this.routeMessage(payload as ServerMessage);
        });

        channel.onLeave(() => {
          this.handleChannelLeave();
        });

        client.connect();
        channel.join();
      } catch (error) {
        Logger.debug('[Runtime]', 'connect failed', error);
        this.connectPromise = null;
        this.handleErrorEvent(error as Error);
        reject(error);
      }
    });

    return this.connectPromise;
  }

  disconnect(): void {
    this.clearReconnectTimer();
    this.reconnectAttempts = 0;
    this.connectPromise = null;
    if (this.channel) {
      try {
        this.channel.leave();
      } catch (error) {
        Logger.debug('[Runtime]', 'channel leave error', error);
      }
      this.channel = null;
    }
    if (this.client) {
      try {
        this.client.disconnect();
      } catch (error) {
        Logger.debug('[Runtime]', 'client disconnect error', error);
      }
      this.client = null;
    }
    this.updateState({ status: 'disconnected' });
    this.events.emit('disconnected', undefined);
  }

  destroy(): void {
    this.disposed = true;
    this.disconnect();
    this.events.clear();
  }

  sendEvent(hid: string, payload: ClientEventMessage['payload'], cseq?: number): void {
    const sid = this.sessionId ?? this.bootPayload?.sid;
    if (!this.channel || !sid) {
      return;
    }
    Logger.debug('[Runtime]', 'send event', { handler: hid, cseq });
    const message: ClientEventMessage = {
      t: 'evt',
      sid,
      hid,
      payload,
      cseq,
    };
    Logger.debug('[Runtime]', '→ sending event', message);
    this.channel.sendMessage('evt', message);
  }

  sendNavigation(path: string, q: string, hash = ''): void {
    const sid = this.sessionId ?? this.bootPayload?.sid;
    if (!this.channel || !sid) {
      return;
    }
    Logger.debug('[Runtime]', 'send navigation', { path, q, hash });
    const message: ClientNavMessage = {
      t: 'nav',
      sid,
      path,
      q,
      hash,
    };
    Logger.debug('[Runtime]', '→ sending navigation', message);
    this.channel.sendMessage('nav', message);
  }

  sendPopNavigation(path: string, q: string, hash = ''): void {
    const sid = this.sessionId ?? this.bootPayload?.sid;
    if (!this.channel || !sid) {
      return;
    }
    Logger.debug('[Runtime]', 'send pop navigation', { path, q, hash });
    const message: ClientNavMessage = {
      t: 'pop',
      sid,
      path,
      q,
      hash,
    };
    Logger.debug('[Runtime]', '→ sending pop navigation', message);
    this.channel.sendMessage('pop', message);
  }

  requestRecover(): void {
    const sid = this.sessionId ?? this.bootPayload?.sid;
    if (!this.channel || !sid) {
      return;
    }
    Logger.debug('[Runtime]', 'request recover', { sid });
    const payload: ClientRecoverMessage = {
      t: 'recover',
      sid,
    };
    Logger.debug('[Runtime]', '→ sending recover', payload);
    this.channel.sendMessage('recover', payload);
  }

  private buildJoinPayload(boot: BootPayload) {
    const ack = this.lastAck || boot.seq || 0;
    return {
      sid: boot.sid,
      ver: boot.ver ?? 0,
      ack,
      loc: {
        path: boot.location?.path ?? '/',
        q: boot.location?.q ?? '',
        hash: boot.location?.hash ?? '',
      },
    };
  }

  private routeMessage(msg: ServerMessage): void {
    if (!msg || typeof (msg as any).t !== 'string') {
      return;
    }
    Logger.debug('[Runtime]', 'received message', { type: (msg as any)?.t });
    this.events.emit('message', msg);
    switch (msg.t) {
      case 'init':
        this.handleInit(msg as InitMessage);
        break;
      case 'frame':
        this.handleFrame(msg as FrameMessage);
        break;
      case 'template':
        this.events.emit('template', msg as TemplateMessage);
        break;
      case 'resume':
        this.events.emit('resume', msg as ResumeMessage);
        break;
      case 'join':
        this.events.emit('join', msg as JoinMessage);
        break;
      case 'diagnostic':
        this.events.emit('diagnostic', msg as DiagnosticMessage);
        break;
      case 'error':
        this.handleErrorMessage(msg as ErrorMessage);
        break;
      case 'upload':
        this.events.emit('upload', msg as UploadControlMessage);
        break;
      case 'domreq':
        this.events.emit('domreq', msg as DOMRequestMessage);
        break;
      default:
        break;
    }
  }

  private handleInit(msg: InitMessage): void {
    this.sessionId = msg.sid;
    this.version = msg.ver;
    Logger.debug('[Runtime]', 'init received', { sid: msg.sid, version: msg.ver, seq: msg.seq });
    if (typeof msg.seq === 'number') {
      this.lastAck = msg.seq;
      this.sendAck(msg.seq);
    }
    this.events.emit('init', msg);
  }

  private handleFrame(msg: FrameMessage): void {
    this.version = msg.ver;
    Logger.debug('[Runtime]', 'frame received', {
      version: msg.ver,
      seq: msg.seq,
      ops: Array.isArray(msg.patch) ? msg.patch.length : 0,
    });
    if (typeof msg.seq === 'number') {
      this.lastAck = msg.seq;
      this.sendAck(msg.seq);
    }
    this.events.emit('frame', msg);
  }

  private handleErrorMessage(msg: ErrorMessage): void {
    const error = new Error(msg.message ?? 'server error');
    error.name = msg.code ?? 'ServerError';
    this.handleErrorEvent(error);
  }

  private handleChannelLeave(): void {
    Logger.debug('[Runtime]', 'channel left, cleaning up');
    this.channel = null;
    this.sessionId = null;
    this.version = 0;
    if (this.client) {
      try {
        this.client.disconnect();
      } catch (error) {
        Logger.debug('[Runtime]', 'client disconnect error', error);
      }
      this.client = null;
    }
    this.updateState({ status: 'disconnected' });
    this.events.emit('disconnected', undefined);
    if (!this.disposed && this.options.reconnect) {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      return;
    }
    if (this.reconnectTimer) {
      return;
    }
    this.reconnectAttempts += 1;
    const delay = this.options.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    Logger.debug('[Runtime]', 'scheduling reconnect', {
      attempt: this.reconnectAttempts,
      delay,
    });
    this.updateState({ status: 'reconnecting', attempt: this.reconnectAttempts });
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      void this.connect();
    }, delay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private sendAck(seq: number): void {
    const sid = this.sessionId ?? this.bootPayload?.sid;
    if (!this.channel || !sid) {
      return;
    }
    Logger.debug('[Runtime]', 'acknowledging frame', { seq });
    const payload: ClientAckMessage = {
      t: 'ack',
      sid,
      seq,
    };
    Logger.debug('[Runtime]', '→ sending ack', payload);
    this.channel.sendMessage('ack', payload);
  }

  private handleErrorEvent(error: Error): void {
    Logger.warn('[Runtime]', 'runtime error', error);
    this.events.emit('error', { error });
    this.updateState({ status: 'error', error });
  }

  private updateState(next: ConnectionState): void {
    this.state = next;
    this.events.emit('state', next);
  }

}

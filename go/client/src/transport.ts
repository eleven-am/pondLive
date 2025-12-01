import { PondClient, ChannelState } from '@eleven-am/pondsocket-client';
import {
    Topic,
    Topics,
    Location,
    ActionFor,
    PayloadFor,
    ClientEvt,
    ClientAck,
    isMessage,
    HandlerEventPayload,
    handlerTopic,
    FramePatchPayload,
    Patch,
    Event,
    ScriptPayload,
} from './protocol';
import { Bus } from './bus';
import {Logger} from "./logger";

export interface TransportConfig {
    endpoint: string;
    sessionId: string;
    version: number;
    lastAck: number;
    location: Location;
    bus: Bus;
}

export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'stalled';

export interface JoinPayload {
    sid: string;
    ver: number;
    ack: number;
    loc: Location;
}

export class Transport {
    private readonly client: PondClient;
    private readonly channel: ReturnType<PondClient['createChannel']>;
    private readonly sessionId: string;
    private readonly bus: Bus;
    private state: ConnectionState = 'disconnected';
    private stateListeners: Array<(state: ConnectionState) => void> = [];

    constructor(config: TransportConfig) {
        this.sessionId = config.sessionId;
        this.bus = config.bus;

        this.client = new PondClient(config.endpoint);

        const joinPayload: JoinPayload = {
            sid: config.sessionId,
            ver: config.version,
            ack: config.lastAck,
            loc: config.location,
        };

        this.channel = this.client.createChannel(`live/${config.sessionId}`, joinPayload);

        this.channel.onMessage((_event: string, payload: unknown) => {
            this.handleMessage(payload);
        });

        this.channel.onChannelStateChange((channelState: ChannelState) => {
            this.handleStateChange(channelState);
        });
    }

    get sid(): string {
        return this.sessionId;
    }

    get connectionState(): ConnectionState {
        return this.state;
    }

    connect(): void {
        this.state = 'connecting';
        this.notifyStateChange();
        this.channel.join();
        this.client.connect();
    }

    disconnect(): void {
        this.channel.leave();
        this.client.disconnect();
        this.state = 'disconnected';
        this.notifyStateChange();
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
        const evt: ClientEvt = {
            t: topic,
            sid: this.sessionId,
            a: String(action),
            p: payload,
        };
        this.sendMessage('evt', evt);
    }

    sendAck(seq: number): void {
        const ack: ClientAck = {
            t: Topics.Ack,
            sid: this.sessionId,
            seq,
        };
        this.sendMessage('ack', ack);
    }

    sendHandler(handlerId: string, payload: HandlerEventPayload): void {
        const evt: ClientEvt = {
            t: handlerTopic(handlerId),
            sid: this.sessionId,
            a: 'invoke',
            p: payload,
        };
        this.sendMessage('evt', evt);
    }

    sendScript(scriptId: string, payload: ScriptPayload): void {
        const evt: ClientEvt = {
            t: `script:${scriptId}`,
            sid: this.sessionId,
            a: 'message',
            p: payload,
        };
        this.sendMessage('evt', evt);
    }

    private handleMessage(payload: unknown): void {
        Logger.info('TRANSPORT','Transport received message:', payload);
        if (!isMessage(payload)) {
            return;
        }

        const { seq, topic, event, data } = payload;

        if (!this.isValidTopic(topic)) {
            return;
        }

        this.publishToBus(topic, event, data, seq);
    }

    private isValidTopic(topic: string): topic is Topic {
        return topic === 'router' || topic === 'dom' || topic === 'frame' || topic === 'ack' || topic.startsWith('script:');
    }

    private publishToBus(topic: Topic, action: string, data: unknown, seq: number): void {
        switch (topic) {
            case 'frame':
                if (action === 'patch') {
                    const payload: FramePatchPayload = {
                        seq,
                        patches: data as Patch[],
                    };
                    this.bus.publish('frame', 'patch', payload);
                }
                break;
            case 'router':
                if (action === 'push') {
                    this.bus.publish('router', 'push', data as PayloadFor<'router', 'push'>);
                } else if (action === 'replace') {
                    this.bus.publish('router', 'replace', data as PayloadFor<'router', 'replace'>);
                } else if (action === 'back') {
                    this.bus.publish('router', 'back', undefined);
                } else if (action === 'forward') {
                    this.bus.publish('router', 'forward', undefined);
                }
                break;
            case 'dom':
                if (action === 'call') {
                    this.bus.publish('dom', 'call', data as PayloadFor<'dom', 'call'>);
                } else if (action === 'set') {
                    this.bus.publish('dom', 'set', data as PayloadFor<'dom', 'set'>);
                } else if (action === 'query') {
                    this.bus.publish('dom', 'query', data as PayloadFor<'dom', 'query'>);
                } else if (action === 'async') {
                    this.bus.publish('dom', 'async', data as PayloadFor<'dom', 'async'>);
                }
                break;
            case 'ack':
                if (action === 'ack') {
                    this.bus.publish('ack', 'ack', data as PayloadFor<'ack', 'ack'>);
                }
                break;
            default:
                if (topic.startsWith('script:') && action === 'send') {
                    const payload = data as ScriptPayload;
                    this.bus.publishScript(payload.scriptId, 'send', payload);
                }
                break;
        }
    }

    private handleStateChange(channelState: ChannelState): void {
        switch (channelState) {
            case ChannelState.JOINED:
                this.state = 'connected';
                break;
            case ChannelState.STALLED:
                this.state = 'stalled';
                break;
            case ChannelState.CLOSED:
                this.state = 'disconnected';
                break;
            case ChannelState.JOINING:
            case ChannelState.IDLE:
                this.state = 'connecting';
                break;
        }
        this.notifyStateChange();
    }

    private notifyStateChange(): void {
        for (const listener of this.stateListeners) {
            try {
                listener(this.state);
            } catch {
                // swallow
            }
        }
    }

    private sendMessage<T extends Event>(type: string, message: T): void {
        Logger.info('TRANSPORT','Transport sending message:', type, message);
        this.channel.sendMessage(type, message);
    }
}

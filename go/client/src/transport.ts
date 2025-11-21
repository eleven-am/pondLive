import {PondClient} from '@eleven-am/pondsocket-client';
import {ClientMessage, ServerMessage} from './protocol';
import {MessageHandler, StateChangeHandler, TransportConfig} from './types';

export class Transport {
    private readonly client: PondClient;
    private readonly channel: ReturnType<PondClient['createChannel']>;
    private readonly _sessionId: string;
    private handler: MessageHandler | null = null;

    constructor(config: TransportConfig) {
        this._sessionId = config.sessionId;

        this.client = new PondClient(config.endpoint);

        const joinPayload = {
            sid: config.sessionId,
            ver: config.version,
            ack: config.ack,
            loc: config.location,
        };

        this.channel = this.client.createChannel(`live/${config.sessionId}`, joinPayload);
        this.channel.join();

        this.channel.onMessage((_event: string, payload: unknown) => {
            this.handler?.(payload as ServerMessage);
        });
    }

    get sessionId(): string {
        return this._sessionId;
    }

    connect(): void {
        this.client.connect();
    }

    disconnect(): void {
        this.channel.leave();
        this.client.disconnect();
    }

    send(msg: ClientMessage): void {
        this.channel.sendMessage(msg.t, msg);
    }

    onMessage(handler: MessageHandler): void {
        this.handler = handler;
    }

    onStateChange(handler: StateChangeHandler): void {
        this.channel.onChannelStateChange(handler);
    }
}

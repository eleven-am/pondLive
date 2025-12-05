import { ScriptMeta, ScriptPayload } from './protocol';
import { Bus, Subscription } from './bus';
import { Logger } from './logger';
import { Transport } from './transport';

export interface ScriptTransport {
    send(event: string, data: unknown): void;
    on(event: string, handler: (data: unknown) => void): void;
}

export interface ScriptInstance {
    cleanup?: () => void;
    eventHandlers: Map<string, (data: unknown) => void>;
    subscription: Subscription;
}

export interface ScriptExecutorConfig {
    bus: Bus;
    transport: Transport;
}


export class ScriptExecutor {
    private readonly bus: Bus;
    private readonly transport: Transport;
    private readonly scripts = new Map<string, ScriptInstance>();

    constructor(config: ScriptExecutorConfig) {
        this.bus = config.bus;
        this.transport = config.transport;
    }

    async execute(meta: ScriptMeta, element: Element): Promise<void> {
        const { scriptId, script } = meta;

        Logger.debug('Script', 'execute called', { scriptId, element: element.tagName, script: script.substring(0, 100) + '...' });

        this.cleanup(scriptId);

        const instance: ScriptInstance = {
            eventHandlers: new Map(),
            subscription: this.bus.subscribeScript(scriptId, 'send', (payload: ScriptPayload) => {
                Logger.debug('Script', 'server message received', { scriptId, event: payload.event, data: payload.data });
                this.handleServerMessage(scriptId, payload.event, payload.data);
            }),
        };

        const transport: ScriptTransport = {
            send: (event: string, data: unknown) => {
                Logger.debug('Script', 'transport.send called', { scriptId, event, data });
                const payload: ScriptPayload = {
                    scriptId,
                    event,
                    data,
                };
                this.transport.sendScript(scriptId, payload);
            },
            on: (event: string, handler: (data: unknown) => void) => {
                Logger.debug('Script', 'transport.on registered', { scriptId, event });
                instance.eventHandlers.set(event, handler);
            },
        };

        try {
            Logger.debug('Script', 'creating function', { scriptId });
            const fn = new Function('element', 'transport', `return (${script})(element, transport);`);
            Logger.debug('Script', 'executing function', { scriptId });
            const cleanup = await fn(element, transport);

            if (typeof cleanup === 'function') {
                Logger.debug('Script', 'cleanup function returned', { scriptId });
                instance.cleanup = cleanup;
            }

            this.scripts.set(scriptId, instance);
            Logger.debug('Script', 'execute complete', { scriptId, handlers: Array.from(instance.eventHandlers.keys()) });
        } catch (err) {
            Logger.error('Script', 'execute failed', { scriptId, error: String(err) });
            instance.subscription.unsubscribe();
            throw err;
        }
    }

    private handleServerMessage(scriptId: string, event: string, data: unknown): void {
        Logger.debug('Script', 'handleServerMessage', { scriptId, event, data });
        const instance = this.scripts.get(scriptId);
        if (!instance) {
            Logger.warn('Script', 'no instance found', { scriptId });
            return;
        }

        const handler = instance.eventHandlers.get(event);
        if (!handler) {
            Logger.warn('Script', 'no handler found', { scriptId, event, availableHandlers: Array.from(instance.eventHandlers.keys()) });
            return;
        }

        try {
            Logger.debug('Script', 'invoking handler', { scriptId, event });
            handler(data);
        } catch (err) {
            Logger.error('Script', 'handler error', { scriptId, event, error: String(err) });
        }
    }

    cleanup(scriptId: string): void {
        Logger.debug('Script', 'cleanup called', { scriptId });
        const instance = this.scripts.get(scriptId);
        if (!instance) {
            Logger.debug('Script', 'cleanup: no instance found', { scriptId });
            return;
        }

        if (instance.cleanup) {
            try {
                Logger.debug('Script', 'running cleanup function', { scriptId });
                instance.cleanup();
            } catch (err) {
                Logger.error('Script', 'cleanup function error', { scriptId, error: String(err) });
            }
        }

        instance.subscription.unsubscribe();
        this.scripts.delete(scriptId);
        Logger.debug('Script', 'cleanup complete', { scriptId });
    }

    destroy(): void {
        for (const scriptId of this.scripts.keys()) {
            this.cleanup(scriptId);
        }
    }
}

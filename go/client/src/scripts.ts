import {ScriptMeta, ScriptTransport, ScriptInstance, ScriptExecutorConfig} from './types';
import {Logger} from './logger';

export class ScriptExecutor {
    private readonly sessionId: string;
    private readonly onMessage: ScriptExecutorConfig['onMessage'];
    private scripts = new Map<string, ScriptInstance>();

    constructor(config: ScriptExecutorConfig) {
        this.sessionId = config.sessionId;
        this.onMessage = config.onMessage;
    }

    async execute(meta: ScriptMeta, element: Element): Promise<void> {
        const {scriptId, script} = meta;

        this.cleanup(scriptId);

        const instance: ScriptInstance = {
            eventHandlers: new Map()
        };

        const transport: ScriptTransport = {
            send: (data) => {
                this.onMessage({
                    t: 'script:message',
                    sid: this.sessionId,
                    scriptId,
                    event: '',
                    data
                });
            },
            on: (event: string, handler: (data: Record<string, unknown>) => void) => {
                instance.eventHandlers.set(event, handler);
            }
        };

        try {
            const scriptFn = new Function('element', 'transport', `return (${script})(element, transport)`);
            const cleanup = await scriptFn(element, transport);

            if (typeof cleanup === 'function') {
                instance.cleanup = cleanup;
            }

            this.scripts.set(scriptId, instance);
            Logger.debug('ScriptExecutor', 'Script executed', scriptId);
        } catch (error) {
            Logger.error('ScriptExecutor', 'Script execution failed', scriptId, error);
        }
    }

    handleEvent(scriptId: string, event: string, data: Record<string, unknown>): void {
        const instance = this.scripts.get(scriptId);
        if (!instance) {
            Logger.warn('ScriptExecutor', 'Script instance not found', scriptId);
            return;
        }

        const handler = instance.eventHandlers.get(event);
        if (!handler) {
            Logger.warn('ScriptExecutor', 'Event handler not found', scriptId, event);
            return;
        }

        try {
            handler(data);
        } catch (error) {
            Logger.error('ScriptExecutor', 'Event handler failed', scriptId, event, error);
        }
    }

    cleanup(scriptId: string): void {
        const instance = this.scripts.get(scriptId);
        if (!instance) return;

        if (instance.cleanup) {
            try {
                instance.cleanup();
            } catch (error) {
                Logger.error('ScriptExecutor', 'Cleanup failed', scriptId, error);
            }
        }

        this.scripts.delete(scriptId);
        Logger.debug('ScriptExecutor', 'Script cleaned up', scriptId);
    }
}

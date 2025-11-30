import { ScriptMeta, ScriptPayload } from './protocol';
import { Bus, Subscription } from './bus';

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
}

type SandboxTarget = Record<string, unknown>;

const SANDBOX_WHITELIST: readonly string[] = [
    'console',
    'setTimeout',
    'clearTimeout',
    'setInterval',
    'clearInterval',
    'requestAnimationFrame',
    'cancelAnimationFrame',
    'Promise',
    'Array',
    'Object',
    'String',
    'Number',
    'Boolean',
    'Math',
    'JSON',
    'Date',
    'Map',
    'Set',
    'WeakMap',
    'WeakSet',
    'Error',
    'TypeError',
    'RangeError',
    'SyntaxError',
    'ReferenceError',
    'parseInt',
    'parseFloat',
    'isNaN',
    'isFinite',
    'encodeURI',
    'decodeURI',
    'encodeURIComponent',
    'decodeURIComponent',
    'atob',
    'btoa',
    'fetch',
    'URL',
    'URLSearchParams',
    'TextEncoder',
    'TextDecoder',
    'Blob',
    'File',
    'FormData',
    'Headers',
    'Request',
    'Response',
    'AbortController',
    'AbortSignal',
    'Symbol',
    'Proxy',
    'Reflect',
    'RegExp',
    'Intl',
    'BigInt',
    'ArrayBuffer',
    'DataView',
    'Uint8Array',
    'Uint16Array',
    'Uint32Array',
    'Int8Array',
    'Int16Array',
    'Int32Array',
    'Float32Array',
    'Float64Array',
    'Uint8ClampedArray',
    'BigInt64Array',
    'BigUint64Array',
];

function createSandbox(element: Element, transport: ScriptTransport): SandboxTarget {
    const target: SandboxTarget = {
        element,
        transport,
    };

    for (const key of SANDBOX_WHITELIST) {
        if (key in globalThis) {
            target[key] = (globalThis as unknown as Record<string, unknown>)[key];
        }
    }

    return new Proxy(target, {
        has: () => true,
        get: (t, prop) => {
            if (typeof prop === 'string' && prop in t) {
                return t[prop];
            }
            return undefined;
        },
        set: (t, prop, value) => {
            if (typeof prop === 'string') {
                t[prop] = value;
            }
            return true;
        },
    });
}

export class ScriptExecutor {
    private readonly bus: Bus;
    private readonly scripts = new Map<string, ScriptInstance>();

    constructor(config: ScriptExecutorConfig) {
        this.bus = config.bus;
    }

    async execute(meta: ScriptMeta, element: Element): Promise<void> {
        const { scriptId, script } = meta;

        this.cleanup(scriptId);

        const instance: ScriptInstance = {
            eventHandlers: new Map(),
            subscription: this.bus.subscribeScript(scriptId, 'send', (payload: ScriptPayload) => {
                this.handleServerMessage(scriptId, payload.event, payload.data);
            }),
        };

        const transport: ScriptTransport = {
            send: (event: string, data: unknown) => {
                this.bus.publishScript(scriptId, 'message', {
                    scriptId,
                    event,
                    data,
                });
            },
            on: (event: string, handler: (data: unknown) => void) => {
                instance.eventHandlers.set(event, handler);
            },
        };

        const sandbox = createSandbox(element, transport);

        try {
            const fn = new Function(
                'sandbox',
                `with(sandbox) { return (${script})(element, transport); }`
            );
            const cleanup = await fn(sandbox);

            if (typeof cleanup === 'function') {
                instance.cleanup = cleanup;
            }

            this.scripts.set(scriptId, instance);
        } catch (err) {
            instance.subscription.unsubscribe();
            throw err;
        }
    }

    private handleServerMessage(scriptId: string, event: string, data: unknown): void {
        const instance = this.scripts.get(scriptId);
        if (!instance) return;

        const handler = instance.eventHandlers.get(event);
        if (!handler) return;

        try {
            handler(data);
        } catch {
        }
    }

    cleanup(scriptId: string): void {
        const instance = this.scripts.get(scriptId);
        if (!instance) return;

        if (instance.cleanup) {
            try {
                instance.cleanup();
            } catch {
            }
        }

        instance.subscription.unsubscribe();
        this.scripts.delete(scriptId);
    }

    destroy(): void {
        for (const scriptId of this.scripts.keys()) {
            this.cleanup(scriptId);
        }
    }
}

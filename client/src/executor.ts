import {
    DOMCallPayload,
    DOMSetPayload,
    DOMQueryPayload,
    DOMAsyncPayload,
    DOMResponsePayload,
    RouterNavPayload,
    RouterPopstatePayload,
} from './protocol';
import { Bus, Subscription } from './bus';
import { Transport } from './transport';

export type RefResolver = (refId: string) => Element | undefined;

export interface ExecutorConfig {
    bus: Bus;
    transport: Transport;
    resolveRef: RefResolver;
}

export class Executor {
    private readonly bus: Bus;
    private readonly transport: Transport;
    private readonly resolveRef: RefResolver;
    private readonly subscriptions: Subscription[] = [];
    private popstateHandler: (() => void) | null = null;

    constructor(config: ExecutorConfig) {
        this.bus = config.bus;
        this.transport = config.transport;
        this.resolveRef = config.resolveRef;

        this.setupDOMSubscriptions();
        this.setupRouterSubscriptions();
        this.setupPopstateListener();
    }

    destroy(): void {
        for (const sub of this.subscriptions) {
            sub.unsubscribe();
        }
        this.subscriptions.length = 0;

        if (this.popstateHandler) {
            window.removeEventListener('popstate', this.popstateHandler);
            this.popstateHandler = null;
        }
    }

    private setupDOMSubscriptions(): void {
        this.subscriptions.push(
            this.bus.subscribe('dom', 'call', (payload) => this.handleCall(payload))
        );
        this.subscriptions.push(
            this.bus.subscribe('dom', 'set', (payload) => this.handleSet(payload))
        );
        this.subscriptions.push(
            this.bus.subscribe('dom', 'query', (payload) => this.handleQuery(payload))
        );
        this.subscriptions.push(
            this.bus.subscribe('dom', 'async', (payload) => this.handleAsync(payload))
        );
    }

    private setupRouterSubscriptions(): void {
        this.subscriptions.push(
            this.bus.subscribe('router', 'push', (payload) => this.handlePush(payload))
        );
        this.subscriptions.push(
            this.bus.subscribe('router', 'replace', (payload) => this.handleReplace(payload))
        );
        this.subscriptions.push(
            this.bus.subscribe('router', 'back', () => this.handleBack())
        );
        this.subscriptions.push(
            this.bus.subscribe('router', 'forward', () => this.handleForward())
        );
    }

    private setupPopstateListener(): void {
        this.popstateHandler = () => {
            const payload: RouterPopstatePayload = {
                path: window.location.pathname,
                query: window.location.search.replace(/^\?/, ''),
                hash: window.location.hash.replace(/^#/, ''),
            };
            this.transport.send('router', 'popstate', payload);
        };
        window.addEventListener('popstate', this.popstateHandler);
    }

    private handleCall(payload: DOMCallPayload): void {
        const el = this.resolveRef(payload.ref);
        if (!el) return;

        const method = (el as unknown as Record<string, unknown>)[payload.method];
        if (typeof method === 'function') {
            method.apply(el, payload.args ?? []);
        }
    }

    private handleSet(payload: DOMSetPayload): void {
        const el = this.resolveRef(payload.ref);
        if (!el) return;

        (el as unknown as Record<string, unknown>)[payload.prop] = payload.value;
    }

    private handleQuery(payload: DOMQueryPayload): void {
        const el = this.resolveRef(payload.ref);
        const response: DOMResponsePayload = { requestId: payload.requestId };

        if (!el) {
            response.error = `ref not found: ${payload.ref}`;
            this.transport.send('dom', 'response', response);
            return;
        }

        const values: Record<string, unknown> = {};
        for (const selector of payload.selectors) {
            values[selector] = this.readProperty(el, selector);
        }

        response.values = values;
        this.transport.send('dom', 'response', response);
    }

    private handleAsync(payload: DOMAsyncPayload): void {
        const el = this.resolveRef(payload.ref);
        const response: DOMResponsePayload = { requestId: payload.requestId };

        if (!el) {
            response.error = `ref not found: ${payload.ref}`;
            this.transport.send('dom', 'response', response);
            return;
        }

        const method = (el as unknown as Record<string, unknown>)[payload.method];
        if (typeof method !== 'function') {
            response.error = `method not found: ${payload.method}`;
            this.transport.send('dom', 'response', response);
            return;
        }

        Promise.resolve(method.apply(el, payload.args ?? []))
            .then((result) => {
                response.result = this.serializeValue(result);
                this.transport.send('dom', 'response', response);
            })
            .catch((err) => {
                response.error = err instanceof Error ? err.message : String(err);
                this.transport.send('dom', 'response', response);
            });
    }

    private handlePush(payload: RouterNavPayload): void {
        const url = this.buildUrl(payload);
        window.history.pushState({}, '', url);
    }

    private handleReplace(payload: RouterNavPayload): void {
        const url = this.buildUrl(payload);
        window.history.replaceState({}, '', url);
    }

    private handleBack(): void {
        window.history.back();
    }

    private handleForward(): void {
        window.history.forward();
    }

    private buildUrl(payload: RouterNavPayload): string {
        let url = payload.path;
        if (payload.query) {
            url += '?' + payload.query;
        }
        if (payload.hash) {
            url += '#' + payload.hash;
        }
        return url;
    }

    private readProperty(el: Element, path: string): unknown {
        const segments = path.split('.');
        let current: unknown = el;

        for (const segment of segments) {
            if (current == null) return undefined;
            current = (current as Record<string, unknown>)[segment];
        }

        return this.serializeValue(current);
    }

    private serializeValue(value: unknown): unknown {
        if (value === null || value === undefined) return null;

        const type = typeof value;
        if (type === 'string' || type === 'number' || type === 'boolean') return value;

        if (Array.isArray(value)) {
            return value.map((v) => this.serializeValue(v)).filter((v) => v !== undefined);
        }

        if (value instanceof Date) return value.toISOString();
        if (value instanceof DOMTokenList) return Array.from(value);
        if (value instanceof Node) return undefined;

        try {
            return JSON.parse(JSON.stringify(value));
        } catch {
            return undefined;
        }
    }
}

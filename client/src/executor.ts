import {
    DOMCallPayload,
    DOMSetPayload,
    DOMQueryPayload,
    DOMAsyncPayload,
    DOMResponsePayload,
    DOMObservePayload,
    DOMUnobservePayload,
    RouterNavPayload,
    RouterPopstatePayload,
    RouterRedirectPayload,
    urlOf,
} from './protocol';
import { Bus, Subscription } from './bus';
import { Transport } from './transport';
import { OBSERVED_EVENTS, ObservedProp, RectWire, SAFE_PROTOCOL, ScrollWire, UNSAFE_URL, refAllows } from './wire';
import { serializeValue } from './serialize';
import { limit } from './rate';

export type RefResolver = (id: number) => Element | undefined;

type El = any;
const rectOf = (el: El): RectWire => {
    const r = el.getBoundingClientRect();
    return { x: r.x, y: r.y, width: r.width, height: r.height, top: r.top, right: r.right, bottom: r.bottom, left: r.left };
};

const READ: Record<ObservedProp, (el: El) => unknown> = {
    currentTime: (e) => e.currentTime,
    duration: (e) => e.duration,
    paused: (e) => e.paused,
    ended: (e) => e.ended,
    volume: (e) => e.volume,
    muted: (e) => e.muted,
    focused: (e) => document.activeElement === e,
    scroll: (e): ScrollWire => ({ scrollTop: e.scrollTop, scrollLeft: e.scrollLeft, scrollHeight: e.scrollHeight, scrollWidth: e.scrollWidth, clientHeight: e.clientHeight, clientWidth: e.clientWidth }),
    rect: rectOf,
};

interface Observer {
    id: number;
    push: () => void;
    stop: () => void;
}

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
    private readonly observers = new Map<string, Observer>();
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
        this.observers.forEach((o) => o.stop());
        this.observers.clear();

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
        this.subscriptions.push(
            this.bus.subscribe('dom', 'observe', (payload) => this.handleObserve(payload))
        );
        this.subscriptions.push(
            this.bus.subscribe('dom', 'unobserve', (payload) => this.handleUnobserve(payload))
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
        this.subscriptions.push(
            this.bus.subscribe('router', 'redirect', (payload) => this.handleRedirect(payload))
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
        const el = refAllows(payload.k, 'method', payload.method) && (this.resolveRef(payload.id) as any);
        if (el && typeof el[payload.method] === 'function') {
            el[payload.method](...(payload.args ?? []));
            this.pushObserved(payload.id);
        }
    }

    private handleSet(payload: DOMSetPayload): void {
        const el = refAllows(payload.k, 'set', payload.prop) && (this.resolveRef(payload.id) as any);
        if (el) {
            el[payload.prop] = payload.value;
            this.pushObserved(payload.id);
        }
    }

    private pushObserved(id: number): void {
        this.observers.forEach((o) => {
            if (o.id === id) o.push();
        });
    }

    private handleObserve(payload: DOMObservePayload): void {
        const key = this.handleUnobserve(payload);
        const el = this.resolveRef(payload.id) as El;
        const prop = payload.prop as ObservedProp;
        const read = READ[prop];
        if (!el || !read || !refAllows(payload.k, 'observe', prop)) return;

        const events = OBSERVED_EVENTS[prop];
        let last: string | undefined;
        const push = () => {
            const value = read(el);
            const text = JSON.stringify(value);
            if (text === last) return;
            last = text;
            this.transport.send('dom', 'value', { id: payload.id, prop: payload.prop, value });
        };
        const fire = limit<unknown>(undefined, payload.ms < 0 ? undefined : payload.ms, push);

        let stop: () => void;
        if (prop === 'rect') {
            const ro = new ResizeObserver(fire);
            ro.observe(el);
            stop = () => ro.disconnect();
        } else {
            events.forEach((name) => el.addEventListener(name, fire));
            stop = () => events.forEach((name) => el.removeEventListener(name, fire));
        }
        this.observers.set(key, { id: payload.id, push, stop });
        push();
    }

    private handleUnobserve(payload: DOMUnobservePayload): string {
        const key = payload.id + ':' + payload.prop;
        this.observers.get(key)?.stop();
        this.observers.delete(key);
        return key;
    }

    private reply(requestId: string, rest: Partial<DOMResponsePayload>): void {
        this.transport.send('dom', 'response', { requestId, ...rest });
    }

    private handleQuery(payload: DOMQueryPayload): void {
        const { requestId, id, selectors } = payload;
        const el = this.resolveRef(id) as Record<string, unknown> | undefined;
        if (!selectors.every((name) => refAllows(payload.k, 'query', name))) return this.reply(requestId, { error: 'property not allowed' });
        if (!el) return this.reply(requestId, { error: `ref not found: ${id}` });
        const values: Record<string, unknown> = {};
        for (const name of selectors) values[name] = serializeValue(el[name]);
        this.reply(requestId, { values });
    }

    private handleAsync(payload: DOMAsyncPayload): void {
        const { requestId, id, method: name } = payload;
        const el = this.resolveRef(id) as Record<string, any> | undefined;
        if (!refAllows(payload.k, 'method', name)) return this.reply(requestId, { error: `method not allowed: ${name}` });
        if (!el) return this.reply(requestId, { error: `ref not found: ${id}` });
        if (typeof el[name] !== 'function') return this.reply(requestId, { error: `method not found: ${name}` });
        Promise.resolve(el[name](...(payload.args ?? [])))
            .then((result) => this.reply(requestId, { result: serializeValue(result) }))
            .catch((err) => this.reply(requestId, { error: err instanceof Error ? err.message : String(err) }));
    }

    private handlePush(payload: RouterNavPayload): void {
        const url = urlOf(payload.path, payload.query, payload.hash);
        window.history.pushState({}, '', url);
    }

    private handleReplace(payload: RouterNavPayload): void {
        const url = urlOf(payload.path, payload.query, payload.hash);
        window.history.replaceState({}, '', url);
    }

    private handleBack(): void {
        window.history.back();
    }

    private handleForward(): void {
        window.history.forward();
    }

    private handleRedirect(payload: RouterRedirectPayload): void {
        const url = payload.url;
        if (!url || UNSAFE_URL.test(url)) return;
        let parsed: URL;
        try {
            parsed = new URL(url, window.location.href);
        } catch {
            return;
        }
        if (!SAFE_PROTOCOL.test(parsed.protocol)) return;
        window.location.assign(url);
    }
}

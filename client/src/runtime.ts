import {
    Boot,
    Message,
    Location,
    StaticTopic,
    SCRIPT_TOPIC,
    urlOf,
    isBoot,
    isScriptMessage,
} from './protocol';
import { ATTR_RESYNCING } from './wire';
import { Bus } from './bus';
import { Transport, ConnectionState } from './transport';
import { Applier } from './applier';
import { Sequencer } from './sequencer';
import { Executor } from './executor';
import { Scripts, lookup } from './scripts';
import { Logger } from './logger';

export interface RuntimeConfig {
    root: Node;
    sessionId: string;
    version: number;
    bundle?: string;
    seq: number;
    endpoint: string;
    token: string;
    debug?: boolean;
}

const RELOAD_JITTER_MIN = 1000;
const RELOAD_JITTER_MAX = 10000;
const MAX_RELOADS = 10;
const RELOAD_TRACKING_KEY = 'pond_reload_count';
const RELOAD_TIMESTAMP_KEY = 'pond_reload_timestamp';
const RELOAD_WINDOW_MS = 60000;

declare global {
    interface Window {
        __POND_RUNTIME__?: Runtime;
        __LIVEUI_BOOT__?: Boot;
    }
}

export class Runtime {
    private readonly bus: Bus;
    private readonly transport: Transport;
    private readonly applier: Applier;
    private readonly executor: Executor;
    private readonly scripts: Scripts;

    private readonly sequencer: Sequencer;
    private readonly root: Node;
    private connectedState = false;

    constructor(config: RuntimeConfig) {
        this.root = config.root;

        Logger.configure({ enabled: config.debug ?? false, level: 'debug' });
        Logger.info('Runtime', 'Initializing', { sid: config.sessionId, ver: config.version });

        this.bus = new Bus();

        this.transport = new Transport({
            endpoint: config.endpoint,
            token: config.token,
            sessionId: config.sessionId,
            version: config.version,
            bundle: config.bundle,
            applied: () => this.sequencer.appliedSeq,
            wantsReset: () => this.sequencer.isResyncing(),
            onMessage: (msg) => this.sequencer.handle(msg),
        });

        this.applier = new Applier(config.root as Element, {
            onEvent: (id, event, data) => this.handleEvent(id, event, data),
            onScript: (el) => this.scripts.reconcile(el),
            onScriptCleanup: (el) => this.scripts.releaseHost(el),
        });

        this.executor = new Executor({
            bus: this.bus,
            transport: this.transport,
            resolveRef: (id) => this.applier.element(id),
        });

        this.scripts = new Scripts({ send: (id, action, payload) => this.transport.sendScript(id, action, payload), lookup });

        this.sequencer = new Sequencer({
            applier: this.applier,
            dispatch: (msg) => this.dispatchMessage(msg),
            sendAck: (seq) => this.transport.sendAck(seq),
            sendResync: (seq, reason) => this.transport.sendResync(seq, reason),
            isConnected: () => this.connectedState,
            applyLocation: (location) => this.applyLocation(location),
            onResyncing: (resyncing) => this.markResyncing(resyncing),
            onFailed: () => this.reloadWithJitter(),
        });
        this.sequencer.boot(config.seq);

        this.transport.onStateChange((state) => this.handleStateChange(state));

        window.__POND_RUNTIME__ = this;
    }

    connect(): void {
        Logger.info('Runtime', 'Connecting');
        this.transport.connect();
    }

    disconnect(): void {
        Logger.info('Runtime', 'Disconnecting');
        this.transport.disconnect();
        this.sequencer.destroy();
        this.executor.destroy();
        this.scripts.destroy();
        this.bus.clear();
        this.applier.destroy();
    }

    connected(): boolean {
        return this.connectedState;
    }

    get seq(): number {
        return this.sequencer.appliedSeq;
    }

    handleBoot(boot: Boot): void {
        Logger.info('Runtime', 'Boot received', { ver: boot.ver, seq: boot.seq });
        this.applier.index();
        this.sequencer.boot(boot.seq);
    }

    private handleEvent(id: number, event: string, data: Record<string, unknown>): void {
        if (this.sequencer.isResyncing()) {
            Logger.warn('Runtime', 'Dropping event while resyncing', { id, event });
            return;
        }
        Logger.debug('Runtime', 'Event', { id, event });
        this.transport.sendHandler(id, event, data);
    }

    private dispatchMessage({ topic, event, data }: Message): void {
        if (topic === SCRIPT_TOPIC) {
            if (event === 'm' && isScriptMessage(data)) this.scripts.deliver(data.i, data.d);
            return;
        }
        this.bus.publish(topic as StaticTopic, event as never, data as never);
    }

    private applyLocation(location: Location): void {
        const params = new URLSearchParams();
        for (const [key, values] of Object.entries(location.query ?? {})) {
            for (const value of values) params.append(key, value);
        }
        history.replaceState(history.state, '', urlOf(location.path, params.toString(), location.hash));
    }

    private markResyncing(resyncing: boolean): void {
        const el = this.root as Element;
        if (typeof el.toggleAttribute !== 'function') return;
        el.toggleAttribute(ATTR_RESYNCING, resyncing);
    }

    private handleStateChange(state: ConnectionState): void {
        Logger.debug('Runtime', 'Connection state', state);

        const wasConnected = this.connectedState;
        this.connectedState = state === 'connected';

        if (!wasConnected && this.connectedState) {
            Logger.info('Runtime', 'Connected');
            this.clearReloadTracking();
            this.sequencer.onConnected();
        } else if (wasConnected && !this.connectedState) {
            Logger.warn('Runtime', 'Disconnected');
            this.sequencer.onDisconnected();
        }

        if (state === 'declined') {
            Logger.warn('Runtime', 'Session declined - session expired or not found');
            this.reloadWithJitter();
        }
    }

    private reloadWithJitter(): void {
        if (this.shouldEnterFailsafeMode()) {
            Logger.error('Runtime', 'Entering failsafe mode - too many consecutive reloads');
            this.enterFailsafeMode();
            return;
        }

        this.incrementReloadCount();

        const jitter = Math.floor(Math.random() * (RELOAD_JITTER_MAX - RELOAD_JITTER_MIN)) + RELOAD_JITTER_MIN;
        Logger.info('Runtime', `Reloading in ${jitter}ms`);

        setTimeout(() => {
            window.location.reload();
        }, jitter);
    }

    private shouldEnterFailsafeMode(): boolean {
        try {
            const lastTimestamp = parseInt(sessionStorage.getItem(RELOAD_TIMESTAMP_KEY) || '0', 10);
            const reloadCount = parseInt(sessionStorage.getItem(RELOAD_TRACKING_KEY) || '0', 10);
            const now = Date.now();

            if (now - lastTimestamp > RELOAD_WINDOW_MS) {
                return false;
            }

            return reloadCount >= MAX_RELOADS;
        } catch {
            return false;
        }
    }

    private incrementReloadCount(): void {
        try {
            const lastTimestamp = parseInt(sessionStorage.getItem(RELOAD_TIMESTAMP_KEY) || '0', 10);
            const now = Date.now();

            if (now - lastTimestamp > RELOAD_WINDOW_MS) {
                sessionStorage.setItem(RELOAD_TRACKING_KEY, '1');
            } else {
                const count = parseInt(sessionStorage.getItem(RELOAD_TRACKING_KEY) || '0', 10);
                sessionStorage.setItem(RELOAD_TRACKING_KEY, String(count + 1));
            }

            sessionStorage.setItem(RELOAD_TIMESTAMP_KEY, String(now));
        } catch {
        }
    }

    private clearReloadTracking(): void {
        try {
            sessionStorage.removeItem(RELOAD_TRACKING_KEY);
            sessionStorage.removeItem(RELOAD_TIMESTAMP_KEY);
        } catch {
        }
    }

    private enterFailsafeMode(): void {
        try {
            sessionStorage.removeItem(RELOAD_TRACKING_KEY);
            sessionStorage.removeItem(RELOAD_TIMESTAMP_KEY);
        } catch {
        }
    }
}

export function boot(bundle?: string): Runtime | null {
    if (typeof window === 'undefined') return null;

    const script = document.getElementById('live-boot');
    let bootData: Boot | null = null;

    if (script?.textContent) {
        try {
            bootData = JSON.parse(script.textContent);
        } catch {
            Logger.error('Runtime', 'Failed to parse boot payload');
        }
    }

    if (!bootData) {
        bootData = window.__LIVEUI_BOOT__ ?? null;
    }

    if (!bootData || !isBoot(bootData)) {
        Logger.error('Runtime', 'No boot payload found');
        return null;
    }

    const config: RuntimeConfig = {
        root: document.documentElement,
        sessionId: bootData.sid,
        version: bootData.ver,
        bundle,
        seq: bootData.seq,
        endpoint: bootData.ws,
        token: bootData.tok,
        debug: bootData.client?.debug,
    };

    const runtime = new Runtime(config);
    runtime.handleBoot(bootData);
    runtime.connect();

    return runtime;
}

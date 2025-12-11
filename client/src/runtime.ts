import {
    Boot,
    Location,
    Patch,
    FramePatchPayload,
    ScriptMeta,
    isBoot,
    isServerError,
} from './protocol';
import { Bus } from './bus';
import { Transport, ConnectionState } from './transport';
import { Patcher } from './patcher';
import { Executor } from './executor';
import { ScriptExecutor } from './scripts';
import { Logger } from './logger';

export interface RuntimeConfig {
    root: Node;
    sessionId: string;
    version: number;
    seq: number;
    endpoint: string;
    location: Location;
    debug?: boolean;
}

const RELOAD_JITTER_MIN = 1000;
const RELOAD_JITTER_MAX = 10000;
const MAX_RELOADS = 10;
const RELOAD_TRACKING_KEY = 'pond_reload_count';
const RELOAD_TIMESTAMP_KEY = 'pond_reload_timestamp';
const RELOAD_WINDOW_MS = 60000;

interface ResumeOK {
    t: 'resume_ok';
    from: number;
    to: number;
}

function isResumeOK(msg: unknown): msg is ResumeOK {
    return (
        typeof msg === 'object' &&
        msg !== null &&
        (msg as ResumeOK).t === 'resume_ok'
    );
}

declare global {
    interface Window {
        __POND_RUNTIME__?: Runtime;
        __LIVEUI_BOOT__?: Boot;
    }
}

export class Runtime {
    private readonly bus: Bus;
    private readonly transport: Transport;
    private readonly patcher: Patcher;
    private readonly executor: Executor;
    private readonly scripts: ScriptExecutor;
    private readonly refs = new Map<string, Element>();

    private cseq = 0;
    private lastSeq = 0;
    private connectedState = false;

    constructor(config: RuntimeConfig) {
        this.lastSeq = config.seq;

        Logger.configure({ enabled: config.debug ?? false, level: 'debug' });
        Logger.info('Runtime', 'Initializing', { sid: config.sessionId, ver: config.version });

        this.bus = new Bus();

        this.transport = new Transport({
            endpoint: config.endpoint,
            sessionId: config.sessionId,
            version: config.version,
            lastAck: config.seq,
            location: config.location,
            bus: this.bus,
        });

        const resolveRef = (refId: string) => this.refs.get(refId);

        this.patcher = new Patcher(config.root, {
            onEvent: (handlerId, data) => this.handleEvent(handlerId, data),
            onRef: (refId, el) => {
                this.refs.set(refId, el);
                Logger.debug('Runtime', 'Ref set', refId);
            },
            onRefDelete: (refId) => {
                this.refs.delete(refId);
                Logger.debug('Runtime', 'Ref deleted', refId);
            },
            onScript: (meta, el) => this.handleScript(meta, el),
            onScriptCleanup: (scriptId) => this.handleScriptCleanup(scriptId),
        });

        this.executor = new Executor({
            bus: this.bus,
            transport: this.transport,
            resolveRef,
        });

        this.scripts = new ScriptExecutor({ bus: this.bus, transport: this.transport });

        this.bus.subscribe('frame', 'patch', (payload) => this.handlePatch(payload));

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
        this.executor.destroy();
        this.scripts.destroy();
        this.bus.clear();
        this.refs.clear();
    }

    connected(): boolean {
        return this.connectedState;
    }

    get seq(): number {
        return this.lastSeq;
    }

    handleBoot(boot: Boot): void {
        Logger.info('Runtime', 'Boot received', { ver: boot.ver, seq: boot.seq, patches: boot.patch?.length ?? 0 });

        if (boot.patch && boot.patch.length > 0) {
            this.applyPatches(boot.patch);
        }

        this.lastSeq = boot.seq;
        this.transport.sendAck(boot.seq);
    }

    handleMessage(msg: unknown): void {
        if (isServerError(msg)) {
            const err = msg as { code: string; message: string };
            Logger.error('Runtime', 'Server error', { code: err.code, message: err.message });
            return;
        }

        if (isResumeOK(msg)) {
            this.handleResumeOK(msg);
            return;
        }
    }

    private handlePatch(payload: FramePatchPayload): void {
        Logger.debug('Runtime', 'Patch received', { seq: payload.seq, count: payload.patches?.length ?? 0 });

        if (payload.patches && payload.patches.length > 0) {
            this.applyPatches(payload.patches);
        }

        this.lastSeq = payload.seq;
        this.transport.sendAck(payload.seq);
    }

    private handleResumeOK(resume: ResumeOK): void {
        Logger.info('Runtime', 'Resume OK', { from: resume.from, to: resume.to });
    }

    private handleEvent(handlerId: string, data: Record<string, unknown>): void {
        this.cseq++;
        Logger.debug('Runtime', 'Event', { handler: handlerId, cseq: this.cseq });

        const payload = {
            ...data,
            cseq: this.cseq,
        };

        this.transport.sendHandler(handlerId, payload);
    }

    private handleScript(meta: ScriptMeta, el: Element): void {
        Logger.debug('Runtime', 'Script execute', meta.scriptId);
        this.scripts.execute(meta, el).catch((err) => {
            Logger.error('Runtime', 'Script error', { scriptId: meta.scriptId, error: String(err) });
        });
    }

    private handleScriptCleanup(scriptId: string): void {
        Logger.debug('Runtime', 'Script cleanup', scriptId);
        this.scripts.cleanup(scriptId);
    }

    private handleStateChange(state: ConnectionState): void {
        Logger.debug('Runtime', 'Connection state', state);

        const wasConnected = this.connectedState;
        this.connectedState = state === 'connected';

        if (!wasConnected && this.connectedState) {
            Logger.info('Runtime', 'Connected');
            this.clearReloadTracking();
        } else if (wasConnected && !this.connectedState) {
            Logger.warn('Runtime', 'Disconnected');
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

    private applyPatches(patches: Patch[]): void {
        this.patcher.apply(patches);
    }
}

export function boot(): Runtime | null {
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
        seq: bootData.seq,
        endpoint: '/live',
        location: bootData.location,
        debug: bootData.client?.debug,
    };

    const runtime = new Runtime(config);
    runtime.handleBoot(bootData);
    runtime.connect();

    return runtime;
}

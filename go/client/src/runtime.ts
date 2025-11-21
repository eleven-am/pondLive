import {Transport} from './transport';
import {Patcher} from './patcher';
import {Router} from './router';
import {Uploader} from './uploader';
import {EffectExecutor} from './effects';
import {ScriptExecutor} from './scripts';
import {Logger} from './logger';
import {ChannelState} from '@eleven-am/pondsocket-client';
import {
    Boot,
    ClientEvent,
    DOMRequest,
    Frame,
    Init,
    Location as ProtoLocation,
    NavMessage,
    ResumeOK,
    ScriptEvent,
    ServerMessage
} from './protocol';
import {Effect, RouterMeta, ScriptMeta, UploadMeta} from './types';

export interface RuntimeConfig {
    root: Node;
    sessionId: string;
    version: number;
    seq: number;
    endpoint: string;
    uploadEndpoint: string;
    location: ProtoLocation;
    debug?: boolean;
}

export class Runtime {
    private connectedState = true;
    private readonly sessionId: string;
    private readonly transport: Transport;
    private readonly patcher: Patcher;
    private readonly router: Router;
    private readonly uploader: Uploader;
    private readonly effects: EffectExecutor;
    private readonly scripts: ScriptExecutor;
    private readonly refs = new Map<string, Element>();

    private cseq = 0;

    constructor(config: RuntimeConfig) {
        this.sessionId = config.sessionId;

        Logger.configure({enabled: config.debug ?? false, level: 'debug'});

        const resolveRef = (refId: string) => this.refs.get(refId);

        this.transport = new Transport({
            endpoint: config.endpoint,
            sessionId: config.sessionId,
            version: config.version,
            ack: config.seq,
            location: config.location
        });

        this.scripts = new ScriptExecutor({
            sessionId: config.sessionId,
            onMessage: (msg) => this.transport.send(msg)
        });

        this.patcher = new Patcher(config.root, {
            onEvent: (_event, handler, data) => this.handleEvent(handler, data),
            onRef: (refId, el) => this.refs.set(refId, el),
            onRefDelete: (refId) => this.refs.delete(refId),
            onRouter: (meta) => this.handleRouterClick(meta),
            onUpload: (meta, files) => this.handleUpload(meta, files),
            onScript: (meta, el) => this.handleScript(meta, el),
            onScriptCleanup: (scriptId) => this.scripts.cleanup(scriptId)
        });

        this.router = new Router((type, path, query, hash) => {
            this.sendNav(type, path, query, hash);
        });

        this.uploader = new Uploader({
            endpoint: config.uploadEndpoint,
            sessionId: config.sessionId,
            onMessage: (msg) => this.transport.send(msg)
        });

        this.effects = new EffectExecutor({
            sessionId: config.sessionId,
            resolveRef,
            onDOMResponse: (res) => this.transport.send(res)
        });

        this.transport.onMessage((msg) => this.handleMessage(msg));
        this.transport.onStateChange((state) => this.handleStateChange(state));
    }

    connect(): void {
        this.transport.connect();
        Logger.info('Runtime', 'Connected');
    }

    disconnect(): void {
        this.transport.disconnect();
        Logger.info('Runtime', 'Disconnected');
    }

    connected(): boolean {
        return this.connectedState;
    }

    private handleMessage(msg: ServerMessage): void {
        Logger.debug('Runtime', 'Received', msg.t);

        switch (msg.t) {
            case 'boot':
                this.handleBoot(msg);
                break;
            case 'init':
                this.handleInit(msg);
                break;
            case 'frame':
                this.handleFrame(msg);
                break;
            case 'resume_ok':
                this.handleResumeOK(msg);
                break;
            case 'dom_req':
                this.handleDOMRequest(msg);
                break;
            case 'script:event':
                this.handleScriptEvent(msg);
                break;
            case 'evt_ack':
                break;
            case 'error':
                Logger.error('Runtime', 'Server error', msg.code, msg.message);
                break;
            case 'diagnostic':
                Logger.warn('Runtime', 'Diagnostic', msg.code, msg.message);
                break;
        }
    }

    handleBoot(boot: Boot): void {
        Logger.info('Runtime', 'Boot received', {ver: boot.ver, seq: boot.seq, patches: boot.patch?.length ?? 0});
        if (boot.patch && boot.patch.length > 0) {
            this.patcher.apply(boot.patch);
        }

        this.sendAck(boot.seq);
    }

    private handleInit(init: Init): void {
        Logger.info('Runtime', 'Init received', {ver: init.ver, seq: init.seq});
        this.sendAck(init.seq);
    }

    private handleFrame(frame: Frame): void {
        Logger.debug('Runtime', 'Frame', {seq: frame.seq, ops: frame.patch?.length ?? 0});

        if (frame.patch && frame.patch.length > 0) {
            this.patcher.apply(frame.patch);
        }

        if (frame.effects && frame.effects.length > 0) {
            this.effects.execute(frame.effects as Effect[]);
        }

        if (frame.nav) {
            this.handleServerNav(frame.nav);
        }

        this.sendAck(frame.seq);
    }

    private handleResumeOK(resume: ResumeOK): void {
        Logger.info('Runtime', 'Resume OK', {from: resume.from, to: resume.to});
    }

    private handleDOMRequest(req: DOMRequest): void {
        this.effects.handleDOMRequest(req);
    }

    private handleServerNav(nav: { push?: string; replace?: string; back?: boolean }): void {
        if (nav.push) {
            window.history.pushState({}, '', nav.push);
        } else if (nav.replace) {
            window.history.replaceState({}, '', nav.replace);
        } else if (nav.back) {
            window.history.back();
        }
    }

    private handleEvent(handler: string, data: unknown): void {
        const event: ClientEvent = {
            t: 'evt',
            sid: this.sessionId,
            hid: handler,
            cseq: ++this.cseq,
            payload: (data as Record<string, unknown>) ?? {}
        };
        this.transport.send(event);
        Logger.debug('Runtime', 'Event sent', handler);
    }

    private handleRouterClick(meta: RouterMeta): void {
        this.router.navigate(meta);
    }

    private handleUpload(meta: UploadMeta, files: FileList): void {
        this.uploader.upload(meta, files);
    }

    private handleScript(meta: ScriptMeta, el: Element): void {
        this.scripts.execute(meta, el);
    }

    private handleScriptEvent(msg: ScriptEvent): void {
        this.scripts.handleEvent(msg.scriptId, msg.event, msg.data);
    }

    private sendNav(type: 'nav' | 'pop', path: string, query: string, hash: string): void {
        const msg: NavMessage = {t: type, sid: this.sessionId, path, q: query, hash};
        this.transport.send(msg);
        Logger.debug('Runtime', 'Nav sent', type, path);
    }

    private sendAck(seq: number): void {
        this.transport.send({t: 'ack', sid: this.sessionId, seq});
    }

    private handleStateChange(state: ChannelState): void {
        Logger.debug('Runtime', 'Channel state', state);
        if (state === ChannelState.STALLED || state === ChannelState.CLOSED) {
            this.connectedState = false;
        }
    }
}

export function boot(): Runtime | null {
    if (typeof window === 'undefined') return null;

    const script = document.getElementById('live-boot');
    let bootData: Boot | null = null;

    if (script?.textContent) {
        try {
            bootData = JSON.parse(script.textContent);
        } catch (e) {
            Logger.error('Runtime', 'Failed to parse boot payload', e);
        }
    }

    if (!bootData) {
        bootData = (window as unknown as { __LIVEUI_BOOT__?: Boot }).__LIVEUI_BOOT__ ?? null;
    }

    if (!bootData) {
        Logger.error('Runtime', 'No boot payload found');
        return null;
    }

    const config: RuntimeConfig = {
        root: document.documentElement,
        sessionId: bootData.sid,
        version: bootData.ver,
        seq: bootData.seq,
        endpoint: bootData.client?.endpoint ?? '/live',
        uploadEndpoint: bootData.client?.upload ?? '/pondlive/upload',
        location: bootData.location,
        debug: bootData.client?.debug
    };

    const runtime = new Runtime(config);
    runtime.handleBoot(bootData);
    runtime.connect();
    return runtime;
}

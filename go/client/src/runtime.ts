import { PondClient, ChannelState } from '@eleven-am/pondsocket-client';
import { BootPayload, FramePayload, InitPayload } from './types';
import { Logger } from './logger';
import { hydrate } from './vdom';
import { Patcher } from './patcher';
import { ClientNode } from './types';
import { EventManager } from './events';
import { Router } from './router';
import { DOMActionExecutor } from './dom_actions';
import { UploadManager, UploadRuntime, ClientConfig } from './uploads';

export class LiveRuntime implements UploadRuntime {
    private client!: PondClient;
    private channel: any;
    private config: ClientConfig = {};
    private root: ClientNode | null = null;
    private patcher!: Patcher;
    private eventManager!: EventManager;
    private router!: Router;
    private uploadManager!: UploadManager;
    private refs = new Map<string, ClientNode>();
    private domActions!: DOMActionExecutor;

    private readonly sessionId: string = '';

    constructor() {
        const boot = this.getBootPayload();
        if (!boot) {
            Logger.error('Runtime', 'No boot payload found');
            return;
        }

        this.sessionId = boot.sid;
        this.config = boot.client || {};

        Logger.configure({ debug: boot.client?.debug });
        Logger.debug('Runtime', 'Booting...', boot);

        this.connect(boot);
        this.hydrate(boot);
    }

    private getBootPayload(): BootPayload | null {
        if (typeof window === 'undefined') return null;
        const script = document.getElementById('live-boot');
        if (script && script.textContent) {
            try {
                return JSON.parse(script.textContent);
            } catch (e) {
                Logger.error('Runtime', 'Failed to parse boot payload', e);
            }
        }
        return (window as any).__LIVEUI_BOOT__ || null;
    }

    private hydrate(boot: BootPayload) {
        try {
            const jsonTree = JSON.parse(boot.json);

            
            
            

            
            function findHtmlElement(node: any): Element | null {
                if (node.tag === 'html') {
                    return document.documentElement;
                }
                if (node.children) {
                    for (const child of node.children) {
                        const result = findHtmlElement(child);
                        if (result) return result;
                    }
                }
                return null;
            }

            const htmlElement = findHtmlElement(jsonTree);
            if (!htmlElement) {
                Logger.error('Runtime', 'Could not find <html> element in JSON tree');
                return;
            }

            
            this.root = this.hydrateWithComponentWrappers(jsonTree, htmlElement);

            
            if (this.eventManager && this.root) {
                this.eventManager.attach(this.root);
            }

            
            if (this.router && this.root) {
                this.attachRouterRecursively(this.root);
            }

            if (this.eventManager && this.router && this.uploadManager) {
                this.patcher = new Patcher(this.root, this.eventManager, this.router, this.uploadManager, this.refs);
            }

            Logger.debug('Runtime', 'Hydration complete');
        } catch (e) {
            Logger.error('Runtime', 'Hydration failed', e);
        }
    }

    private hydrateWithComponentWrappers(jsonNode: any, htmlElement: Element): ClientNode {
        
        if (jsonNode.tag === 'html') {
            return hydrate(jsonNode, htmlElement, this.refs);
        }

        const clientNode: ClientNode = {
            ...jsonNode,
            el: null,
            children: undefined
        };

        if (jsonNode.componentId) {
            clientNode.componentId = jsonNode.componentId;
        }

        if (jsonNode.children && jsonNode.children.length > 0) {
            clientNode.children = [];
            for (const child of jsonNode.children) {
                const childNode = this.hydrateWithComponentWrappers(child, htmlElement);
                clientNode.children.push(childNode);
            }
        }

        // attach outermost HTML element to the first element child under this wrapper
        if (clientNode.el === null && clientNode.children && clientNode.children.length === 1 && clientNode.children[0].tag === 'html') {
            clientNode.el = htmlElement;
        }

        return clientNode;
    }

    private connect(boot: BootPayload) {
        const endpoint = boot.client?.endpoint || '/live';
        this.client = new PondClient(endpoint);

        const joinPayload = {
            sid: boot.sid,
            ver: boot.ver,
            ack: boot.seq,
            loc: boot.location
        };

        this.channel = this.client.createChannel(`live/${boot.sid}`, joinPayload);
        this.eventManager = new EventManager(this.channel, boot.sid);
        this.router = new Router(this.channel, boot.sid);
        this.uploadManager = new UploadManager(this);
        this.domActions = new DOMActionExecutor(this.refs);

        this.channel.onChannelStateChange((state: ChannelState) => {
            Logger.debug('Runtime', 'Channel state:', state);
        });

        this.channel.onMessage((event: string, payload: any) => {
            Logger.debug('WS Recv', event, payload);
            this.handleMessage(payload);
        });

        this.client.connect();
        this.channel.join();
    }

    getSessionId(): string | undefined {
        return this.sessionId;
    }

    getUploadEndpoint(): string {
        return this.config.upload || '/pondlive/upload';
    }

    sendUploadMessage(payload: any) {
        Logger.debug('WS Send', { t: 'upload', ...payload });
        this.channel.sendMessage({ t: 'upload', ...payload });
    }

    private handleMessage(msg: any) {
        switch (msg.t) {
            case 'frame':
                this.handleFrame(msg as FramePayload);
                break;
            case 'init':
                this.handleInit(msg as InitPayload);
                break;
            case 'domreq':
                this.handleDOMRequest(msg);
                break;
            case 'upload':
                this.uploadManager.handleControl(msg);
                break;
            default:
                Logger.debug('Runtime', 'Unknown message type', msg.t);
        }
    }

    private handleFrame(frame: FramePayload) {
        Logger.debug('Runtime', 'Received frame', { seq: frame.seq, ops: frame.patch.length });

        if (this.patcher && frame.patch) {
            // Apply patches in server order to preserve index assumptions (moves/adds/dels).
            for (const op of frame.patch) {
                this.patcher.apply(op);
            }
        }

        if (frame.effects) {
            this.domActions.execute(frame.effects);
        }
    }

    private handleInit(init: InitPayload) {
        Logger.debug('Runtime', 'Re-initialized', init);
        
    }

    private handleDOMRequest(req: any) {
        const { id, ref, props, method, args } = req;

        
        const node = this.refs.get(ref);
        if (!node || !node.el) {
            this.sendDOMResponse({ t: 'domres', id, error: 'ref not found' });
            return;
        }

        const el = node.el as any;

        try {
            let result: any;
            let values: any;

            
            if (props && Array.isArray(props)) {
                values = {};
                for (const prop of props) {
                    values[prop] = el[prop];
                }
            }

            
            if (method && typeof el[method] === 'function') {
                result = el[method](...(args || []));
            }

            this.sendDOMResponse({ t: 'domres', id, result, values });
        } catch (e: any) {
            this.sendDOMResponse({ t: 'domres', id, error: e.message || 'unknown error' });
        }
    }

    private sendDOMResponse(response: any) {
        const payload = { ...response, sid: this.sessionId };
        Logger.debug('WS Send', 'domres', payload);
        this.channel.sendMessage('domres', payload);
    }

    private attachRouterRecursively(node: ClientNode) {
        if (this.router) {
            this.router.attach(node);
        }
        if (node.children) {
            for (const child of node.children) {
                this.attachRouterRecursively(child);
            }
        }
    }
}

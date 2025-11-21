import { ClientNode, HandlerMeta } from './types';
import { Logger } from './logger';
import { extractEventDetail } from './event-detail';

type ListenerRecord = {
    handlerId: string;
    listener: EventListener;
};

export class EventManager {
    private listeners = new WeakMap<Node, Map<string, ListenerRecord[]>>();

    constructor(private channel: any, private sid: string) { }

    attach(node: ClientNode) {
        if (!node) return;

        
        if (node.el && node.handlers && node.handlers.length > 0) {
            this.bindEvents(node);
        }

        
        if (node.children) {
            for (const child of node.children) {
                this.attach(child);
            }
        }
    }

    detach(node: ClientNode) {
        if (!node) return;

        
        if (node.el) {
            this.unbindEvents(node.el);
        }

        
        if (node.children) {
            for (const child of node.children) {
                this.detach(child);
            }
        }
    }

    private bindEvents(node: ClientNode) {
        if (!node.el || !node.handlers) return;

        const el = node.el;
        let nodeListeners = this.listeners.get(el);
        if (!nodeListeners) {
            nodeListeners = new Map();
            this.listeners.set(el, nodeListeners);
        }

        for (const h of node.handlers) {
            if (!h || !h.event || !h.handler) continue;

            const existing = nodeListeners.get(h.event) || [];
            const duplicate = existing.some((rec) => rec.handlerId === h.handler);
            if (duplicate) {
                continue;
            }

            const listener = (e: Event) => {
                const preventDefault = !(h.listen && h.listen.includes('allowDefault'));
                if (preventDefault && e.cancelable) {
                    e.preventDefault();
                }

                this.triggerHandler(h, e, node);

                if (!h.listen || !h.listen.includes('bubble')) {
                    e.stopPropagation();
                }
            };

            el.addEventListener(h.event, listener);
            nodeListeners.set(h.event, [...existing, { handlerId: h.handler, listener }]);
            Logger.debug('Events', 'Attached listener', { event: h.event, handler: h.handler });
        }
    }

    private unbindEvents(el: Node) {
        const nodeListeners = this.listeners.get(el);
        if (!nodeListeners) return;

        for (const [event, records] of nodeListeners.entries()) {
            for (const rec of records) {
                el.removeEventListener(event, rec.listener);
            }
        }
        this.listeners.delete(el);
    }

    private triggerHandler(handler: HandlerMeta, e: Event, node: ClientNode) {
        Logger.debug('Events', 'Triggering handler', { handlerId: handler.handler, type: e.type });

        const refElement = node.el instanceof Element ? node.el : undefined;
        const detail = extractEventDetail(e, handler.props, { refElement });

        const payload: any = {
            name: e.type
        };

        if (detail !== undefined) {
            payload.detail = detail;
        }

        Logger.debug('WS Send', 'evt', {
            t: 'evt',
            sid: this.sid,
            hid: handler.handler,
            payload: payload
        });
        this.channel.sendMessage('evt', {
            t: 'evt',
            sid: this.sid,
            hid: handler.handler,
            payload: payload
        });
    }
}

/**
 * Event Emitter
 *
 * Simple type-safe event emitter for lifecycle hooks
 */

export type EventHandler<T = any> = (data: T) => void;

export class EventEmitter<EventMap extends Record<string, any>> {
    private listeners = new Map<keyof EventMap, Set<EventHandler>>();

    on<K extends keyof EventMap>(event: K, handler: EventHandler<EventMap[K]>): () => void {
        if (!this.listeners.has(event)) {
            this.listeners.set(event, new Set());
        }
        this.listeners.get(event)!.add(handler);

        // Return unsubscribe function
        return () => this.off(event, handler);
    }

    off<K extends keyof EventMap>(event: K, handler: EventHandler<EventMap[K]>): void {
        const handlers = this.listeners.get(event);
        if (handlers) {
            handlers.delete(handler);
            if (handlers.size === 0) {
                this.listeners.delete(event);
            }
        }
    }

    emit<K extends keyof EventMap>(event: K, data: EventMap[K]): void {
        const handlers = this.listeners.get(event);
        if (handlers) {
            handlers.forEach(handler => {
                try {
                    handler(data);
                } catch (error) {
                    console.error(`Error in event handler for ${String(event)}:`, error);
                }
            });
        }
    }

    once<K extends keyof EventMap>(event: K, handler: EventHandler<EventMap[K]>): () => void {
        const wrappedHandler = (data: EventMap[K]) => {
            handler(data);
            this.off(event, wrappedHandler);
        };
        return this.on(event, wrappedHandler);
    }

    removeAllListeners(event?: keyof EventMap): void {
        if (event) {
            this.listeners.delete(event);
        } else {
            this.listeners.clear();
        }
    }

    listenerCount(event: keyof EventMap): number {
        return this.listeners.get(event)?.size ?? 0;
    }
}

import {Logger} from './logger';

export type EventMap = Record<string, any>;

export type Listener<T> = (payload: T) => void;

export class TypedEventEmitter<Events extends EventMap> {
  private listeners: Map<keyof Events, Set<Listener<any>>> = new Map();

  on<K extends keyof Events>(event: K, listener: Listener<Events[K]>): () => void {
    const bucket = this.listeners.get(event) ?? new Set();
    bucket.add(listener as Listener<any>);
    this.listeners.set(event, bucket);
    return () => this.off(event, listener);
  }

  once<K extends keyof Events>(event: K, listener: Listener<Events[K]>): () => void {
    const unsubscribe = this.on(event, (payload) => {
      unsubscribe();
      listener(payload);
    });
    return unsubscribe;
  }

  off<K extends keyof Events>(event: K, listener: Listener<Events[K]>): void {
    const bucket = this.listeners.get(event);
    if (!bucket) {
      return;
    }
    bucket.delete(listener as Listener<any>);
    if (bucket.size === 0) {
      this.listeners.delete(event);
    }
  }

  emit<K extends keyof Events>(event: K, payload: Events[K]): void {
    const bucket = this.listeners.get(event);
    if (!bucket) {
      return;
    }
    for (const listener of Array.from(bucket)) {
      try {
        (listener as Listener<Events[K]>)(payload);
      } catch (error) {
        Logger.error('event listener error', event, error);
      }
    }
  }

  clear(): void {
    this.listeners.clear();
  }
}

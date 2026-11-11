import {
    Topic,
    ActionFor,
    PayloadFor,
} from './protocol';
import { Logger } from './logger';

type SubscriberKey = `${Topic}:${string}`;
type Callback<P> = (payload: P) => void;

interface Subscriber {
    id: number;
    callback: Callback<never>;
}

interface WildcardSubscriber {
    id: number;
    callback: (topic: Topic, action: string, payload: unknown) => void;
}

export interface Subscription {
    unsubscribe: () => void;
}

export class Bus {
    private subscribers = new Map<SubscriberKey, Subscriber[]>();
    private wildcardSubscribers: WildcardSubscriber[] = [];
    private nextSubId = 0;

    subscribe<T extends Topic, A extends ActionFor<T>>(
        topic: T,
        action: A,
        callback: Callback<PayloadFor<T, A>>
    ): Subscription {
        const key: SubscriberKey = `${topic}:${String(action)}`;
        const subId = ++this.nextSubId;
        const sub: Subscriber = {
            id: subId,
            callback: callback as Callback<never>
        };

        const subs = this.subscribers.get(key) ?? [];
        subs.push(sub);
        this.subscribers.set(key, subs);

        return {
            unsubscribe: () => this.unsubscribe(key, subId),
        };
    }

    upsert<T extends Topic, A extends ActionFor<T>>(
        topic: T,
        action: A,
        callback: Callback<PayloadFor<T, A>>
    ): Subscription {
        const key: SubscriberKey = `${topic}:${String(action)}`;
        const subs = this.subscribers.get(key);

        if (subs && subs.length > 0) {
            const first = subs[0];
            first.callback = callback as Callback<never>;
            if (subs.length > 1) {
                this.subscribers.set(key, [first]);
            }
            const subId = first.id;
            return {
                unsubscribe: () => this.unsubscribe(key, subId),
            };
        }

        const subId = ++this.nextSubId;
        const sub: Subscriber = {
            id: subId,
            callback: callback as Callback<never>
        };
        this.subscribers.set(key, [sub]);

        return {
            unsubscribe: () => this.unsubscribe(key, subId),
        };
    }

    subscribeAll(
        callback: (topic: Topic, action: string, payload: unknown) => void
    ): Subscription {
        const subId = ++this.nextSubId;
        const sub: WildcardSubscriber = { id: subId, callback };
        this.wildcardSubscribers.push(sub);

        return {
            unsubscribe: () => this.unsubscribeWildcard(subId),
        };
    }

    publish<T extends Topic, A extends ActionFor<T>>(
        topic: T,
        action: A,
        payload: PayloadFor<T, A>
    ): void {
        this.deliver(topic, String(action), payload);
    }

    subscriberCount<T extends Topic, A extends ActionFor<T>>(topic: T, action: A): number {
        const key: SubscriberKey = `${topic}:${String(action)}`;
        return this.subscribers.get(key)?.length ?? 0;
    }

    clear(): void {
        this.subscribers.clear();
        this.wildcardSubscribers = [];
    }

    private deliver(topic: Topic, action: string, payload: unknown): void {
        const key = `${topic}:${action}` as SubscriberKey;
        const subs = this.subscribers.get(key) ?? [];

        for (const sub of subs) {
            try {
                (sub.callback as Callback<unknown>)(payload);
            } catch (error) {
                Logger.error('Bus', 'Subscriber failed', { topic, action, error });
            }
        }

        for (const sub of this.wildcardSubscribers) {
            try {
                sub.callback(topic, action, payload);
            } catch (error) {
                Logger.error('Bus', 'Subscriber failed', { topic, action, error });
            }
        }
    }

    private unsubscribe(key: SubscriberKey, subId: number): void {
        const subs = this.subscribers.get(key);
        if (!subs) return;

        const idx = subs.findIndex((s) => s.id === subId);
        if (idx !== -1) {
            subs.splice(idx, 1);
            if (subs.length === 0) {
                this.subscribers.delete(key);
            }
        }
    }

    private unsubscribeWildcard(subId: number): void {
        const idx = this.wildcardSubscribers.findIndex((s) => s.id === subId);
        if (idx !== -1) {
            this.wildcardSubscribers.splice(idx, 1);
        }
    }
}

import {
    Topic,
    StaticTopic,
    ScriptTopic,
    HandlerTopic,
    ActionFor,
    PayloadFor,
    ScriptTopicActions,
    HandlerTopicActions,
    scriptTopic,
    handlerTopic,
} from './protocol';

type SubscriberKey = `${StaticTopic}:${string}` | `${ScriptTopic}:${string}` | `${HandlerTopic}:${string}`;
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
        const key: SubscriberKey = `${topic}:${String(action)}`;
        const subs = this.subscribers.get(key) ?? [];

        for (const sub of subs) {
            try {
                (sub.callback as Callback<PayloadFor<T, A>>)(payload);
            } catch {
                // swallow to prevent cascade
            }
        }

        for (const sub of this.wildcardSubscribers) {
            try {
                sub.callback(topic, String(action), payload);
            } catch {
                // swallow
            }
        }
    }

    subscriberCount<T extends Topic, A extends ActionFor<T>>(topic: T, action: A): number {
        const key: SubscriberKey = `${topic}:${String(action)}`;
        return this.subscribers.get(key)?.length ?? 0;
    }

    subscribeScript<A extends keyof ScriptTopicActions>(
        scriptId: string,
        action: A,
        callback: Callback<ScriptTopicActions[A]>
    ): Subscription {
        return this.subscribe(scriptTopic(scriptId), action, callback);
    }

    publishScript<A extends keyof ScriptTopicActions>(
        scriptId: string,
        action: A,
        payload: ScriptTopicActions[A]
    ): void {
        const topic = scriptTopic(scriptId);
        const key: SubscriberKey = `${topic}:${String(action)}`;
        const subs = this.subscribers.get(key) ?? [];

        for (const sub of subs) {
            try {
                (sub.callback as Callback<ScriptTopicActions[A]>)(payload);
            } catch {
            }
        }

        for (const sub of this.wildcardSubscribers) {
            try {
                sub.callback(topic, String(action), payload);
            } catch {
            }
        }
    }

    publishHandler(
        handlerId: string,
        payload: HandlerTopicActions['invoke']
    ): void {
        const topic = handlerTopic(handlerId);
        const key: SubscriberKey = `${topic}:invoke`;
        const subs = this.subscribers.get(key) ?? [];

        for (const sub of subs) {
            try {
                (sub.callback as Callback<HandlerTopicActions['invoke']>)(payload);
            } catch {
            }
        }

        for (const sub of this.wildcardSubscribers) {
            try {
                sub.callback(topic, 'invoke', payload);
            } catch {
            }
        }
    }

    clear(): void {
        this.subscribers.clear();
        this.wildcardSubscribers = [];
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

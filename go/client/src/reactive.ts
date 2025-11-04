/**
 * Reactive Signal System
 *
 * Lightweight reactive primitives for state management
 */

export class Signal<T> {
    private value: T;
    private subscribers = new Set<(value: T) => void>();

    constructor(initial: T) {
        this.value = initial;
    }

    get(): T {
        return this.value;
    }

    set(newValue: T): void {
        if (this.value !== newValue) {
            this.value = newValue;
            this.notify();
        }
    }

    update(updater: (current: T) => T): void {
        this.set(updater(this.value));
    }

    subscribe(fn: (value: T) => void): () => void {
        this.subscribers.add(fn);
        // Immediately call with current value
        fn(this.value);
        // Return unsubscribe function
        return () => this.subscribers.delete(fn);
    }

    private notify(): void {
        this.subscribers.forEach(fn => fn(this.value));
    }
}

export class ComputedSignal<T> {
    private value: T;
    private subscribers = new Set<(value: T) => void>();
    private unsubscribers: Array<() => void> = [];

    constructor(compute: () => T, dependencies: Signal<any>[]) {
        this.value = compute();

        // Subscribe to all dependencies
        dependencies.forEach(dep => {
            const unsub = dep.subscribe(() => {
                const newValue = compute();
                if (this.value !== newValue) {
                    this.value = newValue;
                    this.notify();
                }
            });
            this.unsubscribers.push(unsub);
        });
    }

    get(): T {
        return this.value;
    }

    subscribe(fn: (value: T) => void): () => void {
        this.subscribers.add(fn);
        fn(this.value);
        return () => this.subscribers.delete(fn);
    }

    destroy(): void {
        this.unsubscribers.forEach(unsub => unsub());
        this.unsubscribers = [];
        this.subscribers.clear();
    }

    private notify(): void {
        this.subscribers.forEach(fn => fn(this.value));
    }
}

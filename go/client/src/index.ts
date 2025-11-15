import { LiveRuntime } from './runtime';
import type { RuntimeOptions } from './runtime';
import { HydrationManager } from './hydration';
import { EventDelegation } from './event-delegation';
import type { BootPayload, ConnectionState, EventPayload } from './types';

export class LiveUI {
  private readonly runtime: LiveRuntime;
  private readonly _hydration: HydrationManager;
  private readonly events: EventDelegation;

  constructor(options?: RuntimeOptions) {
    this.runtime = new LiveRuntime(options);
    this._hydration = new HydrationManager(this.runtime);
    this.events = new EventDelegation(this._hydration.getRegistry(), this.runtime);
    this.events.setup();
  }

  connect(): Promise<void> {
    return this.runtime.connect();
  }

  disconnect(): void {
    this.runtime.disconnect();
    this.events.teardown();
  }

  destroy(): void {
    this.runtime.destroy();
    this.events.teardown();
  }

  getState(): ConnectionState {
    return this.runtime.getState();
  }

  getBootPayload(): BootPayload | null {
    return this.runtime.getBootPayload();
  }

  on<K extends Parameters<LiveRuntime['on']>[0]>(
    event: K,
    listener: Parameters<LiveRuntime['on']>[1],
  ): ReturnType<LiveRuntime['on']> {
    return this.runtime.on(event, listener);
  }

  once<K extends Parameters<LiveRuntime['once']>[0]>(
    event: K,
    listener: Parameters<LiveRuntime['once']>[1],
  ): ReturnType<LiveRuntime['once']> {
    return this.runtime.once(event, listener);
  }

  off<K extends Parameters<LiveRuntime['off']>[0]>(
    event: K,
    listener: Parameters<LiveRuntime['off']>[1],
  ): void {
    this.runtime.off(event, listener);
  }

  sendEvent(handlerId: string, payload: EventPayload, cseq?: number): void {
    this.runtime.sendEvent(handlerId, payload, cseq);
  }

  sendNavigation(path: string, q: string, hash?: string): void {
    this.runtime.sendNavigation(path, q, hash);
  }

  getHydrationManager(): HydrationManager {
    return this._hydration;
  }
}

export { LiveRuntime } from './runtime';
export type { RuntimeOptions } from './runtime';
export * from './types';
export default LiveUI;

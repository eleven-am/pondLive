import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../src/patcher', () => ({
  applyOps: vi.fn(),
  configurePatcher: vi.fn(),
  getPatcherConfig: vi.fn(),
  clearPatcherCaches: vi.fn(),
  getPatcherStats: vi.fn(),
  morphElement: vi.fn(),
  computeInverseOps: vi.fn().mockReturnValue([]),
}));

vi.mock('@eleven-am/pondsocket-client', () => {
  return {
    default: vi.fn().mockImplementation(() => ({
      createChannel: vi.fn().mockReturnValue({
        onJoin: vi.fn(),
        onMessage: vi.fn(),
        onLeave: vi.fn(),
        join: vi.fn(),
        send: vi.fn(),
        leave: vi.fn(),
      }),
      connect: vi.fn(),
      disconnect: vi.fn(),
    })),
  };
});

import { applyOps } from '../src/patcher';
import LiveUI from '../src/index';
import type { ConnectionState } from '../src/types';

describe('LiveUI batching and state management', () => {
  const originalRAF = (globalThis as any).requestAnimationFrame;
  const originalCancelRAF = (globalThis as any).cancelAnimationFrame;

  beforeEach(() => {
    vi.useFakeTimers();
    (globalThis as any).requestAnimationFrame = undefined;
    (globalThis as any).cancelAnimationFrame = undefined;
  });

  afterEach(() => {
    vi.useRealTimers();
    (globalThis as any).requestAnimationFrame = originalRAF;
    (globalThis as any).cancelAnimationFrame = originalCancelRAF;
    vi.restoreAllMocks();
  });

  it('updates connection state when reconnect attempt changes', () => {
    const live = new LiveUI({ autoConnect: false });
    const changes: Array<{ from: ConnectionState; to: ConnectionState }> = [];
    live.on('stateChanged', change => changes.push(change));

    (live as any).setState({ status: 'reconnecting', attempt: 1 });
    expect(live.getConnectionState()).toEqual({ status: 'reconnecting', attempt: 1 });

    (live as any).setState({ status: 'reconnecting', attempt: 2 });
    expect(live.getConnectionState()).toEqual({ status: 'reconnecting', attempt: 2 });

    const attempts = changes.map(change => (change.to as any).attempt).filter(Boolean);
    expect(attempts).toEqual([1, 2]);
  });

  it('falls back to setTimeout when requestAnimationFrame is unavailable', () => {
    const live = new LiveUI({ autoConnect: false });
    const setTimeoutSpy = vi.spyOn(globalThis, 'setTimeout');
    const clearTimeoutSpy = vi.spyOn(globalThis, 'clearTimeout');

    const nowValues = [0, 5, 0, 15];
    vi.spyOn(performance, 'now').mockImplementation(() => {
      const value = nowValues.shift();
      return value !== undefined ? value : 0;
    });

    (live as any).patchQueue = [["setText", 0, 'hello']];
    (live as any).scheduleBatch();

    expect(setTimeoutSpy).toHaveBeenCalled();
    expect((live as any).batchScheduled).toBe(true);

    vi.runOnlyPendingTimers();

    expect(applyOps).toHaveBeenCalledWith([["setText", 0, 'hello']]);
    expect(live.getMetrics().averagePatchTime).toBe(5);

    // Queue a second batch to validate rolling average
    (live as any).patchQueue = [["setText", 0, 'world']];
    (live as any).scheduleBatch();
    vi.runOnlyPendingTimers();

    expect(applyOps).toHaveBeenCalledWith([["setText", 0, 'world']]);
    expect(live.getMetrics().averagePatchTime).toBe(10);

    // Schedule a third batch but disconnect before it runs to ensure timers are cleared
    (live as any).patchQueue = [["setText", 0, '!']];
    (live as any).scheduleBatch();

    live.disconnect();
    expect(clearTimeoutSpy).toHaveBeenCalled();
  });
});

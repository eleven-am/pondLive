import { describe, it, expect, beforeEach, vi } from 'vitest';
import type { BootPayload } from '../src';

interface MockChannel {
  emitState(state: string): void;
  emitMessage(payload: any): void;
  sentMessages: { event: string; payload: any }[];
}

const mockControl: { channels: MockChannel[] } = { channels: [] };

vi.mock('@eleven-am/pondsocket-client', () => {
  const ChannelState = { JOINED: 'JOINED' } as const;
  class MockChannelImpl {
    public sentMessages: { event: string; payload: any }[] = [];
    private stateHandlers: Array<(state: string) => void> = [];
    private messageHandlers: Array<(event: string, payload: any) => void> = [];
    private leaveHandlers: Array<() => void> = [];

    constructor(public topic: string, public payload: any) {}

    onChannelStateChange(handler: (state: string) => void) {
      this.stateHandlers.push(handler);
    }

    onMessage(handler: (event: string, payload: any) => void) {
      this.messageHandlers.push(handler);
    }

    onLeave(handler: () => void) {
      this.leaveHandlers.push(handler);
    }

    sendMessage(event: string, payload: any) {
      this.sentMessages.push({ event, payload });
    }

    join() {}
    leave() {
      this.leaveHandlers.forEach((handler) => handler());
    }

    emitState(state: string) {
      this.stateHandlers.forEach((handler) => handler(state));
    }

    emitMessage(payload: any) {
      this.messageHandlers.forEach((handler) => handler(payload.t ?? '', payload));
    }
  }

  class MockPondClient {
    createChannel(topic: string, payload: any) {
      const channel = new MockChannelImpl(topic, payload);
      mockControl.channels.push(channel);
      return channel;
    }
    connect() {}
    disconnect() {}
  }

  return { PondClient: MockPondClient, ChannelState };
});

import { LiveRuntime } from '../src';

const baseBoot: BootPayload = {
  t: 'boot',
  sid: 'sid-123',
  ver: 1,
  seq: 1,
  location: { path: '/', q: '', hash: '' },
  s: [],
  d: [],
  slots: [],
};

describe('LiveRuntime', () => {
  beforeEach(() => {
    mockControl.channels.length = 0;
    (window as any).__LIVEUI_BOOT__ = { ...baseBoot };
  });

  it('connects and emits init/frame events', async () => {
    const runtime = new LiveRuntime({ autoConnect: false, reconnect: false });
    const initSpy = vi.fn();
    runtime.on('init', initSpy);

    const connectPromise = runtime.connect();
    const channel = mockControl.channels[mockControl.channels.length - 1];
    channel.emitState('JOINED');
    await connectPromise;

    const initMessage = {
      t: 'init',
      sid: baseBoot.sid,
      ver: baseBoot.ver,
      location: baseBoot.location,
      s: [],
      d: [],
      slots: [],
      seq: 5,
    };
    channel.emitMessage(initMessage);
    expect(initSpy).toHaveBeenCalledWith(initMessage);

    const ack = channel.sentMessages[channel.sentMessages.length - 1];
    expect(ack?.event).toBe('ack');
    expect(ack?.payload.seq).toBe(5);
  });

  it('sends events through the channel', async () => {
    const runtime = new LiveRuntime({ autoConnect: false, reconnect: false });
    const connectPromise = runtime.connect();
    const channel = mockControl.channels[mockControl.channels.length - 1];
    channel.emitState('JOINED');
    await connectPromise;

    runtime.sendEvent('handler-1', { name: 'click' });
    const evtMessage = channel.sentMessages.find((msg) => msg.event === 'evt');
    expect(evtMessage?.payload.hid).toBe('handler-1');
  });
});

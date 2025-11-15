import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { HydrationManager } from '../src/hydration';
import { LiveRuntime } from '../src/runtime';
import type { FrameMessage, CookieEffect } from '../src/types';

describe('CookieEffect', () => {
  let manager: HydrationManager;
  let runtime: LiveRuntime;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    document.body.innerHTML = '';
    runtime = new LiveRuntime({ autoConnect: false });
    manager = new HydrationManager(runtime);

    fetchMock = vi.fn(() =>
      Promise.resolve({
        ok: true,
        status: 200,
      } as Response)
    );
    global.fetch = fetchMock;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const createFrameWithEffect = (effect: CookieEffect): FrameMessage => ({
    t: 'frame',
    sid: 'test',
    ver: 1,
    seq: 1,
    effects: [effect],
  });

  it('should make POST request to endpoint with credentials', async () => {
    const effect: CookieEffect = {
      type: 'cookies',
      endpoint: '/api/set-cookies',
      sid: 'session-123',
      token: 'token-abc',
    };

    const frame = createFrameWithEffect(effect);
    (manager as any).applyFrame(frame);

    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/set-cookies',
      expect.objectContaining({
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          sid: 'session-123',
          token: 'token-abc',
        }),
        credentials: 'include',
      })
    );
  });

  it('should use custom method if provided', async () => {
    const effect: CookieEffect = {
      type: 'cookies',
      endpoint: '/api/set-cookies',
      sid: 'session-123',
      token: 'token-abc',
      method: 'PUT',
    };

    const frame = createFrameWithEffect(effect);
    (manager as any).applyFrame(frame);

    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/set-cookies',
      expect.objectContaining({
        method: 'PUT',
      })
    );
  });

  it('should skip if endpoint is missing', async () => {
    const effect = {
      type: 'cookies',
      endpoint: '',
      sid: 'session-123',
      token: 'token-abc',
    } as CookieEffect;

    const frame = createFrameWithEffect(effect);
    (manager as any).applyFrame(frame);

    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('should skip if sid is missing', async () => {
    const effect = {
      type: 'cookies',
      endpoint: '/api/set-cookies',
      sid: '',
      token: 'token-abc',
    } as CookieEffect;

    const frame = createFrameWithEffect(effect);
    (manager as any).applyFrame(frame);

    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('should skip if token is missing', async () => {
    const effect = {
      type: 'cookies',
      endpoint: '/api/set-cookies',
      sid: 'session-123',
      token: '',
    } as CookieEffect;

    const frame = createFrameWithEffect(effect);
    (manager as any).applyFrame(frame);

    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('should handle fetch failure gracefully', async () => {
    fetchMock.mockRejectedValueOnce(new Error('Network error'));

    const effect: CookieEffect = {
      type: 'cookies',
      endpoint: '/api/set-cookies',
      sid: 'session-123',
      token: 'token-abc',
    };

    const frame = createFrameWithEffect(effect);

    expect(() => (manager as any).applyFrame(frame)).not.toThrow();

    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(fetchMock).toHaveBeenCalled();
  });

  it('should handle non-ok response gracefully', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 401,
    } as Response);

    const effect: CookieEffect = {
      type: 'cookies',
      endpoint: '/api/set-cookies',
      sid: 'session-123',
      token: 'token-abc',
    };

    const frame = createFrameWithEffect(effect);

    expect(() => (manager as any).applyFrame(frame)).not.toThrow();

    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(fetchMock).toHaveBeenCalled();
  });

  it('should include credentials for cross-origin requests', async () => {
    const effect: CookieEffect = {
      type: 'cookies',
      endpoint: 'https://other-domain.com/api/set-cookies',
      sid: 'session-123',
      token: 'token-abc',
    };

    const frame = createFrameWithEffect(effect);
    (manager as any).applyFrame(frame);

    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(fetchMock).toHaveBeenCalledWith(
      'https://other-domain.com/api/set-cookies',
      expect.objectContaining({
        credentials: 'include',
      })
    );
  });

  it('should handle multiple cookie effects in sequence', async () => {
    const frame: FrameMessage = {
      t: 'frame',
      sid: 'test',
      ver: 1,
      seq: 1,
      effects: [
        {
          type: 'cookies',
          endpoint: '/api/set-cookies-1',
          sid: 'session-1',
          token: 'token-1',
        },
        {
          type: 'cookies',
          endpoint: '/api/set-cookies-2',
          sid: 'session-2',
          token: 'token-2',
        },
      ],
    };

    (manager as any).applyFrame(frame);

    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/set-cookies-1',
      expect.objectContaining({
        body: JSON.stringify({
          sid: 'session-1',
          token: 'token-1',
        }),
      })
    );
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/set-cookies-2',
      expect.objectContaining({
        body: JSON.stringify({
          sid: 'session-2',
          token: 'token-2',
        }),
      })
    );
  });
});

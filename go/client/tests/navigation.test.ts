import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { HydrationManager } from '../src/hydration';
import { LiveRuntime } from '../src/runtime';
import type { FrameMessage } from '../src/types';

describe('Navigation (frame.nav)', () => {
  let manager: HydrationManager;
  let runtime: LiveRuntime;
  let pushStateSpy: ReturnType<typeof vi.spyOn>;
  let replaceStateSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    document.body.innerHTML = '';
    runtime = new LiveRuntime({ autoConnect: false });
    manager = new HydrationManager(runtime);
    pushStateSpy = vi.spyOn(history, 'pushState');
    replaceStateSpy = vi.spyOn(history, 'replaceState');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const createFrameWithNav = (nav: { push?: string; replace?: string; back?: boolean }): FrameMessage => ({
    t: 'frame',
    sid: 'test',
    ver: 1,
    seq: 1,
    nav,
  });

  describe('pushState navigation', () => {
    it('should push new URL to history', () => {
      const frame = createFrameWithNav({ push: '/new-page' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/new-page');
      expect(replaceStateSpy).not.toHaveBeenCalled();
    });

    it('should push URL with query parameters', () => {
      const frame = createFrameWithNav({ push: '/search?q=test&page=2' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/search?q=test&page=2');
    });

    it('should push URL with hash', () => {
      const frame = createFrameWithNav({ push: '/page#section' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/page#section');
    });

    it('should push absolute URL', () => {
      const frame = createFrameWithNav({ push: 'https://example.com/page' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', 'https://example.com/page');
    });

    it('should push relative URL', () => {
      const frame = createFrameWithNav({ push: '../parent/page' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '../parent/page');
    });
  });

  describe('replaceState navigation', () => {
    it('should replace current URL in history', () => {
      const frame = createFrameWithNav({ replace: '/updated-page' });
      (manager as any).applyFrame(frame);

      expect(replaceStateSpy).toHaveBeenCalledWith(null, '', '/updated-page');
      expect(pushStateSpy).not.toHaveBeenCalled();
    });

    it('should replace URL with query parameters', () => {
      const frame = createFrameWithNav({ replace: '/items?sort=name&order=asc' });
      (manager as any).applyFrame(frame);

      expect(replaceStateSpy).toHaveBeenCalledWith(null, '', '/items?sort=name&order=asc');
    });

    it('should replace URL with hash', () => {
      const frame = createFrameWithNav({ replace: '/doc#intro' });
      (manager as any).applyFrame(frame);

      expect(replaceStateSpy).toHaveBeenCalledWith(null, '', '/doc#intro');
    });

    it('should replace with absolute URL', () => {
      const frame = createFrameWithNav({ replace: 'https://example.com/other' });
      (manager as any).applyFrame(frame);

      expect(replaceStateSpy).toHaveBeenCalledWith(null, '', 'https://example.com/other');
    });
  });

  describe('priority and validation', () => {
    it('should prefer replace over push when both provided', () => {
      const frame = createFrameWithNav({ push: '/push-url', replace: '/replace-url' });
      (manager as any).applyFrame(frame);

      expect(replaceStateSpy).toHaveBeenCalledWith(null, '', '/replace-url');
      expect(pushStateSpy).not.toHaveBeenCalled();
    });

    it('should skip navigation when nav is empty object', () => {
      const frame = createFrameWithNav({});
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).not.toHaveBeenCalled();
      expect(replaceStateSpy).not.toHaveBeenCalled();
    });

    it('should skip navigation when push is empty string', () => {
      const frame = createFrameWithNav({ push: '' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).not.toHaveBeenCalled();
      expect(replaceStateSpy).not.toHaveBeenCalled();
    });

    it('should skip navigation when replace is empty string', () => {
      const frame = createFrameWithNav({ replace: '' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).not.toHaveBeenCalled();
      expect(replaceStateSpy).not.toHaveBeenCalled();
    });

    it('should skip navigation when nav is undefined', () => {
      const frame: FrameMessage = {
        t: 'frame',
        sid: 'test',
        ver: 1,
        seq: 1,
      };
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).not.toHaveBeenCalled();
      expect(replaceStateSpy).not.toHaveBeenCalled();
    });
  });

  describe('error handling', () => {
    it('should not throw when history.pushState fails', () => {
      pushStateSpy.mockImplementation(() => {
        throw new Error('Security error');
      });

      const frame = createFrameWithNav({ push: '/fail' });

      expect(() => (manager as any).applyFrame(frame)).not.toThrow();
    });

    it('should not throw when history.replaceState fails', () => {
      replaceStateSpy.mockImplementation(() => {
        throw new Error('Security error');
      });

      const frame = createFrameWithNav({ replace: '/fail' });

      expect(() => (manager as any).applyFrame(frame)).not.toThrow();
    });
  });

  describe('combined with other frame operations', () => {
    it('should apply navigation after effects', () => {
      const frame: FrameMessage = {
        t: 'frame',
        sid: 'test',
        ver: 1,
        seq: 1,
        effects: [
          {
            type: 'metadata',
            title: 'New Page',
          },
        ],
        nav: { push: '/new-page' },
      };

      (manager as any).applyFrame(frame);

      expect(document.title).toBe('New Page');
      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/new-page');
    });

    it('should apply navigation with patch operations', () => {
      const frame: FrameMessage = {
        t: 'frame',
        sid: 'test',
        ver: 1,
        seq: 1,
        patch: [],
        nav: { replace: '/updated' },
      };

      (manager as any).applyFrame(frame);

      expect(replaceStateSpy).toHaveBeenCalledWith(null, '', '/updated');
    });
  });

  describe('URL encoding', () => {
    it('should handle URLs with special characters', () => {
      const frame = createFrameWithNav({ push: '/search?q=hello world&lang=en' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/search?q=hello world&lang=en');
    });

    it('should handle URLs with encoded characters', () => {
      const frame = createFrameWithNav({ push: '/search?q=hello%20world' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/search?q=hello%20world');
    });

    it('should handle URLs with international characters', () => {
      const frame = createFrameWithNav({ push: '/ページ' });
      (manager as any).applyFrame(frame);

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/ページ');
    });
  });

  describe('back navigation', () => {
    let backSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
      backSpy = vi.spyOn(history, 'back');
    });

    it('should call history.back()', () => {
      const frame = createFrameWithNav({ back: true });
      (manager as any).applyFrame(frame);

      expect(backSpy).toHaveBeenCalled();
      expect(pushStateSpy).not.toHaveBeenCalled();
      expect(replaceStateSpy).not.toHaveBeenCalled();
    });

    it('should prioritize back over push and replace', () => {
      const frame = createFrameWithNav({ push: '/page1', replace: '/page2', back: true });
      (manager as any).applyFrame(frame);

      expect(backSpy).toHaveBeenCalled();
      expect(pushStateSpy).not.toHaveBeenCalled();
      expect(replaceStateSpy).not.toHaveBeenCalled();
    });

    it('should not throw when history.back fails', () => {
      backSpy.mockImplementation(() => {
        throw new Error('Cannot go back');
      });

      const frame = createFrameWithNav({ back: true });

      expect(() => (manager as any).applyFrame(frame)).not.toThrow();
    });

    it('should handle back in sequence with other navigations', () => {
      const frame1 = createFrameWithNav({ push: '/page1' });
      const frame2 = createFrameWithNav({ push: '/page2' });
      const frame3 = createFrameWithNav({ back: true });
      const frame4 = createFrameWithNav({ push: '/page3' });

      (manager as any).applyFrame(frame1);
      (manager as any).applyFrame(frame2);
      (manager as any).applyFrame(frame3);
      (manager as any).applyFrame(frame4);

      expect(pushStateSpy).toHaveBeenCalledTimes(3);
      expect(backSpy).toHaveBeenCalledTimes(1);
    });
  });

  describe('multiple navigation in sequence', () => {
    it('should apply multiple push navigations', () => {
      const frame1 = createFrameWithNav({ push: '/page1' });
      const frame2 = createFrameWithNav({ push: '/page2' });
      const frame3 = createFrameWithNav({ push: '/page3' });

      (manager as any).applyFrame(frame1);
      (manager as any).applyFrame(frame2);
      (manager as any).applyFrame(frame3);

      expect(pushStateSpy).toHaveBeenCalledTimes(3);
      expect(pushStateSpy).toHaveBeenNthCalledWith(1, null, '', '/page1');
      expect(pushStateSpy).toHaveBeenNthCalledWith(2, null, '', '/page2');
      expect(pushStateSpy).toHaveBeenNthCalledWith(3, null, '', '/page3');
    });

    it('should apply mixed push and replace navigations', () => {
      const frame1 = createFrameWithNav({ push: '/page1' });
      const frame2 = createFrameWithNav({ replace: '/page1-updated' });
      const frame3 = createFrameWithNav({ push: '/page2' });

      (manager as any).applyFrame(frame1);
      (manager as any).applyFrame(frame2);
      (manager as any).applyFrame(frame3);

      expect(pushStateSpy).toHaveBeenCalledTimes(2);
      expect(replaceStateSpy).toHaveBeenCalledTimes(1);
      expect(pushStateSpy).toHaveBeenNthCalledWith(1, null, '', '/page1');
      expect(replaceStateSpy).toHaveBeenCalledWith(null, '', '/page1-updated');
      expect(pushStateSpy).toHaveBeenNthCalledWith(2, null, '', '/page2');
    });
  });
});

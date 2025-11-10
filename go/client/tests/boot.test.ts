import { describe, it, expect, beforeEach, vi } from 'vitest';
import { BootHandler } from '../src/boot';
import * as dom from '../src/dom-index';
import type { BootPayload } from '../src/types';

vi.mock('../src/dom-index');
vi.mock('../src/events');

describe('BootHandler', () => {
  let bootHandler: BootHandler;

  beforeEach(() => {
    bootHandler = new BootHandler({ debug: false });
    vi.clearAllMocks();

    // Reset DOM
    document.body.innerHTML = '';
    delete (window as any).__LIVEUI_BOOT__;
  });

  describe('load', () => {
    it('should load explicit boot payload', () => {
      const payload: BootPayload = {
        t: 'boot',
        sid: 'test-session',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
      };

      const result = bootHandler.load(payload);
      expect(result).toEqual(payload);
      expect(bootHandler.get()).toEqual(payload);
    });

    it('should detect boot from window global', () => {
      const payload: BootPayload = {
        t: 'boot',
        sid: 'global-session',
        ver: 2,
        seq: 5,
        s: [],
        d: [],
        slots: [],
        handlers: {},
        location: { path: '/test', q: '', hash: '' },
      };

      (window as any).__LIVEUI_BOOT__ = payload;

      const result = bootHandler.load();
      expect(result).toEqual(payload);
    });

    it('should detect boot from DOM script tag', () => {
      const payload: BootPayload = {
        t: 'boot',
        sid: 'dom-session',
        ver: 3,
        seq: 10,
        s: [],
        d: [],
        slots: [],
        handlers: {},
        location: { path: '/page', q: 'foo=bar', hash: '' },
        client: { endpoint: '/ws' },
      };

      const script = document.createElement('script');
      script.id = 'live-boot';
      script.type = 'application/json'; // Prevent JSDOM from executing as JS
      script.textContent = JSON.stringify(payload);
      document.body.appendChild(script);

      const result = bootHandler.load();
      expect(result).toEqual(payload);
    });

    it('should expose client configuration when present', () => {
      const payload: BootPayload = {
        t: 'boot',
        sid: 'client-session',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
        client: { endpoint: '/socket' },
      };

      bootHandler.load(payload);
      expect(bootHandler.get()?.client?.endpoint).toBe('/socket');
    });

    it('should return null for invalid payload', () => {
      const result = bootHandler.load({ sid: '' } as any);
      expect(result).toBeNull();
    });

    it('should return null when no payload found', () => {
      const result = bootHandler.load();
      expect(result).toBeNull();
    });
  });

  describe('ensure', () => {
    it('should return boot payload if loaded', () => {
      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
      };

      bootHandler.load(payload);
      expect(bootHandler.ensure()).toEqual(payload);
    });

    it('should throw if no boot payload', () => {
      expect(() => bootHandler.ensure()).toThrow('boot payload is required');
    });
  });

  describe('getJoinLocation', () => {
    it('should return current browser location', () => {
      Object.defineProperty(window, 'location', {
        value: {
          pathname: '/test',
          search: '?foo=bar',
          hash: '#section',
        },
        writable: true,
      });

      const location = bootHandler.getJoinLocation();
      expect(location).toEqual({
        path: '/test',
        q: 'foo=bar',
        hash: 'section',
      });
    });

    it('should use boot fallback when no browser location', () => {
      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [],
        handlers: {},
        location: { path: '/fallback', q: 'a=b', hash: 'h' },
      };

      bootHandler.load(payload);

      // Mock no browser location
      Object.defineProperty(window, 'location', {
        value: {
          pathname: '',
          search: '',
          hash: '',
        },
        writable: true,
      });

      const location = bootHandler.getJoinLocation();
      expect(location.path).toBe('/fallback');
    });
  });

  describe('DOM registration', () => {
    it('should register slots from boot payload', () => {
      const registerSlotMock = vi.spyOn(dom, 'registerSlot');

      document.body.innerHTML = '<div id="root"><span id="slot1"></span><div id="slot2"></div></div>';
      const slot1 = document.getElementById('slot1');
      const slot2 = document.getElementById('slot2');

      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [
          { anchorId: 1, path: [0, 0] },
          { anchorId: 2, path: [0, 1] },
        ],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
      };

      bootHandler.load(payload);

      expect(registerSlotMock).toHaveBeenCalledTimes(2);
      expect(registerSlotMock).toHaveBeenCalledWith(1, slot1);
      expect(registerSlotMock).toHaveBeenCalledWith(2, slot2);
    });

    it('should register list containers from slot metadata', () => {
      const registerListMock = vi.spyOn(dom, 'registerList');

      document.body.innerHTML = '<div id="root"><ul id="list"></ul></div>';
      const listEl = document.getElementById('list') as Element;

      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [
          {
            anchorId: 1,
            path: [0, 0],
            list: { path: [0, 0] },
          },
        ],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
      };

      bootHandler.load(payload);

      expect(registerListMock).toHaveBeenCalledWith(1, listEl);
    });
  });

  describe('location sync', () => {
    it('should sync browser location with boot location', () => {
      const replaceStateSpy = vi.spyOn(window.history, 'replaceState');

      Object.defineProperty(window, 'location', {
        value: {
          pathname: '/wrong',
          search: '',
          hash: '',
        },
        writable: true,
      });

      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [],
        handlers: {},
        location: { path: '/correct', q: 'foo=bar', hash: 'section' },
      };

      bootHandler.load(payload);

      expect(replaceStateSpy).toHaveBeenCalledWith({}, '', '/correct?foo=bar#section');
    });

    it('should not sync if locations match', () => {
      const replaceStateSpy = vi.spyOn(window.history, 'replaceState');

      Object.defineProperty(window, 'location', {
        value: {
          pathname: '/same',
          search: '?foo=bar',
          hash: '#section',
        },
        writable: true,
      });

      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [],
        handlers: {},
        location: { path: '/same', q: 'foo=bar', hash: 'section' },
      };

      bootHandler.load(payload);

      expect(replaceStateSpy).not.toHaveBeenCalled();
    });
  });
});

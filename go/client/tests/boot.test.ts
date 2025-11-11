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
    it('should register slots from manifest data', () => {
      const registerSlotMock = vi.spyOn(dom, 'registerSlot');

      document.body.replaceChildren();
      const host = document.createElement('div');
      const textNode = document.createTextNode('value');
      host.appendChild(textNode);
      document.body.append(host);

      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [
          { anchorId: 1 },
          { anchorId: 2 },
        ],
        slotPaths: [
          { slot: 1, componentId: 'comp', elementPath: [], textChildIndex: 0 },
          { slot: 2, componentId: 'comp', elementPath: [] },
        ],
        componentPaths: [
          { componentId: 'comp', firstChild: [0], lastChild: [0] },
        ],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
      };

      bootHandler.load(payload);

      expect(registerSlotMock).toHaveBeenCalledWith(1, textNode);
      expect(registerSlotMock).toHaveBeenCalledWith(2, host);
    });

    it('should register list containers from manifest data', () => {
      const registerListMock = vi.spyOn(dom, 'registerList');

      document.body.replaceChildren();
      const list = document.createElement('ul');
      document.body.append(list);

      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [],
        listPaths: [
          { slot: 4, componentId: 'comp', elementPath: [] },
        ],
        componentPaths: [
          { componentId: 'comp', firstChild: [0], lastChild: [0] },
        ],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
      };

      bootHandler.load(payload);

      expect(registerListMock).toHaveBeenCalledWith(4, list);
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

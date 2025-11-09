import { describe, it, expect, beforeEach, vi } from 'vitest';
import { BootHandler } from '../src/boot';
import * as dom from '../src/dom-index';
import type { BootPayload } from '../src/types';

function computeAnchorDescriptor(node: Node): { parentPath: number[]; childIndex: number } {
  const path: number[] = [];
  let current: Node | null = node;
  while (current && current.parentNode) {
    const parent = current.parentNode;
    const index = Array.from(parent.childNodes).indexOf(current);
    if (index < 0) {
      break;
    }
    path.unshift(index);
    current = parent;
  }
  const parentPath = path.slice(0, -1);
  const childIndex = path[path.length - 1] ?? 0;
  return { parentPath, childIndex };
}

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

      const slot1 = document.createElement('div');
      document.body.appendChild(slot1);

      const slot2 = document.createElement('div');
      document.body.appendChild(slot2);

      const anchor1 = computeAnchorDescriptor(slot1);
      const anchor2 = computeAnchorDescriptor(slot2);

      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [
          { anchorId: 1, parentPath: anchor1.parentPath, childIndex: anchor1.childIndex },
          { anchorId: 2, parentPath: anchor2.parentPath, childIndex: anchor2.childIndex },
        ],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
      };

      bootHandler.load(payload);

      expect(registerSlotMock).toHaveBeenCalledTimes(2);
      expect(registerSlotMock).toHaveBeenCalledWith(1, slot1);
      expect(registerSlotMock).toHaveBeenCalledWith(2, slot2);
    });

    it('should use anchor descriptors when slot attributes are absent', () => {
      const registerSlotMock = vi.spyOn(dom, 'registerSlot');

      const container = document.createElement('section');
      const textNode = document.createTextNode('hello');
      container.appendChild(textNode);
      document.body.appendChild(container);

      const { parentPath, childIndex } = computeAnchorDescriptor(textNode);

      const payload: BootPayload = {
        t: 'boot',
        sid: 'meta',
        ver: 1,
        seq: 0,
        s: [],
        d: [],
        slots: [
          {
            anchorId: 5,
            parentPath,
            childIndex,
          },
        ],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
      };

      bootHandler.load(payload);

      expect(registerSlotMock).toHaveBeenCalledWith(5, textNode);
    });

    it('should initialize list slots', () => {
      const registerListMock = vi.spyOn(dom, 'registerList');

      const listA = document.createElement('ul');
      const listB = document.createElement('ul');
      document.body.appendChild(listA);
      document.body.appendChild(listB);

      const anchorA = computeAnchorDescriptor(listA);
      const anchorB = computeAnchorDescriptor(listB);

      const payload: BootPayload = {
        t: 'boot',
        sid: 'test',
        ver: 1,
        seq: 0,
        s: [],
        d: [
          { kind: 'text' },
          { kind: 'list', list: [] },
          { kind: 'attrs' },
          { kind: 'list', list: [] },
        ],
        slots: [
          { anchorId: 1, parentPath: anchorA.parentPath, childIndex: anchorA.childIndex },
          { anchorId: 3, parentPath: anchorB.parentPath, childIndex: anchorB.childIndex },
        ],
        handlers: {},
        location: { path: '/', q: '', hash: '' },
      };

      bootHandler.load(payload);

      expect(registerListMock).toHaveBeenCalledWith(1, listA, expect.any(Map));
      expect(registerListMock).toHaveBeenCalledWith(3, listB, expect.any(Map));
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

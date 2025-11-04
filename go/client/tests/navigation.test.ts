import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupEventDelegation, registerNavigationHandler, unregisterNavigationHandler } from '../src/events';

describe('Navigation Interception', () => {
  let navigationCallback: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    document.body.innerHTML = '';
    navigationCallback = vi.fn(() => true);
    vi.clearAllMocks();

    // Mock history
    window.history.pushState = vi.fn();
    window.history.replaceState = vi.fn();

    // Setup event delegation for navigation to work
    setupEventDelegation(() => {});
  });

  describe('Navigation Handler Registration', () => {
    it('should register navigation handler', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/test';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).toHaveBeenCalledWith('/test', '', '');
      expect(event.defaultPrevented).toBe(true);
    });

    it('should unregister navigation handler', () => {
      registerNavigationHandler(navigationCallback);
      unregisterNavigationHandler();

      const anchor = document.createElement('a');
      anchor.href = '/test';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
    });
  });

  describe('Same-Origin Checks', () => {
    it('should intercept same-origin links', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = window.location.origin + '/page';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).toHaveBeenCalled();
      expect(event.defaultPrevented).toBe(true);
    });

    it('should not intercept different-origin links', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = 'https://external.com/page';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
      expect(event.defaultPrevented).toBe(false);
    });
  });

  describe('Mouse Button Checks', () => {
    it('should only intercept primary (left) mouse button', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/test';
      document.body.appendChild(anchor);

      // Middle click
      const middleClick = new MouseEvent('click', { bubbles: true, cancelable: true, button: 1 });
      anchor.dispatchEvent(middleClick);
      expect(navigationCallback).not.toHaveBeenCalled();

      // Right click
      const rightClick = new MouseEvent('click', { bubbles: true, cancelable: true, button: 2 });
      anchor.dispatchEvent(rightClick);
      expect(navigationCallback).not.toHaveBeenCalled();

      // Left click
      const leftClick = new MouseEvent('click', { bubbles: true, cancelable: true, button: 0 });
      anchor.dispatchEvent(leftClick);
      expect(navigationCallback).toHaveBeenCalled();
    });
  });

  describe('Modifier Key Checks', () => {
    it('should not intercept with ctrl key', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/test';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true, ctrlKey: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
    });

    it('should not intercept with meta key', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/test';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true, metaKey: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
    });

    it('should not intercept with shift key', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/test';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true, shiftKey: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
    });

    it('should not intercept with alt key', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/test';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true, altKey: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
    });
  });

  describe('Target Attribute Checks', () => {
    it('should not intercept links with target="_blank"', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/test';
      anchor.target = '_blank';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
    });

    it('should intercept links with target="_self"', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/test';
      anchor.target = '_self';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).toHaveBeenCalled();
    });

    it('should intercept links with no target attribute', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/test';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).toHaveBeenCalled();
    });
  });

  describe('Hash Navigation', () => {
    it('should parse hash from URL', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = '/page#section';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).toHaveBeenCalledWith('/page', '', 'section');
    });

    it('should not intercept hash-only navigation on same page', () => {
      registerNavigationHandler(navigationCallback);

      Object.defineProperty(window, 'location', {
        value: {
          pathname: '/page',
          search: '',
          hash: '',
          origin: window.location.origin,
        },
        writable: true,
      });

      const anchor = document.createElement('a');
      anchor.href = '/page#section';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
    });
  });

  describe('Query String Handling', () => {
    it('should parse query string from URL', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = window.location.origin + '/page?foo=bar&baz=qux';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).toHaveBeenCalledWith('/page', 'foo=bar&baz=qux', '');
    });

    it('should handle both query and hash', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = window.location.origin + '/page?foo=bar#section';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).toHaveBeenCalledWith('/page', 'foo=bar', 'section');
    });
  });

  describe('Invalid URLs', () => {
    it('should not intercept invalid URLs', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = 'javascript:void(0)';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
    });

    it('should not intercept mailto links', () => {
      registerNavigationHandler(navigationCallback);

      const anchor = document.createElement('a');
      anchor.href = 'mailto:test@example.com';
      document.body.appendChild(anchor);

      const event = new MouseEvent('click', { bubbles: true, cancelable: true });
      anchor.dispatchEvent(event);

      expect(navigationCallback).not.toHaveBeenCalled();
    });
  });
});

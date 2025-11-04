import { vi } from 'vitest';

if (typeof window !== 'undefined' && !('scrollTo' in window)) {
  // jsdom doesn't implement scrollTo by default
  // eslint-disable-next-line @typescript-eslint/no-empty-function
  (window as any).scrollTo = () => {};
}

if (typeof window !== 'undefined' && !('history' in window)) {
  (window as any).history = {
    pushState: vi.fn(),
    replaceState: vi.fn(),
  };
}

if (typeof window !== 'undefined') {
  window.history.pushState = window.history.pushState || vi.fn();
  window.history.replaceState = window.history.replaceState || vi.fn();
}

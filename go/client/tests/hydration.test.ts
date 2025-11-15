import { describe, it, expect, beforeEach } from 'vitest';
import { HydrationManager } from '../src/hydration';
import type { TemplatePayload } from '../src/types';

interface HandlerMap {
  init?: (payload: TemplatePayload) => void;
  template?: (payload: TemplatePayload) => void;
  frame?: (payload: any) => void;
}

describe('HydrationManager', () => {
  let handlers: HandlerMap;
  beforeEach(() => {
    handlers = {};
    document.body.innerHTML = '<main>original</main>';
  });

  it('replaces body HTML when init template arrives', () => {
    const runtime = {
      on(event: keyof HandlerMap, cb: any) {
        handlers[event] = cb;
        return () => {
          delete handlers[event];
        };
      },
      getBootPayload() {
        return null;
      },
    } as any;

    new HydrationManager(runtime);

    const payload: TemplatePayload = {
      html: '<div id="boot">hello</div>',
      s: [],
      d: [],
      slots: [],
    };

    handlers.init?.(payload);
    expect(document.body.innerHTML).toContain('hello');
  });
});

import { describe, expect, it, vi, beforeEach } from 'vitest';
import LiveUI from '../src/index';
import type { BootPayload, ResumeMessage } from '../src/types';

describe('LiveUI resume handling', () => {
  let boot: BootPayload;

  beforeEach(() => {
    document.body.innerHTML = '<div></div>';
    boot = {
      t: 'boot',
      sid: 'test-sid',
      ver: 1,
      seq: 1,
      html: '<div></div>',
      s: [],
      d: [],
      slots: [{ anchorId: 0 }],
      slotPaths: [
        { slot: 0, componentId: 'root', elementPath: [], textChildIndex: -1 },
      ],
      componentPaths: [
        { componentId: 'root', firstChild: [0], lastChild: [0] },
      ],
      handlers: {},
      location: { path: '/', q: '', hash: '' },
    };
  });

  it('emits resumed event when resume message arrives', () => {
    const client = new LiveUI({ autoConnect: false, boot });
    const handler = vi.fn();
    client.on('resumed', handler);

    const resume: ResumeMessage = { t: 'resume', sid: 'test-sid', from: 2, to: 3 };
    (client as any).handleMessage(resume);

    expect(handler).toHaveBeenCalledWith({ from: 2, to: 3 });
  });
});

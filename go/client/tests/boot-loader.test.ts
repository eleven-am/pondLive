import { describe, it, expect, beforeEach } from 'vitest';
import { BootLoader } from '../src/boot';
import type { BootPayload } from '../src/types';

const basePayload: BootPayload = {
  t: 'boot',
  sid: 'test-sid',
  ver: 1,
  seq: 1,
  location: { path: '/', q: '', hash: '' },
  s: [],
  d: [],
  slots: [],
};

describe('BootLoader', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
    document.head.innerHTML = '';
    delete (window as any).__LIVEUI_BOOT__;
  });

  it('returns explicit payload', () => {
    const loader = new BootLoader();
    const payload = loader.load(basePayload);
    expect(payload).toEqual(basePayload);
    expect(loader.ensure()).toEqual(basePayload);
  });

  it('reads payload from script tag', () => {
    const script = document.createElement('script');
    script.id = 'live-boot';
    script.type = 'application/json';
    script.textContent = JSON.stringify(basePayload);
    document.body.appendChild(script);

    const loader = new BootLoader();
    const payload = loader.load();
    expect(payload).toEqual(basePayload);
  });
});

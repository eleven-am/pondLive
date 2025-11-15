import type {BootPayload} from './types';
import {Logger} from './logger';

export type BootSource = BootPayload | null | undefined;

export interface BootLoaderOptions {
  debug?: boolean;
  scriptId?: string;
}

export class BootLoader {
  private payload: BootPayload | null = null;
  private readonly options: Required<BootLoaderOptions>;

  constructor(options?: BootLoaderOptions) {
    this.options = {
      debug: options?.debug ?? false,
      scriptId: options?.scriptId ?? 'live-boot',
    };
  }

  load(explicit?: BootSource): BootPayload | null {
    const candidate = explicit ?? this.readWindowPayload() ?? this.readScriptPayload();
    if (candidate && typeof candidate.sid === 'string') {
      this.payload = candidate;
      this.cacheToWindow(candidate);
      if (this.options.debug) {
        Logger.debug('[boot]', 'payload loaded', {
          sid: candidate.sid,
          version: candidate.ver,
          hasHtml: Boolean(candidate.html),
        });
      }
      return this.payload;
    }
    if (this.options.debug) {
      this.log('boot payload unavailable');
    }
    return this.payload;
  }

  get(): BootPayload | null {
    return this.payload;
  }

  ensure(): BootPayload {
    const boot = this.payload ?? this.load();
    if (!boot || typeof boot.sid !== 'string') {
      throw new Error('[LiveUI] boot payload is required before connecting');
    }
    return boot;
  }

  private readWindowPayload(): BootPayload | null {
    if (typeof window === 'undefined') {
      return null;
    }
    const globalAny = window as typeof window & { __LIVEUI_BOOT__?: BootPayload };
    const payload = globalAny.__LIVEUI_BOOT__;
    if (payload && typeof payload.sid === 'string') {
      return payload;
    }
    return null;
  }

  private readScriptPayload(): BootPayload | null {
    if (typeof document === 'undefined') {
      return null;
    }
    const script = document.getElementById(this.options.scriptId);
    const content = script?.textContent;
    if (!content) {
      return null;
    }
    try {
        return JSON.parse(content) as BootPayload;
    } catch (error) {
      this.log('failed to parse boot payload', error);
      return null;
    }
  }

  private cacheToWindow(payload: BootPayload): void {
    if (typeof window === 'undefined') {
      return;
    }
    const globalAny = window as typeof window & { __LIVEUI_BOOT__?: BootPayload };
    globalAny.__LIVEUI_BOOT__ = payload;
  }

  private log(message: string, error?: unknown): void {
    if (!this.options.debug) {
      return;
    }
    if (error) {
      Logger.warn('[boot]', message, error);
    } else {
      Logger.warn('[boot]', message);
    }
  }
}

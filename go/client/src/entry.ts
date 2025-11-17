import LiveUI from './index';
import type { LiveUIOptions, BootPayload } from './types';
import { Logger } from './logger';

type LiveUIInstance = InstanceType<typeof LiveUI>;

type DevtoolsHook = {
  installed?: boolean;
  instance?: LiveUIInstance;
};

type LiveUIWindow = Window & {
  LiveUI?: typeof LiveUI & { boot?: typeof bootClient; instance?: LiveUIInstance };
  LiveUIInstance?: LiveUIInstance;
  __LIVEUI_BOOT__?: BootPayload;
  __LIVEUI_OPTIONS__?: LiveUIOptions;
  __LIVEUI_DEVTOOLS__?: DevtoolsHook;
};

let bootPromise: Promise<LiveUIInstance> | null = null;

function getWindow(): LiveUIWindow | null {
  if (typeof window === 'undefined') {
    return null;
  }
  return window as LiveUIWindow;
}

function detectBootPayload(target: LiveUIWindow): BootPayload | null {
  if (target.__LIVEUI_BOOT__ && typeof target.__LIVEUI_BOOT__ === 'object') {
    Logger.debug('[Entry]', 'Boot payload from window', {
      sid: target.__LIVEUI_BOOT__.sid,
      hasListPaths: Array.isArray(target.__LIVEUI_BOOT__.listPaths),
      listPathsLength: Array.isArray(target.__LIVEUI_BOOT__.listPaths) ? target.__LIVEUI_BOOT__.listPaths.length : 0,
    });
    return target.__LIVEUI_BOOT__!;
  }
  if (typeof document === 'undefined') {
    return null;
  }
  const script = document.getElementById('live-boot');
  const content = script?.textContent?.trim();
  if (!content) {
    Logger.debug('[Entry]', 'No boot script content found');
    return null;
  }
  Logger.debug('[Entry]', 'Boot script found', { contentLength: content.length });
  try {
    const payload = JSON.parse(content) as BootPayload;
    target.__LIVEUI_BOOT__ = payload;
    Logger.debug('[Entry]', 'Boot payload parsed successfully', {
      sid: payload.sid,
      hasListPaths: Array.isArray(payload.listPaths),
      listPathsLength: Array.isArray(payload.listPaths) ? payload.listPaths.length : 0,
      listPaths: payload.listPaths,
      hasComponentPaths: Array.isArray(payload.componentPaths),
      componentPathsLength: Array.isArray(payload.componentPaths) ? payload.componentPaths.length : 0,
    });
    return payload;
  } catch (error) {
    Logger.error('Failed to parse boot payload', error);
    return null;
  }
}

function attachGlobals(target: LiveUIWindow, instance: LiveUIInstance): void {
  const LiveUIExport = LiveUI as typeof LiveUI & { boot?: typeof bootClient; instance?: LiveUIInstance };
  LiveUIExport.boot = bootClient;
  LiveUIExport.instance = instance;
  target.LiveUI = LiveUIExport;
  target.LiveUIInstance = instance;
  if (target.__LIVEUI_DEVTOOLS__) {
    target.__LIVEUI_DEVTOOLS__.installed = true;
    target.__LIVEUI_DEVTOOLS__.instance = instance;
  }
}

function createClient(target: LiveUIWindow): LiveUIInstance {
  const inlineOptions = { ...(target.__LIVEUI_OPTIONS__ ?? {}) } as LiveUIOptions;
  const bootPayload = detectBootPayload(target);
  const resolvedBoot = inlineOptions.boot ?? bootPayload ?? null;
  if (resolvedBoot) {
    inlineOptions.boot = resolvedBoot;
    target.__LIVEUI_BOOT__ = resolvedBoot;
    if (typeof resolvedBoot.client?.debug === 'boolean') {
      inlineOptions.debug = resolvedBoot.client.debug;
    }
  }
  if (typeof inlineOptions.debug === 'undefined') {
    inlineOptions.debug = false;
  }
  target.__LIVEUI_OPTIONS__ = inlineOptions;

  const autoConnect = inlineOptions.autoConnect !== false;
  inlineOptions.autoConnect = false;

  const client = new LiveUI(inlineOptions);
  attachGlobals(target, client);
  Logger.debug('[Entry]', 'LiveUI client created', {
    autoConnect,
    debug: inlineOptions.debug,
    hasBoot: Boolean(resolvedBoot),
  });

  if (autoConnect && resolvedBoot) {
    void client.connect().catch((error) => {
      Logger.error('Failed to connect after boot', error);
    });
  }

  return client;
}

function scheduleBoot(target: LiveUIWindow): Promise<LiveUIInstance> {
  return new Promise((resolve, reject) => {
    const start = () => {
      try {
        const instance = createClient(target);
        resolve(instance);
      } catch (error) {
        reject(error);
      }
    };
    if (typeof document !== 'undefined' && document.readyState === 'loading') {
      const handler = () => {
        document.removeEventListener('DOMContentLoaded', handler);
        start();
      };
      document.addEventListener('DOMContentLoaded', handler);
    } else {
      start();
    }
  });
}

export function bootClient(options: { force?: boolean } = {}): Promise<LiveUIInstance> {
  const target = getWindow();
  if (!target) {
    return Promise.reject(new Error('[LiveUI] window is not available in this environment'));
  }
  if (options.force) {
    bootPromise = null;
  }
  if (!bootPromise) {
    bootPromise = scheduleBoot(target);
  }
  return bootPromise;
}

if (typeof window !== 'undefined') {
  void bootClient().catch((error) => {
    Logger.error('Boot failed', error);
  });
}

export * from './index';

import LiveUI, {
  applyOps,
  clearPatcherCaches,
  configurePatcher,
  dom,
  getPatcherConfig,
  getPatcherStats,
  morphElement,
} from './index';
import type { BootPayload, LiveUIOptions } from './types';

type LiveUIInstance = InstanceType<typeof LiveUI>;

type LiveUIPatcherExports = {
  configure: typeof configurePatcher;
  getConfig: typeof getPatcherConfig;
  clearCaches: typeof clearPatcherCaches;
  getStats: typeof getPatcherStats;
  morphElement: typeof morphElement;
};

type LiveUIAugmented = typeof LiveUI & {
  instance?: LiveUIInstance;
  dom?: typeof dom;
  applyOps?: typeof applyOps;
  patcher?: LiveUIPatcherExports;
  boot?: typeof bootClient;
};

type DevtoolsHook = {
  installed?: boolean;
  instance?: LiveUIInstance;
};

type LiveUIWindow = Window & {
  LiveUI?: LiveUIAugmented;
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
  const existing = target.__LIVEUI_BOOT__;
  if (existing && typeof existing === 'object' && typeof existing.sid === 'string') {
    return existing;
  }

  if (typeof document === 'undefined') {
    return null;
  }

  const script = document.getElementById('live-boot');
  const content = script?.textContent;
  if (!content) {
    return null;
  }

  try {
    const payload = JSON.parse(content) as BootPayload;
    target.__LIVEUI_BOOT__ = payload;
    return payload;
  } catch (error) {
    console.error('[LiveUI] Failed to parse boot payload', error);
    return null;
  }
}

function attachGlobals(target: LiveUIWindow, instance: LiveUIInstance): void {
  const augmented = (LiveUI as LiveUIAugmented);

  Object.assign(augmented, {
    instance,
    dom,
    applyOps,
    patcher: {
      configure: configurePatcher,
      getConfig: getPatcherConfig,
      clearCaches: clearPatcherCaches,
      getStats: getPatcherStats,
      morphElement,
    } satisfies LiveUIPatcherExports,
    boot: bootClient,
  });

  target.LiveUI = augmented;
  target.LiveUIInstance = instance;

  if (target.__LIVEUI_DEVTOOLS__) {
    target.__LIVEUI_DEVTOOLS__.installed = true;
    target.__LIVEUI_DEVTOOLS__.instance = instance;
  }
}

function createClient(target: LiveUIWindow): LiveUIInstance {
  const inlineOptions = { ...(target.__LIVEUI_OPTIONS__ ?? {}) } as LiveUIOptions;
  const bootPayload = detectBootPayload(target);
  const inlineBoot = inlineOptions.boot;
  const resolvedBootPayload = bootPayload ?? inlineBoot ?? null;
  const shouldAutoConnect = inlineOptions.autoConnect !== false;

  const options: LiveUIOptions = { ...inlineOptions, autoConnect: false };
  if (resolvedBootPayload) {
    options.boot = resolvedBootPayload;
    if (!bootPayload) {
      target.__LIVEUI_BOOT__ = resolvedBootPayload;
    }
  }

  const client = new LiveUI(options);
  attachGlobals(target, client);

  if (shouldAutoConnect) {
    if (resolvedBootPayload) {
      void client.connect().catch((error) => {
        console.error('[LiveUI] Failed to connect during boot', error);
      });
    } else {
      console.warn('[LiveUI] Boot payload missing; auto-connect skipped.');
    }
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
        console.error('[LiveUI] Boot failed', error);
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

export function bootClient({ force = false }: { force?: boolean } = {}): Promise<LiveUIInstance> {
  const globalWindow = getWindow();
  if (!globalWindow) {
    return Promise.reject(new Error('LiveUI: window is not available for bootstrapping.'));
  }

  if (force) {
    bootPromise = null;
  }

  if (!bootPromise) {
    bootPromise = scheduleBoot(globalWindow);
  }

  return bootPromise;
}

if (typeof window !== 'undefined') {
  void bootClient().catch(() => {
    /* Error already logged in scheduleBoot */
  });
}

export * from './index';
export { default } from './index';

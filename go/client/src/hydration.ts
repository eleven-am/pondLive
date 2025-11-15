import type { TemplatePayload, FrameMessage, DOMRequestMessage, Effect, DOMActionEffect, CookieEffect } from './types';
import type { LiveRuntime } from './runtime';
import { DomRegistry } from './dom-registry';
import { RefRegistry } from './refs';
import { Patcher } from './patcher';
import { registerSlotTable } from './events';
import { applyRouterBindings } from './router-bindings';
import type { ComponentPathDescriptor } from './types';
import type { ComponentRange } from './manifest';
import { UploadManager } from './uploads';
import { MetadataManager } from './metadata';
import { Logger } from './logger';

export interface HydrationOptions {
  root?: ParentNode | null;
}

export class HydrationManager {
  private readonly runtime: LiveRuntime;
  private readonly options: HydrationOptions;
  private readonly dom = new DomRegistry();
  private readonly refs: RefRegistry;
  private readonly uploads: UploadManager;
  private readonly patcher: Patcher;
  private readonly metadata: MetadataManager;
  private componentRanges?: Map<string, ComponentRange>;

  constructor(runtime: LiveRuntime, options?: HydrationOptions) {
    this.runtime = runtime;
    this.options = options ?? {};
    this.uploads = new UploadManager(runtime);
    this.refs = new RefRegistry(runtime);
    this.metadata = new MetadataManager();
    this.patcher = new Patcher(this.dom, this.refs, this.uploads);
    this.runtime.on('init', (msg) => this.applyTemplate(msg));
    this.runtime.on('template', (msg) => this.applyTemplate(msg));
    this.runtime.on('frame', (msg) => this.applyFrame(msg));
    this.runtime.on('upload', (msg) => this.uploads.handleControl(msg));
    this.runtime.on('domreq', (msg) => this.handleDOMRequest(msg));
    const boot = this.runtime.getBootPayload();
    if (boot) {
      this.applyTemplate(boot);
    }
  }

  private applyTemplate(payload: TemplatePayload): void {
    if (typeof document === 'undefined') {
      return;
    }
    if (typeof payload.html !== 'string') {
      return;
    }
    const root = this.resolveRoot();
    if (!root) {
      return;
    }
    const componentPaths = payload.componentPaths as ComponentPathDescriptor[] | undefined;
    Logger.debug('[Hydration]', 'applying template', {
      slotPaths: payload.slotPaths?.length ?? 0,
      listPaths: payload.listPaths?.length ?? 0,
      componentPaths: componentPaths?.length ?? 0,
      htmlLength: payload.html?.length ?? 0,
    });
    if (root instanceof Document) {
      const container = root.body ?? root.documentElement ?? (root as unknown as HTMLElement);
      if (container) {
        container.innerHTML = payload.html;
        pruneWhitespace(container);
      }
      const overrides = this.dom.prime(componentPaths, { root });
      this.componentRanges = overrides;
      this.primeRegistries(payload, container ?? root, overrides);
      return;
    }
    if (root instanceof ShadowRoot) {
      root.innerHTML = payload.html;
      pruneWhitespace(root);
      const overrides = this.dom.prime(componentPaths, { root });
      this.componentRanges = overrides;
      this.primeRegistries(payload, root, overrides);
      return;
    }
    (root as Element).innerHTML = payload.html;
    pruneWhitespace(root as ParentNode);
    const overrides = this.dom.prime(componentPaths, { root });
    this.componentRanges = overrides;
    this.primeRegistries(payload, root as Element, overrides);
  }

  private applyFrame(frame: FrameMessage): void {
    Logger.debug('[Hydration]', 'applying frame', {
      seq: frame.seq,
      ver: frame.ver,
      patchOps: Array.isArray(frame.patch) ? frame.patch.length : 0,
      effects: frame.effects?.length ?? 0,
    });
    this.patcher.applyFrame(frame);
    if (frame.bindings?.slots) {
      registerSlotTable(frame.bindings.slots);
    }
    if (frame.bindings?.router) {
      applyRouterBindings(frame.bindings.router, this.componentRanges);
    }
    if (frame.bindings?.refs) {
      this.refs.registerBindings(frame.bindings.refs, this.componentRanges);
    }
    if (frame.bindings?.uploads) {
      this.uploads.registerBindings(frame.bindings.uploads, this.componentRanges);
    }
    if (frame.bindings?.slots === undefined && frame.bindings?.router) {
      // no-op; placeholder for incremental updates
    }
    this.refs.apply(frame.refs);
    if (frame.effects) {
      this.applyEffects(frame.effects);
    }
    if (frame.nav) {
      this.applyNavigation(frame.nav);
    }
    Logger.debug('[Hydration]', 'frame applied', {
      hasSlots: Boolean(frame.bindings?.slots),
      hasRouter: Boolean(frame.bindings?.router?.length),
      hasRefs: Boolean(frame.bindings?.refs?.length),
      hasUploads: Boolean(frame.bindings?.uploads?.length),
      refDeltaAdd: frame.refs?.add ? Object.keys(frame.refs.add).length : 0,
      refDeltaDel: frame.refs?.del?.length ?? 0,
      effectsApplied: frame.effects?.length ?? 0,
      navApplied: Boolean(frame.nav?.push || frame.nav?.replace),
    });
  }

  private applyEffects(effects: Effect[]): void {
    for (const effect of effects) {
      if (effect.type === 'metadata') {
        this.metadata.applyEffect(effect);
      } else if (effect.type === 'dom') {
        this.applyDOMAction(effect);
      } else if (effect.type === 'cookies') {
        this.applyCookieEffect(effect);
      }
    }
  }

  private applyDOMAction(effect: DOMActionEffect): void {
    if (!effect.ref || !effect.kind) {
      return;
    }
    const element = this.refs.get(effect.ref);
    if (!element) {
      Logger.debug('[Hydration]', 'DOM action skipped (element not found)', { ref: effect.ref, kind: effect.kind });
      return;
    }
    try {
      const kind = effect.kind;
      if (kind === 'dom.call' && effect.method) {
        if (typeof (element as any)[effect.method] === 'function') {
          const args = Array.isArray(effect.args) ? effect.args : [];
          (element as any)[effect.method](...args);
          Logger.debug('[Hydration]', 'DOM action call', { ref: effect.ref, method: effect.method });
        }
      } else if (kind === 'dom.set' && effect.prop) {
        (element as any)[effect.prop] = effect.value;
        Logger.debug('[Hydration]', 'DOM action set', { ref: effect.ref, prop: effect.prop });
      } else if (kind === 'dom.toggle' && effect.prop) {
        (element as any)[effect.prop] = effect.value;
        Logger.debug('[Hydration]', 'DOM action toggle', { ref: effect.ref, prop: effect.prop, value: effect.value });
      } else if (kind === 'dom.class' && effect.class && element instanceof Element) {
        if (effect.on === true) {
          element.classList.add(effect.class);
        } else {
          element.classList.remove(effect.class);
        }
        Logger.debug('[Hydration]', 'DOM action class', { ref: effect.ref, class: effect.class, on: effect.on });
      } else if (kind === 'dom.scroll' && element instanceof Element) {
        const options: ScrollIntoViewOptions = {};
        if (effect.behavior) options.behavior = effect.behavior as ScrollBehavior;
        if (effect.block) options.block = effect.block as ScrollLogicalPosition;
        if (effect.inline) options.inline = effect.inline as ScrollLogicalPosition;
        element.scrollIntoView(options);
        Logger.debug('[Hydration]', 'DOM action scroll', { ref: effect.ref, options });
      }
    } catch (error) {
      Logger.warn('[Hydration]', 'DOM action failed', { ref: effect.ref, kind: effect.kind, error });
    }
  }

  private applyCookieEffect(effect: CookieEffect): void {
    if (typeof window === 'undefined' || typeof fetch !== 'function') {
      return;
    }
    if (!effect.endpoint || !effect.sid || !effect.token) {
      Logger.debug('[Hydration]', 'Cookie effect skipped (missing required fields)', effect);
      return;
    }
    const method = effect.method || 'POST';
    Logger.debug('[Hydration]', 'Applying cookie effect', { endpoint: effect.endpoint, method });
    fetch(effect.endpoint, {
      method,
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        sid: effect.sid,
        token: effect.token,
      }),
      credentials: 'include',
    })
      .then((response) => {
        if (response.ok) {
          Logger.debug('[Hydration]', 'Cookie effect succeeded', { endpoint: effect.endpoint });
        } else {
          Logger.warn('[Hydration]', 'Cookie effect failed', { endpoint: effect.endpoint, status: response.status });
        }
      })
      .catch((error) => {
        Logger.warn('[Hydration]', 'Cookie effect error', { endpoint: effect.endpoint, error });
      });
  }

  private applyNavigation(nav: { push?: string; replace?: string; back?: boolean }): void {
    if (typeof window === 'undefined' || typeof history === 'undefined') {
      return;
    }
    if (!nav.push && !nav.replace && !nav.back) {
      return;
    }
    if (nav.back) {
      try {
        history.back();
        Logger.debug('[Hydration]', 'Navigation back');
      } catch (error) {
        Logger.warn('[Hydration]', 'Navigation back failed', { error });
      }
      return;
    }
    const url = nav.replace || nav.push;
    if (!url || typeof url !== 'string') {
      Logger.debug('[Hydration]', 'Navigation skipped (invalid URL)', nav);
      return;
    }
    try {
      if (nav.replace) {
        history.replaceState(null, '', nav.replace);
        Logger.debug('[Hydration]', 'Navigation replace', { url: nav.replace });
      } else {
        history.pushState(null, '', nav.push!);
        Logger.debug('[Hydration]', 'Navigation push', { url: nav.push });
      }
    } catch (error) {
      Logger.warn('[Hydration]', 'Navigation failed', { url, error });
    }
  }

  private resolveRoot(): Element | Document | ShadowRoot | null {
    const root = this.options.root ?? document.body ?? document;
    if (!root) {
      return null;
    }
    if (root instanceof Element || root instanceof Document || root instanceof ShadowRoot) {
      return root;
    }
    return document.body ?? document;
  }

  private primeRegistries(payload: TemplatePayload, root: ParentNode | null, overrides?: Map<string, any>): void {
    if (!root) {
      return;
    }
    this.dom.reset();
    this.refs.clear();
    this.refs.apply(payload.refs);
    if (payload.bindings?.slots) {
      registerSlotTable(payload.bindings.slots);
    } else {
      registerSlotTable(undefined);
    }
    const listRowIndex = buildListRowIndex(payload);
    this.dom.registerSlotAnchors(payload.slotPaths, overrides);
    if (Array.isArray(payload.slots)) {
      this.dom.registerSlots(payload.slots);
    }
    this.dom.registerListContainers(payload.listPaths, overrides, listRowIndex);
    if (payload.bindings?.router) {
      applyRouterBindings(payload.bindings.router, overrides);
    }
    if (payload.bindings?.refs) {
      this.refs.registerBindings(payload.bindings.refs, overrides);
    }
    this.uploads.prime(payload.bindings?.uploads ?? null, overrides);
    Logger.debug('[Hydration]', 'registries primed', {
      slots: payload.slotPaths?.length ?? 0,
      lists: payload.listPaths?.length ?? 0,
      routers: payload.bindings?.router?.length ?? 0,
      refs: payload.bindings?.refs?.length ?? 0,
      uploads: payload.bindings?.uploads?.length ?? 0,
    });
  }

  private handleDOMRequest(msg: DOMRequestMessage): void {
    if (!msg || !msg.id || !msg.ref) {
      this.runtime.sendDOMResponse({ id: msg?.id ?? '', error: 'invalid request' });
      return;
    }
    const element = this.refs.get(msg.ref);
    if (!element) {
      this.runtime.sendDOMResponse({ id: msg.id, error: 'element not found' });
      return;
    }
    try {
      let result: any;
      const values: Record<string, any> = {};
      if (msg.method && typeof (element as any)[msg.method] === 'function') {
        const args = Array.isArray(msg.args) ? msg.args : [];
        result = (element as any)[msg.method](...args);
      }
      if (Array.isArray(msg.props)) {
        msg.props.forEach((prop) => {
          if (prop) {
            values[prop] = (element as any)[prop];
          }
        });
      }
      this.runtime.sendDOMResponse({
        id: msg.id,
        result,
        values: Object.keys(values).length > 0 ? values : undefined,
      });
    } catch (error) {
      this.runtime.sendDOMResponse({
        id: msg.id,
        error: error instanceof Error ? error.message : 'unknown error',
      });
    }
  }

  getRegistry(): DomRegistry {
    return this.dom;
  }
}

function pruneWhitespace(root: ParentNode | null): void {
  if (!root || typeof Node === 'undefined') {
    return;
  }
  const doc = (root instanceof Document ? root : root.ownerDocument) ?? document;
  if (!doc || typeof doc.createTreeWalker !== 'function') {
    return;
  }
  const walker = doc.createTreeWalker(root, NodeFilter.SHOW_TEXT, null);
  const removals: Node[] = [];
  let current = walker.nextNode();
  while (current) {
    if (current.nodeType === Node.TEXT_NODE) {
      const text = current.textContent ?? '';
      if (!text.trim()) {
        removals.push(current);
      }
    }
    current = walker.nextNode();
  }
  removals.forEach((node) => {
    if (node.parentNode) {
      node.parentNode.removeChild(node);
    }
  });
}

function buildListRowIndex(payload: TemplatePayload): Map<number, { key: string; count: number }[]> {
  const map = new Map<number, { key: string; count: number }[]>();
  if (!payload || !Array.isArray(payload.d)) {
    return map;
  }
  const slots = Array.isArray(payload.slots) ? payload.slots : [];
  payload.d.forEach((slot, index) => {
    if (!slot || slot.kind !== 'list' || !Array.isArray(slot.list) || slot.list.length === 0) {
      return;
    }
    const entries = slot.list
      .map((row) => {
        if (!row || typeof row.key !== 'string' || row.key.length === 0) {
          return undefined;
        }
        const count = Math.max(1, Number(row.rootCount) || 1);
        return { key: row.key, count };
      })
      .filter((entry): entry is { key: string; count: number } => Boolean(entry));
    if (entries.length === 0) {
      return;
    }
    const slotId = slots[index]?.anchorId ?? index;
    map.set(slotId, entries);
  });
  return map;
}

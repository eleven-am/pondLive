import type { RefBindingDescriptor, RefDelta, RefMeta } from './types';
import type { ComponentRange } from './manifest';
import { resolveNodeInComponent } from './manifest';
import type { LiveRuntime } from './runtime';
import { Logger } from './logger';

interface RefListeners {
  element: Element;
  listeners: Map<string, EventListener>;
}

export class RefRegistry {
  private meta = new Map<string, RefMeta>();
  private bindings = new Map<string, RefListeners>();

  constructor(private runtime: LiveRuntime) {}

  clear(): void {
    Array.from(this.bindings.keys()).forEach((id) => this.detach(id));
    this.meta.clear();
    Logger.debug('[Refs]', 'cleared all bindings');
  }

  apply(delta?: RefDelta | null): void {
    if (!delta) {
      return;
    }
    if (Array.isArray(delta.del)) {
      for (const id of delta.del) {
        this.detach(id);
        this.meta.delete(id);
      }
    }
    if (delta.add) {
      for (const [id, meta] of Object.entries(delta.add)) {
        if (id) {
          this.meta.set(id, meta);
        }
      }
    }
    Logger.debug('[Refs]', 'applied ref delta', {
      added: delta.add ? Object.keys(delta.add).length : 0,
      removed: delta.del?.length ?? 0,
    });
  }

  registerBindings(descriptors?: RefBindingDescriptor[] | null, overrides?: Map<string, ComponentRange>): void {
    if (!Array.isArray(descriptors)) {
      return;
    }
    Logger.debug('[Refs]', 'registering ref bindings', { count: descriptors.length });
    descriptors.forEach((descriptor, index) => {
      if (!descriptor || typeof descriptor.refId !== 'string' || descriptor.refId.length === 0) {
        return;
      }
      Logger.debug('[Refs]', 'processing ref binding', {
        index,
        descriptor: {
          componentId: descriptor.componentId,
          path: descriptor.path,
          refId: descriptor.refId,
        },
        hasRefId: !!descriptor.refId,
        refIdLength: descriptor.refId?.length ?? 0,
      });
      Logger.debug('[Refs]', 'resolving node for ref', {
        refId: descriptor.refId,
        componentId: descriptor.componentId,
        path: descriptor.path,
      });
      const node = resolveNodeInComponent(descriptor.componentId, descriptor.path, overrides);
      Logger.debug('[Refs]', 'node resolved', {
        refId: descriptor.refId,
        node,
        isElement: node instanceof Element,
        nodeType: node?.nodeType,
        nodeName: node?.nodeName,
      });
      if (node instanceof Element) {
        Logger.debug('[Refs]', 'attaching ref', { refId: descriptor.refId, node });
        this.attach(descriptor.refId, node);
      } else {
        Logger.debug('[Refs]', 'node is not Element, detaching', { refId: descriptor.refId, node });
        this.detach(descriptor.refId);
      }
    });
  }

  get(id: string): Element | undefined {
    return this.bindings.get(id)?.element;
  }

  private attach(refId: string, element: Element): void {
    const meta = this.meta.get(refId);
    if (!meta) {
      return;
    }
    const existing = this.bindings.get(refId);
    if (existing && existing.element === element) {
      return;
    }
    this.detach(refId);
    const listeners = new Map<string, EventListener>();
    this.bindings.set(refId, { element, listeners });
    Logger.debug('[Refs]', 'attached ref', {
      refId,
      tag: element.tagName,
    });
  }

  private detach(refId: string): void {
    const binding = this.bindings.get(refId);
    if (!binding) {
      return;
    }
    binding.listeners.forEach((listener, event) => {
      binding.element.removeEventListener(event, listener);
    });
    this.bindings.delete(refId);
    Logger.debug('[Refs]', 'detached ref', { refId });
  }
}

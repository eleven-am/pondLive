import type { RouterBindingDescriptor } from './types';
import type { ComponentRange } from './manifest';
import { resolveNodeInComponent, getComponentRange } from './manifest';
import { Logger } from './logger';

export type RouterMeta = {
  path?: string;
  query?: string;
  hash?: string;
  replace?: string;
};

const routerMeta = new WeakMap<Element, RouterMeta>();

export function applyRouterBindings(
  descriptors?: RouterBindingDescriptor[] | null,
  overrides?: Map<string, ComponentRange>,
): void {
  if (!Array.isArray(descriptors)) {
    return;
  }
  let applied = 0;
  descriptors.forEach((descriptor, index) => {
    Logger.debug('[Router]', `processing binding ${index}`, {
      componentId: descriptor?.componentId?.substring(0, 12) + '...',
      pathLength: descriptor?.path?.length,
      pathValue: descriptor?.pathValue,
    });
    if (!descriptor || typeof descriptor.componentId !== 'string') {
      Logger.debug('[Router]', `skipping binding ${index}: invalid descriptor`);
      return;
    }
    const range = overrides?.get(descriptor.componentId) ?? getComponentRange(descriptor.componentId);
    if (range) {
      Logger.debug('[Router]', `component range for binding ${index}`, {
        container: range.container.nodeName,
        start: range.startIndex,
        end: range.endIndex,
        childCount: range.container.childNodes.length,
      });
    } else {
      Logger.debug('[Router]', `no component range found for binding ${index}`);
    }
    let node = resolveNodeInComponent(descriptor.componentId, descriptor.path, overrides);
    Logger.debug('[Router]', `resolved node for binding ${index}`, {
      nodeType: node?.nodeName,
      isElement: node instanceof Element,
    });

    // If path resolution failed but the component range container is an Element,
    // use the container itself. This handles cases where the component's root
    // element IS the element we want to bind to (e.g., RouterLink's <a> tag).
    if (!(node instanceof Element) && range && range.container instanceof Element) {
      Logger.debug('[Router]', `binding ${index} falling back to range container`, {
        containerNodeName: range.container.nodeName,
      });
      node = range.container;
    }

    if (!(node instanceof Element)) {
      Logger.debug('[Router]', `binding ${index} failed: node is not an Element`);
      return;
    }
    routerMeta.set(node, {
      path: descriptor.pathValue ?? undefined,
      query: descriptor.query ?? undefined,
      hash: descriptor.hash ?? undefined,
      replace: descriptor.replace ?? undefined,
    });
    // Mark element for debugging
    (node as any).__routerDebug = {
      path: descriptor.pathValue,
      appliedAt: Date.now(),
      componentId: descriptor.componentId,
    };
    Logger.debug('[Router]', `binding ${index} applied successfully`, {
      path: descriptor.pathValue,
      nodeName: node.nodeName,
      isConnected: node.isConnected,
      hasParent: !!node.parentElement,
    });
    applied += 1;
  });
  Logger.debug('[Router]', 'applied router bindings', { count: applied });
}

export function getRouterMeta(element: Element | null): RouterMeta | undefined {
  if (!element) {
    return undefined;
  }
  let current: Element | null = element;
  while (current) {
    const meta = routerMeta.get(current);
    if (meta?.path) {
      return meta;
    }
    current = current.parentElement;
  }
  return undefined;
}

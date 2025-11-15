import type { RouterBindingDescriptor } from './types';
import type { ComponentRange } from './manifest';
import { resolveNodeInComponent } from './manifest';
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
  descriptors.forEach((descriptor) => {
    if (!descriptor || typeof descriptor.componentId !== 'string') {
      return;
    }
    const node = resolveNodeInComponent(descriptor.componentId, descriptor.path, overrides);
    if (!(node instanceof Element)) {
      return;
    }
    routerMeta.set(node, {
      path: descriptor.pathValue ?? undefined,
      query: descriptor.query ?? undefined,
      hash: descriptor.hash ?? undefined,
      replace: descriptor.replace ?? undefined,
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

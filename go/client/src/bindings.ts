import type { ComponentRange } from "./componentRanges";
import { resolveComponentPathNode } from "./manifest";
import { applyRouterAttribute } from "./events";
import { attachRefToElement } from "./refs";
import type {
  RefBindingDescriptor,
  RouterBindingDescriptor,
} from "./types";

export interface BindingResolveOptions {
  overrides?: Map<string, ComponentRange>;
  fallbackRange?: ComponentRange | null;
}

function resolveBindingTarget(
  componentId: string,
  path: number[] | undefined,
  options?: BindingResolveOptions,
): Element | null {
  const node = resolveComponentPathNode(componentId, path, {
    overrides: options?.overrides,
    fallbackRange: options?.fallbackRange ?? null,
  });
  if (node instanceof Element) {
    return node;
  }
  return null;
}

export function applyRouterBindings(
  bindings: RouterBindingDescriptor[] | undefined | null,
  options?: BindingResolveOptions,
): void {
  if (!Array.isArray(bindings) || bindings.length === 0) {
    return;
  }
  for (const binding of bindings) {
    if (!binding) {
      continue;
    }
    const element = resolveBindingTarget(
      binding.componentId ?? "",
      binding.path ?? undefined,
      options,
    );
    if (!element) {
      continue;
    }
    if (binding.pathValue !== undefined) {
      applyRouterAttribute(element, "path", binding.pathValue ?? "");
    }
    if (binding.query !== undefined) {
      applyRouterAttribute(element, "query", binding.query ?? "");
    }
    if (binding.hash !== undefined) {
      applyRouterAttribute(element, "hash", binding.hash ?? "");
    }
    if (binding.replace !== undefined) {
      applyRouterAttribute(element, "replace", binding.replace ?? "");
    }
  }
}

export function applyRefBindings(
  bindings: RefBindingDescriptor[] | undefined | null,
  options?: BindingResolveOptions,
): void {
  if (!Array.isArray(bindings) || bindings.length === 0) {
    return;
  }
  for (const binding of bindings) {
    if (!binding || typeof binding.refId !== "string" || binding.refId.length === 0) {
      continue;
    }
    const element = resolveBindingTarget(
      binding.componentId ?? "",
      binding.path ?? undefined,
      options,
    );
    if (!element) {
      continue;
    }
    attachRefToElement(binding.refId, element);
  }
}

import type { ComponentMarkerDescriptor } from "./types";

export interface ComponentMarkerBounds {
  container: ParentNode;
  start: number;
  end: number;
}

const componentMarkerIndex = new Map<string, ComponentMarkerBounds>();
const hasDocumentConstructor = typeof Document !== "undefined";

function cloneDescriptors(
  descriptors: Record<string, ComponentMarkerDescriptor>,
): Record<string, ComponentMarkerDescriptor> {
  const out: Record<string, ComponentMarkerDescriptor> = {};
  for (const [id, descriptor] of Object.entries(descriptors)) {
    if (!id || !descriptor) continue;
    const start = Number(descriptor.start);
    const end = Number(descriptor.end);
    if (!Number.isFinite(start) || !Number.isFinite(end)) continue;
    const clone: ComponentMarkerDescriptor = {
      start,
      end,
    };
    if ("container" in descriptor) {
      const normalized = normalizePath(descriptor.container);
      if (normalized === null) {
        continue;
      }
      if (normalized.length > 0) {
        clone.container = normalized;
      }
    }
    out[id] = clone;
  }
  return out;
}

function normalizePath(path: unknown): number[] | null {
  if (path == null) {
    return [];
  }
  if (!Array.isArray(path)) {
    return null;
  }
  const result: number[] = [];
  for (const segment of path) {
    const value = Number(segment);
    if (!Number.isFinite(value)) {
      return null;
    }
    result.push(Math.trunc(value));
  }
  return result;
}

function assignMarkers(
  descriptors: Record<string, ComponentMarkerDescriptor>,
  root: ParentNode | null | undefined,
  resetExisting: boolean,
): void {
  const target = root ?? (typeof document !== "undefined" ? document : null);
  if (!target || !isParentNode(target)) {
    return;
  }

  if (resetExisting) {
    componentMarkerIndex.clear();
  }

  for (const [id, descriptor] of Object.entries(descriptors)) {
    if (!id || !descriptor) continue;
    const start = Number(descriptor.start);
    const end = Number(descriptor.end);
    if (!Number.isFinite(start) || !Number.isFinite(end)) continue;
    if (end < start) continue;
    const normalizedStart = Math.max(0, Math.trunc(start));
    const normalizedEnd = Math.max(normalizedStart, Math.trunc(end));
    const path = Array.isArray(descriptor.container)
      ? descriptor.container
      : [];
    const container = resolveContainer(target, path);
    if (!container) continue;
    registerComponentMarker(id, container, normalizedStart, normalizedEnd);
  }
}

function resolveContainer(root: ParentNode, path: number[]): ParentNode | null {
  if (path.length === 0) {
    return root;
  }
  let current: Node | null = root;
  for (const index of path) {
    current = getRenderableChild(current, index);
    if (!current) {
      return null;
    }
  }
  return isParentNode(current) ? current : null;
}

function getRenderableChild(parent: Node | null, targetIndex: number): Node | null {
  if (!isParentNode(parent) || targetIndex < 0) {
    return null;
  }
  let index = 0;
  for (let child = parent.firstChild; child; child = child.nextSibling) {
    if (!isRenderableNode(child)) {
      continue;
    }
    if (index === targetIndex) {
      return child;
    }
    index += 1;
  }
  return null;
}

function isRenderableNode(node: Node | null): boolean {
  return !!node && node.nodeType !== Node.COMMENT_NODE;
}

function isParentNode(node: Node | null | undefined): node is ParentNode {
  if (!node) return false;
  if (node instanceof Element || node instanceof DocumentFragment) {
    return true;
  }
  if (hasDocumentConstructor && node instanceof Document) {
    return true;
  }
  return false;
}

export function resetComponentMarkers(): void {
  componentMarkerIndex.clear();
}

export function initializeComponentMarkers(
  descriptors: Record<string, ComponentMarkerDescriptor> | null | undefined,
  root?: ParentNode | null,
): void {
  if (!descriptors) {
    resetComponentMarkers();
    return;
  }
  assignMarkers(cloneDescriptors(descriptors), root ?? null, true);
}

export function registerComponentMarkers(
  descriptors: Record<string, ComponentMarkerDescriptor> | null | undefined,
  root: ParentNode | null | undefined,
): void {
  if (!descriptors) {
    return;
  }
  assignMarkers(cloneDescriptors(descriptors), root, false);
}

export function registerComponentMarker(
  id: string,
  container: ParentNode | null | undefined,
  start: number,
  end: number,
): void {
  if (!id || !container) return;
  const normalizedStart = Math.max(0, Math.trunc(Number(start)));
  const normalizedEnd = Math.max(normalizedStart, Math.trunc(Number(end)));
  componentMarkerIndex.set(id, {
    container,
    start: normalizedStart,
    end: normalizedEnd,
  });
}

export function getComponentBounds(id: string): ComponentMarkerBounds | null {
  if (!id) return null;
  const entry = componentMarkerIndex.get(id);
  if (!entry || !isParentNode(entry.container)) {
    return null;
  }
  return {
    container: entry.container,
    start: entry.start,
    end: entry.end,
  };
}

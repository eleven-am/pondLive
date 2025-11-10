import { getComponentBounds } from "./componentMarkers";
import type { ComponentMarkerBounds } from "./componentMarkers";
import type { SlotMeta } from "./types";

export function resolveSlotTarget(slot: SlotMeta): Node | null {
  if (!slot || typeof slot.anchorId !== "number") {
    return null;
  }
  const anchor = resolveSlotPath(slot.component, slot.path);
  if (!anchor) {
    return null;
  }
  if (typeof slot.text !== "number") {
    return anchor;
  }
  if (!isParentNode(anchor)) {
    return null;
  }
  return getRenderableChild(anchor, slot.text);
}

export function resolveListContainer(slot: SlotMeta): Element | null {
  if (!slot || !slot.list) {
    return null;
  }
  const component = slot.list.component ?? slot.component;
  const path = slot.list.path ?? slot.path;
  const target = resolveSlotPath(component, path);
  return target instanceof Element ? target : null;
}

export function resolveSlotPath(
  componentId?: string,
  path?: number[],
): Node | null {
  const segments = Array.isArray(path) ? path : [];
  if (typeof componentId === "string" && componentId.length > 0) {
    const bounds = getComponentBounds(componentId);
    if (!bounds) {
      return null;
    }
    return resolveWithinComponent(bounds, segments);
  }
  return resolveFromDocument(segments);
}

function resolveWithinComponent(
  bounds: ComponentMarkerBounds,
  path: number[],
): Node | null {
  const { container, start, end } = bounds;
  if (!isParentNode(container) || start < 0 || end < start) {
    return null;
  }
  if (path.length === 0) {
    if (end <= start) {
      return null;
    }
    return getRenderableChild(container, start);
  }
  const headIndex = start + path[0];
  if (headIndex < start || headIndex >= end) {
    return null;
  }
  let current: Node | null = getRenderableChild(container, headIndex);
  if (!current) {
    return null;
  }
  for (let i = 1; i < path.length; i += 1) {
    current = getRenderableChildFromNode(current, path[i]);
    if (!current) {
      return null;
    }
  }
  return current;
}

function resolveFromDocument(path: number[]): Node | null {
  if (typeof document === "undefined") {
    return null;
  }
  let current: Node = document.body ?? document;
  if (path.length === 0) {
    const first = getRenderableChildFromNode(current, 0);
    return first ?? null;
  }
  for (const index of path) {
    const next = getRenderableChildFromNode(current, index);
    if (!next) {
      return null;
    }
    current = next;
  }
  return current;
}

function getRenderableChildFromNode(node: Node, targetIndex: number): Node | null {
  if (!isParentNode(node) || targetIndex < 0) {
    return null;
  }
  return getRenderableChild(node, targetIndex);
}

function getRenderableChild(parent: ParentNode, targetIndex: number): Node | null {
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

function isParentNode(node: Node): node is ParentNode {
  return node instanceof Element || node instanceof Document || node instanceof DocumentFragment;
}

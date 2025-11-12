import type {
  SlotPathDescriptor,
  ListPathDescriptor,
  ComponentPathDescriptor,
} from './types';
import {
  ComponentRange,
  getComponentRange,
  registerComponentRanges,
} from './componentRanges';

function debugLog(...args: any[]): void {
  if (typeof console === 'undefined') {
    return;
  }
  console.log('[liveui][manifest]', ...args);
}

function debugWarn(...args: any[]): void {
  if (typeof console === 'undefined') {
    return;
  }
  console.warn('[liveui][manifest]', ...args);
}

function ensureArray(path?: number[] | null): number[] {
  if (!Array.isArray(path)) {
    return [];
  }
  return path.filter((index) => Number.isInteger(index));
}

function getRootContainer(root?: ParentNode | null): ParentNode {
  if (root && root instanceof Document) {
    return root.body ?? root;
  }
  if (root && 'childNodes' in root) {
    return root;
  }
  if (typeof document !== 'undefined') {
    return document.body ?? document;
  }
  throw new Error('liveui: unable to resolve root container for manifest resolution');
}

function clampIndex(container: ParentNode, index: number): number {
  const maxIndex = container.childNodes.length - 1;
  if (maxIndex < 0) {
    return 0;
  }
  if (index < 0) {
    return 0;
  }
  if (index > maxIndex) {
    return maxIndex;
  }
  return index;
}

function computeBaseRange(root: ParentNode): ComponentRange {
  const count = root.childNodes.length;
  return {
    container: root,
    startIndex: 0,
    endIndex: count > 0 ? count - 1 : -1,
  };
}

function resolveContainerFromParent(
  parentRange: ComponentRange,
  parentPath: number[] | undefined,
): ParentNode | null {
  if (!parentRange || parentRange.endIndex < parentRange.startIndex) {
    return null;
  }
  const steps = ensureArray(parentPath);
  if (steps.length <= 1) {
    return parentRange.container;
  }
  const containerSteps = steps.slice(0, -1);
  let current: ParentNode = parentRange.container;
  let node: Node | null = null;
  for (let i = 0; i < containerSteps.length; i++) {
    const step = containerSteps[i];
    const targetIndex = i === 0 ? parentRange.startIndex + step : step;
    node = current.childNodes.item(targetIndex) ?? null;
    if (!(node instanceof Element || node instanceof DocumentFragment)) {
      return null;
    }
    current = node;
  }
  return current;
}

function deriveRangeFromDescriptor(
  descriptor: ComponentPathDescriptor,
  parentRange: ComponentRange,
): ComponentRange | null {
  if (!parentRange || parentRange.endIndex < parentRange.startIndex) {
    return null;
  }
  const anchorPath = ensureArray(descriptor.parentPath);
  const anchor = resolveNodeInRange(parentRange, anchorPath);
  if (!anchor) {
    return null;
  }
  const container =
    (anchor.parentNode as ParentNode | null) ?? parentRange.container;
  const startIndex = getNodeIndex(container, anchor);
  if (startIndex < 0) {
    return null;
  }
  return {
    container,
    startIndex,
    endIndex: startIndex,
  };
}

function getNodeIndex(container: ParentNode, target: Node | null): number {
  if (!container || !target) {
    return -1;
  }
  const nodes = container.childNodes;
  for (let i = 0; i < nodes.length; i++) {
    if (nodes.item(i) === target) {
      return i;
    }
  }
  return -1;
}

export function computeComponentRanges(
  descriptors: ComponentPathDescriptor[] | undefined,
  options?: {
    root?: ParentNode | null;
    baseId?: string | null;
    baseRange?: ComponentRange | null;
  },
): Map<string, ComponentRange> {
  const ranges = new Map<string, ComponentRange>();
  if (!Array.isArray(descriptors) || descriptors.length === 0) {
    return ranges;
  }
  const rootContainer = getRootContainer(options?.root ?? null);
  const defaultRange = options?.baseRange ?? computeBaseRange(rootContainer);
  const baseId = options?.baseId ?? '';
  if (options?.baseRange && baseId) {
    ranges.set(baseId, options.baseRange);
  }
  const pending = new Map<string, ComponentPathDescriptor>();
  for (const descriptor of descriptors) {
    if (!descriptor || typeof descriptor.componentId !== 'string') continue;
    pending.set(descriptor.componentId, descriptor);
  }
  let progressed = true;
  while (pending.size > 0 && progressed) {
    progressed = false;
    for (const [id, descriptor] of Array.from(pending.entries())) {
      const parentId = descriptor.parentId ?? '';
      let parentRange: ComponentRange | null = null;
      if (parentId) {
        parentRange = ranges.get(parentId) ?? getComponentRange(parentId);
      } else {
        parentRange = baseId ? ranges.get(baseId) ?? null : null;
        if (!parentRange) {
          parentRange = defaultRange;
        }
      }
      if (!parentRange) {
        continue;
      }
      const derived = deriveRangeFromDescriptor(descriptor, parentRange);
      if (!derived) {
        pending.delete(id);
        continue;
      }
      ranges.set(id, derived);
      pending.delete(id);
      progressed = true;
    }
  }
  return ranges;
}

export function resolveComponentRanges(
  descriptors: ComponentPathDescriptor[] | undefined,
  options?: {
    root?: ParentNode | null;
    baseId?: string | null;
    baseRange?: ComponentRange | null;
  },
): void {
  const ranges = computeComponentRanges(descriptors, options);
  registerComponentRanges(ranges);
}

export function resolveComponentPathNode(
  componentId: string,
  path?: number[] | null,
  options?: {
    overrides?: Map<string, ComponentRange>;
    fallbackRange?: ComponentRange | null;
  },
): Node | null {
  let range: ComponentRange | null = null;
  if (componentId && componentId.length > 0) {
    range = options?.overrides?.get(componentId) ?? getComponentRange(componentId);
  } else {
    range = options?.fallbackRange ?? null;
  }
  if (!range) {
    return null;
  }
  return resolveNodeInRange(range, path ?? undefined);
}

function resolveNodeInRange(
  range: ComponentRange | null,
  path: number[] | undefined,
): Node | null {
  if (!range || range.endIndex < range.startIndex) return null;
  const steps = ensureArray(path);
  const rootIndex = clampIndex(range.container, range.startIndex);
  let current: Node | null = range.container.childNodes.item(rootIndex) ?? null;
  if (!current) {
    return null;
  }
  if (steps.length === 0) {
    return current;
  }
  for (const step of steps) {
    if (!(current instanceof Element || current instanceof DocumentFragment)) {
      return null;
    }
    const childIndex = clampIndex(current, step);
    current = current.childNodes.item(childIndex) ?? null;
    if (!current) {
      return null;
    }
  }
  return current;
}

export function resolveSlotAnchors(
  descriptors: SlotPathDescriptor[] | undefined,
  overrides?: Map<string, ComponentRange>,
): Map<number, Node> {
  const anchors = new Map<number, Node>();
  if (!Array.isArray(descriptors)) {
    return anchors;
  }
  for (const descriptor of descriptors) {
    if (!descriptor) continue;
    const slotId = Number(descriptor.slot);
    if (!Number.isInteger(slotId) || slotId < 0 || anchors.has(slotId)) {
      continue;
    }
    const range =
      overrides?.get(descriptor.componentId) ??
      getComponentRange(descriptor.componentId);
    if (!range) {
      debugWarn(
        'slot range missing',
        descriptor.componentId,
        descriptor,
        Array.from(overrides?.keys() ?? []),
      );
      continue;
    }
    const anchor = resolveNodeInRange(range, descriptor.elementPath);
    if (!anchor) {
      debugWarn('slot anchor not found', descriptor, range);
      continue;
    }
    let target: Node | null = anchor;
    const textIndex = Number(descriptor.textChildIndex);
    if (Number.isInteger(textIndex) && textIndex >= 0) {
      if (anchor instanceof Text) {
        target = anchor;
      } else if (anchor instanceof Element || anchor instanceof DocumentFragment) {
        let textNode = anchor.childNodes.item(textIndex) ?? null;
        if (!textNode && typeof document !== 'undefined') {
          textNode = document.createTextNode('');
          const before =
            textIndex < anchor.childNodes.length
              ? anchor.childNodes.item(textIndex)
              : null;
          anchor.insertBefore(textNode, before);
        }
        target = textNode;
      } else {
        target = null;
      }
    }
    if (!target) {
      debugWarn('slot target missing', descriptor, {
        anchorName:
          anchor instanceof Element ? anchor.tagName : anchor?.nodeName,
      });
      continue;
    }
    debugLog('slot resolved', slotId, descriptor.componentId);
    anchors.set(slotId, target);
  }
  return anchors;
}

export function resolveListContainers(
  descriptors: ListPathDescriptor[] | undefined,
  overrides?: Map<string, ComponentRange>,
): Map<number, Element> {
  const lists = new Map<number, Element>();
  if (!Array.isArray(descriptors)) {
    return lists;
  }
  for (const descriptor of descriptors) {
    if (!descriptor) continue;
    const slotId = Number(descriptor.slot);
    if (!Number.isInteger(slotId) || slotId < 0 || lists.has(slotId)) {
      continue;
    }
    const range =
      overrides?.get(descriptor.componentId) ??
      getComponentRange(descriptor.componentId);
    if (!range) {
      debugWarn(
        'list range missing',
        descriptor.componentId,
        descriptor,
        Array.from(overrides?.keys() ?? []),
      );
      continue;
    }
    const node = resolveNodeInRange(range, descriptor.elementPath);
    if (!(node instanceof Element)) {
      debugWarn('list container not found', descriptor, range);
      continue;
    }
    debugLog('list container resolved', slotId, descriptor.componentId);
    lists.set(slotId, node);
  }
  return lists;
}

export function applyComponentRanges(
  descriptors: ComponentPathDescriptor[] | undefined,
  options?: {
    root?: ParentNode | null;
    baseId?: string | null;
    baseRange?: ComponentRange | null;
  },
): Map<string, ComponentRange> {
  const ranges = computeComponentRanges(descriptors, options);
  registerComponentRanges(ranges);
  return ranges;
}

export function resolveNodeInComponent(
  componentId: string,
  path: number[] | undefined,
  overrides?: Map<string, ComponentRange>,
): Node | null {
  if (typeof componentId !== "string" || componentId.length === 0) {
    return null;
  }
  const range = overrides?.get(componentId) ?? getComponentRange(componentId);
  if (!range) {
    return null;
  }
  return resolveNodeInRange(range, path);
}

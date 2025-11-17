import type {
  SlotPathDescriptor,
  ListPathDescriptor,
  ComponentPathDescriptor,
  PathSegmentDescriptor,
} from './types';
import { Logger } from './logger';

export interface ComponentRange {
  container: ParentNode;
  startIndex: number;
  endIndex: number;
}

const componentRanges = new Map<string, ComponentRange>();

export function registerComponentRanges(ranges: Map<string, ComponentRange>): void {
  ranges.forEach((range, id) => {
    if (id) {
      componentRanges.set(id, range);
    }
  });
  // DEBUG: Log all registered component ranges
  const rangeData = Array.from(componentRanges.entries()).map(([id, range]) => ({
    id: id.substring(0, 12) + '...',
    container: range.container.nodeName,
    start: range.startIndex,
    end: range.endIndex,
    childCount: range.container.childNodes.length,
  }));
  Logger.debug('[Manifest]', 'registered component ranges', { count: componentRanges.size, ranges: rangeData });
}

export function getComponentRange(id: string): ComponentRange | undefined {
  return componentRanges.get(id);
}

// DEBUG: Expose component ranges for debugging
if (typeof window !== 'undefined') {
  (window as any).__DEBUG_COMPONENT_RANGES__ = componentRanges;
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
    const range = overrides?.get(descriptor.componentId) ?? getComponentRange(descriptor.componentId);
    if (!range) {
      continue;
    }
    const anchor = resolveNodeBySegments(range, descriptor.path);
    if (!anchor) {
      continue;
    }
    let target: Node | null = anchor;
    const textIndex = Number(descriptor.textChildIndex);
    if (Number.isInteger(textIndex) && textIndex >= 0) {
      if (anchor instanceof Text) {
        target = anchor;
      } else if (anchor instanceof Element || anchor instanceof DocumentFragment) {
        const existing = anchor.childNodes.item(textIndex);
        if (existing) {
          target = existing;
        } else if (typeof document !== 'undefined') {
          const textNode = document.createTextNode('');
          const before = anchor.childNodes.item(textIndex) ?? null;
          anchor.insertBefore(textNode, before);
          target = textNode;
        }
      } else {
        target = null;
      }
    }
    if (!target) {
      continue;
    }
    anchors.set(slotId, target);
  }
  return anchors;
}

export function resolveListContainers(
  descriptors: ListPathDescriptor[] | undefined,
  overrides?: Map<string, ComponentRange>,
): Map<number, Element> {
  const containers = new Map<number, Element>();
  Logger.debug('[Manifest]', 'resolveListContainers START', {
    descriptorsIsArray: Array.isArray(descriptors),
    descriptorsLength: Array.isArray(descriptors) ? descriptors.length : 'N/A',
    hasOverrides: Boolean(overrides),
  });
  if (!Array.isArray(descriptors)) {
    Logger.debug('[Manifest]', 'resolveListContainers: descriptors not an array, returning empty');
    return containers;
  }
  for (let i = 0; i < descriptors.length; i++) {
    const descriptor = descriptors[i];
    Logger.debug('[Manifest]', `resolveListContainers: processing descriptor[${i}]`, {
      descriptor,
      hasDescriptor: Boolean(descriptor),
    });
    if (!descriptor) continue;
    const slotId = Number(descriptor.slot);
    if (!Number.isInteger(slotId) || slotId < 0 || containers.has(slotId)) {
      Logger.debug('[Manifest]', `resolveListContainers: skipping descriptor[${i}] - invalid slot`, {
        slotId,
        isInteger: Number.isInteger(slotId),
        alreadyHas: containers.has(slotId),
      });
      continue;
    }
    const range = overrides?.get(descriptor.componentId) ?? getComponentRange(descriptor.componentId);
    if (!range) {
      Logger.debug('[Manifest]', `resolveListContainers: descriptor[${i}] - NO RANGE FOUND`, {
        componentId: descriptor.componentId,
        hasOverride: Boolean(overrides?.get(descriptor.componentId)),
        hasComponentRange: Boolean(getComponentRange(descriptor.componentId)),
      });
      continue;
    }
    Logger.debug('[Manifest]', `resolveListContainers: descriptor[${i}] - found range`, {
      componentId: descriptor.componentId,
      range,
      atRoot: descriptor.atRoot,
    });
    if (descriptor.atRoot) {
      const element = resolveRangeContainerElement(range);
      if (element) {
        Logger.debug('[Manifest]', `resolveListContainers: descriptor[${i}] - resolved atRoot element`, {
          slotId,
          elementNodeName: element.nodeName,
        });
        containers.set(slotId, element);
      } else {
        Logger.debug('[Manifest]', `resolveListContainers: descriptor[${i}] - atRoot but no container element`);
      }
      continue;
    }
    const node = resolveNodeBySegments(range, descriptor.path);
    Logger.debug('[Manifest]', `resolveListContainers: descriptor[${i}] - resolved node by segments`, {
      nodeType: node?.nodeName,
      isElement: node instanceof Element,
    });
    if (!(node instanceof Element)) {
      Logger.debug('[Manifest]', `resolveListContainers: descriptor[${i}] - node not an Element, skipping`);
      continue;
    }
    containers.set(slotId, node);
    Logger.debug('[Manifest]', `resolveListContainers: descriptor[${i}] - SUCCESS`, {
      slotId,
      elementNodeName: node.nodeName,
    });
  }
  Logger.debug('[Manifest]', 'resolveListContainers END', { containerCount: containers.size });
  return containers;
}

export function applyComponentRanges(
  descriptors: ComponentPathDescriptor[] | undefined,
  options?: { root?: ParentNode | null },
): Map<string, ComponentRange> {
  const ranges = computeComponentRanges(descriptors, options);
  registerComponentRanges(ranges);
  return ranges;
}

function computeComponentRanges(
  descriptors: ComponentPathDescriptor[] | undefined,
  options?: { root?: ParentNode | null },
): Map<string, ComponentRange> {
  const ranges = new Map<string, ComponentRange>();
  if (!Array.isArray(descriptors)) {
    return ranges;
  }
  const root = options?.root ?? document.body ?? document;
  const baseRange: ComponentRange | null = root
    ? { container: root, startIndex: 0, endIndex: root.childNodes.length - 1 }
    : null;
  const pending = new Map<string, ComponentPathDescriptor>();
  descriptors.forEach((descriptor) => {
    if (descriptor && typeof descriptor.componentId === 'string') {
      pending.set(descriptor.componentId, descriptor);
    }
  });
  let progressed = true;
  while (pending.size > 0 && progressed) {
    progressed = false;
    for (const [id, descriptor] of Array.from(pending.entries())) {
      const parentRange = descriptor.parentId ? ranges.get(descriptor.parentId) : baseRange;
      if (!parentRange) {
        continue;
      }
      Logger.debug('[Manifest]', 'computing component range', {
        id,
        parentId: descriptor.parentId,
        parentPath: descriptor.parentPath,
        firstChild: descriptor.firstChild,
        lastChild: descriptor.lastChild,
        parentRangeContainer: parentRange.container.nodeName,
        parentRangeStart: parentRange.startIndex,
        parentRangeEnd: parentRange.endIndex,
      });
      const firstNode = resolveNodeForComponent(parentRange, descriptor.parentPath, descriptor.firstChild);
      const lastNode = resolveNodeForComponent(parentRange, descriptor.parentPath, descriptor.lastChild);
      Logger.debug('[Manifest]', 'resolved boundary nodes', {
        id,
        firstNode: firstNode?.nodeName,
        lastNode: lastNode?.nodeName,
      });
      const container = chooseComponentContainer(firstNode, lastNode, parentRange.container);
      const topLevelFirst = ascendToContainer(firstNode, container) ?? firstNode;
      const topLevelLast = ascendToContainer(lastNode, container) ?? lastNode ?? topLevelFirst;
      let startIndex = topLevelFirst ? getNodeIndex(container, topLevelFirst) : parentRange.startIndex;
      if (startIndex < 0) {
        startIndex = parentRange.startIndex;
      }
      let endIndex = topLevelLast ? getNodeIndex(container, topLevelLast) : startIndex;
      if (endIndex < startIndex) {
        endIndex = startIndex;
      }
      Logger.debug('[Manifest]', 'computed component range', {
        id,
        container: container.nodeName,
        startIndex,
        endIndex,
      });
      ranges.set(id, { container, startIndex, endIndex });
      pending.delete(id);
      progressed = true;
    }
  }
  return ranges;
}

export function resolveNodeInComponent(
  componentId: string,
  path?: PathSegmentDescriptor[] | null,
  overrides?: Map<string, ComponentRange>,
): Node | null {
  const range = overrides?.get(componentId) ?? getComponentRange(componentId);
  if (!range) {
    return null;
  }
  return resolveNodeBySegments(range, path);
}

function resolveNodeBySegments(
  range: ComponentRange,
  segments?: PathSegmentDescriptor[] | null,
): Node | null {
  if (!range || range.endIndex < range.startIndex) {
    Logger.debug('[Manifest]', 'resolveNodeBySegments: invalid range', { range });
    return null;
  }
  const container = range.container;
  if (!container) {
    Logger.debug('[Manifest]', 'resolveNodeBySegments: no container', { range });
    return null;
  }
  Logger.debug('[Manifest]', 'resolveNodeBySegments START', {
    segments,
    rangeInfo: {
      containerNodeName: container.nodeName,
      startIndex: range.startIndex,
      endIndex: range.endIndex,
      childCount: container.childNodes.length,
    },
  });
  if (!Array.isArray(segments) || segments.length === 0) {
    const root = resolveRangeRoot(range);
    Logger.debug('[Manifest]', 'resolveNodeBySegments: using range root', { result: root?.nodeName });
    return root;
  }
  let current: Node | null = null;
  let activeRange: ComponentRange = range;

  for (let i = 0; i < segments.length; i++) {
    const segment = segments[i];
    if (!segment) {
      continue;
    }
    if (segment.kind === 'range') {
      current = resolveRangeChild(activeRange, segment.index);
      Logger.debug('[Manifest]', 'resolveNodeBySegments: range segment', {
        step: i,
        offset: segment.index,
        node: current?.nodeName,
      });
      if (!current) {
        return null;
      }
      // After resolving a range segment, if current is an Element/Fragment,
      // create a new range context for potential subsequent segments
      if (current instanceof Element || current instanceof DocumentFragment) {
        activeRange = {
          container: current,
          startIndex: 0,
          endIndex: current.childNodes.length - 1,
        };
      } else if (i < segments.length - 1) {
        // If we resolved to a non-Element/Fragment and there are more segments,
        // fail now since we can't navigate further from text nodes, etc.
        Logger.debug('[Manifest]', 'resolveNodeBySegments: range resolved to non-Element with remaining segments', {
          step: i,
          current: current?.nodeName,
          remainingSegments: segments.length - 1 - i,
        });
        return null;
      }
      continue;
    }
    if (!current) {
      current = resolveRangeChild(activeRange, segment.index);
      Logger.debug('[Manifest]', 'resolveNodeBySegments: selecting top-level child', {
        step: i,
        offset: segment.index,
        node: current?.nodeName,
      });
      if (!current) {
        return null;
      }
      continue;
    }
    if (!(current instanceof Element || current instanceof DocumentFragment)) {
      Logger.debug('[Manifest]', 'resolveNodeBySegments: current not Element/Fragment', {
        step: i,
        current: current?.nodeName,
      });
      return null;
    }
    const clamped = clampIndex(current, segment.index);
    const next: Node | null = current.childNodes.item(clamped) ?? null;
    Logger.debug('[Manifest]', 'resolveNodeBySegments: navigating dom segment', {
      step: i,
      index: segment.index,
      clamped,
      currentNodeName: current.nodeName,
      nextNodeName: next?.nodeName,
    });
    current = next;
    if (!current) {
      return null;
    }
  }
  Logger.debug('[Manifest]', 'resolveNodeBySegments END', { result: current?.nodeName });
  return current ?? resolveRangeRoot(range);
}

function resolveNodeForComponent(
  range: ComponentRange,
  parentPath?: PathSegmentDescriptor[] | null,
  childPath?: PathSegmentDescriptor[] | null,
): Node | null {
  const combined = combineSegments(parentPath, childPath);
  if (combined) {
    return resolveNodeBySegments(range, combined);
  }
  return resolveNodeBySegments(range, parentPath) ?? resolveNodeBySegments(range, childPath);
}

function combineSegments(
  base?: PathSegmentDescriptor[] | null,
  extra?: PathSegmentDescriptor[] | null,
): PathSegmentDescriptor[] | undefined {
  const result: PathSegmentDescriptor[] = [];
  if (Array.isArray(base) && base.length > 0) {
    result.push(...base);
  }
  if (Array.isArray(extra) && extra.length > 0) {
    result.push(...extra);
  }
  return result.length > 0 ? result : undefined;
}

function chooseComponentContainer(
  firstNode: Node | null,
  lastNode: Node | null,
  fallback: ParentNode | null,
): ParentNode {
  const firstAncestors = collectAncestorParents(firstNode);
  if (!lastNode) {
    return firstAncestors[0] ?? (fallback as ParentNode);
  }
  const lastAncestors = collectAncestorParents(lastNode);
  for (const ancestor of firstAncestors) {
    if (lastAncestors.includes(ancestor)) {
      return ancestor;
    }
  }
  return firstAncestors[0] ?? lastAncestors[0] ?? (fallback as ParentNode);
}

function collectAncestorParents(node: Node | null): ParentNode[] {
  const parents: ParentNode[] = [];
  let current: Node | null = node;
  while (current && current.parentNode) {
    const parent = current.parentNode;
    parents.push(parent);
    current = parent;
  }
  return parents;
}

function ascendToContainer(node: Node | null, container: ParentNode | null): Node | null {
  if (!node || !container) {
    return node;
  }
  let current: Node | null = node;
  while (current && current.parentNode && current.parentNode !== container) {
    current = current.parentNode;
  }
  return current;
}

function clampIndex(container: ParentNode, index: number): number {
  const max = container.childNodes.length - 1;
  if (max < 0) {
    return 0;
  }
  if (index < 0) {
    return 0;
  }
  if (index > max) {
    return max;
  }
  return index;
}

function getNodeIndex(container: ParentNode, node: Node | null): number {
  if (!container || !node) {
    return -1;
  }
  const nodes = container.childNodes;
  for (let i = 0; i < nodes.length; i++) {
    if (nodes.item(i) === node) {
      return i;
    }
  }
  return -1;
}

function resolveRangeRoot(range: ComponentRange): Node | null {
  return resolveRangeChild(range, 0);
}

function resolveRangeChild(range: ComponentRange, offset: number): Node | null {
  const container = range.container;
  if (!container || container.childNodes.length === 0) {
    return null;
  }
  const start = clampIndex(container, range.startIndex);
  const end = Math.max(start, clampIndex(container, range.endIndex));
  const span = end - start;
  const normalizedOffset = Number.isFinite(offset) ? offset : 0;
  const clampedOffset = normalizedOffset <= 0 ? 0 : normalizedOffset >= span ? span : normalizedOffset;
  const index = start + clampedOffset;
  return container.childNodes.item(index) ?? null;
}

function resolveRangeContainerElement(range: ComponentRange): Element | null {
  const container = range.container;
  if (container instanceof Element) {
    return container;
  }
  if (container instanceof Document) {
    return container.body ?? container.documentElement ?? null;
  }
  if (container instanceof DocumentFragment) {
    const first = container.firstElementChild;
    return first ?? null;
  }
  return null;
}

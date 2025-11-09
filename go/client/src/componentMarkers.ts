import type { ComponentMarkerDescriptor } from "./types";

interface ComponentMarkerBounds {
  start: Comment | null;
  end: Comment | null;
}

const componentMarkerIndex = new Map<string, ComponentMarkerBounds>();
const markerComments = new WeakSet<Comment>();

function ensureDocument(root: ParentNode | null | undefined): Document | null {
  if (!root) {
    return typeof document !== "undefined" ? document : null;
  }
  if (root instanceof Document) {
    return root;
  }
  const node = root as unknown as Node;
  if (node && node.ownerDocument) {
    return node.ownerDocument;
  }
  return typeof document !== "undefined" ? document : null;
}

function cloneDescriptors(
  descriptors: Record<string, ComponentMarkerDescriptor>,
): Record<string, ComponentMarkerDescriptor> {
  const out: Record<string, ComponentMarkerDescriptor> = {};
  for (const [id, descriptor] of Object.entries(descriptors)) {
    if (!descriptor) continue;
    const start = Number(descriptor.start);
    const end = Number(descriptor.end);
    if (!Number.isFinite(start) || !Number.isFinite(end)) continue;
    let parentPath: number[] | undefined;
    if (Array.isArray(descriptor.parentPath)) {
      const parsed: number[] = [];
      let valid = true;
      for (const segment of descriptor.parentPath) {
        const index = Number(segment);
        if (!Number.isFinite(index)) {
          valid = false;
          break;
        }
        parsed.push(index);
      }
      if (!valid) continue;
      parentPath = parsed;
    }
    out[id] = parentPath ? { parentPath, start, end } : { start, end };
  }
  return out;
}

function assignMarkers(
  descriptors: Record<string, ComponentMarkerDescriptor>,
  root: ParentNode | null | undefined,
  resetExisting: boolean,
): void {
  const target = root ?? (typeof document !== "undefined" ? document : null);
  if (!target) {
    return;
  }
  const doc = ensureDocument(target);
  if (!doc) {
    return;
  }

  if (resetExisting) {
    componentMarkerIndex.clear();
  }

  for (const [id, descriptor] of Object.entries(descriptors)) {
    if (!descriptor) continue;
    const bounds = createMarkerBounds(descriptor, target, doc);
    if (!bounds) continue;
    registerComponentMarker(id, bounds.start, bounds.end);
  }
}

function isParentNode(value: Node | null): value is ParentNode {
  if (!value) return false;
  return (
    value instanceof Element ||
    value instanceof Document ||
    value instanceof DocumentFragment
  );
}

function markMarkerNode(comment: Comment | null): Comment | null {
  if (!comment) {
    return null;
  }
  comment.data = "";
  markerComments.add(comment);
  return comment;
}

function isMarkerComment(node: Node | null | undefined): node is Comment {
  if (!node || !(node instanceof Comment)) {
    return false;
  }
  return markerComments.has(node);
}

function countRenderableChildren(parent: ParentNode): number {
  let count = 0;
  for (let i = 0; i < parent.childNodes.length; i++) {
    const child = parent.childNodes.item(i);
    if (!child) continue;
    if (isMarkerComment(child)) continue;
    count++;
  }
  return count;
}

export function getRenderableChild(
  parent: ParentNode,
  index: number,
): Node | null {
  if (!Number.isInteger(index) || index < 0) {
    return null;
  }
  let remaining = index;
  for (let i = 0; i < parent.childNodes.length; i++) {
    const child = parent.childNodes.item(i);
    if (!child) continue;
    if (isMarkerComment(child)) {
      continue;
    }
    if (remaining === 0) {
      return child;
    }
    remaining--;
  }
  return null;
}

export function resolveParentNode(
  root: ParentNode,
  path: number[] | undefined,
): ParentNode | null {
  if (!Array.isArray(path) || path.length === 0) {
    return root;
  }
  let current: ParentNode | null = root;
  for (const segment of path) {
    if (!current) return null;
    const index = Number(segment);
    if (!Number.isInteger(index) || index < 0) {
      return null;
    }
    const next = getRenderableChild(current, index);
    if (!isParentNode(next)) {
      return null;
    }
    current = next;
  }
  return current;
}

function createMarkerBounds(
  descriptor: ComponentMarkerDescriptor,
  root: ParentNode,
  doc: Document,
): { start: Comment; end: Comment } | null {
  const parent = resolveParentNode(root, descriptor.parentPath);
  if (!parent) {
    return null;
  }

  const endIndex = Number(descriptor.end);
  if (!Number.isInteger(endIndex) || endIndex < 0) {
    return null;
  }
  const startIndex = Number(descriptor.start);
  if (!Number.isInteger(startIndex) || startIndex < 0) {
    return null;
  }

  const available = countRenderableChildren(parent);
  if (endIndex > available || startIndex > available) {
    return null;
  }

  const end = markMarkerNode(doc.createComment(""));
  if (!end) {
    return null;
  }
  const endRef = getRenderableChild(parent, endIndex);
  parent.insertBefore(end, endRef);

  const start = markMarkerNode(doc.createComment(""));
  if (!start) {
    end.remove();
    return null;
  }
  const startRef = getRenderableChild(parent, startIndex);
  parent.insertBefore(start, startRef ?? end);

  return { start, end };
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
  start: Comment | null,
  end: Comment | null,
): void {
  if (!id) return;
  const entry = componentMarkerIndex.get(id) ?? { start: null, end: null };
  if (start) {
    const marked = markMarkerNode(start);
    if (marked) {
      entry.start = marked;
    }
  }
  if (end) {
    const marked = markMarkerNode(end);
    if (marked) {
      entry.end = marked;
    }
  }
  componentMarkerIndex.set(id, entry);
}

export function getComponentBounds(
  id: string,
): { start: Comment; end: Comment } | null {
  if (!id) return null;
  const entry = componentMarkerIndex.get(id);
  if (entry && entry.start?.isConnected && entry.end?.isConnected) {
    return { start: entry.start, end: entry.end };
  }
  if (entry?.start && entry?.end) {
    return { start: entry.start, end: entry.end };
  }
  return null;
}

import type { ComponentMarkerDescriptor } from "./types";

interface ComponentMarkerBounds {
  start: Comment | null;
  end: Comment | null;
}

const componentMarkerIndex = new Map<string, ComponentMarkerBounds>();

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
    out[id] = { start, end };
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

  const startLookup = new Map<number, string>();
  const endLookup = new Map<number, string>();
  for (const [id, descriptor] of Object.entries(descriptors)) {
    if (!descriptor) continue;
    const start = Number(descriptor.start);
    const end = Number(descriptor.end);
    if (!Number.isFinite(start) || !Number.isFinite(end)) continue;
    startLookup.set(start, id);
    endLookup.set(end, id);
  }

  if (startLookup.size === 0 && endLookup.size === 0) {
    return;
  }

  const walker = doc.createTreeWalker(
    target instanceof Document ? target : (target as unknown as Node),
    NodeFilter.SHOW_COMMENT,
  );

  let index = 0;
  let current = walker.nextNode() as Comment | null;
  while (current) {
    const startId = startLookup.get(index);
    if (startId) {
      registerComponentMarker(startId, current, componentMarkerIndex.get(startId)?.end ?? null);
    }
    const endId = endLookup.get(index);
    if (endId) {
      registerComponentMarker(endId, componentMarkerIndex.get(endId)?.start ?? null, current);
    }
    current.data = "";
    index++;
    current = walker.nextNode() as Comment | null;
  }
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
    entry.start = start;
    start.data = "";
  }
  if (end) {
    entry.end = end;
    end.data = "";
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

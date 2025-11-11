export interface ComponentRange {
  container: ParentNode;
  startIndex: number;
  endIndex: number;
}

const rangeIndex = new Map<string, ComponentRange>();

export function resetComponentRanges(): void {
  rangeIndex.clear();
}

export function registerComponentRange(
  id: string,
  range: ComponentRange | null | undefined,
): ComponentRange | null {
  if (!id || !range || !range.container) return null;
  const { container } = range;
  let { startIndex, endIndex } = range;
  if (!Number.isFinite(startIndex) || !Number.isFinite(endIndex)) {
    return null;
  }
  const normalizedStart = Math.max(0, Math.trunc(startIndex));
  let normalizedEnd = Math.trunc(endIndex);
  if (!Number.isFinite(normalizedEnd)) {
    return null;
  }
  if (normalizedEnd < normalizedStart) {
    normalizedEnd = normalizedStart - 1;
  }
  const normalized: ComponentRange = {
    container,
    startIndex: normalizedStart,
    endIndex: normalizedEnd,
  };
  rangeIndex.set(id, normalized);
  return normalized;
}

export function registerComponentRanges(
  descriptors: Record<string, ComponentRange> | Map<string, ComponentRange> | undefined | null,
): void {
  if (!descriptors) return;
  if (descriptors instanceof Map) {
    for (const [id, range] of Array.from(descriptors.entries())) {
      const normalized = registerComponentRange(id, range);
      if (normalized) {
        descriptors.set(id, normalized);
      } else {
        descriptors.delete(id);
      }
    }
    return;
  }
  for (const [id, range] of Object.entries(descriptors)) {
    const normalized = registerComponentRange(id, range);
    if (normalized) {
      descriptors[id] = normalized;
    } else {
      delete descriptors[id];
    }
  }
}

export function getComponentRange(id: string): ComponentRange | null {
  if (!id) return null;
  return rangeIndex.get(id) ?? null;
}

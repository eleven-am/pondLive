import type {
  SlotMeta,
  SlotPathDescriptor,
  ListPathDescriptor,
  ComponentPathDescriptor,
} from './types';
import { resolveSlotAnchors, resolveListContainers, applyComponentRanges, ComponentRange } from './manifest';
import { Logger } from './logger';

type RowRecord = { nodes: Node[] };
type RowPrimeInfo = { key: string; count: number };

export class DomRegistry {
  private slots = new Map<number, Node>();
  private lists = new Map<
    number,
    { container: Element; rows: Map<string, RowRecord>; order: string[] }
  >();

  reset(): void {
    this.slots.clear();
    this.lists.clear();
    Logger.debug('[DomRegistry]', 'reset');
  }

  prime(componentPaths?: ComponentPathDescriptor[], options?: { root?: ParentNode | null }) {
    return applyComponentRanges(componentPaths, options);
  }

  registerSlotAnchors(descriptors?: SlotPathDescriptor[], overrides?: Map<string, ComponentRange>): void {
    const anchors = resolveSlotAnchors(descriptors, overrides);
    anchors.forEach((node, id) => this.slots.set(id, node));
    Logger.debug('[DomRegistry]', 'registered slot anchors', { count: anchors.size });
  }

  registerListContainers(
    descriptors?: ListPathDescriptor[],
    overrides?: Map<string, ComponentRange>,
    rowMeta?: Map<number, RowPrimeInfo[]>,
  ): void {
    const containers = resolveListContainers(descriptors, overrides);
    containers.forEach((element, id) => {
      const info = rowMeta?.get(id);
      const { rows, order } = collectRows(element, info);
      this.lists.set(id, { container: element, rows, order });
    });
    Logger.debug('[DomRegistry]', 'registered list containers', { count: containers.size });
  }

  registerSlots(slotDescriptors: SlotMeta[]): void {
    if (!Array.isArray(slotDescriptors)) {
      return;
    }
    slotDescriptors.forEach((slot) => {
      if (slot && typeof slot.anchorId === 'number' && !this.slots.has(slot.anchorId)) {
        const placeholder = typeof document !== 'undefined' ? document.createTextNode('') : null;
        if (placeholder) {
          this.slots.set(slot.anchorId, placeholder);
        }
      }
    });
    Logger.debug('[DomRegistry]', 'registered additional slots', { total: this.slots.size });
  }

  getSlot(id: number): Node | undefined {
    return this.slots.get(id);
  }

  registerLists(
    listDescriptors?: ListPathDescriptor[],
    overrides?: Map<string, ComponentRange>,
    rowMeta?: Map<number, RowPrimeInfo[]>,
  ): void {
    this.registerListContainers(listDescriptors, overrides, rowMeta);
  }

  getList(id: number): Element | undefined {
    return this.lists.get(id)?.container;
  }

  getRow(slotId: number, key: string): RowRecord | undefined {
    return this.lists.get(slotId)?.rows.get(key);
  }

  insertRow(slotId: number, key: string, nodes: Node[], index: number): void {
    if (!key || nodes.length === 0) {
      return;
    }
    const list = this.lists.get(slotId);
    if (!list) {
      return;
    }
    list.rows.set(key, { nodes: [...nodes] });
    const clamped = clampIndex(list.order.length, index);
    list.order.splice(clamped, 0, key);
    Logger.debug('[DomRegistry]', 'inserted row', { slotId, key, index: clamped });
  }

  deleteRow(slotId: number, key: string): void {
    const list = this.lists.get(slotId);
    if (!list) {
      return;
    }
    list.rows.delete(key);
    const idx = list.order.indexOf(key);
    if (idx >= 0) {
      list.order.splice(idx, 1);
    }
    Logger.debug('[DomRegistry]', 'deleted row', { slotId, key });
  }

  moveRow(slotId: number, key: string, toIndex: number): void {
    const list = this.lists.get(slotId);
    if (!list) {
      return;
    }
    const current = list.order.indexOf(key);
    if (current === -1) {
      return;
    }
    list.order.splice(current, 1);
    const clamped = clampIndex(list.order.length, toIndex);
    list.order.splice(clamped, 0, key);
    Logger.debug('[DomRegistry]', 'moved row', { slotId, key, to: clamped });
  }

  getRowKeyAt(slotId: number, index: number): string | undefined {
    const list = this.lists.get(slotId);
    if (!list) {
      return undefined;
    }
    const order = list.order ?? [];
    if (index < 0 || index >= order.length) {
      return undefined;
    }
    return order[index];
  }

  getRowFirstNode(slotId: number, key: string): Node | undefined {
    const record = this.getRow(slotId, key);
    return record?.nodes[0];
  }
}

function collectRows(
  container: Element | null,
  meta?: RowPrimeInfo[],
): { rows: Map<string, RowRecord>; order: string[] } {
  const rows = new Map<string, RowRecord>();
  const order: string[] = [];
  if (!container || !Array.isArray(meta) || meta.length === 0) {
    return { rows, order };
  }
  const nodes = Array.from(container.childNodes);
  let cursor = 0;
  for (const info of meta) {
    if (!info || !info.key) {
      continue;
    }
    const count = Math.max(1, Number(info.count) || 1);
    const span: Node[] = [];
    for (let i = 0; i < count && cursor < nodes.length; i += 1, cursor += 1) {
      const node = nodes[cursor];
      if (node) {
        span.push(node);
      }
    }
    if (span.length === 0) {
      continue;
    }
    rows.set(info.key, { nodes: span });
    order.push(info.key);
  }
  return { rows, order };
}

function clampIndex(length: number, index: number): number {
  if (!Number.isFinite(index) || index < 0) {
    return 0;
  }
  if (index > length) {
    return length;
  }
  return index;
}

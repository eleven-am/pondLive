import { DomRegistry } from './dom-registry';
import { registerBindingsForSlot } from './events';
import { applyRouterBindings } from './router-bindings';
import { applyComponentRanges } from './manifest';
import { RefRegistry } from './refs';
import { UploadManager } from './uploads';
import { Logger } from './logger';
import type {
  DiffOp,
  FrameMessage,
  ListChildOp,
  ListInsOp,
  ListDelOp,
  ListMovOp,
  ComponentPathDescriptor,
  SlotPathDescriptor,
  ListPathDescriptor,
  BindingsPayload,
} from './types';

export class Patcher {
  constructor(private dom: DomRegistry, private refs?: RefRegistry, private uploads?: UploadManager) {}

  applyFrame(frame: FrameMessage): void {
    if (!Array.isArray(frame.patch)) {
      return;
    }
    Logger.debug('[Patcher]', 'applying frame patch', { opCount: frame.patch.length });
    this.applyOps(frame.patch);
  }

  applyOps(ops: DiffOp[]): void {
    Logger.debug('[Patcher]', 'applying ops', { count: ops.length });
    for (const op of ops) {
      if (!Array.isArray(op) || op.length === 0) {
        continue;
      }
      const kind = op[0];
      switch (kind) {
        case 'setText':
          this.applySetText(op[1], op[2]);
          break;
        case 'setAttrs':
          this.applySetAttrs(op[1], op[2] || {});
          break;
        case 'list':
          this.applyList(op[1], op.slice(2) as ListChildOp[]);
          break;
      }
    }
  }

  private applySetText(slotId: number, value: string): void {
    const node = this.dom.getSlot(slotId);
    if (!node) {
      Logger.debug('[Patcher]', 'setText skipped (missing slot)', { slotId });
      return;
    }
    if (node instanceof Text) {
      node.textContent = value ?? '';
      Logger.debug('[Patcher]', 'setText applied', { slotId, nodeType: 'Text' });
      return;
    }
    if (node instanceof Element) {
      node.textContent = value ?? '';
      Logger.debug('[Patcher]', 'setText applied', { slotId, nodeType: node.tagName });
    }
  }

  private applySetAttrs(slotId: number, attrs: Record<string, string>): void {
    const node = this.dom.getSlot(slotId);
    if (!(node instanceof Element)) {
      Logger.debug('[Patcher]', 'setAttrs skipped (non-element)', { slotId });
      return;
    }
    Object.entries(attrs ?? {}).forEach(([key, value]) => {
      if (value === null || value === undefined || value === '') {
        node.removeAttribute(key);
      } else {
        node.setAttribute(key, value);
      }
    });
    Logger.debug('[Patcher]', 'setAttrs applied', { slotId, keys: Object.keys(attrs ?? {}) });
  }

  private applyList(slotId: number, childOps: ListChildOp[]): void {
    const container = this.dom.getList(slotId);
    if (!(container instanceof Element)) {
      Logger.debug('[Patcher]', 'list op skipped (no container)', { slotId });
      return;
    }
    Logger.debug('[Patcher]', 'list patch', { slotId, opCount: childOps.length });
    childOps.forEach((op) => {
      if (!Array.isArray(op)) {
        return;
      }
      switch (op[0]) {
        case 'del':
          this.applyListDelete(slotId, container, op as ListDelOp);
          break;
        case 'ins':
          this.applyListInsert(slotId, container, op as ListInsOp);
          break;
        case 'mov':
          this.applyListMove(slotId, container, op as ListMovOp);
          break;
      }
    });
  }

  private applyListDelete(slotId: number, container: Element, op: ListDelOp): void {
    const key = op[1];
    const record = this.dom.getRow(slotId, key);
    if (!record) {
      Logger.debug('[Patcher]', 'list delete skipped (missing row)', { slotId, key });
      return;
    }
    record.nodes.forEach((node) => {
      if (node.parentNode === container) {
        container.removeChild(node);
      }
    });
    this.dom.deleteRow(slotId, key);
    Logger.debug('[Patcher]', 'list delete applied', { slotId, key });
  }

  private applyListInsert(slotId: number, container: Element, op: ListInsOp): void {
    const [_, index, payload] = op;
    const html = payload?.html ?? '';
    if (!html) {
      Logger.debug('[Patcher]', 'list insert skipped (no html)', { slotId, index });
      return;
    }
    const template = document.createElement('template');
    template.innerHTML = html;
    const fragment = template.content;
    const nodes = Array.from(fragment.childNodes);
    if (nodes.length === 0) {
      Logger.debug('[Patcher]', 'list insert skipped (no nodes)', { slotId, index });
      return;
    }
    const beforeKey = this.dom.getRowKeyAt(slotId, index);
    const before = beforeKey ? this.dom.getRowFirstNode(slotId, beforeKey) : null;
    container.insertBefore(fragment, before ?? null);
    this.dom.insertRow(slotId, payload?.key ?? '', nodes, index);
    const root = nodes.find((node): node is Element => node instanceof Element) ?? null;
    if (root) {
      this.registerRowMetadata(
        root,
        payload?.componentPaths ?? [],
        payload?.slotPaths,
        payload?.listPaths,
        payload?.bindings,
      );
    }
    Logger.debug('[Patcher]', 'list insert applied', {
      slotId,
      index,
      key: payload?.key,
      nodeCount: nodes.length,
    });
  }

  private applyListMove(slotId: number, container: Element, op: ListMovOp): void {
    const from = op[1];
    const to = op[2];
    if (from === to) {
      return;
    }
    const currentKey = this.dom.getRowKeyAt(slotId, from);
    if (!currentKey) {
      Logger.debug('[Patcher]', 'list move skipped (missing source key)', { slotId, from, to });
      return;
    }
    const record = this.dom.getRow(slotId, currentKey);
    if (!record) {
      Logger.debug('[Patcher]', 'list move skipped (missing record)', { slotId, from, to });
      return;
    }
    const targetKey = this.dom.getRowKeyAt(slotId, to);
    const beforeNode = targetKey ? this.dom.getRowFirstNode(slotId, targetKey) : null;
    record.nodes.forEach((node) => {
      if (node.parentNode !== container) {
        return;
      }
      container.insertBefore(node, beforeNode ?? null);
    });
    this.dom.moveRow(slotId, currentKey, to);
    Logger.debug('[Patcher]', 'list move applied', { slotId, from, to, key: currentKey });
  }
  private registerRowMetadata(
    root: Element,
    componentPaths?: ComponentPathDescriptor[] | null,
    slotPaths?: SlotPathDescriptor[] | null,
    listPaths?: ListPathDescriptor[] | null,
    bindings?: BindingsPayload | null,
  ): void {
    const overrides = applyComponentRanges(componentPaths ?? [], { root });
    this.dom.registerSlotAnchors(slotPaths ?? undefined, overrides);
    this.dom.registerLists(listPaths ?? undefined, overrides);
    if (bindings?.slots) {
      Object.entries(bindings.slots).forEach(([slot, specs]) => {
        registerBindingsForSlot(Number(slot), specs);
      });
    }
    if (bindings?.router) {
      applyRouterBindings(bindings.router, overrides);
    }
    if (bindings?.refs) {
      this.refs?.registerBindings(bindings.refs, overrides);
    }
    if (bindings?.uploads) {
      this.uploads?.registerBindings(bindings.uploads, overrides, { replace: false });
    }
    Logger.debug('[Patcher]', 'row metadata registered', {
      slots: bindings?.slots ? Object.keys(bindings.slots).length : 0,
      routers: bindings?.router?.length ?? 0,
      refs: bindings?.refs?.length ?? 0,
      uploads: bindings?.uploads?.length ?? 0,
    });
  }
}

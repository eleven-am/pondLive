import { describe, it, expect, beforeEach, vi } from 'vitest';
import { DomRegistry } from '../src/dom-registry';
import { Patcher } from '../src/patcher';
import type { ListInsOp, FrameMessage, ComponentPathDescriptor } from '../src/types';
import { getSlotBindings, registerSlotTable } from '../src/events';
import { RefRegistry } from '../src/refs';

function createFrame(patch: any[]): FrameMessage {
  return {
    t: 'frame',
    sid: 'sid',
    ver: 1,
    patch: patch as any,
    handlers: {},
    refs: {},
    bindings: {},
  };
}

describe('Patcher', () => {
  beforeEach(() => {
    registerSlotTable(undefined);
  });

  it('applies setText operations to slot nodes', () => {
    const registry = new DomRegistry();
    const slot = document.createElement('span');
    registry.registerSlotAnchors([{ slot: 1, componentId: '', textChildIndex: 0 }], undefined);
    registry['slots'].set(1, slot);
    const patcher = new Patcher(registry);
    patcher.applyFrame(createFrame([
      ['setText', 1, 'hello'],
    ]));
    expect(slot.textContent).toBe('hello');
  });

  it('inserts list rows and registers nested metadata', () => {
    const registry = new DomRegistry();
    const list = document.createElement('div');
    registry.registerLists([{ slot: 5, componentId: '', path: [] }], undefined);
    registry['lists'].set(5, { container: list, rows: new Map(), order: [] } as any);
    const patcher = new Patcher(registry);
    const insert: ListInsOp = ['ins', 0, {
      key: 'row-1',
      html: '<div class="row"><span>Row</span></div>',
      componentPaths: [],
      bindings: { slots: { 9: [{ event: 'click', handler: 'h5' }] } },
    }];
    patcher.applyFrame(createFrame([
      ['list', 5, insert],
    ]));
    expect(list.innerHTML).toContain('Row');
    const rowRecord = registry.getRow(5, 'row-1');
    expect(rowRecord?.nodes[0]).toBeInstanceOf(Element);
    expect(getSlotBindings(9)).toMatchObject([{ event: 'click', handler: 'h5' }]);
  });

  it('registers ref bindings for inserted rows', () => {
    const registry = new DomRegistry();
    const runtimeStub = { sendEvent: vi.fn() } as any;
    const refs = new RefRegistry(runtimeStub);
    refs.apply({
      add: {
        'ref:btn': {
          tag: 'button',
          events: { click: { props: [] } },
        },
      },
    });
    const list = document.createElement('div');
    registry.registerLists([{ slot: 7, componentId: '', path: [] }], undefined);
    registry['lists'].set(7, { container: list, rows: new Map(), order: [] } as any);
    const patcher = new Patcher(registry, refs);
    const componentPaths: ComponentPathDescriptor[] = [
      {
        componentId: 'cmp-row',
        firstChild: [{ kind: 'dom', index: 0 }],
        lastChild: [{ kind: 'dom', index: 0 }],
      },
    ];
    const insert: ListInsOp = ['ins', 0, {
      key: 'alpha',
      html: '<div class="row"><button type="button">Click</button></div>',
      componentPaths,
      bindings: {
        refs: [
          {
            componentId: 'cmp-row',
            refId: 'ref:btn',
            path: [{ kind: 'dom', index: 0 }],
          },
        ],
      },
    }];
    patcher.applyOps([
      ['list', 7, insert],
    ]);
    const button = list.querySelector('button');
    expect(button).toBeInstanceOf(HTMLButtonElement);
    expect(refs.get('ref:btn')).toBe(button);
  });

  it('handles fragment rows with multiple root nodes', () => {
    const registry = new DomRegistry();
    const list = document.createElement('div');
    registry.registerLists([{ slot: 11, componentId: '', path: [] }], undefined);
    registry['lists'].set(11, { container: list, rows: new Map(), order: [] } as any);
    const patcher = new Patcher(registry);
    const insert: ListInsOp = ['ins', 0, {
      key: 'multi',
      html: '<p>First</p><p>Second</p>',
      componentPaths: [],
    }];
    patcher.applyOps([
      ['list', 11, insert],
    ]);
    expect(list.querySelectorAll('p')).toHaveLength(2);
    const record = registry.getRow(11, 'multi');
    expect(record?.nodes.length).toBe(2);
    patcher.applyOps([
      ['list', 11, ['del', 'multi']],
    ]);
    expect(list.childNodes.length).toBe(0);
  });
});

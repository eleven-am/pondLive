import { describe, it, expect, beforeEach, vi } from 'vitest';
import { Patcher } from '../src/patcher';
import { EventManager } from '../src/events';
import { Router } from '../src/router';
import { RefRegistry } from '../src/refs';
import { ClientNode, Patch } from '../src/types';
import { hydrate } from '../src/vdom';

describe('Patcher', () => {
  let root: ClientNode;
  let container: HTMLElement;
  let events: any;
  let router: any;
  let uploads: any;
  let refs: Map<string, ClientNode>;
  let patcher: Patcher;

  beforeEach(() => {
    container = document.createElement('div');
    container.innerHTML = '<div id="target">Original</div>';

    const json: StructuredNode = {
      tag: 'div',
      children: [
        {
          tag: 'div',
          attrs: { id: ['target'] },
          children: [{ text: 'Original' }]
        }
      ]
    };

    // Mocks
    events = { attach: vi.fn(), detach: vi.fn() } as any;
    router = { attach: vi.fn(), detach: vi.fn() } as any; // Updated router mock
    uploads = { bind: vi.fn(), unbind: vi.fn() } as any; // New uploads mock
    refs = new Map<string, ClientNode>() as any; // Updated refs to be a Map

    root = hydrate(json, container, refs);
    patcher = new Patcher(root, events, router, uploads, refs); // Updated Patcher constructor call
  });

  it('setText', () => {
    const patch: Patch = {
      op: 'setText',
      path: [0, 0], // div -> text
      value: 'Updated'
    };
    patcher.apply(patch);
    expect(container.textContent).toBe('Updated');
    expect(root.children![0].children![0].text).toBe('Updated');
  });

  it('setAttr', () => {
    const patch: Patch = {
      op: 'setAttr',
      path: [0], // div
      value: { class: ['new-class', 'another'] }
    };
    patcher.apply(patch);
    const div = container.firstElementChild!;
    expect(div.getAttribute('class')).toBe('new-class another');
  });

  it('delAttr', () => {
    const div = container.firstElementChild!;
    div.setAttribute('data-test', 'value');

    const patch: Patch = {
      op: 'delAttr',
      path: [0],
      name: 'data-test'
    };
    patcher.apply(patch);
    expect(div.hasAttribute('data-test')).toBe(false);
  });

  it('addChild', () => {
    const patch: Patch = {
      op: 'addChild',
      path: [0], // Add to div
      index: 1,
      value: { tag: 'span', children: [{ text: 'New' }] }
    };
    patcher.apply(patch);

    const div = container.firstElementChild!;
    expect(div.children).toHaveLength(1); // Original text + new span? No, text is node, span is element.
    // Original: <div>"Original"</div>
    // Added: <span>"New"</span> at index 1

    expect(div.childNodes).toHaveLength(2);
    expect(div.childNodes[1].nodeName).toBe('SPAN');
    expect(div.childNodes[1].textContent).toBe('New');

    expect(events.attach).toHaveBeenCalled();
    expect(router.attach).toHaveBeenCalled();
  });

  it('delChild', () => {
    const patch: Patch = {
      op: 'delChild',
      path: [0], // div
      index: 0 // delete text node "Original"
    };
    patcher.apply(patch);

    const div = container.firstElementChild!;
    expect(div.childNodes).toHaveLength(0);
    expect(div.textContent).toBe('');

    expect(events.detach).toHaveBeenCalled();
    expect(router.detach).toHaveBeenCalled();
  });

  it('replaceNode', () => {
    const patch: Patch = {
      op: 'replaceNode',
      path: [0], // Replace the div
      value: { tag: 'p', children: [{ text: 'Replaced' }] }
    };
    patcher.apply(patch);

    expect(container.firstElementChild!.tagName).toBe('P');
    expect(container.textContent).toBe('Replaced');

    expect(events.detach).toHaveBeenCalled();
    expect(events.attach).toHaveBeenCalled();
  });

  it('setStyle', () => {
    const patch: Patch = {
      op: 'setStyle',
      path: [0],
      value: { color: 'red', 'font-size': '12px' }
    };
    patcher.apply(patch);
    const div = container.firstElementChild as HTMLElement;
    expect(div.style.color).toBe('red');
    expect(div.style.fontSize).toBe('12px');
  });

  it('setStyleDecl', () => {
    // Setup style element
    const styleEl = document.createElement('style');
    styleEl.textContent = '.test { color: blue; }';
    container.appendChild(styleEl);
    // We need to manually attach it to a ClientNode
    const styleNode: ClientNode = { tag: 'style', el: styleEl };
    root.children!.push(styleNode);

    // In JSDOM, style sheets might need some help or might not parse fully without layout.
    // But let's try.
    // Note: JSDOM support for CSSStyleSheet is limited.
    // We might need to mock the sheet if JSDOM doesn't parse it.
    if (!styleEl.sheet) {
      // Mock sheet
      const sheet = {
        cssRules: [
          {
            selectorText: '.test',
            style: { setProperty: vi.fn(), removeProperty: vi.fn() }
          }
        ]
      } as any;
      Object.defineProperty(styleEl, 'sheet', { value: sheet });
    }

    const patch: Patch = {
      op: 'setStyleDecl',
      path: [1], // The style node we added
      selector: '.test',
      name: 'color',
      value: 'green'
    };
    patcher.apply(patch);

    const rule = styleEl.sheet!.cssRules[0] as CSSStyleRule;
    // If mocked
    if (vi.isMockFunction(rule.style.setProperty)) {
      expect(rule.style.setProperty).toHaveBeenCalledWith('color', 'green');
    } else {
      expect(rule.style.color).toBe('green');
    }
  });

  it('setRef', () => {
    const patch: Patch = {
      op: 'setRef',
      path: [0],
      value: 'my-ref'
    };
    patcher.apply(patch);
    expect(refs.get('my-ref')).toBe(root.children![0]);
  });
});

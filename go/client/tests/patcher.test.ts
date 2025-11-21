import { describe, it, expect, beforeEach, vi } from 'vitest';
import { Patcher } from '../src/patcher';
import { EventManager } from '../src/events';
import { Router } from '../src/router';
import { ClientNode, Patch, StructuredNode, Stylesheet } from '../src/types';
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

    events = { attach: vi.fn(), detach: vi.fn() } as any;
    router = { attach: vi.fn(), detach: vi.fn() } as any;
    uploads = { bind: vi.fn(), unbind: vi.fn() } as any;
    refs = new Map<string, ClientNode>() as any;

    root = hydrate(json, container, refs);
    patcher = new Patcher(root, events, router, uploads, refs);
  });

  it('setText', () => {
    const patch: Patch = {
      op: 'setText',
      path: [0, 0],
      value: 'Updated'
    };
    patcher.apply(patch);
    expect(container.textContent).toBe('Updated');
    expect(root.children![0].children![0].text).toBe('Updated');
  });

  it('setAttr', () => {
    const patch: Patch = {
      op: 'setAttr',
      path: [0],
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
      path: [0],
      index: 1,
      value: { tag: 'span', children: [{ text: 'New' }] }
    };
    patcher.apply(patch);

    const div = container.firstElementChild!;
    expect(div.children).toHaveLength(1);
    expect(div.childNodes).toHaveLength(2);
    expect(div.childNodes[1].nodeName).toBe('SPAN');
    expect(div.childNodes[1].textContent).toBe('New');

    expect(events.attach).toHaveBeenCalled();
    expect(router.attach).toHaveBeenCalled();
  });

  it('delChild', () => {
    const patch: Patch = {
      op: 'delChild',
      path: [0],
      index: 0
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
      path: [0],
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
    const styleEl = document.createElement('style');
    styleEl.textContent = '.test { color: blue; }';
    container.appendChild(styleEl);
    const styleNode: ClientNode = { tag: 'style', el: styleEl };
    root.children!.push(styleNode);

    if (!styleEl.sheet) {
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
      path: [1],
      selector: '.test',
      name: 'color',
      value: 'green'
    };
    patcher.apply(patch);

    const rule = styleEl.sheet!.cssRules[0] as CSSStyleRule;
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

  it('reorders keyed children using moveChild with key', () => {
    container.innerHTML = '<ul><li>A</li><li>B</li></ul>';
    const json: StructuredNode = {
      tag: 'ul',
      children: [
        { tag: 'li', key: 'a', children: [{ text: 'A' }] },
        { tag: 'li', key: 'b', children: [{ text: 'B' }] }
      ]
    };
    root = hydrate(json, container.firstChild as Node, refs);
    patcher = new Patcher(root, events, router, uploads, refs);

    const movePatch: Patch = {
      op: 'moveChild',
      path: [],
      value: { key: 'b', oldIdx: 1, newIdx: 0 }
    };

    patcher.apply(movePatch);

    const ul = container.firstElementChild as HTMLUListElement;
    expect(ul.children[0].textContent).toBe('B');
    expect(ul.children[1].textContent).toBe('A');
  });

  it('renders style element with stylesheet rules', () => {
    const styleJson: StructuredNode = {
      tag: 'style',
      stylesheet: {
        rules: [
          { selector: '.card', props: { color: 'red', background: '#fff' } },
          { selector: '.btn', props: { padding: '10px' } }
        ]
      }
    };

    const patch: Patch = {
      op: 'addChild',
      path: [],
      index: 0,
      value: styleJson
    };

    patcher.apply(patch);

    const styleEl = root.children![0].el as HTMLStyleElement;
    expect(styleEl.tagName).toBe('STYLE');
    expect(styleEl.textContent).toContain('.card');
    expect(styleEl.textContent).toContain('color: red;');
    expect(styleEl.textContent).toContain('background: #fff;');
    expect(styleEl.textContent).toContain('.btn');
    expect(styleEl.textContent).toContain('padding: 10px;');
  });

  it('renders style element with media blocks', () => {
    const styleJson: StructuredNode = {
      tag: 'style',
      stylesheet: {
        rules: [
          { selector: '.card', props: { color: 'blue' } }
        ],
        mediaBlocks: [
          {
            query: '(max-width: 768px)',
            rules: [
              { selector: '.card', props: { 'font-size': '14px' } }
            ]
          }
        ]
      }
    };

    const patch: Patch = {
      op: 'addChild',
      path: [],
      index: 0,
      value: styleJson
    };

    patcher.apply(patch);

    const styleEl = root.children![0].el as HTMLStyleElement;
    expect(styleEl.textContent).toContain('.card { color: blue; }');
    expect(styleEl.textContent).toContain('@media (max-width: 768px)');
    expect(styleEl.textContent).toContain('.card { font-size: 14px; }');
  });

  it('renders style element with multiple media blocks', () => {
    const styleJson: StructuredNode = {
      tag: 'style',
      stylesheet: {
        rules: [
          { selector: '.container', props: { width: '100%' } }
        ],
        mediaBlocks: [
          {
            query: '(min-width: 768px)',
            rules: [
              { selector: '.container', props: { width: '750px' } }
            ]
          },
          {
            query: '(min-width: 1024px)',
            rules: [
              { selector: '.container', props: { width: '960px' } },
              { selector: '.sidebar', props: { display: 'block' } }
            ]
          }
        ]
      }
    };

    const patch: Patch = {
      op: 'addChild',
      path: [],
      index: 0,
      value: styleJson
    };

    patcher.apply(patch);

    const styleEl = root.children![0].el as HTMLStyleElement;
    const content = styleEl.textContent!;

    expect(content).toContain('.container { width: 100%; }');
    expect(content).toContain('@media (min-width: 768px)');
    expect(content).toContain('@media (min-width: 1024px)');
    expect(content).toContain('.sidebar { display: block; }');
  });
});

import { describe, it, beforeEach, expect } from 'vitest';
import {
  resolveSlotAnchors,
  resolveListContainers,
  registerComponentRanges,
  applyComponentRanges,
} from '../src/manifest';
import type { ComponentRange } from '../src/manifest';

describe('manifest resolution', () => {
  let host: HTMLElement;

  beforeEach(() => {
    host = document.createElement('div');
  });

  it('resolves slot anchors relative to component ranges', () => {
    const section = document.createElement('section');
    const paragraph = document.createElement('p');
    paragraph.textContent = 'Intro';
    const span = document.createElement('span');
    span.id = 'target';
    const emphasis = document.createElement('em');
    emphasis.textContent = 'Nested';
    span.append(emphasis);
    section.append(paragraph, span);
    const article = document.createElement('article');
    const button = document.createElement('button');
    button.textContent = 'Action';
    article.append(button);
    host.replaceChildren(section, article);
    const range: ComponentRange = {
      container: host,
      startIndex: 0,
      endIndex: host.childNodes.length - 1,
    };
    registerComponentRanges(new Map([['cmp:root', range]]));
    const anchors = resolveSlotAnchors(
      [
        {
          slot: 1,
          componentId: 'cmp:root',
          path: [
            { kind: 'dom', index: 0 },
            { kind: 'dom', index: 1 },
          ],
          textChildIndex: 0,
        },
      ],
      undefined,
    );
    expect(anchors.get(1)).toBe((host.querySelector('#target') as Element).childNodes.item(0));
  });

  it('falls back to the range root when slot path is omitted', () => {
    const first = document.createElement('div');
    first.id = 'first-child';
    const second = document.createElement('div');
    host.replaceChildren(first, second);
    const range: ComponentRange = {
      container: host,
      startIndex: 0,
      endIndex: host.childNodes.length - 1,
    };
    registerComponentRanges(new Map([['cmp:root-fallback', range]]));
    const anchors = resolveSlotAnchors(
      [
        {
          slot: 3,
          componentId: 'cmp:root-fallback',
          path: [],
        },
      ],
      undefined,
    );
    expect(anchors.get(3)).toBe(first);
  });

  it('resolves slot anchors in nested containers with dom-only paths', () => {
    const section = document.createElement('section');
    const first = document.createElement('div');
    first.textContent = 'alpha';
    const second = document.createElement('div');
    const button = document.createElement('button');
    button.id = 'range-target';
    second.append(button);
    section.append(first, second);
    host.replaceChildren(section);
    const range: ComponentRange = {
      container: host,
      startIndex: 0,
      endIndex: host.childNodes.length - 1,
    };
    registerComponentRanges(new Map([['cmp:range', range]]));
    const anchors = resolveSlotAnchors(
      [
        {
          slot: 9,
          componentId: 'cmp:range',
          path: [
            { kind: 'dom', index: 0 },
            { kind: 'dom', index: 1 },
            { kind: 'dom', index: 0 },
          ],
        },
      ],
      undefined,
    );
    expect(anchors.get(9)).toBe(button);
  });

  it('resolves list containers using top-level offsets', () => {
    const first = document.createElement('ul');
    first.id = 'first';
    const second = document.createElement('ul');
    second.id = 'second';
    const third = document.createElement('ul');
    third.id = 'third';
    host.replaceChildren(first, second, third);
    const range: ComponentRange = {
      container: host,
      startIndex: 0,
      endIndex: host.childNodes.length - 1,
    };
    registerComponentRanges(new Map([['cmp:list', range]]));
    const containers = resolveListContainers(
      [
        {
          slot: 5,
          componentId: 'cmp:list',
          path: [{ kind: 'dom', index: 1 }],
        },
      ],
      undefined,
    );
    expect(containers.get(5)).toBe(host.querySelector('#second'));
  });

  it('resolves list containers nested multiple levels deep', () => {
    const section = document.createElement('section');
    const wrapper = document.createElement('div');
    const list = document.createElement('ul');
    list.id = 'deep-list';
    wrapper.append(document.createElement('p'), list);
    section.append(wrapper);
    host.replaceChildren(section);
    const range: ComponentRange = {
      container: host,
      startIndex: 0,
      endIndex: host.childNodes.length - 1,
    };
    registerComponentRanges(new Map([['cmp:deep-list', range]]));
    const containers = resolveListContainers(
      [
        {
          slot: 7,
          componentId: 'cmp:deep-list',
          path: [
            { kind: 'dom', index: 0 },
            { kind: 'dom', index: 0 },
            { kind: 'dom', index: 1 },
          ],
        },
      ],
      undefined,
    );
    expect(containers.get(7)).toBe(list);
  });

  it('resolves root list containers when descriptor marks atRoot', () => {
    host.innerHTML = `
      <div id="root"></div>
    `;
    const root = host.querySelector('#root') as HTMLElement;
    const range: ComponentRange = {
      container: root,
      startIndex: 0,
      endIndex: -1,
    };
    registerComponentRanges(new Map([['cmp:root', range]]));
    const containers = resolveListContainers(
      [{ slot: 2, componentId: 'cmp:root', atRoot: true }],
      undefined,
    );
    expect(containers.get(2)).toBe(root);
  });

  it('computes component ranges when last child paths are nested', () => {
    const heading = document.createElement('h2');
    heading.textContent = 'Heading';
    const wrapper = document.createElement('div');
    const inner = document.createElement('div');
    const nested = document.createElement('div');
    const button = document.createElement('button');
    nested.append(button);
    inner.append(document.createTextNode('prefix'), nested);
    wrapper.append(inner);
    host.replaceChildren(heading, wrapper);
    const ranges = applyComponentRanges(
      [
        {
          componentId: 'cmp:home',
          parentId: '',
          parentPath: [],
          firstChild: [{ kind: 'dom', index: 0 }],
          lastChild: [
            { kind: 'dom', index: 1 },
            { kind: 'dom', index: 0 },
            { kind: 'dom', index: 1 },
            { kind: 'dom', index: 0 },
          ],
        },
      ],
      { root: host },
    );
    const cmpRange = ranges.get('cmp:home');
    expect(cmpRange?.container).toBe(host);
    expect(cmpRange?.startIndex).toBe(0);
    expect(cmpRange?.endIndex).toBe(1);
  });

  it('computes component ranges using parentPath to select nested containers', () => {
    const wrapper = document.createElement('div');
    const target = document.createElement('section');
    const spanA = document.createElement('span');
    const spanB = document.createElement('span');
    target.append(spanA, spanB);
    wrapper.append(target, document.createElement('footer'));
    host.replaceChildren(document.createElement('header'), wrapper);
    const ranges = applyComponentRanges(
      [
        {
          componentId: 'cmp:nested-parent',
          parentId: '',
          parentPath: [
            { kind: 'dom', index: 1 },
            { kind: 'dom', index: 0 },
          ],
          firstChild: [{ kind: 'dom', index: 0 }],
          lastChild: [{ kind: 'dom', index: 1 }],
        },
      ],
      { root: host },
    );
    const cmpRange = ranges.get('cmp:nested-parent');
    expect(cmpRange?.container).toBe(target);
    expect(cmpRange?.startIndex).toBe(0);
    expect(cmpRange?.endIndex).toBe(1);
  });

});

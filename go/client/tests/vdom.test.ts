import { describe, it, expect, beforeEach } from 'vitest';
import { hydrate } from '../src/vdom';
import { ClientNode, StructuredNode } from '../src/types';

describe('hydrate', () => {
    let refs: Map<string, ClientNode>;

    beforeEach(() => {
        refs = new Map();
        document.body.innerHTML = '';
    });

    it('hydrates a simple text node', () => {
        document.body.innerHTML = 'Hello';
        const json: StructuredNode = { text: 'Hello' };

        // hydrate expects a node, so we pass the text node
        const textNode = document.body.firstChild!;
        const clientNode = hydrate(json, textNode, refs);

        expect(clientNode.el).toBe(textNode);
        expect(clientNode.text).toBe('Hello');
        expect((textNode as any).__pondNode).toBe(clientNode);
    });

    it('hydrates an element with attributes', () => {
        document.body.innerHTML = '<div id="test" class="foo"></div>';
        const json: StructuredNode = {
            tag: 'div',
            attrs: { id: ['test'], class: ['foo'] }
        };

        const div = document.body.firstChild!;
        const clientNode = hydrate(json, div, refs);

        expect(clientNode.el).toBe(div);
        expect(clientNode.tag).toBe('div');
        expect(clientNode.attrs).toEqual({ id: ['test'], class: ['foo'] });
    });

    it('hydrates nested children', () => {
        document.body.innerHTML = '<div><span>Child</span></div>';
        const json: StructuredNode = {
            tag: 'div',
            children: [
                {
                    tag: 'span',
                    children: [{ text: 'Child' }]
                }
            ]
        };

        const div = document.body.firstChild!;
        const clientNode = hydrate(json, div, refs);

        expect(clientNode.children).toHaveLength(1);
        expect(clientNode.children![0].tag).toBe('span');
        expect(clientNode.children![0].el).toBe(div.firstChild);

        const spanNode = clientNode.children![0];
        expect(spanNode.children).toHaveLength(1);
        expect(spanNode.children![0].text).toBe('Child');
        expect(spanNode.children![0].el).toBe(div.firstChild!.firstChild);
    });

    it('registers refs during hydration', () => {
        document.body.innerHTML = '<button>Click</button>';
        const json: StructuredNode = {
            tag: 'button',
            refId: 'btn-1',
            children: [{ text: 'Click' }]
        };

        const btn = document.body.firstChild!;
        const clientNode = hydrate(json, btn, refs);

        expect(refs.get('btn-1')).toBe(clientNode);
        expect(clientNode.el).toBe(btn);
    });

    it('handles virtual component wrappers (flattening)', () => {
        // Server sends: Component(div, span) -> flattened in DOM as <div>...</div><span>...</span>
        // But wait, hydrate takes a single root DOM node.
        // If the component is a fragment, it doesn't map to a single DOM node.
        // However, our hydrate function currently takes (json, dom).
        // This implies 1:1 mapping at the root.
        // The component wrapper logic in vdom.ts handles children that are wrappers.

        document.body.innerHTML = '<div>A</div><div>B</div>';

        // Parent container
        const container = document.createElement('div');
        container.innerHTML = '<div>A</div><div>B</div>';

        const json: StructuredNode = {
            tag: 'div',
            children: [
                {
                    // Virtual wrapper
                    componentId: 'comp-1',
                    children: [
                        { tag: 'div', children: [{ text: 'A' }] },
                        { tag: 'div', children: [{ text: 'B' }] }
                    ]
                }
            ]
        };

        const clientNode = hydrate(json, container, refs);

        // The wrapper should be in clientNode.children
        expect(clientNode.children).toHaveLength(1);
        const wrapper = clientNode.children![0];
        expect(wrapper.componentId).toBe('comp-1');
        expect(wrapper.el).toBeNull();

        // The wrapper should have 2 children, mapping to the DOM nodes
        expect(wrapper.children).toHaveLength(2);
        expect(wrapper.children![0].tag).toBe('div');
        expect(wrapper.children![0].children![0].text).toBe('A');
        expect(wrapper.children![1].tag).toBe('div');
        expect(wrapper.children![1].children![0].text).toBe('B');
    });
});

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { DOMActionExecutor } from '../src/dom_actions';
import { ClientNode, DOMActionEffect } from '../src/types';

describe('DOMActionExecutor', () => {
    let refs: Map<string, ClientNode>;
    let executor: DOMActionExecutor;
    let channel: any;

    beforeEach(() => {
        refs = new Map();
        executor = new DOMActionExecutor(refs);
    });

    it('executes dom.call', () => {
        const el = document.createElement('input');
        el.focus = vi.fn();
        const node: ClientNode = { tag: 'input', el: el };
        refs.set('my-input', node);

        const effect: DOMActionEffect = {
            type: 'dom',
            kind: 'dom.call',
            ref: 'my-input',
            method: 'focus'
        };

        executor.execute([effect]);
        expect(el.focus).toHaveBeenCalled();
    });

    it('executes dom.set', () => {
        const el = document.createElement('input');
        const node: ClientNode = { tag: 'input', el: el };
        refs.set('my-input', node);

        const effect: DOMActionEffect = {
            type: 'dom',
            kind: 'dom.set',
            ref: 'my-input',
            prop: 'value',
            value: 'hello'
        };

        executor.execute([effect]);
        expect(el.value).toBe('hello');
    });

    it('executes dom.class', () => {
        const el = document.createElement('div');
        const node: ClientNode = { tag: 'div', el: el };
        refs.set('my-div', node);

        const effect: DOMActionEffect = {
            type: 'dom',
            kind: 'dom.class',
            ref: 'my-div',
            class: 'active',
            on: true
        };

        executor.execute([effect]);
        expect(el.classList.contains('active')).toBe(true);

        effect.on = false;
        executor.execute([effect]);
        expect(el.classList.contains('active')).toBe(false);
    });
});

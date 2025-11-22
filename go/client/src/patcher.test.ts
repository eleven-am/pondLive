import { describe, it, expect, beforeEach, vi } from 'vitest';
import { Patcher } from './patcher';
import { Patch, PatcherCallbacks } from './types';

describe('Patcher', () => {
    let root: HTMLElement;
    let callbacks: PatcherCallbacks;
    let patcher: Patcher;

    beforeEach(() => {
        document.body.innerHTML = '<div id="root"></div>';
        root = document.getElementById('root')!;

        callbacks = {
            onEvent: vi.fn(),
            onRef: vi.fn(),
            onRefDelete: vi.fn(),
            onRouter: vi.fn(),
            onScript: vi.fn(),
            onScriptCleanup: vi.fn(),
        };

        patcher = new Patcher(root, callbacks);
    });

    describe('setText', () => {
        it('should set text content', () => {
            root.innerHTML = '<span>old</span>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setText', value: 'new' }
            ];
            patcher.apply(patches);
            expect(root.querySelector('span')!.textContent).toBe('new');
        });
    });

    describe('setComment', () => {
        it('should set comment content', () => {
            root.innerHTML = '<!--old-->';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setComment', value: 'new comment' }
            ];
            patcher.apply(patches);
            expect(root.childNodes[0].textContent).toBe('new comment');
        });
    });

    describe('setAttr', () => {
        it('should set class attribute', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setAttr', value: { class: ['foo', 'bar'] } }
            ];
            patcher.apply(patches);
            expect(root.querySelector('div')!.className).toBe('foo bar');
        });

        it('should set input value', () => {
            root.innerHTML = '<input type="text">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setAttr', value: { value: ['hello'] } }
            ];
            patcher.apply(patches);
            expect((root.querySelector('input') as HTMLInputElement).value).toBe('hello');
        });

        it('should set checkbox checked', () => {
            root.innerHTML = '<input type="checkbox">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setAttr', value: { checked: ['true'] } }
            ];
            patcher.apply(patches);
            expect((root.querySelector('input') as HTMLInputElement).checked).toBe(true);
        });

        it('should set option selected', () => {
            root.innerHTML = '<select><option value="a">A</option></select>';
            const patches: Patch[] = [
                { seq: 0, path: [0, 0], op: 'setAttr', value: { selected: ['true'] } }
            ];
            patcher.apply(patches);
            expect((root.querySelector('option') as HTMLOptionElement).selected).toBe(true);
        });

        it('should set boolean attribute with empty value', () => {
            root.innerHTML = '<button></button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setAttr', value: { disabled: [] } }
            ];
            patcher.apply(patches);
            expect(root.querySelector('button')!.getAttribute('disabled')).toBe('');
        });

        it('should set regular attribute', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setAttr', value: { 'data-id': ['123'] } }
            ];
            patcher.apply(patches);
            expect(root.querySelector('div')!.getAttribute('data-id')).toBe('123');
        });
    });

    describe('delAttr', () => {
        it('should remove class attribute', () => {
            root.innerHTML = '<div class="foo bar"></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'delAttr', name: 'class' }
            ];
            patcher.apply(patches);
            expect(root.querySelector('div')!.hasAttribute('class')).toBe(false);
        });

        it('should clear input value', () => {
            root.innerHTML = '<input type="text" value="hello">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'delAttr', name: 'value' }
            ];
            patcher.apply(patches);
            expect((root.querySelector('input') as HTMLInputElement).value).toBe('');
        });

        it('should uncheck checkbox', () => {
            root.innerHTML = '<input type="checkbox" checked>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'delAttr', name: 'checked' }
            ];
            patcher.apply(patches);
            expect((root.querySelector('input') as HTMLInputElement).checked).toBe(false);
        });

        it('should remove regular attribute', () => {
            root.innerHTML = '<div data-id="123"></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'delAttr', name: 'data-id' }
            ];
            patcher.apply(patches);
            expect(root.querySelector('div')!.hasAttribute('data-id')).toBe(false);
        });
    });

    describe('setStyle', () => {
        it('should set inline styles', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setStyle', value: { color: 'red', 'font-size': '16px' } }
            ];
            patcher.apply(patches);
            const el = root.querySelector('div') as HTMLElement;
            expect(el.style.color).toBe('red');
            expect(el.style.fontSize).toBe('16px');
        });
    });

    describe('delStyle', () => {
        it('should remove inline style', () => {
            root.innerHTML = '<div style="color: red; font-size: 16px;"></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'delStyle', name: 'color' }
            ];
            patcher.apply(patches);
            const el = root.querySelector('div') as HTMLElement;
            expect(el.style.color).toBe('');
            expect(el.style.fontSize).toBe('16px');
        });
    });

    describe('setHandlers', () => {
        it('should attach event handlers', () => {
            root.innerHTML = '<button>Click</button>';
            const patches: Patch[] = [
                {
                    seq: 0,
                    path: [0],
                    op: 'setHandlers',
                    value: [{ event: 'click', handler: 'h1', props: [] }]
                }
            ];
            patcher.apply(patches);

            const button = root.querySelector('button')!;
            button.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', {});
        });

        it('should extract event props', () => {
            root.innerHTML = '<input type="text">';
            const patches: Patch[] = [
                {
                    seq: 0,
                    path: [0],
                    op: 'setHandlers',
                    value: [{ event: 'input', handler: 'h1', props: ['target.value'] }]
                }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input') as HTMLInputElement;
            input.value = 'hello';
            input.dispatchEvent(new Event('input'));

            expect(callbacks.onEvent).toHaveBeenCalledWith('input', 'h1', { 'target.value': 'hello' });
        });

        it('should replace old handlers', () => {
            root.innerHTML = '<button>Click</button>';
            const patches1: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: [] }] }
            ];
            const patches2: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h2', props: [] }] }
            ];

            patcher.apply(patches1);
            patcher.apply(patches2);

            const button = root.querySelector('button')!;
            button.click();

            expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h2', {});
        });

        it('should preventDefault by default', () => {
            root.innerHTML = '<form><button type="submit">Submit</button></form>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'submit', handler: 'h1', props: [] }] }
            ];
            patcher.apply(patches);

            const form = root.querySelector('form')!;
            const event = new Event('submit', { bubbles: true, cancelable: true });
            form.dispatchEvent(event);

            expect(event.defaultPrevented).toBe(true);
        });

        it('should allow default when listen includes allowDefault', () => {
            root.innerHTML = '<form><button type="submit">Submit</button></form>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'submit', handler: 'h1', props: [], listen: ['allowDefault'] }] }
            ];
            patcher.apply(patches);

            const form = root.querySelector('form')!;
            const event = new Event('submit', { bubbles: true, cancelable: true });
            form.dispatchEvent(event);

            expect(event.defaultPrevented).toBe(false);
        });

        it('should stopPropagation by default', () => {
            root.innerHTML = '<div id="outer"><button>Click</button></div>';
            let outerClicked = false;
            root.querySelector('#outer')!.addEventListener('click', () => { outerClicked = true; });

            const patches: Patch[] = [
                { seq: 0, path: [0, 0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: [] }] }
            ];
            patcher.apply(patches);

            const button = root.querySelector('button')!;
            button.click();

            expect(outerClicked).toBe(false);
        });

        it('should allow bubble when listen includes bubble', () => {
            root.innerHTML = '<div id="outer"><button>Click</button></div>';
            let outerClicked = false;
            root.querySelector('#outer')!.addEventListener('click', () => { outerClicked = true; });

            const patches: Patch[] = [
                { seq: 0, path: [0, 0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: [], listen: ['bubble'] }] }
            ];
            patcher.apply(patches);

            const button = root.querySelector('button')!;
            button.click();

            expect(outerClicked).toBe(true);
        });
    });

    describe('setRouter', () => {
        it('should attach router click handler', () => {
            root.innerHTML = '<a href="/home">Home</a>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setRouter', value: { pathValue: '/home', replace: false } }
            ];
            patcher.apply(patches);

            const link = root.querySelector('a')!;
            const event = new MouseEvent('click', { bubbles: true, cancelable: true });
            link.dispatchEvent(event);

            expect(callbacks.onRouter).toHaveBeenCalledWith({ pathValue: '/home', replace: false });
        });

        it('should prevent default on click', () => {
            root.innerHTML = '<a href="/home">Home</a>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setRouter', value: { pathValue: '/home' } }
            ];
            patcher.apply(patches);

            const link = root.querySelector('a')!;
            const event = new MouseEvent('click', { bubbles: true, cancelable: true });
            link.dispatchEvent(event);

            expect(event.defaultPrevented).toBe(true);
        });
    });

    describe('delRouter', () => {
        it('should remove router handler', () => {
            root.innerHTML = '<a href="/home">Home</a>';
            const patches1: Patch[] = [
                { seq: 0, path: [0], op: 'setRouter', value: { pathValue: '/home' } }
            ];
            const patches2: Patch[] = [
                { seq: 0, path: [0], op: 'delRouter' }
            ];

            patcher.apply(patches1);
            patcher.apply(patches2);

            const link = root.querySelector('a')!;
            link.click();

            expect(callbacks.onRouter).not.toHaveBeenCalled();
        });
    });

    describe('setRef', () => {
        it('should call onRef callback', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setRef', value: 'myRef' }
            ];
            patcher.apply(patches);

            expect(callbacks.onRef).toHaveBeenCalledWith('myRef', root.querySelector('div'));
        });
    });

    describe('delRef', () => {
        it('should call onRefDelete with refId', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'delRef', value: 'myRef' }
            ];
            patcher.apply(patches);

            expect(callbacks.onRefDelete).toHaveBeenCalledWith('myRef');
        });
    });

    describe('replaceNode', () => {
        it('should replace element with new element', () => {
            root.innerHTML = '<div>old</div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'replaceNode', value: { tag: 'span', children: [{ text: 'new' }] } }
            ];
            patcher.apply(patches);

            expect(root.innerHTML).toBe('<span>new</span>');
        });

        it('should replace element with text node', () => {
            root.innerHTML = '<div>old</div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'replaceNode', value: { text: 'just text' } }
            ];
            patcher.apply(patches);

            expect(root.textContent).toBe('just text');
        });

        it('should replace and apply attrs', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                {
                    seq: 0,
                    path: [0],
                    op: 'replaceNode',
                    value: { tag: 'span', attrs: { class: ['foo', 'bar'], id: ['test'] } }
                }
            ];
            patcher.apply(patches);

            const span = root.querySelector('span')!;
            expect(span.className).toBe('foo bar');
            expect(span.id).toBe('test');
        });
    });

    describe('addChild', () => {
        it('should add child at index', () => {
            root.innerHTML = '<ul><li>1</li><li>3</li></ul>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'addChild', index: 1, value: { tag: 'li', children: [{ text: '2' }] } }
            ];
            patcher.apply(patches);

            const items = root.querySelectorAll('li');
            expect(items.length).toBe(3);
            expect(items[1].textContent).toBe('2');
        });

        it('should append child at end', () => {
            root.innerHTML = '<ul><li>1</li></ul>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'addChild', index: 1, value: { tag: 'li', children: [{ text: '2' }] } }
            ];
            patcher.apply(patches);

            const items = root.querySelectorAll('li');
            expect(items.length).toBe(2);
            expect(items[1].textContent).toBe('2');
        });
    });

    describe('delChild', () => {
        it('should remove child at index', () => {
            root.innerHTML = '<ul><li>1</li><li>2</li><li>3</li></ul>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'delChild', index: 1 }
            ];
            patcher.apply(patches);

            const items = root.querySelectorAll('li');
            expect(items.length).toBe(2);
            expect(items[0].textContent).toBe('1');
            expect(items[1].textContent).toBe('3');
        });
    });

    describe('moveChild', () => {
        it('should move child to new position', () => {
            root.innerHTML = '<ul><li>1</li><li>2</li><li>3</li></ul>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'moveChild', value: { fromIndex: 2, newIdx: 0 } }
            ];
            patcher.apply(patches);

            const items = root.querySelectorAll('li');
            expect(items[0].textContent).toBe('3');
            expect(items[1].textContent).toBe('1');
            expect(items[2].textContent).toBe('2');
        });
    });

    describe('sequencing', () => {
        it('should apply patches in seq order', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 2, path: [0], op: 'setAttr', value: { class: ['third'] } },
                { seq: 0, path: [0], op: 'setAttr', value: { class: ['first'] } },
                { seq: 1, path: [0], op: 'setAttr', value: { class: ['second'] } },
            ];
            patcher.apply(patches);

            // Last applied wins
            expect(root.querySelector('div')!.className).toBe('third');
        });

        it('should handle multiple deletes in correct order', () => {
            root.innerHTML = '<ul><li>1</li><li>2</li><li>3</li></ul>';
            // Delete from end to start to avoid index shifting issues
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'delChild', index: 2 },
                { seq: 1, path: [0], op: 'delChild', index: 0 },
            ];
            patcher.apply(patches);

            const items = root.querySelectorAll('li');
            expect(items.length).toBe(1);
            expect(items[0].textContent).toBe('2');
        });
    });

    describe('path resolution', () => {
        it('should resolve nested paths', () => {
            root.innerHTML = '<div><ul><li>item</li></ul></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0, 0, 0], op: 'setText', value: 'updated' }
            ];
            patcher.apply(patches);

            expect(root.querySelector('li')!.textContent).toBe('updated');
        });

        it('should handle invalid path gracefully', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0, 5, 10], op: 'setText', value: 'nope' }
            ];
            // Should not throw
            expect(() => patcher.apply(patches)).not.toThrow();
        });
    });

    describe('extractEventData', () => {
        it('should extract target.value from input', () => {
            root.innerHTML = '<input type="text">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'input', handler: 'h1', props: ['target.value'] }] }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input') as HTMLInputElement;
            input.value = 'test value';
            input.dispatchEvent(new Event('input'));

            expect(callbacks.onEvent).toHaveBeenCalledWith('input', 'h1', { 'target.value': 'test value' });
        });

        it('should extract event root alias', () => {
            root.innerHTML = '<button>Click</button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['event.type'] }] }
            ];
            patcher.apply(patches);

            root.querySelector('button')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', { 'event.type': 'click' });
        });

        it('should extract currentTarget alias', () => {
            root.innerHTML = '<button data-id="123">Click</button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['currentTarget.dataset.id'] }] }
            ];
            patcher.apply(patches);

            root.querySelector('button')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', { 'currentTarget.dataset.id': '123' });
        });

        it('should extract element alias (same as currentTarget)', () => {
            root.innerHTML = '<button data-value="abc">Click</button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['element.dataset.value'] }] }
            ];
            patcher.apply(patches);

            root.querySelector('button')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', { 'element.dataset.value': 'abc' });
        });

        it('should extract ref alias (same as element)', () => {
            root.innerHTML = '<button data-ref="xyz">Click</button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['ref.dataset.ref'] }] }
            ];
            patcher.apply(patches);

            root.querySelector('button')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', { 'ref.dataset.ref': 'xyz' });
        });

        it('should handle nested property paths', () => {
            root.innerHTML = '<input type="text">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'focus', handler: 'h1', props: ['target.style.color'] }] }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input') as HTMLInputElement;
            input.style.color = 'red';
            input.dispatchEvent(new Event('focus'));

            expect(callbacks.onEvent).toHaveBeenCalledWith('focus', 'h1', { 'target.style.color': 'red' });
        });

        it('should return null for nonexistent properties', () => {
            root.innerHTML = '<input type="text">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'input', handler: 'h1', props: ['target.nonexistent'] }] }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input')!;
            input.dispatchEvent(new Event('input'));

            expect(callbacks.onEvent).toHaveBeenCalledWith('input', 'h1', { 'target.nonexistent': null });
        });

        it('should serialize Date to ISO string', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['detail.date'] }] }
            ];
            patcher.apply(patches);

            const date = new Date('2024-01-15T12:00:00.000Z');
            const event = new CustomEvent('click', { detail: { date } });
            root.querySelector('div')!.dispatchEvent(event);

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', { 'detail.date': '2024-01-15T12:00:00.000Z' });
        });

        it('should serialize DOMTokenList to array', () => {
            root.innerHTML = '<div class="foo bar baz"></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['target.classList'] }] }
            ];
            patcher.apply(patches);

            root.querySelector('div')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', { 'target.classList': ['foo', 'bar', 'baz'] });
        });

        it('should handle arrays', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['detail.items'] }] }
            ];
            patcher.apply(patches);

            const event = new CustomEvent('click', { detail: { items: [1, 2, 3] } });
            root.querySelector('div')!.dispatchEvent(event);

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', { 'detail.items': [1, 2, 3] });
        });

        it('should serialize objects via JSON', () => {
            root.innerHTML = '<div></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['detail.obj'] }] }
            ];
            patcher.apply(patches);

            const event = new CustomEvent('click', { detail: { obj: { a: 1, b: 'two' } } });
            root.querySelector('div')!.dispatchEvent(event);

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', { 'detail.obj': { a: 1, b: 'two' } });
        });

        it('should filter out Node values', () => {
            root.innerHTML = '<div><span>child</span></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['target.firstChild'] }] }
            ];
            patcher.apply(patches);

            root.querySelector('div')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', {});
        });

        it('should handle multiple props', () => {
            root.innerHTML = '<input type="text" data-id="42">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'input', handler: 'h1', props: ['target.value', 'target.dataset.id', 'event.type'] }] }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input') as HTMLInputElement;
            input.value = 'hello';
            input.dispatchEvent(new Event('input'));

            expect(callbacks.onEvent).toHaveBeenCalledWith('input', 'h1', {
                'target.value': 'hello',
                'target.dataset.id': '42',
                'event.type': 'input'
            });
        });

        it('should extract checkbox checked state', () => {
            root.innerHTML = '<input type="checkbox">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'change', handler: 'h1', props: ['target.checked'] }] }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input') as HTMLInputElement;
            input.checked = true;
            input.dispatchEvent(new Event('change'));

            expect(callbacks.onEvent).toHaveBeenCalledWith('change', 'h1', { 'target.checked': true });
        });

        it('should handle empty props array', () => {
            root.innerHTML = '<button>Click</button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: [] }] }
            ];
            patcher.apply(patches);

            root.querySelector('button')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', {});
        });

        it('should handle undefined props', () => {
            root.innerHTML = '<button>Click</button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1' }] }
            ];
            patcher.apply(patches);

            root.querySelector('button')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', {});
        });

        it('should handle direct event properties without alias', () => {
            root.innerHTML = '<button>Click</button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['bubbles', 'cancelable'] }] }
            ];
            patcher.apply(patches);

            const event = new MouseEvent('click', { bubbles: true, cancelable: true });
            root.querySelector('button')!.dispatchEvent(event);

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', { bubbles: true, cancelable: true });
        });

        it('should handle keyboard event properties', () => {
            root.innerHTML = '<input type="text">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'keydown', handler: 'h1', props: ['event.key', 'event.code', 'event.shiftKey'] }] }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input')!;
            const event = new KeyboardEvent('keydown', { key: 'Enter', code: 'Enter', shiftKey: true });
            input.dispatchEvent(event);

            expect(callbacks.onEvent).toHaveBeenCalledWith('keydown', 'h1', {
                'event.key': 'Enter',
                'event.code': 'Enter',
                'event.shiftKey': true
            });
        });

        it('should handle whitespace in prop paths', () => {
            root.innerHTML = '<input type="text">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'input', handler: 'h1', props: [' target . value '] }] }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input') as HTMLInputElement;
            input.value = 'trimmed';
            input.dispatchEvent(new Event('input'));

            expect(callbacks.onEvent).toHaveBeenCalledWith('input', 'h1', { ' target . value ': 'trimmed' });
        });
    });

    describe('createNode', () => {
        it('should create element with handlers', () => {
            root.innerHTML = '';
            const patches: Patch[] = [
                {
                    seq: 0,
                    path: [],
                    op: 'addChild',
                    index: 0,
                    value: {
                        tag: 'button',
                        children: [{ text: 'Click' }],
                        handlers: [{ event: 'click', handler: 'h1', props: [] }]
                    }
                }
            ];
            patcher.apply(patches);

            const button = root.querySelector('button')!;
            button.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('click', 'h1', {});
        });

        it('should create element with router', () => {
            root.innerHTML = '';
            const patches: Patch[] = [
                {
                    seq: 0,
                    path: [],
                    op: 'addChild',
                    index: 0,
                    value: {
                        tag: 'a',
                        attrs: { href: ['/about'] },
                        router: { pathValue: '/about' }
                    }
                }
            ];
            patcher.apply(patches);

            const link = root.querySelector('a')!;
            link.click();

            expect(callbacks.onRouter).toHaveBeenCalled();
        });

        it('should create element with refId', () => {
            root.innerHTML = '';
            const patches: Patch[] = [
                {
                    seq: 0,
                    path: [],
                    op: 'addChild',
                    index: 0,
                    value: { tag: 'div', refId: 'myDiv' }
                }
            ];
            patcher.apply(patches);

            expect(callbacks.onRef).toHaveBeenCalledWith('myDiv', root.querySelector('div'));
        });

        it('should create comment node', () => {
            root.innerHTML = '';
            const patches: Patch[] = [
                { seq: 0, path: [], op: 'addChild', index: 0, value: { comment: 'a comment' } }
            ];
            patcher.apply(patches);

            expect(root.childNodes[0].nodeType).toBe(8); // Comment node
            expect(root.childNodes[0].textContent).toBe('a comment');
        });

        it('should set unsafeHTML', () => {
            root.innerHTML = '';
            const patches: Patch[] = [
                {
                    seq: 0,
                    path: [],
                    op: 'addChild',
                    index: 0,
                    value: { tag: 'div', unsafeHTML: '<strong>bold</strong>' }
                }
            ];
            patcher.apply(patches);

            expect(root.querySelector('div')!.innerHTML).toBe('<strong>bold</strong>');
        });
    });

    describe('Script Cleanup System', () => {
        describe('setScript and delScript', () => {
            it('should call onScript when setScript patch is applied', () => {
                root.innerHTML = '<div></div>';
                const patches: Patch[] = [
                    {
                        seq: 0,
                        path: [0],
                        op: 'setScript',
                        value: { scriptId: 'script-1', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(patches);

                expect(callbacks.onScript).toHaveBeenCalledWith(
                    { scriptId: 'script-1', script: '(el, t) => {}' },
                    root.querySelector('div')
                );
            });

            it('should call onScriptCleanup when delScript patch is applied', () => {
                root.innerHTML = '<div></div>';
                const setPatches: Patch[] = [
                    {
                        seq: 0,
                        path: [0],
                        op: 'setScript',
                        value: { scriptId: 'script-1', script: '(el, t) => {}' }
                    }
                ];
                const delPatches: Patch[] = [
                    { seq: 1, path: [0], op: 'delScript' }
                ];

                patcher.apply(setPatches);
                expect(callbacks.onScriptCleanup).not.toHaveBeenCalled();

                patcher.apply(delPatches);
                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');
            });

            it('should handle delScript on element without script', () => {
                root.innerHTML = '<div></div>';
                const patches: Patch[] = [
                    { seq: 0, path: [0], op: 'delScript' }
                ];

                expect(() => patcher.apply(patches)).not.toThrow();
                expect(callbacks.onScriptCleanup).not.toHaveBeenCalled();
            });
        });

        describe('cleanup on delChild', () => {
            it('should call cleanup when element with script is removed', () => {
                root.innerHTML = '<div><span></span></div>';

                const setScript: Patch[] = [
                    {
                        seq: 0,
                        path: [0, 0],
                        op: 'setScript',
                        value: { scriptId: 'script-1', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScript);

                const delChild: Patch[] = [
                    { seq: 1, path: [0], op: 'delChild', index: 0 }
                ];
                patcher.apply(delChild);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');
                expect(root.querySelector('span')).toBeNull();
            });

            it('should cleanup multiple nested scripts when parent is removed', () => {
                root.innerHTML = '<div><ul><li></li><li></li></ul></div>';

                const setScripts: Patch[] = [
                    {
                        seq: 0,
                        path: [0, 0],
                        op: 'setScript',
                        value: { scriptId: 'script-ul', script: '(el, t) => {}' }
                    },
                    {
                        seq: 1,
                        path: [0, 0, 0],
                        op: 'setScript',
                        value: { scriptId: 'script-li-1', script: '(el, t) => {}' }
                    },
                    {
                        seq: 2,
                        path: [0, 0, 1],
                        op: 'setScript',
                        value: { scriptId: 'script-li-2', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScripts);

                const delChild: Patch[] = [
                    { seq: 3, path: [0], op: 'delChild', index: 0 }
                ];
                patcher.apply(delChild);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-ul');
                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-li-1');
                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-li-2');
                expect(callbacks.onScriptCleanup).toHaveBeenCalledTimes(3);
            });

            it('should handle removing element without scripts', () => {
                root.innerHTML = '<div><span>text</span></div>';

                const patches: Patch[] = [
                    { seq: 0, path: [0], op: 'delChild', index: 0 }
                ];

                expect(() => patcher.apply(patches)).not.toThrow();
                expect(callbacks.onScriptCleanup).not.toHaveBeenCalled();
            });

            it('should cleanup deeply nested scripts', () => {
                root.innerHTML = '<div><section><article><p></p></article></section></div>';

                const setScript: Patch[] = [
                    {
                        seq: 0,
                        path: [0, 0, 0, 0],
                        op: 'setScript',
                        value: { scriptId: 'deep-script', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScript);

                const delChild: Patch[] = [
                    { seq: 1, path: [0], op: 'delChild', index: 0 }
                ];
                patcher.apply(delChild);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('deep-script');
            });
        });

        describe('cleanup on replaceNode', () => {
            it('should call cleanup when replacing element with script', () => {
                root.innerHTML = '<div></div>';

                const setScript: Patch[] = [
                    {
                        seq: 0,
                        path: [0],
                        op: 'setScript',
                        value: { scriptId: 'script-1', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScript);

                const replace: Patch[] = [
                    {
                        seq: 1,
                        path: [0],
                        op: 'replaceNode',
                        value: { tag: 'span', children: [{ text: 'new' }] }
                    }
                ];
                patcher.apply(replace);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');
                expect(root.querySelector('div')).toBeNull();
                expect(root.querySelector('span')).not.toBeNull();
            });

            it('should cleanup nested scripts when replacing parent', () => {
                root.innerHTML = '<div><span></span><p></p></div>';

                const setScripts: Patch[] = [
                    {
                        seq: 0,
                        path: [0, 0],
                        op: 'setScript',
                        value: { scriptId: 'script-span', script: '(el, t) => {}' }
                    },
                    {
                        seq: 1,
                        path: [0, 1],
                        op: 'setScript',
                        value: { scriptId: 'script-p', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScripts);

                const replace: Patch[] = [
                    {
                        seq: 2,
                        path: [0],
                        op: 'replaceNode',
                        value: { tag: 'article', children: [{ text: 'new content' }] }
                    }
                ];
                patcher.apply(replace);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-span');
                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-p');
                expect(callbacks.onScriptCleanup).toHaveBeenCalledTimes(2);
            });

            it('should handle replacing element without scripts', () => {
                root.innerHTML = '<div>old</div>';

                const patches: Patch[] = [
                    {
                        seq: 0,
                        path: [0],
                        op: 'replaceNode',
                        value: { tag: 'span', children: [{ text: 'new' }] }
                    }
                ];

                expect(() => patcher.apply(patches)).not.toThrow();
                expect(callbacks.onScriptCleanup).not.toHaveBeenCalled();
            });

            it('should cleanup script on element being replaced and not affect new element', () => {
                root.innerHTML = '<div></div>';

                const setScript: Patch[] = [
                    {
                        seq: 0,
                        path: [0],
                        op: 'setScript',
                        value: { scriptId: 'old-script', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScript);

                const replace: Patch[] = [
                    {
                        seq: 1,
                        path: [0],
                        op: 'replaceNode',
                        value: {
                            tag: 'div',
                            script: { scriptId: 'new-script', script: '(el, t) => {}' }
                        }
                    }
                ];
                patcher.apply(replace);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('old-script');
                expect(callbacks.onScript).toHaveBeenCalledWith(
                    { scriptId: 'new-script', script: '(el, t) => {}' },
                    root.querySelector('div')
                );
            });
        });

        describe('complex cleanup scenarios', () => {
            it('should handle mixed content with some scripts and some non-scripts', () => {
                root.innerHTML = '<div><span></span><p></p><a></a></div>';

                const setScripts: Patch[] = [
                    {
                        seq: 0,
                        path: [0, 0],
                        op: 'setScript',
                        value: { scriptId: 'script-span', script: '(el, t) => {}' }
                    },
                    {
                        seq: 1,
                        path: [0, 2],
                        op: 'setScript',
                        value: { scriptId: 'script-a', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScripts);

                const delChild: Patch[] = [
                    { seq: 2, path: [], op: 'delChild', index: 0 }
                ];
                patcher.apply(delChild);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-span');
                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-a');
                expect(callbacks.onScriptCleanup).toHaveBeenCalledTimes(2);
            });

            it('should cleanup script when element tree is removed', () => {
                root.innerHTML = '<div id="app"><header><nav></nav></header><main><section></section></main></div>';

                const setScripts: Patch[] = [
                    {
                        seq: 0,
                        path: [0, 0, 0],
                        op: 'setScript',
                        value: { scriptId: 'nav-script', script: '(el, t) => {}' }
                    },
                    {
                        seq: 1,
                        path: [0, 1, 0],
                        op: 'setScript',
                        value: { scriptId: 'section-script', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScripts);

                const delChild: Patch[] = [
                    { seq: 2, path: [], op: 'delChild', index: 0 }
                ];
                patcher.apply(delChild);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('nav-script');
                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('section-script');
                expect(callbacks.onScriptCleanup).toHaveBeenCalledTimes(2);
            });

            it('should only cleanup once when same script is explicitly deleted then element is removed', () => {
                root.innerHTML = '<div><span></span></div>';

                const setScript: Patch[] = [
                    {
                        seq: 0,
                        path: [0, 0],
                        op: 'setScript',
                        value: { scriptId: 'script-1', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScript);

                const delScript: Patch[] = [
                    { seq: 1, path: [0, 0], op: 'delScript' }
                ];
                patcher.apply(delScript);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledTimes(1);
                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');

                const delChild: Patch[] = [
                    { seq: 2, path: [0], op: 'delChild', index: 0 }
                ];
                patcher.apply(delChild);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledTimes(1);
            });

            it('should handle text and comment nodes in tree without errors', () => {
                root.innerHTML = '<div>text<!--comment--><span></span></div>';

                const setScript: Patch[] = [
                    {
                        seq: 0,
                        path: [0, 2],
                        op: 'setScript',
                        value: { scriptId: 'script-1', script: '(el, t) => {}' }
                    }
                ];
                patcher.apply(setScript);

                const delChild: Patch[] = [
                    { seq: 1, path: [], op: 'delChild', index: 0 }
                ];

                expect(() => patcher.apply(delChild)).not.toThrow();
                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');
            });
        });

        describe('script lifecycle integration', () => {
            it('should cleanup old script when setScript is called on same element', () => {
                root.innerHTML = '<div></div>';

                const setScript1: Patch[] = [
                    {
                        seq: 0,
                        path: [0],
                        op: 'setScript',
                        value: { scriptId: 'script-1', script: '(el, t) => { /* v1 */ }' }
                    }
                ];
                patcher.apply(setScript1);

                expect(callbacks.onScript).toHaveBeenCalledTimes(1);

                const setScript2: Patch[] = [
                    {
                        seq: 1,
                        path: [0],
                        op: 'setScript',
                        value: { scriptId: 'script-2', script: '(el, t) => { /* v2 */ }' }
                    }
                ];
                patcher.apply(setScript2);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');
                expect(callbacks.onScript).toHaveBeenCalledTimes(2);
            });

            it('should cleanup script when delScript patch is applied', () => {
                root.innerHTML = '<div></div>';

                const setScript: Patch[] = [
                    {
                        seq: 0,
                        path: [0],
                        op: 'setScript',
                        value: { scriptId: 'script-1', script: '(el, t) => { /* v1 */ }' }
                    }
                ];
                patcher.apply(setScript);
                expect(callbacks.onScript).toHaveBeenCalledTimes(1);

                const delScript: Patch[] = [
                    {
                        seq: 1,
                        path: [0],
                        op: 'delScript'
                    }
                ];
                patcher.apply(delScript);

                expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');
            });

            it('should create element with script via addChild', () => {
                root.innerHTML = '<div></div>';

                const addChild: Patch[] = [
                    {
                        seq: 0,
                        path: [0],
                        op: 'addChild',
                        index: 0,
                        value: {
                            tag: 'span',
                            script: { scriptId: 'child-script', script: '(el, t) => {}' }
                        }
                    }
                ];
                patcher.apply(addChild);

                expect(callbacks.onScript).toHaveBeenCalledWith(
                    { scriptId: 'child-script', script: '(el, t) => {}' },
                    root.querySelector('span')
                );
            });
        });
    });
});

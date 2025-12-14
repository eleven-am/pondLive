import { describe, it, expect, beforeEach, vi } from 'vitest';
import { Patcher, PatcherCallbacks } from './patcher';
import { Patch } from './protocol';

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
                    value: [{ event: 'click', handler: 'h1' }]
                }
            ];
            patcher.apply(patches);

            const button = root.querySelector('button')!;
            button.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('h1', {});
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

            expect(callbacks.onEvent).toHaveBeenCalledWith('h1', { 'target.value': 'hello' });
        });

        it('should replace old handlers', () => {
            root.innerHTML = '<button>Click</button>';
            const patches1: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1' }] }
            ];
            const patches2: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h2' }] }
            ];

            patcher.apply(patches1);
            patcher.apply(patches2);

            const button = root.querySelector('button')!;
            button.click();

            expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
            expect(callbacks.onEvent).toHaveBeenCalledWith('h2', {});
        });

        it('should handle prevent option', () => {
            root.innerHTML = '<form><button type="submit">Submit</button></form>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'submit', handler: 'h1', prevent: true }] }
            ];
            patcher.apply(patches);

            const form = root.querySelector('form')!;
            const event = new Event('submit', { cancelable: true });
            form.dispatchEvent(event);

            expect(event.defaultPrevented).toBe(true);
        });

        it('should handle stop option', () => {
            root.innerHTML = '<div id="outer"><button>Click</button></div>';
            let outerClicked = false;
            root.querySelector('#outer')!.addEventListener('click', () => { outerClicked = true; });

            const patches: Patch[] = [
                { seq: 0, path: [0, 0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', stop: true }] }
            ];
            patcher.apply(patches);

            const button = root.querySelector('button')!;
            button.click();

            expect(outerClicked).toBe(false);
        });

        it('should handle debounce option', async () => {
            vi.useFakeTimers();
            root.innerHTML = '<input type="text">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'input', handler: 'h1', debounce: 100 }] }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input')!;
            input.dispatchEvent(new Event('input'));
            input.dispatchEvent(new Event('input'));
            input.dispatchEvent(new Event('input'));

            expect(callbacks.onEvent).not.toHaveBeenCalled();

            vi.advanceTimersByTime(100);

            expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
            vi.useRealTimers();
        });

        it('should handle throttle option', () => {
            vi.useFakeTimers();
            root.innerHTML = '<input type="text">';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'input', handler: 'h1', throttle: 100 }] }
            ];
            patcher.apply(patches);

            const input = root.querySelector('input')!;
            input.dispatchEvent(new Event('input'));
            input.dispatchEvent(new Event('input'));
            input.dispatchEvent(new Event('input'));

            expect(callbacks.onEvent).toHaveBeenCalledTimes(1);

            vi.advanceTimersByTime(100);
            input.dispatchEvent(new Event('input'));

            expect(callbacks.onEvent).toHaveBeenCalledTimes(2);
            vi.useRealTimers();
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

    describe('setScript', () => {
        it('should call onScript callback', () => {
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

        it('should cleanup old script when setting new', () => {
            root.innerHTML = '<div></div>';
            const patches1: Patch[] = [
                { seq: 0, path: [0], op: 'setScript', value: { scriptId: 'script-1', script: '(el, t) => {}' } }
            ];
            const patches2: Patch[] = [
                { seq: 1, path: [0], op: 'setScript', value: { scriptId: 'script-2', script: '(el, t) => {}' } }
            ];

            patcher.apply(patches1);
            patcher.apply(patches2);

            expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');
        });
    });

    describe('delScript', () => {
        it('should call onScriptCleanup', () => {
            root.innerHTML = '<div></div>';
            const patches1: Patch[] = [
                { seq: 0, path: [0], op: 'setScript', value: { scriptId: 'script-1', script: '(el, t) => {}' } }
            ];
            const patches2: Patch[] = [
                { seq: 1, path: [0], op: 'delScript' }
            ];

            patcher.apply(patches1);
            patcher.apply(patches2);

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

        it('should cleanup scripts in replaced tree', () => {
            root.innerHTML = '<div></div>';
            const patches1: Patch[] = [
                { seq: 0, path: [0], op: 'setScript', value: { scriptId: 'script-1', script: '(el, t) => {}' } }
            ];
            const patches2: Patch[] = [
                { seq: 1, path: [0], op: 'replaceNode', value: { tag: 'span' } }
            ];

            patcher.apply(patches1);
            patcher.apply(patches2);

            expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');
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

        it('should cleanup scripts in deleted tree', () => {
            root.innerHTML = '<div><span></span></div>';
            const patches1: Patch[] = [
                { seq: 0, path: [0, 0], op: 'setScript', value: { scriptId: 'script-1', script: '(el, t) => {}' } }
            ];
            const patches2: Patch[] = [
                { seq: 1, path: [0], op: 'delChild', index: 0 }
            ];

            patcher.apply(patches1);
            patcher.apply(patches2);

            expect(callbacks.onScriptCleanup).toHaveBeenCalledWith('script-1');
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

        it('should find SSR element by signature when not in keyedElements', () => {
            root.innerHTML = '<div><a href="/google">Google</a><a href="/stripe">Stripe</a><a href="/netflix">Netflix</a></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'addChild', index: 0, value: { tag: 'a', attrs: { href: ['/cvs/master'] }, children: [{ text: 'Master CV' }] } },
                { seq: 1, path: [0], op: 'moveChild', value: { fromIndex: 0, newIdx: 1, key: 'E:a|href=/google' } },
            ];
            patcher.apply(patches);

            const links = root.querySelectorAll('a');
            expect(links.length).toBe(4);
            expect(links[0].getAttribute('href')).toBe('/cvs/master');
            expect(links[1].getAttribute('href')).toBe('/google');
            expect(links[2].getAttribute('href')).toBe('/stripe');
            expect(links[3].getAttribute('href')).toBe('/netflix');
        });

        it('should handle navigation scenario with signature-based moves', () => {
            root.innerHTML = '<div><a href="/applications/google">Google CL</a><a href="/applications/stripe">Stripe CL</a><a href="/applications/netflix">Netflix CL</a><a href="/applications/apple">Apple CL</a></div>';

            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'delChild', index: 3 },
                { seq: 1, path: [0], op: 'addChild', index: 0, value: { tag: 'a', attrs: { href: ['/cvs/master'] }, children: [{ text: 'Master CV' }] } },
                { seq: 2, path: [0], op: 'moveChild', value: { fromIndex: 0, newIdx: 1, key: 'E:a|href=/applications/google' } },
                { seq: 3, path: [0], op: 'moveChild', value: { fromIndex: 1, newIdx: 2, key: 'E:a|href=/applications/stripe' } },
                { seq: 4, path: [0], op: 'moveChild', value: { fromIndex: 2, newIdx: 3, key: 'E:a|href=/applications/netflix' } },
            ];
            patcher.apply(patches);

            const links = root.querySelectorAll('a');
            expect(links.length).toBe(4);
            expect(links[0].getAttribute('href')).toBe('/cvs/master');
            expect(links[1].getAttribute('href')).toBe('/applications/google');
            expect(links[2].getAttribute('href')).toBe('/applications/stripe');
            expect(links[3].getAttribute('href')).toBe('/applications/netflix');
        });

        it('should find SSR element by explicit key (K: signature)', () => {
            root.innerHTML = '<ul><li data-key="item-a">A</li><li data-key="item-b">B</li><li data-key="item-c">C</li></ul>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'addChild', index: 0, value: { tag: 'li', attrs: { 'data-key': ['item-new'] }, children: [{ text: 'New' }] } },
                { seq: 1, path: [0], op: 'moveChild', value: { fromIndex: 0, newIdx: 1, key: 'K:item-a' } },
                { seq: 2, path: [0], op: 'moveChild', value: { fromIndex: 1, newIdx: 2, key: 'K:item-b' } },
            ];
            patcher.apply(patches);

            const items = root.querySelectorAll('li');
            expect(items.length).toBe(4);
            expect(items[0].getAttribute('data-key')).toBe('item-new');
            expect(items[1].getAttribute('data-key')).toBe('item-a');
            expect(items[2].getAttribute('data-key')).toBe('item-b');
            expect(items[3].getAttribute('data-key')).toBe('item-c');
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

            expect(root.querySelector('div')!.className).toBe('third');
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
            expect(() => patcher.apply(patches)).not.toThrow();
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
                        handlers: [{ event: 'click', handler: 'h1' }]
                    }
                }
            ];
            patcher.apply(patches);

            const button = root.querySelector('button')!;
            button.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('h1', {});
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

            expect(root.childNodes[0].nodeType).toBe(8);
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

            expect(callbacks.onEvent).toHaveBeenCalledWith('h1', { 'target.value': 'test value' });
        });

        it('should extract event.type', () => {
            root.innerHTML = '<button>Click</button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['event.type'] }] }
            ];
            patcher.apply(patches);

            root.querySelector('button')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('h1', { 'event.type': 'click' });
        });

        it('should extract currentTarget properties', () => {
            root.innerHTML = '<button data-id="123">Click</button>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['currentTarget.dataset.id'] }] }
            ];
            patcher.apply(patches);

            root.querySelector('button')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('h1', { 'currentTarget.dataset.id': '123' });
        });

        it('should serialize DOMTokenList to array', () => {
            root.innerHTML = '<div class="foo bar baz"></div>';
            const patches: Patch[] = [
                { seq: 0, path: [0], op: 'setHandlers', value: [{ event: 'click', handler: 'h1', props: ['target.classList'] }] }
            ];
            patcher.apply(patches);

            root.querySelector('div')!.click();

            expect(callbacks.onEvent).toHaveBeenCalledWith('h1', { 'target.classList': ['foo', 'bar', 'baz'] });
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

            expect(callbacks.onEvent).toHaveBeenCalledWith('h1', { 'target.checked': true });
        });
    });
});

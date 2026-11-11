import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Applier, ApplierCallbacks, EditError } from './applier';
import { Edit } from './protocol';
import { Logger } from './logger';

const REGION_LIST =
    '<ul data-pi="1"><!--r:2-->' +
    '<li data-pi="3">a</li>' +
    '<li data-pi="4">b</li>' +
    '<li data-pi="5">c</li>' +
    '<!--/r:2--></ul>';

let callbacks: { onEvent: ReturnType<typeof vi.fn>; onScript: ReturnType<typeof vi.fn>; onScriptCleanup: ReturnType<typeof vi.fn> };

function mount(body: string, head = ''): Applier {
    document.head.innerHTML = head;
    document.body.innerHTML = body;
    const applier = new Applier(document.documentElement, callbacks as ApplierCallbacks);
    applier.index();
    return applier;
}

function text(selector: string): string {
    return document.querySelector(selector)!.textContent!;
}

function order(parent: string): string[] {
    return Array.from(document.querySelector(parent)!.children).map((c) => c.textContent!);
}

function failure(applier: Applier, edits: unknown[]): EditError {
    try {
        applier.apply(edits as Edit[]);
    } catch (e) {
        expect(e).toBeInstanceOf(EditError);
        return e as EditError;
    }
    throw new Error('expected the edit to fail');
}

beforeEach(() => {
    Logger.configure({ enabled: false });
    callbacks = { onEvent: vi.fn(), onScript: vi.fn(), onScriptCleanup: vi.fn() };
});

afterEach(() => {
    document.head.innerHTML = '';
    document.body.innerHTML = '';
    document.documentElement.removeAttribute('data-ran');
});

describe('Applier index', () => {
    it('indexes elements by data-pi', () => {
        const applier = mount('<div data-pi="7"><span data-pi="9"></span></div>');

        expect(applier.node(7)).toBe(document.querySelector('div'));
        expect(applier.node(9)).toBe(document.querySelector('span'));
        expect(applier.element(9)).toBe(document.querySelector('span'));
        expect(applier.node(8)).toBeUndefined();
    });

    it('resolves a text marker to the text node after it', () => {
        const applier = mount('<p>x<!--t:5-->hello<!---->world</p>');

        const slot = applier.node(5)!;
        expect(slot.nodeType).toBe(Node.TEXT_NODE);
        expect(slot.nodeValue).toBe('hello');
        expect(applier.element(5)).toBeUndefined();
    });

    it('creates the text node the parser dropped for an empty bound text', () => {
        const applier = mount('<p><!--t:5--></p><p><!--t:6--><!----></p>');

        const first = applier.node(5)!;
        expect(first.nodeType).toBe(Node.TEXT_NODE);
        expect(first.nodeValue).toBe('');
        expect(first.previousSibling!.nodeValue).toBe('t:5');

        const second = applier.node(6)!;
        expect(second.nodeType).toBe(Node.TEXT_NODE);
        expect(second.nextSibling!.nodeType).toBe(Node.COMMENT_NODE);
        expect(second.previousSibling!.nodeValue).toBe('t:6');
    });

    it('indexes regions, nested regions and their end markers', () => {
        const applier = mount('<div><!--r:1--><!--r:2--><i data-pi="3"></i><!--/r:2--><b data-pi="4"></b><!--/r:1--></div>');

        expect(applier.node(1)!.nodeType).toBe(Node.COMMENT_NODE);
        expect(applier.node(2)!.nodeType).toBe(Node.COMMENT_NODE);
        applier.apply([['ins', 2, 0, '<u data-pi="9"></u>']]);
        expect(document.querySelector('div')!.innerHTML).toBe(
            '<!--r:1--><!--r:2--><i data-pi="3"></i><u data-pi="9"></u><!--/r:2--><b data-pi="4"></b><!--/r:1-->',
        );
    });

    it('indexes a raw host as an element and leaves its content alone', () => {
        const applier = mount('<div data-pi="1" data-praw="">raw <b>content</b></div>');

        expect(applier.element(1)).toBe(document.querySelector('div'));
        applier.apply([['h', 1, '<i>new</i>']]);
        expect(text('div')).toBe('new');
    });

    it('logs structural problems at boot without throwing', () => {
        const warn = vi.spyOn(Logger, 'warn').mockImplementation(() => {});
        const applier = mount('<a data-pi="1"></a><a data-pi="1"></a><a data-pi="x"></a><a data-pi="0"></a>');

        expect(applier.node(1)).toBe(document.querySelectorAll('a')[0]);
        expect(warn).toHaveBeenCalledTimes(3);
    });

    it('re-indexing drops ids that are gone', () => {
        const applier = mount('<div data-pi="1"></div>');
        document.body.innerHTML = '<div data-pi="2"></div>';

        applier.index();

        expect(applier.node(1)).toBeUndefined();
        expect(applier.node(2)).toBe(document.querySelector('div'));
    });
});

describe('Applier edits', () => {
    it('t sets nodeValue on a text slot and textContent on an element', () => {
        const applier = mount('<p data-pi="1">old</p><span>a<!--t:2-->b</span>');
        const slot = applier.node(2)!;

        applier.apply([['t', 1, 'new'], ['t', 2, 'X']]);

        expect(text('p')).toBe('new');
        expect(text('span')).toBe('aX');
        expect(applier.node(2)).toBe(slot);
        expect(slot.nodeValue).toBe('X');
    });

    it('t on an empty slot fills the created text node', () => {
        const applier = mount('<p><!--t:2--></p>');

        applier.apply([['t', 2, 'filled']]);

        expect(text('p')).toBe('filled');
    });

    it('a sets and replaces attributes', () => {
        const applier = mount('<div data-pi="1" class="x"></div>');

        applier.apply([['a', 1, 'class', 'y z'], ['a', 1, 'title', 'hi']]);

        const el = document.querySelector('div')!;
        expect(el.getAttribute('class')).toBe('y z');
        expect(el.getAttribute('title')).toBe('hi');
    });

    it('a sets class on svg elements as an attribute', () => {
        const applier = mount('<svg><circle data-pi="1"></circle></svg>');

        applier.apply([['a', 1, 'class', 'big']]);

        expect(document.querySelector('circle')!.getAttribute('class')).toBe('big');
    });

    it('a on value, checked and selected also sets the property', () => {
        const applier = mount(
            '<input data-pi="1"><input data-pi="2" type="checkbox"><select><option data-pi="3" value="a">a</option><option data-pi="4" value="b">b</option></select><textarea data-pi="5"></textarea>',
        );

        applier.apply([
            ['a', 1, 'value', 'typed'],
            ['a', 2, 'checked', ''],
            ['a', 4, 'selected', ''],
            ['a', 5, 'value', 'area'],
        ]);

        expect((document.querySelectorAll('input')[0]).value).toBe('typed');
        expect((document.querySelectorAll('input')[1]).checked).toBe(true);
        expect(document.querySelector('select')!.value).toBe('b');
        expect(document.querySelector('textarea')!.value).toBe('area');
    });

    it('a sets the value property only when it differs', () => {
        const applier = mount('<input data-pi="1">');
        const input = document.querySelector('input')!;
        input.value = 'same';
        const set = vi.fn();
        const proto = Object.getPrototypeOf(input) as object;
        const desc = Object.getOwnPropertyDescriptor(proto, 'value')!;
        Object.defineProperty(input, 'value', {
            configurable: true,
            get: () => desc.get!.call(input) as string,
            set: (v: string) => {
                set(v);
                desc.set!.call(input, v);
            },
        });

        applier.apply([['a', 1, 'value', 'same']]);
        expect(set).not.toHaveBeenCalled();

        applier.apply([['a', 1, 'value', 'other']]);
        expect(set).toHaveBeenCalledTimes(1);
        expect(input.value).toBe('other');
    });

    it('ra removes the attribute and clears checked and selected', () => {
        const applier = mount('<input data-pi="1" type="checkbox" checked title="t"><select multiple><option data-pi="2" selected>a</option><option>b</option></select>');
        const input = document.querySelector('input')!;

        applier.apply([['ra', 1, 'title'], ['ra', 1, 'checked'], ['ra', 2, 'selected']]);

        expect(input.hasAttribute('title')).toBe(false);
        expect(input.checked).toBe(false);
        expect(document.querySelector('option')!.selected).toBe(false);
    });

    it('st sets a style property and removes it on an empty value', () => {
        const applier = mount('<div data-pi="1"></div>');
        const el = document.querySelector('div')!;

        applier.apply([['st', 1, 'color', 'red'], ['st', 1, '--gap', '4px']]);
        expect(el.style.color).toBe('red');
        expect(el.style.getPropertyValue('--gap')).toBe('4px');

        applier.apply([['st', 1, 'color', '']]);
        expect(el.style.color).toBe('');
    });

    it('h sets the innerHTML of a raw host', () => {
        const applier = mount('<div data-pi="1" data-praw=""></div>');

        applier.apply([['h', 1, '<b>bold</b>']]);

        expect(document.querySelector('div')!.innerHTML).toBe('<b>bold</b>');
    });

    it('clr empties a region and de-indexes what was inside', () => {
        const applier = mount(REGION_LIST);

        applier.apply([['clr', 2]]);

        expect(document.querySelector('ul')!.innerHTML).toBe('<!--r:2--><!--/r:2-->');
        expect(applier.node(3)).toBeUndefined();
        expect(applier.node(4)).toBeUndefined();
        expect(applier.node(2)).toBeDefined();
        applier.apply([['ins', 2, 0, '<li data-pi="6">z</li>']]);
        expect(order('ul')).toEqual(['z']);
    });

    it('applies the edits of a frame in order', () => {
        const applier = mount(REGION_LIST);

        applier.apply([['ins', 2, 0, '<li data-pi="6">d</li>'], ['t', 6, 'D'], ['rm', 3]]);

        expect(order('ul')).toEqual(['b', 'c', 'D']);
    });
});

describe('Applier ins', () => {
    it('inserts before a reference unit and at the region end', () => {
        const applier = mount(REGION_LIST);

        applier.apply([
            ['ins', 2, 4, '<li data-pi="6">x</li>'],
            ['ins', 2, 0, '<li data-pi="7">y</li>'],
            ['ins', 2, 3, '<li data-pi="8">w</li>'],
        ]);

        expect(order('ul')).toEqual(['w', 'a', 'x', 'b', 'c', 'y']);
        expect(applier.node(6)).toBe(document.querySelectorAll('li')[2]);
    });

    it('inserts a multi-node unit and a nested region unit', () => {
        const applier = mount(REGION_LIST);

        applier.apply([['ins', 2, 0, '<!--r:9--><li data-pi="10">p</li>tail<!--/r:9-->']]);

        expect(document.querySelector('ul')!.innerHTML).toContain('<!--r:9--><li data-pi="10">p</li>tail<!--/r:9--><!--/r:2-->');
        expect(applier.node(9)!.nodeType).toBe(Node.COMMENT_NODE);
        applier.apply([['ins', 9, 0, '<em data-pi="11">q</em>']]);
        expect(document.querySelector('ul')!.innerHTML).toContain('<li data-pi="10">p</li>tail<em data-pi="11">q</em><!--/r:9-->');
    });

    it('parses table rows in a tbody', () => {
        const applier = mount('<table><tbody data-pi="1"><!--r:2--><!--/r:2--></tbody></table>');

        applier.apply([['ins', 2, 0, '<tr data-pi="3"><td>c</td></tr>']]);

        const row = document.querySelector('tbody')!.firstElementChild!;
        expect(row.localName).toBe('tr');
        expect(applier.node(3)).toBe(row);
        expect(row.firstElementChild!.localName).toBe('td');
    });

    it('parses options in a select', () => {
        const applier = mount('<select data-pi="1"><!--r:2--><!--/r:2--></select>');

        applier.apply([['ins', 2, 0, '<option data-pi="3" value="a">A</option><option data-pi="4" value="b">B</option>']]);

        expect(document.querySelector('select')!.options.length).toBe(2);
    });

    it('creates svg elements in the svg namespace', () => {
        const applier = mount('<svg data-pi="1"><!--r:2--><!--/r:2--></svg>');

        applier.apply([['ins', 2, 0, '<circle data-pi="3" r="4"></circle><g data-pi="5"><rect data-pi="6"></rect></g>']]);

        const svgNS = 'http://www.w3.org/2000/svg';
        expect((applier.node(3) as Element).namespaceURI).toBe(svgNS);
        expect((applier.node(5) as Element).namespaceURI).toBe(svgNS);
        expect((applier.node(6) as Element).namespaceURI).toBe(svgNS);
        expect((applier.node(3) as Element).getAttribute('r')).toBe('4');
    });

    it('creates html elements inside foreignObject', () => {
        const applier = mount('<svg><foreignObject data-pi="1"><!--r:2--><!--/r:2--></foreignObject></svg>');

        applier.apply([['ins', 2, 0, '<div data-pi="3"></div>']]);

        expect((applier.node(3) as Element).namespaceURI).toBe('http://www.w3.org/1999/xhtml');
    });

    it('creates mathml elements in the mathml namespace', () => {
        const applier = mount('<math data-pi="1"><!--r:2--><!--/r:2--></math>');

        applier.apply([['ins', 2, 0, '<mi data-pi="3">x</mi>']]);

        expect((applier.node(3) as Element).namespaceURI).toBe('http://www.w3.org/1998/Math/MathML');
    });

    it('data-psel selects exactly the listed options of a multiple select', () => {
        const applier = mount('<select multiple data-pi="1"><option value="a" selected>A</option><option value="b">B</option><option value="c">C</option></select>');

        applier.apply([['a', 1, 'data-psel', '["b","c"]']]);

        const picked = Array.from((applier.node(1) as HTMLSelectElement).selectedOptions).map((o) => o.value);
        expect(picked).toEqual(['b', 'c']);
    });

    it('target.selected sends the values of the selected options', () => {
        mount('<select multiple data-pi="1" data-pon=\'[{"e":"change","p":"selected"}]\'><option value="a" selected>A</option><option value="b" selected>B</option><option value="c">C</option></select>');

        const select = document.querySelector('select')!;
        select.dispatchEvent(new Event('change'));

        expect(callbacks.onEvent).toHaveBeenCalledWith(1, 'change', { selected: ['a', 'b'] });
    });

    it('indexes ids, text markers, regions and handlers inside the inserted html', () => {
        const applier = mount(REGION_LIST);

        applier.apply([
            ['ins', 2, 0, '<li data-pi="6" data-pon=\'[{"e":"click"}]\'><b><!--t:7-->hi</b><!--r:8--><!--/r:8--></li>'],
        ]);

        expect(applier.node(7)!.nodeValue).toBe('hi');
        expect(applier.node(8)!.nodeType).toBe(Node.COMMENT_NODE);
        (applier.node(6) as HTMLElement).click();
        expect(callbacks.onEvent).toHaveBeenCalledWith(6, 'click', {});
        applier.apply([['t', 7, 'bye'], ['ins', 8, 0, '<i data-pi="9"></i>']]);
        expect(text('li:last-child')).toBe('bye');
    });

    it('starts data-pscript scripts after the unit is in the document', () => {
        const applier = mount(REGION_LIST);
        let connected: boolean | null = null;
        callbacks.onScript.mockImplementation((el: Element) => {
            connected = el.isConnected;
        });

        applier.apply([['ins', 2, 0, `<li data-pi="6" data-pscript="k:6"></li>`]]);

        expect(callbacks.onScript).toHaveBeenCalledWith(applier.node(6));
        expect(connected).toBe(true);
    });
});

describe('Applier mv', () => {
    it('moves a unit before a reference and keeps the node identity', () => {
        const applier = mount(REGION_LIST);
        const [a, b, c] = Array.from(document.querySelectorAll('li'));

        applier.apply([['mv', 5, 2, 3]]);
        expect(order('ul')).toEqual(['c', 'a', 'b']);

        applier.apply([['mv', 5, 2, 0]]);
        expect(order('ul')).toEqual(['a', 'b', 'c']);

        expect(applier.node(3)).toBe(a);
        expect(applier.node(4)).toBe(b);
        expect(applier.node(5)).toBe(c);
        expect(Array.from(document.querySelectorAll('li'))).toEqual([a, b, c]);
    });

    it('moves relative to the ref, not to an index', () => {
        const applier = mount(REGION_LIST);

        applier.apply([['mv', 3, 2, 5]]);

        expect(order('ul')).toEqual(['b', 'a', 'c']);
    });

    it('moves a region unit together with everything inside it', () => {
        const applier = mount(
            '<ul data-pi="1"><!--r:2-->' +
                '<!--r:3--><li data-pi="4">x</li>mid<li data-pi="5">y</li><!--/r:3-->' +
                '<li data-pi="6">z</li>' +
                '<!--/r:2--></ul>',
        );
        const x = applier.node(4);

        applier.apply([['mv', 3, 2, 0]]);

        expect(document.querySelector('ul')!.innerHTML).toBe(
            '<!--r:2--><li data-pi="6">z</li><!--r:3--><li data-pi="4">x</li>mid<li data-pi="5">y</li><!--/r:3--><!--/r:2-->',
        );
        expect(applier.node(4)).toBe(x);
    });

    it('keeps listeners and scripts on a moved unit', () => {
        const applier = mount(
            `<ul data-pi="1"><!--r:2--><li data-pi="3" data-pon='[{"e":"click"}]'>a</li><li data-pi="4">b</li><!--/r:2--></ul>`,
        );

        applier.apply([['mv', 3, 2, 0]]);
        (applier.node(3) as HTMLElement).click();

        expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
        expect(callbacks.onScriptCleanup).not.toHaveBeenCalled();
    });

    it('rejects moving a unit before itself', () => {
        const applier = mount(REGION_LIST);

        expect(failure(applier, [['mv', 4, 2, 4]]).reason).toBe('unresolved');
    });

    it('rejects a ref that is not inside the region', () => {
        const applier = mount(
            '<ul data-pi="1"><!--r:2--><li data-pi="3">a</li><!--/r:2--><li data-pi="4">out</li></ul>',
        );

        expect(failure(applier, [['mv', 3, 2, 4]]).reason).toBe('unresolved');
        expect(failure(applier, [['ins', 2, 4, '<li data-pi="9"></li>']]).reason).toBe('unresolved');
    });

    it('rejects a unit that is not inside the region', () => {
        const applier = mount(
            '<ul data-pi="1"><!--r:2--><li data-pi="3">a</li><!--/r:2--><li data-pi="4">out</li></ul>',
        );

        expect(failure(applier, [['mv', 4, 2, 0]]).reason).toBe('unresolved');
    });
});

describe('Applier rm', () => {
    it('removes an element unit and never reuses its ids', () => {
        const applier = mount(REGION_LIST);

        applier.apply([['rm', 4]]);

        expect(order('ul')).toEqual(['a', 'c']);
        expect(applier.node(4)).toBeUndefined();
        expect(failure(applier, [['t', 4, 'x']]).reason).toBe('unresolved');
        expect(failure(applier, [['rm', 4]]).reason).toBe('unresolved');
    });

    it('removes a region unit including its end marker', () => {
        const applier = mount('<ul data-pi="1"><!--r:2--><!--r:3--><li data-pi="4">x</li>mid<!--/r:3--><li data-pi="5">y</li><!--/r:2--></ul>');

        applier.apply([['rm', 3]]);

        expect(document.querySelector('ul')!.innerHTML).toBe('<!--r:2--><li data-pi="5">y</li><!--/r:2-->');
        expect(applier.node(3)).toBeUndefined();
        expect(applier.node(4)).toBeUndefined();
        expect(applier.node(5)).toBeDefined();
    });

    it('walks the removed subtree: ids, text slots, nested regions and scripts are released', () => {
        const script = 'k:1';
        const applier = mount(
            `<ul data-pi="1"><!--r:2-->` +
                `<li data-pi="3"><b data-pi="4"><!--t:5-->t</b><div data-pi="6" data-pscript='${script}'></div><!--r:7--><i data-pi="8"></i><!--/r:7--></li>` +
                `<li data-pi="9">keep</li>` +
                `<!--/r:2--></ul>`,
        );
        expect(callbacks.onScript).toHaveBeenCalledTimes(1);
        const host = applier.node(6);

        applier.apply([['rm', 3]]);

        for (const id of [3, 4, 5, 6, 7, 8]) {
            expect(applier.node(id)).toBeUndefined();
        }
        expect(applier.node(9)).toBeDefined();
        expect(callbacks.onScriptCleanup).toHaveBeenCalledTimes(1);
        expect(callbacks.onScriptCleanup).toHaveBeenCalledWith(host);
        expect(failure(applier, [['ins', 7, 0, '<i data-pi="10"></i>']]).reason).toBe('unresolved');
    });

    it('clr releases scripts too', () => {
        const applier = mount(`<ul data-pi="1"><!--r:2--><li data-pi="3" data-pscript="k:1"></li><!--/r:2--></ul>`);
        const li = applier.node(3);

        applier.apply([['clr', 2]]);

        expect(callbacks.onScriptCleanup).toHaveBeenCalledWith(li);
    });

    it('rejects a text slot as a unit', () => {
        const applier = mount('<p><!--t:2-->x</p>');

        expect(failure(applier, [['rm', 2]]).reason).toBe('unresolved');
    });
});

describe('Applier strictness', () => {
    it.each([
        ['an unknown id', ['t', 99, 'x']],
        ['t on a region marker', ['t', 2, 'x']],
        ['a on a text slot', ['a', 6, 'title', 'x']],
        ['ra on a region marker', ['ra', 2, 'title']],
        ['st on a text slot', ['st', 6, 'color', 'red']],
        ['h on an element that is not a raw host', ['h', 1, '<b></b>']],
        ['ins into an element instead of a region', ['ins', 1, 0, '<b></b>']],
        ['ins into an unknown region', ['ins', 99, 0, '<b></b>']],
        ['ins before an unknown ref', ['ins', 2, 99, '<b></b>']],
        ['mv of an unknown unit', ['mv', 99, 2, 0]],
        ['mv into an element instead of a region', ['mv', 3, 1, 0]],
        ['rm of an unknown unit', ['rm', 99]],
        ['clr of an element', ['clr', 1]],
    ])('%s is unresolved', (_label, edit) => {
        const applier = mount(`${REGION_LIST}<p><!--t:6-->x</p>`);

        expect(failure(applier, [edit]).reason).toBe('unresolved');
    });

    it.each([
        ['not an array', 'nope'],
        ['an empty tuple', []],
        ['a non-string op', [5, 1]],
        ['an unknown op', ['zz', 1]],
        ['too few arguments', ['t', 1]],
        ['too many arguments', ['t', 1, 'x', 'y']],
        ['a string id', ['t', '1', 'x']],
        ['a zero id', ['t', 0, 'x']],
        ['a negative id', ['rm', -1]],
        ['a fractional id', ['rm', 1.5]],
        ['an id beyond uint32', ['rm', 4294967296]],
        ['a non-string value', ['t', 1, 5]],
        ['a non-string html', ['ins', 2, 0, 5]],
        ['a negative ref', ['mv', 3, 2, -1]],
        ['a null argument', ['a', 1, 'title', null]],
        ['an inherited op name', ['constructor', 1]],
    ])('%s is an apply-error', (_label, edit) => {
        const applier = mount(REGION_LIST);

        expect(failure(applier, [edit]).reason).toBe('apply-error');
    });

    it('a duplicate id in inserted html is an apply-error and inserts nothing', () => {
        const applier = mount(REGION_LIST);

        const error = failure(applier, [['ins', 2, 0, '<li data-pi="3">dup</li>']]);

        expect(error.reason).toBe('apply-error');
        expect(order('ul')).toEqual(['a', 'b', 'c']);
    });

    it('malformed handler json in inserted html is an apply-error', () => {
        const applier = mount(REGION_LIST);

        expect(failure(applier, [['ins', 2, 0, '<li data-pi="6" data-pon="{nope"></li>']]).reason).toBe('apply-error');
        expect(failure(applier, [['ins', 2, 0, '<li data-pi="7" data-pon=\'[{"x":1}]\'></li>']]).reason).toBe('apply-error');
    });

    it('a handler element without an id is an apply-error', () => {
        const applier = mount(REGION_LIST);

        expect(failure(applier, [['ins', 2, 0, '<li data-pon=\'[{"e":"click"}]\'></li>']]).reason).toBe('apply-error');
    });

    it('does not start scripts of an insert that failed', () => {
        const applier = mount(REGION_LIST);

        failure(applier, [['ins', 2, 0, `<li data-pi="6" data-pscript="k:1"></li><li data-pi="3"></li>`]]);
        applier.apply([['t', 3, 'ok']]);

        expect(callbacks.onScript).not.toHaveBeenCalled();
    });

    it('stops at the first failing edit of a frame', () => {
        const applier = mount(REGION_LIST);

        failure(applier, [['t', 3, 'first'], ['t', 99, 'x'], ['t', 4, 'never']]);

        expect(order('ul')).toEqual(['first', 'b', 'c']);
    });
});

describe('Applier exit attributes', () => {
    it('sets and removes data-exiting and inert like any attribute', () => {
        const applier = mount(REGION_LIST);
        const row = document.querySelector('li')!;

        applier.apply([['a', 3, 'data-exiting', ''], ['a', 3, 'inert', '']]);
        expect(row.hasAttribute('data-exiting')).toBe(true);
        expect(row.hasAttribute('inert')).toBe(true);

        applier.apply([['ra', 3, 'data-exiting'], ['ra', 3, 'inert']]);
        expect(row.hasAttribute('data-exiting')).toBe(false);
        expect(row.hasAttribute('inert')).toBe(false);
        expect(applier.node(3)).toBe(row);
    });
});

describe('Applier listeners', () => {
    function button(spec: unknown[], extra = ''): HTMLElement {
        const applier = mount(`<button data-pi="1" ${extra} data-pon='${JSON.stringify(spec)}'>go</button>`);
        void applier;
        return document.querySelector('button')!;
    }

    it('sends the element id and event name', () => {
        const el = button([{ e: 'click' }]);

        el.click();

        expect(callbacks.onEvent).toHaveBeenCalledWith(1, 'click', {});
    });

    it('binds several events to the same id', () => {
        const el = button([{ e: 'click' }, { e: 'input' }]);

        el.click();
        el.dispatchEvent(new Event('input'));
        el.dispatchEvent(new Event('change'));

        expect(callbacks.onEvent.mock.calls.map((c) => c[1])).toEqual(['click', 'input']);
    });

    it('prevent and stop', () => {
        const el = button([{ e: 'click', pv: 1, sp: 1 }]);
        const outer = vi.fn();
        document.body.addEventListener('click', outer);
        const event = new MouseEvent('click', { bubbles: true, cancelable: true });

        el.dispatchEvent(event);
        document.body.removeEventListener('click', outer);

        expect(event.defaultPrevented).toBe(true);
        expect(outer).not.toHaveBeenCalled();
        expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
    });

    it('does not prevent or stop by default', () => {
        const el = button([{ e: 'click' }]);
        const outer = vi.fn();
        document.body.addEventListener('click', outer);
        const event = new MouseEvent('click', { bubbles: true, cancelable: true });

        el.dispatchEvent(event);
        document.body.removeEventListener('click', outer);

        expect(event.defaultPrevented).toBe(false);
        expect(outer).toHaveBeenCalledTimes(1);
    });

    it('once fires a single time', () => {
        const el = button([{ e: 'click', o: 1 }]);

        el.click();
        el.click();

        expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
    });

    it('a once listener with a key filter is not consumed by another key', () => {
        mount(`<input data-pi="1" data-pon='[{"e":"keydown","o":1,"k":["Enter"],"p":"key"}]'>`);
        const input = document.querySelector('input')!;

        input.dispatchEvent(new KeyboardEvent('keydown', { key: 'a' }));
        input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
        input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));

        expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
        expect(callbacks.onEvent).toHaveBeenCalledWith(1, 'keydown', { key: 'Enter' });
    });

    it('capture runs before the target phase of a child', () => {
        mount(`<div data-pi="1" data-pon='[{"e":"click","c":1}]'><span data-pi="2" data-pon='[{"e":"click"}]'>x</span></div>`);

        document.querySelector('span')!.click();

        expect(callbacks.onEvent.mock.calls.map((c) => c[0])).toEqual([1, 2]);
    });

    it('bubbling handlers run after the child', () => {
        mount(`<div data-pi="1" data-pon='[{"e":"click"}]'><span data-pi="2" data-pon='[{"e":"click"}]'>x</span></div>`);

        document.querySelector('span')!.click();

        expect(callbacks.onEvent.mock.calls.map((c) => c[0])).toEqual([2, 1]);
    });

    it('passive listeners cannot prevent default', () => {
        const el = button([{ e: 'wheel', ps: 1, pv: 1 }]);
        const event = new Event('wheel', { cancelable: true });

        el.dispatchEvent(event);

        expect(event.defaultPrevented).toBe(false);
        expect(callbacks.onEvent).toHaveBeenCalledWith(1, 'wheel', {});
    });

    it('a non-passive wheel listener may prevent default', () => {
        const el = button([{ e: 'wheel', pv: 1 }]);
        const event = new Event('wheel', { cancelable: true });

        el.dispatchEvent(event);

        expect(event.defaultPrevented).toBe(true);
    });

    it('listens to non-bubbling events directly: focus and mouseenter', () => {
        mount(`<div><input data-pi="1" data-pon='[{"e":"focus"},{"e":"blur"}]'><p data-pi="2" data-pon='[{"e":"mouseenter"}]'>hover</p></div>`);

        document.querySelector('input')!.dispatchEvent(new FocusEvent('focus', { bubbles: false }));
        document.querySelector('input')!.dispatchEvent(new FocusEvent('blur', { bubbles: false }));
        document.querySelector('p')!.dispatchEvent(new MouseEvent('mouseenter', { bubbles: false }));

        expect(callbacks.onEvent.mock.calls.map((c) => [c[0], c[1]])).toEqual([
            [1, 'focus'],
            [1, 'blur'],
            [2, 'mouseenter'],
        ]);
    });

    it('sends the listed fields, from the event first and then the target, and omits what is missing', () => {
        mount(`<input data-pi="1" data-pon='[{"e":"input","p":"value type clientX missing"}]'>`);
        const input = document.querySelector('input')!;
        input.value = 'hey';

        input.dispatchEvent(new Event('input'));

        expect(callbacks.onEvent).toHaveBeenCalledWith(1, 'input', { value: 'hey', type: 'input' });
    });

    it('debounce sends only the last event, with that event data', () => {
        vi.useFakeTimers();
        try {
            mount(`<input data-pi="1" data-pon='[{"e":"input","p":"value","db":300}]'>`);
            const input = document.querySelector('input')!;

            for (const value of ['a', 'ab', 'abc']) {
                input.value = value;
                input.dispatchEvent(new Event('input'));
                vi.advanceTimersByTime(100);
            }
            expect(callbacks.onEvent).not.toHaveBeenCalled();
            vi.advanceTimersByTime(300);

            expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
            expect(callbacks.onEvent).toHaveBeenCalledWith(1, 'input', { value: 'abc' });
        } finally {
            vi.useRealTimers();
        }
    });

    it('throttle sends the first event and drops the rest of the window', () => {
        vi.useFakeTimers();
        try {
            mount(`<div data-pi="1" data-pon='[{"e":"scroll","th":100}]'></div>`);
            const el = document.querySelector('div')!;

            el.dispatchEvent(new Event('scroll'));
            vi.advanceTimersByTime(50);
            el.dispatchEvent(new Event('scroll'));
            vi.advanceTimersByTime(60);
            el.dispatchEvent(new Event('scroll'));

            expect(callbacks.onEvent).toHaveBeenCalledTimes(2);
        } finally {
            vi.useRealTimers();
        }
    });

    it('a key filter drops other keys before preventing, stopping or sending', () => {
        mount(`<input data-pi="1" data-pon='[{"e":"keydown","pv":1,"sp":1,"k":["Enter","Escape"],"p":"key"}]'>`);
        const input = document.querySelector('input')!;
        const other = new KeyboardEvent('keydown', { key: 'a', cancelable: true });
        const enter = new KeyboardEvent('keydown', { key: 'Enter', cancelable: true });

        input.dispatchEvent(other);
        input.dispatchEvent(enter);

        expect(other.defaultPrevented).toBe(false);
        expect(enter.defaultPrevented).toBe(true);
        expect(callbacks.onEvent.mock.calls).toEqual([[1, 'keydown', { key: 'Enter' }]]);
    });

    it('the pointer is captured inside the pointerdown listener when cp is set', () => {
        mount(`<div data-pi="1" data-pon='[{"e":"pointerdown","cp":1}]'></div>`);
        const el = document.querySelector('div')!;
        const capture = vi.fn();
        (el as any).setPointerCapture = capture;
        const e = new Event('pointerdown');
        Object.defineProperty(e, 'pointerId', { value: 9 });

        el.dispatchEvent(e);

        expect(capture).toHaveBeenCalledWith(9);
    });

    it('~change listens to input on a text field and to change on a checkbox', () => {
        mount(`<div><input data-pi="1" data-pon='[{"e":"~change","p":"value"}]'><input data-pi="2" type="checkbox" data-pon='[{"e":"~change","p":"checked"}]'></div>`);
        const [text, box] = Array.from(document.querySelectorAll('input'));
        text.value = 'hi';
        text.dispatchEvent(new Event('change'));
        expect(callbacks.onEvent).not.toHaveBeenCalled();
        text.dispatchEvent(new Event('input'));
        box.checked = true;
        box.dispatchEvent(new Event('input'));
        box.dispatchEvent(new Event('change'));

        expect(callbacks.onEvent.mock.calls).toEqual([
            [1, '~change', { value: 'hi' }],
            [2, '~change', { checked: true }],
        ]);
    });

    it('throttle delivers the last value of a window after the first one', () => {
        vi.useFakeTimers();
        try {
            mount(`<div data-pi="1" data-pon='[{"e":"mousemove","th":100,"p":"clientX"}]'></div>`);
            const el = document.querySelector('div')!;
            for (const x of [1, 2, 3, 4]) el.dispatchEvent(new MouseEvent('mousemove', { clientX: x }));
            expect(callbacks.onEvent.mock.calls.map((c) => c[2].clientX)).toEqual([1]);
            vi.advanceTimersByTime(100);
            expect(callbacks.onEvent.mock.calls.map((c) => c[2].clientX)).toEqual([1, 4]);
        } finally {
            vi.useRealTimers();
        }
    });

    it('a click sends the debounced input before itself', () => {
        vi.useFakeTimers();
        try {
            mount(`<div><input data-pi="1" data-pon='[{"e":"input","db":300,"p":"value"}]'><button data-pi="2" data-pon='[{"e":"click"}]'>go</button></div>`);
            const input = document.querySelector('input')!;
            input.value = 'typed';
            input.dispatchEvent(new Event('input'));

            document.querySelector('button')!.click();

            expect(callbacks.onEvent.mock.calls.map((c) => [c[0], c[1]])).toEqual([[1, 'input'], [2, 'click']]);
            vi.advanceTimersByTime(1000);
            expect(callbacks.onEvent).toHaveBeenCalledTimes(2);
        } finally {
            vi.useRealTimers();
        }
    });

    it('once removes the listener after the first event', () => {
        mount(`<button data-pi="1" data-pon='[{"e":"click","o":1}]'>go</button>`);
        const button = document.querySelector('button')!;
        button.click();
        button.click();
        expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
    });

    it('load is sent for an image that finished loading before it was bound', async () => {
        document.head.innerHTML = '';
        document.body.innerHTML = '<img data-pi="1" data-pon=\'[{"e":"load"}]\'>';
        const img = document.querySelector('img')!;
        Object.defineProperty(img, 'complete', { value: true });
        Object.defineProperty(img, 'naturalWidth', { value: 10 });
        new Applier(document.documentElement, callbacks as ApplierCallbacks).index();

        await Promise.resolve();

        expect(callbacks.onEvent).toHaveBeenCalledWith(1, 'load', {});
    });

    it('a real load that was already queued does not send a second load', async () => {
        document.head.innerHTML = '';
        document.body.innerHTML = '<img data-pi="1" data-pon=\'[{"e":"load"}]\'>';
        const img = document.querySelector('img')!;
        Object.defineProperty(img, 'complete', { value: true });
        Object.defineProperty(img, 'naturalWidth', { value: 10 });
        new Applier(document.documentElement, callbacks as ApplierCallbacks).index();

        await Promise.resolve();
        img.dispatchEvent(new Event('load'));

        expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
    });

    it('a real load of the first URL is sent again after the src moved away and came back', async () => {
        document.head.innerHTML = '';
        document.body.innerHTML = '<img data-pi="1" data-pon=\'[{"e":"load"}]\'>';
        const img = document.querySelector('img')!;
        let current = 'http://x/a.png';
        Object.defineProperty(img, 'currentSrc', { get: () => current });
        Object.defineProperty(img, 'complete', { value: true });
        Object.defineProperty(img, 'naturalWidth', { value: 10 });
        const applier = new Applier(document.documentElement, callbacks as ApplierCallbacks);
        applier.index();
        await Promise.resolve();
        expect(callbacks.onEvent).toHaveBeenCalledTimes(1);

        applier.apply([['a', 1, 'src', 'http://x/b.png']]);
        current = 'http://x/b.png';
        img.dispatchEvent(new Event('load'));
        expect(callbacks.onEvent).toHaveBeenCalledTimes(2);

        applier.apply([['a', 1, 'src', 'http://x/a.png']]);
        current = 'http://x/a.png';
        img.dispatchEvent(new Event('load'));
        expect(callbacks.onEvent).toHaveBeenCalledTimes(3);
    });

    it('a real load of the same URL is sent when the src attribute is removed and set again', async () => {
        document.head.innerHTML = '';
        document.body.innerHTML = '<img data-pi="1" data-pon=\'[{"e":"load"}]\'>';
        const img = document.querySelector('img')!;
        Object.defineProperty(img, 'currentSrc', { value: 'http://x/a.png' });
        Object.defineProperty(img, 'complete', { value: true });
        Object.defineProperty(img, 'naturalWidth', { value: 10 });
        const applier = new Applier(document.documentElement, callbacks as ApplierCallbacks);
        applier.index();
        await Promise.resolve();

        applier.apply([['ra', 1, 'src']]);
        img.dispatchEvent(new Event('load'));
        expect(callbacks.onEvent).toHaveBeenCalledTimes(2);
    });

    it('load is not invented for an image that has not loaded', async () => {
        document.body.innerHTML = '<img data-pi="1" data-pon=\'[{"e":"load"}]\'>';
        new Applier(document.documentElement, callbacks as ApplierCallbacks).index();

        await Promise.resolve();

        expect(callbacks.onEvent).not.toHaveBeenCalled();
    });

    it('binds an element only once when it is indexed again', () => {
        const applier = mount(`<button data-pi="1" data-pon='[{"e":"click"}]'>go</button>`);

        applier.index();
        applier.index();
        document.querySelector('button')!.click();

        expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
    });
});

describe('Applier scripts', () => {
    const ran = () => document.documentElement.getAttribute('data-ran');

    it('inserted <script> does not execute', () => {
        const applier = mount(REGION_LIST);

        applier.apply([['ins', 2, 0, '<li data-pi="6"><script>document.documentElement.setAttribute("data-ran", "1");</script></li>']]);

        expect(document.querySelector('li script')).not.toBeNull();
        expect(ran()).toBeNull();
    });

    it('does not copy or recreate script attributes on insert', () => {
        const applier = mount(REGION_LIST);

        applier.apply([['ins', 2, 0, '<li data-pi="6"><script type="text/plain" data-x="1">document.documentElement.setAttribute("data-ran", "1");</script></li>']]);

        const script = document.querySelector('li script')!;
        expect(script.getAttribute('type')).toBe('text/plain');
        expect(script.getAttribute('data-x')).toBe('1');
        expect(ran()).toBeNull();
    });

    it('does not execute a script inside a raw host in an insert', () => {
        const applier = mount(REGION_LIST);

        applier.apply([['ins', 2, 0, '<li data-pi="6" data-praw=""><script>document.documentElement.setAttribute("data-ran", "1");</script></li>']]);

        expect(document.querySelector('li[data-praw] script')).not.toBeNull();
        expect(ran()).toBeNull();
    });

    it('does not execute scripts set through h', () => {
        const applier = mount('<div data-pi="1" data-praw=""></div>');

        applier.apply([['h', 1, '<script>document.documentElement.setAttribute("data-ran", "1");</script><b>x</b>']]);

        expect(ran()).toBeNull();
        expect(document.querySelector('div b')).not.toBeNull();
    });

    it('does not execute scripts brought in by a reset morph', () => {
        const applier = mount('<p data-pi="1">x</p>');

        applier.morph('<html><head></head><body><p data-pi="1">x</p><script>document.documentElement.setAttribute("data-ran", "1");</script></body></html>');

        expect(document.querySelector('body script')).not.toBeNull();
        expect(ran()).toBeNull();
    });

    it('asks for a reconcile of data-pscript elements at boot and again on re-index', () => {
        const applier = mount(`<div data-pi="1" data-pscript="k:1"></div>`);

        applier.index();

        expect(callbacks.onScript).toHaveBeenCalledTimes(2);
        expect(callbacks.onScript).toHaveBeenCalledWith(document.querySelector('div'));
    });

    it('destroy cleans up every host', () => {
        const applier = mount(`<div data-pi="1" data-pscript="k:1"></div>`);

        applier.destroy();

        expect(callbacks.onScriptCleanup).toHaveBeenCalledWith(document.querySelector('div'));
    });

    it('asks for one reconcile per host after the whole frame, for attribute edits', () => {
        const applier = mount(`<div data-pi="1" data-pscript="k:1" data-pprops='{"1":1}'></div>`);
        callbacks.onScript.mockClear();
        const seen: string[] = [];
        callbacks.onScript.mockImplementation((el: Element) => seen.push(el.getAttribute('data-pprops')!));

        applier.apply([
            ['a', 1, 'data-pprops', '{"1":2}'],
            ['a', 1, 'data-pprops', '{"1":3}'],
        ]);

        expect(seen).toEqual(['{"1":3}']);
    });

    it('removing data-pscript asks the host to be reconciled so its instances go', () => {
        const applier = mount(`<div data-pi="1" data-pscript="k:1"></div>`);
        callbacks.onScript.mockClear();

        applier.apply([['ra', 1, 'data-pscript']]);

        expect(callbacks.onScript).toHaveBeenCalledWith(document.querySelector('div'));
    });
});

describe('Applier client-owned subtrees', () => {
    const OWNED =
        '<div data-pi="1" data-pown=""><span data-pi="2">a</span><!--r:3--><i data-pi="4">x</i><!--/r:3--><!--t:5-->t</div><p data-pi="6">p</p>';

    it('an edit inside data-pown is an EditError', () => {
        const applier = mount(OWNED);

        for (const edit of [
            ['a', 2, 'title', 'x'],
            ['ra', 2, 'title'],
            ['st', 2, 'color', 'red'],
            ['t', 5, 'new'],
            ['ins', 3, 0, '<b data-pi="9"></b>'],
            ['mv', 4, 3, 0],
            ['rm', 4],
            ['clr', 3],
        ]) {
            const e = failure(applier, [edit]);
            expect(e.message).toContain('client-owned');
        }
        expect(document.querySelector('span')!.hasAttribute('title')).toBe(false);
        expect(document.querySelector('i')).not.toBeNull();
    });

    it('the owner element itself and its siblings stay server-owned', () => {
        const applier = mount(OWNED);

        applier.apply([['a', 1, 'title', 'owner'], ['t', 6, 'q'], ['rm', 1]]);

        expect(document.querySelector('p')!.textContent).toBe('q');
        expect(document.querySelector('[data-pown]')).toBeNull();
    });
});

describe('Applier morph', () => {
    const doc = (body: string, head = '') => `<!DOCTYPE html><html><head>${head}</head><body>${body}</body></html>`;

    it('keeps the identity of every dynamic node and updates what changed', () => {
        const applier = mount(REGION_LIST + '<p data-pi="6"><!--t:7-->old</p>');
        const before = [3, 4, 5, 6].map((id) => applier.node(id));
        const slot = applier.node(7);

        applier.morph(
            doc(
                '<ul data-pi="1"><!--r:2--><li data-pi="3">A</li><li data-pi="4">B</li><li data-pi="5">C</li><!--/r:2--></ul><p data-pi="6"><!--t:7-->new</p>',
            ),
        );

        expect([3, 4, 5, 6].map((id) => applier.node(id))).toEqual(before);
        before.forEach((n, i) => expect(applier.node([3, 4, 5, 6][i])).toBe(n));
        expect(order('ul')).toEqual(['A', 'B', 'C']);
        expect(text('p')).toBe('new');
        expect(applier.node(7)).toBe(slot);
    });

    it('follows a reordered tree by data-pi instead of rebuilding it', () => {
        const applier = mount(REGION_LIST);
        const [a, b, c] = Array.from(document.querySelectorAll('li'));

        applier.morph(
            doc('<ul data-pi="1"><!--r:2--><li data-pi="5">c</li><li data-pi="3">a</li><li data-pi="4">b</li><!--/r:2--></ul>'),
        );

        expect(Array.from(document.querySelectorAll('li'))).toEqual([c, a, b]);
        expect(applier.node(3)).toBe(a);
    });

    it('removes units the reset no longer has and adds new ones', () => {
        const applier = mount(REGION_LIST);

        applier.morph(
            doc('<ul data-pi="1"><!--r:2--><li data-pi="3">a</li><li data-pi="9">new</li><!--/r:2--></ul>'),
        );

        expect(order('ul')).toEqual(['a', 'new']);
        expect(applier.node(4)).toBeUndefined();
        expect(applier.node(9)).toBe(document.querySelectorAll('li')[1]);
    });

    it('re-indexes to exactly what a fresh index of the reset document finds', () => {
        const applier = mount(REGION_LIST);
        const html = doc(
            '<ul data-pi="1"><!--r:2--><li data-pi="3">a</li><li data-pi="9"><!--t:10--></li><!--/r:2--></ul><div><!--r:11--><!--/r:11--></div>',
        );

        applier.morph(html);

        const fresh = new Applier(document.documentElement, callbacks as ApplierCallbacks);
        fresh.index();
        for (const id of [1, 2, 3, 4, 5, 9, 10, 11]) {
            expect(applier.node(id)).toBe(fresh.node(id));
        }
        applier.apply([['t', 10, 'filled'], ['ins', 11, 0, '<i data-pi="12"></i>']]);
        expect(document.querySelector('div')!.querySelector('i')).not.toBeNull();
    });

    it('does not bind a kept element twice', () => {
        const applier = mount(`<button data-pi="1" data-pon='[{"e":"click"}]'>go</button>`);
        const button = document.querySelector('button')!;

        applier.morph(doc(`<button data-pi="1" data-pon='[{"e":"click"}]'>go</button>`));
        applier.morph(doc(`<button data-pi="1" data-pon='[{"e":"click"}]'>go</button>`));
        button.click();

        expect(document.querySelector('button')).toBe(button);
        expect(callbacks.onEvent).toHaveBeenCalledTimes(1);
    });

    it('binds elements that are new in the reset', () => {
        const applier = mount('<p data-pi="1"></p>');

        applier.morph(doc(`<p data-pi="1"></p><button data-pi="2" data-pon='[{"e":"click"}]'>go</button>`));
        document.querySelector('button')!.click();

        expect(callbacks.onEvent).toHaveBeenCalledWith(2, 'click', {});
    });

    it('reconciles kept and new script hosts and cleans up none of the kept ones', () => {
        const applier = mount(`<div data-pi="1" data-pscript="k:1"></div>`);
        const kept = document.querySelector('div');
        callbacks.onScript.mockClear();

        applier.morph(doc(`<div data-pi="1" data-pscript="k:1"></div><div data-pi="2" data-pscript="k:2"></div>`));

        expect(callbacks.onScript.mock.calls.map((c) => (c[0] as Element).getAttribute('data-pi')).sort()).toEqual(['1', '2']);
        expect(document.querySelector('div')).toBe(kept);
        expect(callbacks.onScriptCleanup).not.toHaveBeenCalled();
    });

    it('cleans up the scripts of elements the reset removed', () => {
        const applier = mount(`<div data-pi="1" data-pscript="k:1"></div><p data-pi="2"></p>`);
        const gone = document.querySelector('div');

        applier.morph(doc('<p data-pi="2"></p>'));

        expect(callbacks.onScriptCleanup).toHaveBeenCalledTimes(1);
        expect(callbacks.onScriptCleanup).toHaveBeenCalledWith(gone);
    });

    it('asks a kept host to reconcile when the reset changes its data-pscript, and when it drops it', () => {
        const applier = mount(`<div data-pi="1" data-pscript="k:1"></div><p data-pi="2" data-pscript="k:2"></p>`);
        callbacks.onScript.mockClear();

        applier.morph(doc(`<div data-pi="1" data-pscript="k:3"></div><p data-pi="2"></p>`));

        const asked = callbacks.onScript.mock.calls.map((c) => (c[0] as Element).tagName);
        expect(asked.sort()).toEqual(['DIV', 'P']);
        expect(document.querySelector('div')!.getAttribute('data-pscript')).toBe('k:3');
    });

    it('morph keeps children under data-pown and still updates its attributes', () => {
        const applier = mount(`<div data-pi="1" data-pown="" title="a"><span class="client">made by script</span></div>`);
        const span = document.querySelector('span');

        applier.morph(doc(`<div data-pi="1" data-pown="" title="b"><b>server child</b></div>`));

        expect(document.querySelector('div')!.getAttribute('title')).toBe('b');
        expect(document.querySelector('span')).toBe(span);
        expect(document.querySelector('div b')).toBeNull();
    });

    it('morphs the children of an element that is not client-owned', () => {
        const applier = mount(`<div data-pi="1" title="a"><span>old</span></div>`);

        applier.morph(doc(`<div data-pi="1" title="b"><b>new</b></div>`));

        expect(document.querySelector('div b')).not.toBeNull();
        expect(document.querySelector('div span')).toBeNull();
    });

    it('carries the exit attributes and drops them when the reset has none', () => {
        const applier = mount(REGION_LIST);
        applier.apply([['a', 4, 'data-exiting', ''], ['a', 4, 'inert', '']]);
        const row = applier.node(4) as HTMLElement;

        applier.morph(
            doc(
                '<ul data-pi="1"><!--r:2--><li data-pi="3">a</li><li data-pi="4" data-exiting="" inert="">b</li><li data-pi="5">c</li><!--/r:2--></ul>',
            ),
        );
        expect(row.hasAttribute('data-exiting') && row.hasAttribute('inert')).toBe(true);

        applier.morph(doc('<ul data-pi="1"><!--r:2--><li data-pi="3">a</li><li data-pi="4">b</li><li data-pi="5">c</li><!--/r:2--></ul>'));
        expect(row.hasAttribute('data-exiting') || row.hasAttribute('inert')).toBe(false);
        expect(applier.node(4)).toBe(row);
    });

    it('preserves what the user typed into an input the reset leaves without a value', () => {
        const applier = mount('<input data-pi="1">');
        const input = document.querySelector('input')!;
        input.value = 'typed';

        applier.morph(doc('<input data-pi="1">'));

        expect(document.querySelector('input')).toBe(input);
        expect(input.value).toBe('typed');
    });

    it('lets the reset win when it carries a value', () => {
        const applier = mount('<input data-pi="1">');
        const input = document.querySelector('input')!;
        input.value = 'typed';

        applier.morph(doc('<input data-pi="1" value="server">'));

        expect(input.value).toBe('server');
    });

    it('preserves a dirty textarea the reset leaves empty', () => {
        const applier = mount('<textarea data-pi="1"></textarea>');
        const area = document.querySelector('textarea')!;
        area.value = 'notes';

        applier.morph(doc('<textarea data-pi="1"></textarea>'));

        expect(area.value).toBe('notes');
    });

    it('keeps the user choice in a select, falls back to the first option, and honours an explicit selected', () => {
        const options = '<option value="a">a</option><option value="b">b</option><option value="c">c</option>';
        const applier = mount(`<select data-pi="1">${options}</select>`);
        const select = document.querySelector('select')!;
        select.value = 'b';

        applier.morph(doc(`<select data-pi="1">${options}</select>`));
        expect(document.querySelector('select')).toBe(select);
        expect(select.value).toBe('b');

        applier.morph(doc('<select data-pi="1"><option value="x">x</option><option value="y">y</option></select>'));
        expect(select.value).toBe('x');
        expect(select.querySelector('option[selected]')).toBeNull();

        applier.morph(doc('<select data-pi="1"><option value="x">x</option><option value="y" selected>y</option></select>'));
        expect(select.value).toBe('y');
    });

    it('syncs html attributes from the reset document', () => {
        const applier = mount('<p data-pi="1"></p>');
        document.documentElement.setAttribute('class', 'old');

        applier.morph('<!DOCTYPE html><html class="new"><head></head><body><p data-pi="1"></p></body></html>');

        expect(document.documentElement.getAttribute('class')).toBe('new');
        document.documentElement.removeAttribute('class');
    });
});

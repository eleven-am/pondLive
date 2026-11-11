import { describe, it, expect, afterEach } from 'vitest';
import { domEvent, readFields } from './fields';

afterEach(() => {
    document.body.innerHTML = '';
});

function fire(el: Element, e: Event): Record<string, unknown> & { names: string } {
    let out: any;
    el.addEventListener(e.type, (ev) => (out = ev));
    el.dispatchEvent(e);
    return out;
}

describe('readFields', () => {
    it('reads event fields first and target fields second, and skips missing ones', () => {
        const input = document.createElement('input');
        document.body.append(input);
        input.value = 'x';
        const e = fire(input, new MouseEvent('click', { clientX: 4, ctrlKey: true })) as unknown as Event;
        expect(readFields(e, 'clientX ctrlKey value nope')).toEqual({ clientX: 4, ctrlKey: true, value: 'x' });
    });

    it('an event field wins over a target property of the same name', () => {
        const img = document.createElement('img');
        img.setAttribute('width', '50');
        expect(readFields({ target: img, width: 3 } as unknown as Event, 'width')).toEqual({ width: 3 });
        expect(readFields({ target: img } as unknown as Event, 'width')).toEqual({ width: 50 });
    });

    it('value reads innerText for contenteditable', () => {
        const div = document.createElement('div');
        Object.defineProperty(div, 'isContentEditable', { value: true });
        Object.defineProperty(div, 'innerText', { value: 'typed' });
        expect(readFields({ target: div } as unknown as Event, 'value')).toEqual({ value: 'typed' });
    });

    it('reads checked from the target', () => {
        const box = document.createElement('input');
        box.type = 'checkbox';
        box.checked = true;
        expect(readFields({ target: box } as unknown as Event, 'checked')).toEqual({ checked: true });
    });

    it('selected lists the values of the selected options', () => {
        document.body.innerHTML = '<select multiple><option value="a" selected>A</option><option value="b">B</option><option value="c" selected>C</option></select>';
        const select = document.querySelector('select')!;
        expect(readFields({ target: select } as unknown as Event, 'selected')).toEqual({ selected: ['a', 'c'] });
    });

    it('files sends metadata only', () => {
        const input = document.createElement('input');
        input.type = 'file';
        const file = new File(['abcd'], 'f.txt', { type: 'text/plain', lastModified: 5 });
        Object.defineProperty(input, 'files', { value: [file] });
        expect(readFields({ target: input } as unknown as Event, 'files')).toEqual({ files: [{ n: 'f.txt', s: 4, t: 'text/plain', m: 5 }] });
    });

    it('touches are compact [id, x, y] triples of the changed touches', () => {
        const e = { target: document.body, changedTouches: [{ identifier: 7, clientX: 1, clientY: 2 }] } as unknown as Event;
        expect(readFields(e, 'touches')).toEqual({ touches: [[7, 1, 2]] });
    });

    it('text reads the clipboard or drag text', () => {
        const clip = { target: document.body, clipboardData: { getData: (t: string) => (t === 'text/plain' ? 'pasted' : '') } } as unknown as Event;
        const drag = { target: document.body, dataTransfer: { getData: () => 'dropped' } } as unknown as Event;
        expect(readFields(clip, 'text')).toEqual({ text: 'pasted' });
        expect(readFields(drag, 'text')).toEqual({ text: 'dropped' });
    });

    it('fullscreen is true only when the target is the fullscreen element', () => {
        const el = document.createElement('div');
        Object.defineProperty(document, 'fullscreenElement', { value: el, configurable: true });
        expect(readFields({ target: el } as unknown as Event, 'fullscreen')).toEqual({ fullscreen: true });
        expect(readFields({ target: document.body } as unknown as Event, 'fullscreen')).toEqual({ fullscreen: false });
        Object.defineProperty(document, 'fullscreenElement', { value: null, configurable: true });
    });

    it('returnValue is the dialog result, not the legacy event property', () => {
        const dialog = document.createElement('dialog');
        dialog.returnValue = 'ok';
        expect(readFields({ target: dialog, returnValue: true } as unknown as Event, 'returnValue')).toEqual({ returnValue: 'ok' });
    });
});

describe('domEvent', () => {
    const cases: Array<[string, () => Element, string]> = [
        ['text input', () => Object.assign(document.createElement('input'), { type: 'text' }), 'input'],
        ['number input', () => Object.assign(document.createElement('input'), { type: 'number' }), 'input'],
        ['range input', () => Object.assign(document.createElement('input'), { type: 'range' }), 'input'],
        ['textarea', () => document.createElement('textarea'), 'input'],
        ['checkbox', () => Object.assign(document.createElement('input'), { type: 'checkbox' }), 'change'],
        ['radio', () => Object.assign(document.createElement('input'), { type: 'radio' }), 'change'],
        ['file', () => Object.assign(document.createElement('input'), { type: 'file' }), 'change'],
        ['select', () => document.createElement('select'), 'change'],
        ['select multiple', () => Object.assign(document.createElement('select'), { multiple: true }), 'change'],
        ['contenteditable', () => Object.defineProperty(document.createElement('div'), 'isContentEditable', { value: true }), 'input'],
    ];
    for (const [name, make, want] of cases) {
        it(`~change on a ${name} listens to ${want}`, () => {
            expect(domEvent(make(), '~change')).toBe(want);
        });
    }

    it('leaves every other event name alone', () => {
        expect(domEvent(document.createElement('input'), 'change')).toBe('change');
        expect(domEvent(document.createElement('input'), 'click')).toBe('click');
    });
});

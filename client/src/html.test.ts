import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const MARK = '<!--tt-->';

let createPolicy: ReturnType<typeof vi.fn>;

async function load() {
    vi.resetModules();
    const html = await import('./html');
    const applier = await import('./applier');
    return { ...html, Applier: applier.Applier };
}

beforeEach(() => {
    createPolicy = vi.fn((_name: string, rules: { createHTML(s: string): string }) => ({
        createHTML: (s: string) => rules.createHTML(s) + MARK,
    }));
    vi.stubGlobal('trustedTypes', { createPolicy });
});

afterEach(() => {
    vi.unstubAllGlobals();
    document.body.innerHTML = '';
});

function mount(Applier: any, body: string) {
    document.body.innerHTML = body;
    const applier = new Applier(document.documentElement, { onEvent: () => {}, onScript: () => {}, onScriptCleanup: () => {} });
    applier.index();
    return applier;
}

const hasMark = (node: Node) => Array.from(node.childNodes).some((n) => n.nodeType === Node.COMMENT_NODE && n.nodeValue === 'tt');

describe('Trusted Types', () => {
    it('toHTML creates the pondlive policy once, lazily', async () => {
        const { toHTML } = await load();
        expect(createPolicy).not.toHaveBeenCalled();

        toHTML('a');
        toHTML('b');

        expect(createPolicy).toHaveBeenCalledTimes(1);
        expect(createPolicy.mock.calls[0][0]).toBe('pondlive');
    });

    it('returns the input when the browser has no Trusted Types', async () => {
        vi.unstubAllGlobals();
        const { toHTML } = await load();

        expect(toHTML('<b>x</b>')).toBe('<b>x</b>');
    });

    it('falls back to the input when the policy cannot be created', async () => {
        createPolicy.mockImplementation(() => {
            throw new TypeError('policy exists');
        });
        const { toHTML } = await load();

        expect(toHTML('<b>x</b>')).toBe('<b>x</b>');
        expect(toHTML('<i>y</i>')).toBe('<i>y</i>');
        expect(createPolicy).toHaveBeenCalledTimes(1);
    });

    it('every HTML sink receives the pondlive policy output; createPolicy is called once', async () => {
        const { Applier } = await load();
        const applier = mount(Applier, '<ul data-pi="1"><!--r:2--><!--/r:2--></ul><div data-pi="5" data-praw=""></div>');

        applier.apply([['ins', 2, 0, '<li data-pi="3">x</li>']]);
        expect(hasMark(document.querySelector('ul')!)).toBe(true);

        applier.apply([['h', 5, '<b>raw</b>']]);
        expect(hasMark(document.querySelector('div')!)).toBe(true);

        const parse = vi.spyOn(DOMParser.prototype, 'parseFromString');
        applier.morph('<!DOCTYPE html><html><head></head><body><p data-pi="9">m</p></body></html>');
        expect(parse).toHaveBeenCalledTimes(1);
        expect(parse.mock.calls[0][0].endsWith(MARK)).toBe(true);

        expect(createPolicy).toHaveBeenCalledTimes(1);
    });
});

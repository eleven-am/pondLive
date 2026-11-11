import { describe, it, expect, vi, beforeEach } from 'vitest';

const boot = vi.fn();

vi.mock('./runtime', () => ({ boot: (...args: unknown[]) => boot(...args), Runtime: class {} }));

async function load() {
    vi.resetModules();
    return import('./index');
}

beforeEach(() => {
    boot.mockClear();
});

describe('core entry', () => {
    it('does not boot on load', async () => {
        document.body.innerHTML = '<script id="live-boot" type="application/json">{}</script>';

        await load();

        expect(boot).not.toHaveBeenCalled();
    });

    it('start() twice boots once and hands the hash to boot', async () => {
        const { start } = await load();

        start('abc123def456');
        start('abc123def456');
        start('other');

        expect(boot).toHaveBeenCalledTimes(1);
        expect(boot).toHaveBeenCalledWith('abc123def456');
    });

    it('define before start works, define after start throws', async () => {
        const { define, start } = await load();

        expect(() => define('k1', () => {})).not.toThrow();
        start('h');

        expect(() => define('k2', () => {})).toThrow(/after start/);
    });

    it('define is visible on the module namespace the page global exposes', async () => {
        const mod = await load();

        expect(typeof mod.define).toBe('function');
        expect(typeof mod.start).toBe('function');
    });

    it('waits for DOMContentLoaded when the document is still loading', async () => {
        const { start } = await load();
        const state = vi.spyOn(document, 'readyState', 'get').mockReturnValue('loading');

        start('h');
        expect(boot).not.toHaveBeenCalled();
        document.dispatchEvent(new Event('DOMContentLoaded'));

        expect(boot).toHaveBeenCalledWith('h');
        state.mockRestore();
    });

    it('repairs a bare # in the URL once, at start', async () => {
        const { start } = await load();
        history.replaceState(null, '', '/p?q=1#');
        expect(window.location.href.endsWith('#')).toBe(true);

        start('h');

        expect(window.location.href.endsWith('#')).toBe(false);
        expect(window.location.pathname + window.location.search).toBe('/p?q=1');
        history.replaceState(null, '', '/');
    });
});

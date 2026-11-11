import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Applier } from './applier';
import { Scripts, ScriptDef, ScriptAction } from './scripts';
import { Logger } from './logger';

type Sent = { id: number; action: ScriptAction; payload: any };

let sent: Sent[];
let defs: Map<string, ScriptDef>;
let scripts: Scripts;
let applier: Applier;

function mount(body: string): void {
    document.body.innerHTML = body;
    applier = new Applier(document.documentElement, {
        onEvent: () => {},
        onScript: (el) => scripts.reconcile(el),
        onScriptCleanup: (el) => scripts.releaseHost(el),
    });
    applier.index();
}

function errors(): Sent[] {
    return sent.filter((s) => s.action === 'err');
}

beforeEach(() => {
    vi.useFakeTimers();
    Logger.configure({ enabled: false });
    sent = [];
    defs = new Map();
    scripts = new Scripts({ send: (id, action, payload) => sent.push({ id, action, payload }), lookup: (k) => defs.get(k) });
});

afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = '';
    document.documentElement.removeAttribute('data-pscript');
    document.documentElement.removeAttribute('data-pprops');
});

describe('Scripts attach', () => {
    it('attaches synchronously and gives the script its element and channel', () => {
        const seen: unknown[] = [];
        defs.set('k', (el, io) => {
            seen.push(el, typeof io.on, typeof io.send, typeof io.onProps, io.props);
        });

        mount('<div data-pi="1" data-pscript="k:1"></div>');

        expect(seen).toEqual([document.querySelector('div'), 'function', 'function', 'function', undefined]);
    });

    it('a message in the same batch as the ins reaches the handler', () => {
        const got: unknown[] = [];
        defs.set('k', (_el, io) => {
            io.on((m) => got.push(m));
        });
        mount('<ul data-pi="1"><!--r:2--><!--/r:2--></ul>');

        applier.apply([['ins', 2, 0, '<li data-pi="3" data-pscript="k:42"></li>']]);
        scripts.deliver(42, { n: 1 });

        expect(got).toEqual([{ n: 1 }]);
    });

    it('defers attaches until the whole frame is applied', () => {
        const order: string[] = [];
        defs.set('k', (el) => {
            order.push(`attach:${el.getAttribute('data-pi')}:${document.querySelector('[data-pi="5"]')?.textContent}`);
        });
        mount('<ul data-pi="1"><!--r:2--><!--/r:2--></ul><p data-pi="5"><!--t:6-->a</p>');

        applier.apply([
            ['ins', 2, 0, '<li data-pi="3" data-pscript="k:1"></li>'],
            ['t', 6, 'finished'],
        ]);

        expect(order).toEqual(['attach:3:finished']);
    });

    it('messages before on() queue in order; the 33rd is dropped and reported once', () => {
        let io!: Parameters<ScriptDef>[1];
        defs.set('k', (_el, channel) => {
            io = channel;
        });
        mount('<div data-pi="1" data-pscript="k:5"></div>');

        for (let n = 1; n <= 40; n++) scripts.deliver(5, n);
        const got: unknown[] = [];
        io.on((m) => got.push(m));

        expect(got).toEqual(Array.from({ length: 32 }, (_, n) => n + 1));
        expect(errors()).toHaveLength(1);
        expect(errors()[0]).toEqual({ id: 5, action: 'err', payload: { n: 'queue', m: expect.any(String) } });
    });

    it('delivers straight to the handler once registered, and a later on() replaces it', () => {
        let io!: Parameters<ScriptDef>[1];
        defs.set('k', (_el, channel) => {
            io = channel;
        });
        mount('<div data-pi="1" data-pscript="k:5"></div>');
        const first = vi.fn();
        const second = vi.fn();

        io.on(first);
        scripts.deliver(5, 'a');
        io.on(second);
        scripts.deliver(5, 'b');

        expect(first.mock.calls).toEqual([['a']]);
        expect(second.mock.calls).toEqual([['b']]);
    });

    it('two instances on one element are independent', () => {
        const a: unknown[] = [];
        const b: unknown[] = [];
        const cleanups: string[] = [];
        defs.set('ka', (_el, io) => {
            io.on((m) => a.push(m));
            return () => cleanups.push('a');
        });
        defs.set('kb', (_el, io) => {
            io.on((m) => b.push(m));
            return () => cleanups.push('b');
        });
        mount('<div data-pi="1" data-pscript="ka:1 kb:2"></div>');

        scripts.deliver(2, 'to-b');
        scripts.deliver(1, 'to-a');
        applier.apply([['a', 1, 'data-pscript', 'ka:1']]);

        expect(a).toEqual(['to-a']);
        expect(b).toEqual(['to-b']);
        expect(cleanups).toEqual(['b']);
    });

    it('document-scoped instance gets documentElement', () => {
        let host: Element | undefined;
        defs.set('k', (el) => {
            host = el;
        });
        document.documentElement.setAttribute('data-pscript', 'k:9');

        mount('<p data-pi="1"></p>');

        expect(host).toBe(document.documentElement);
    });

    it('sends the instance id and the payload', () => {
        defs.set('k', (_el, io) => io.send({ kind: 'ready' }));

        mount('<div data-pi="1" data-pscript="k:5"></div>');

        expect(sent).toEqual([{ id: 5, action: 'm', payload: { kind: 'ready' } }]);
    });

    it('does not send after the instance is gone', () => {
        let io!: Parameters<ScriptDef>[1];
        defs.set('k', (_el, channel) => {
            io = channel;
        });
        mount('<div data-pi="1" data-pscript="k:5"></div>');
        applier.apply([['ra', 1, 'data-pscript']]);

        io.send('late');

        expect(sent).toEqual([]);
    });

    it('ignores malformed tokens and keeps the valid ones', () => {
        const ran: string[] = [];
        defs.set('k', (_el) => {
            ran.push('k');
        });

        mount('<div data-pi="1" data-pscript="nope :3 k:x  k:7 k:0"></div>');

        expect(ran).toEqual(['k']);
    });
});

describe('Scripts identity', () => {
    it('changing data-pscript on a kept element cleans up the old instance and attaches the new one', () => {
        const log: string[] = [];
        defs.set('k', (_el, io) => {
            log.push('attach');
            io.on((m) => log.push(`msg:${String(m)}`));
            return () => log.push('cleanup');
        });
        mount('<div data-pi="1" data-pscript="k:1"></div>');
        const el = document.querySelector('div');

        applier.apply([['a', 1, 'data-pscript', 'k:2']]);
        scripts.deliver(1, 'old');
        scripts.deliver(2, 'new');

        expect(document.querySelector('div')).toBe(el);
        expect(log).toEqual(['attach', 'cleanup', 'attach', 'msg:new']);
    });

    it('a kept element with the same data-pscript is not attached again', () => {
        const attach = vi.fn();
        defs.set('k', attach);
        mount('<div data-pi="1" data-pscript="k:1"></div>');

        applier.index();
        applier.morph('<!DOCTYPE html><html><head></head><body><div data-pi="1" data-pscript="k:1"></div></body></html>');

        expect(attach).toHaveBeenCalledTimes(1);
    });

    it('a reset that changes data-pscript swaps the instances', () => {
        const log: string[] = [];
        defs.set('k', () => {
            log.push('attach');
            return () => log.push('cleanup');
        });
        mount('<div data-pi="1" data-pscript="k:1"></div>');

        applier.morph('<!DOCTYPE html><html><head></head><body><div data-pi="1" data-pscript="k:2"></div></body></html>');

        expect(log).toEqual(['attach', 'cleanup', 'attach']);
    });

    it('removing the element cleans up', () => {
        const cleanup = vi.fn();
        defs.set('k', () => cleanup);
        mount('<ul data-pi="1"><!--r:2--><li data-pi="3" data-pscript="k:1"></li><!--/r:2--></ul>');

        applier.apply([['rm', 3]]);

        expect(cleanup).toHaveBeenCalledTimes(1);
    });
});

describe('Scripts props', () => {
    it('hands the initial props to the script', () => {
        let props: unknown;
        defs.set('k', (_el, io) => {
            props = io.props;
        });

        mount(`<div data-pi="1" data-pscript="k:5" data-pprops='{"5":{"u":"/a","n":1}}'></div>`);

        expect(props).toEqual({ u: '/a', n: 1 });
    });

    it('onProps runs once after the frame, not for unchanged JSON', () => {
        const calls: Array<{ props: unknown; text: string }> = [];
        defs.set('k', (_el, io) => {
            io.onProps((p) => calls.push({ props: p, text: document.querySelector('[data-pi="6"]')!.textContent! }));
        });
        mount(`<div data-pi="1" data-pscript="k:5" data-pprops='{"5":{"n":1}}'></div><p data-pi="6"><!--t:7-->a</p>`);

        applier.apply([
            ['a', 1, 'data-pprops', '{"5":{"n":2}}'],
            ['t', 7, 'done'],
        ]);
        expect(calls).toEqual([{ props: { n: 2 }, text: 'done' }]);

        applier.apply([['a', 1, 'data-pprops', '{"5":{"n":2}}']]);
        applier.index();
        applier.morph(
            '<!DOCTYPE html><html><head></head><body><div data-pi="1" data-pscript="k:5" data-pprops=\'{"5":{"n":2}}\'></div><p data-pi="6"><!--t:7-->done</p></body></html>',
        );
        expect(calls).toHaveLength(1);
    });

    it('onProps only fires for the instance whose props changed', () => {
        const a = vi.fn();
        const b = vi.fn();
        defs.set('ka', (_el, io) => io.onProps(a));
        defs.set('kb', (_el, io) => io.onProps(b));
        mount(`<div data-pi="1" data-pscript="ka:1 kb:2" data-pprops='{"1":1,"2":1}'></div>`);

        applier.apply([['a', 1, 'data-pprops', '{"1":1,"2":9}']]);

        expect(a).not.toHaveBeenCalled();
        expect(b.mock.calls).toEqual([[9]]);
    });

    it('io.props reads the latest value', () => {
        let io!: Parameters<ScriptDef>[1];
        defs.set('k', (_el, channel) => {
            io = channel;
        });
        mount(`<div data-pi="1" data-pscript="k:5" data-pprops='{"5":1}'></div>`);

        applier.apply([['a', 1, 'data-pprops', '{"5":2}']]);

        expect(io.props).toBe(2);
    });

    it('a malformed data-pprops is treated as no props', () => {
        let props: unknown = 'unset';
        defs.set('k', (_el, io) => {
            props = io.props;
        });

        mount(`<div data-pi="1" data-pscript="k:5" data-pprops="{nope"></div>`);

        expect(props).toBeUndefined();
    });
});

describe('Scripts errors', () => {
    it('def throw, handler throw, cleanup throw -> err capped at 1 KiB, at most 1/s', () => {
        const huge = 'x'.repeat(5000);
        let io!: Parameters<ScriptDef>[1];
        defs.set('k', (_el, channel) => {
            io = channel;
            channel.on(() => {
                throw new Error(huge);
            });
            channel.onProps(() => {
                throw new Error('props');
            });
            return () => {
                throw new Error('cleanup');
            };
        });
        defs.set('bad', () => {
            throw new Error(huge);
        });

        mount('<div data-pi="1" data-pscript="bad:1 k:2"></div>');
        expect(errors()).toEqual([{ id: 1, action: 'err', payload: { n: 'def', m: expect.any(String) } }]);
        expect(new TextEncoder().encode(errors()[0].payload.m).length).toBeLessThanOrEqual(1024);

        scripts.deliver(2, 1);
        expect(errors().map((e) => e.payload.n)).toEqual(['def', 'on']);
        expect(new TextEncoder().encode(errors()[1].payload.m).length).toBeLessThanOrEqual(1024);

        scripts.deliver(2, 2);
        scripts.deliver(2, 3);
        expect(errors()).toHaveLength(2);

        vi.advanceTimersByTime(1000);
        scripts.deliver(2, 4);
        expect(errors()).toHaveLength(3);

        vi.advanceTimersByTime(1000);
        applier.apply([['a', 1, 'data-pprops', '{"2":1}']]);
        expect(errors().at(-1)!.payload.n).toBe('props');

        vi.advanceTimersByTime(1000);
        applier.apply([['ra', 1, 'data-pscript']]);
        expect(errors().at(-1)!.payload.n).toBe('cleanup');
        expect(io).toBeDefined();
    });

    it('caps multi-byte messages without splitting a character', () => {
        defs.set('k', () => {
            throw new Error('é'.repeat(2000));
        });

        mount('<div data-pi="1" data-pscript="k:1"></div>');

        const m: string = errors()[0].payload.m;
        expect(new TextEncoder().encode(m).length).toBeLessThanOrEqual(1024);
        expect(m).not.toContain('�');
    });

    it('a rejected async def is reported', async () => {
        vi.useRealTimers();
        defs.set('k', async () => {
            throw new Error('late');
        });

        mount('<div data-pi="1" data-pscript="k:1"></div>');
        await Promise.resolve();
        await Promise.resolve();

        expect(errors()[0].payload.n).toBe('def');
    });

    it('unknown key reports undefined', () => {
        mount('<div data-pi="1" data-pscript="nokey:3"></div>');

        expect(errors()).toEqual([{ id: 3, action: 'err', payload: { n: 'undefined', m: expect.stringContaining('nokey') } }]);
    });

    it('a throwing queued handler during drain does not lose the rest', () => {
        let io!: Parameters<ScriptDef>[1];
        defs.set('k', (_el, channel) => {
            io = channel;
        });
        mount('<div data-pi="1" data-pscript="k:5"></div>');
        scripts.deliver(5, 1);
        scripts.deliver(5, 2);
        const got: unknown[] = [];

        io.on((m) => {
            got.push(m);
            if (m === 1) throw new Error('first');
        });

        expect(got).toEqual([1, 2]);
        expect(errors()[0].payload.n).toBe('on');
    });
});

describe('Scripts async', () => {
    it('async def resolving after removal runs cleanup at once', async () => {
        vi.useRealTimers();
        const cleanup = vi.fn();
        let resolve!: (fn: () => void) => void;
        defs.set('k', () => new Promise<() => void>((r) => (resolve = r)));
        mount('<div data-pi="1" data-pscript="k:1"></div>');
        applier.apply([['ra', 1, 'data-pscript']]);

        resolve(cleanup);
        await Promise.resolve();
        await Promise.resolve();

        expect(cleanup).toHaveBeenCalledTimes(1);
    });

    it('async def resolving while attached keeps cleanup for later', async () => {
        vi.useRealTimers();
        const cleanup = vi.fn();
        defs.set('k', async () => cleanup);
        mount('<div data-pi="1" data-pscript="k:1"></div>');
        await Promise.resolve();
        await Promise.resolve();
        expect(cleanup).not.toHaveBeenCalled();

        applier.apply([['ra', 1, 'data-pscript']]);

        expect(cleanup).toHaveBeenCalledTimes(1);
    });

    it('messages delivered while an async def is pending wait for on()', async () => {
        vi.useRealTimers();
        const got: unknown[] = [];
        defs.set('k', async (_el, io) => {
            await Promise.resolve();
            io.on((m) => got.push(m));
        });
        mount('<div data-pi="1" data-pscript="k:1"></div>');

        scripts.deliver(1, 'early');
        await Promise.resolve();
        await Promise.resolve();

        expect(got).toEqual(['early']);
    });
});

describe('Scripts destroy', () => {
    it('destroy cleans up every instance once', () => {
        const cleanup = vi.fn();
        defs.set('k', () => cleanup);
        mount('<div data-pi="1" data-pscript="k:1"></div><div data-pi="2" data-pscript="k:2"></div>');

        scripts.destroy();
        scripts.destroy();

        expect(cleanup).toHaveBeenCalledTimes(2);
    });

    it('a message for an unknown instance is dropped quietly', () => {
        expect(() => scripts.deliver(99, 1)).not.toThrow();
        expect(sent).toEqual([]);
    });
});

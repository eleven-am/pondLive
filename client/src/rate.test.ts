import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { flush, limit } from './rate';

let sent: Array<[string, unknown]>;

beforeEach(() => {
    sent = [];
    vi.useFakeTimers();
});

afterEach(() => {
    flush();
    vi.useRealTimers();
    vi.unstubAllGlobals();
});

const sender = (name: string) => (d: unknown) => sent.push([name, d]);

describe('limit', () => {
    it('sends every event at once when no rate is set', () => {
        const send = limit<number>(undefined, undefined, sender('a'));
        send(1);
        send(2);
        expect(sent).toEqual([['a', 1], ['a', 2]]);
    });

    it('debounce sends only the last value after the quiet period', () => {
        const send = limit<number>(50, undefined, sender('a'));
        send(1);
        vi.advanceTimersByTime(40);
        send(2);
        vi.advanceTimersByTime(40);
        send(3);
        expect(sent).toEqual([]);
        vi.advanceTimersByTime(50);
        expect(sent).toEqual([['a', 3]]);
    });

    it('throttle sends the first value at once and the last value of the window when it ends', () => {
        const send = limit<number>(undefined, 100, sender('a'));
        send(1);
        send(2);
        send(3);
        expect(sent).toEqual([['a', 1]]);
        vi.advanceTimersByTime(100);
        expect(sent).toEqual([['a', 1], ['a', 3]]);
        vi.advanceTimersByTime(100);
        expect(sent).toHaveLength(2);
        send(4);
        expect(sent.at(-1)).toEqual(['a', 4]);
    });

    it('throttle keeps limiting after a trailing send', () => {
        const send = limit<number>(undefined, 100, sender('a'));
        send(1);
        send(2);
        vi.advanceTimersByTime(100);
        send(3);
        expect(sent).toEqual([['a', 1], ['a', 2]]);
        vi.advanceTimersByTime(100);
        expect(sent).toEqual([['a', 1], ['a', 2], ['a', 3]]);
    });

    it('frame mode (0) waits for an animation frame and still delivers the last value', () => {
        const frames: Array<() => void> = [];
        vi.stubGlobal('requestAnimationFrame', (cb: () => void) => frames.push(cb));
        vi.stubGlobal('cancelAnimationFrame', () => {});
        const send = limit<number>(undefined, 0, sender('a'));
        for (let i = 1; i <= 30; i++) send(i);
        expect(sent).toEqual([['a', 1]]);
        expect(frames).toHaveLength(1);
        frames[0]();
        expect(sent).toEqual([['a', 1], ['a', 30]]);
    });

    it('an immediate event is never sent before an earlier pending one', () => {
        const typing = limit<string>(300, undefined, sender('input'));
        const click = limit<string>(undefined, undefined, sender('click'));
        typing('hello');
        click('submit');
        expect(sent).toEqual([['input', 'hello'], ['click', 'submit']]);
        vi.advanceTimersByTime(500);
        expect(sent).toHaveLength(2);
    });

    it('pending slots are sent in the order of their last event', () => {
        const a = limit<number>(100, undefined, sender('a'));
        const b = limit<number>(100, undefined, sender('b'));
        a(1);
        b(2);
        a(3);
        vi.advanceTimersByTime(200);
        expect(sent).toEqual([['b', 2], ['a', 3]]);
    });

    it('a timer that fires first sends the older pending slots with it', () => {
        const a = limit<number>(100, undefined, sender('a'));
        const b = limit<number>(20, undefined, sender('b'));
        a(1);
        b(2);
        vi.advanceTimersByTime(20);
        expect(sent).toEqual([['a', 1], ['b', 2]]);
    });

    it('flush sends everything pending and a second flush sends nothing', () => {
        const send = limit<number>(100, undefined, sender('a'));
        send(1);
        flush();
        flush();
        expect(sent).toEqual([['a', 1]]);
        vi.advanceTimersByTime(200);
        expect(sent).toHaveLength(1);
    });
});

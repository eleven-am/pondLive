import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { readFileSync, readdirSync } from 'fs';
import { join } from 'path';
import { Runtime } from '../../src/runtime';
import { Applier } from '../../src/applier';
import { Logger } from '../../src/logger';
import { FakeSocket } from './fake-socket';

interface Step {
    dir: 'connect' | 'disconnect' | 'client' | 'server';
    frame?: Record<string, unknown>;
}

interface Transcript {
    name: string;
    description: string;
    sid: string;
    ver: number;
    bootSeq: number;
    html: string;
    steps: Step[];
}

const dir = join(__dirname, '..', 'fixtures', 'protocol');
const transcripts: Transcript[] = readdirSync(dir)
    .filter((f) => f.endsWith('.json'))
    .sort()
    .map((f) => JSON.parse(readFileSync(join(dir, f), 'utf8')));

function normalize(frame: Record<string, unknown>): Record<string, unknown> {
    const out = JSON.parse(JSON.stringify(frame)) as Record<string, unknown>;
    if (out.tok !== undefined) out.tok = '<tok>';
    return out;
}

function withoutBS(frame: Record<string, unknown>): Record<string, unknown> {
    const { bs: _bs, ...rest } = frame;
    return rest;
}

describe('protocol transcripts (Go to client wire parity)', () => {
    let runtime: Runtime;
    let cursor: number;
    let acceptedCS: number;

    beforeEach(() => {
        vi.useFakeTimers();
        vi.spyOn(Math, 'random').mockReturnValue(0);
        Logger.configure({ enabled: false });
        FakeSocket.instances = [];
        vi.stubGlobal('WebSocket', FakeSocket);
        cursor = 0;
        acceptedCS = 0;
    });

    afterEach(() => {
        runtime.disconnect();
        vi.unstubAllGlobals();
        vi.useRealTimers();
    });

    function nextClientFrame(): Record<string, unknown> | undefined {
        const sent = FakeSocket.latest().frames().filter((f) => f.t !== 'ack');
        return sent[cursor++];
    }

    function bootRuntime(t: Transcript): void {
        document.open();
        document.write(t.html);
        document.close();
        runtime = new Runtime({
            root: document.documentElement,
            sessionId: t.sid,
            version: t.ver,
            seq: t.bootSeq,
            endpoint: '/live',
            token: '<tok>',
        });
        runtime.handleBoot({
            t: 'boot',
            sid: t.sid,
            ver: t.ver,
            seq: t.bootSeq,
            tok: '<tok>',
            ws: '/live',
            location: { path: '/', query: {}, hash: '' },
        });
    }

    for (const t of transcripts) {
        it(`${t.name}: ${t.description}`, () => {
            const applySpy = vi.spyOn(Applier.prototype, 'apply');
            bootRuntime(t);
            const button = document.querySelector('button')!;
            let connects = 0;

            for (const step of t.steps) {
                switch (step.dir) {
                    case 'connect': {
                        if (connects === 0) {
                            runtime.connect();
                        } else {
                            vi.advanceTimersByTime(60000);
                        }
                        connects++;
                        cursor = 0;
                        break;
                    }
                    case 'disconnect': {
                        FakeSocket.latest().drop(1006);
                        break;
                    }
                    case 'client': {
                        const expected = normalize(step.frame!);
                        if (expected.t === 'join') {
                            FakeSocket.latest().open();
                            const actual = normalize(nextClientFrame()!);
                            expect(withoutBS(actual)).toEqual(withoutBS(expected));
                            expect(actual.bs).toBe(runtime.seq);
                        } else if (expected.t === 'resync') {
                            FakeSocket.latest().receive({ seq: 1, topic: 'frame', event: 'patch', data: [['t', 999, 'x']] });
                            const actual = normalize(nextClientFrame()!);
                            expect(actual).toEqual(expected);
                        } else if (typeof expected.cs === 'number' && expected.cs <= acceptedCS) {
                            break;
                        } else {
                            document.querySelector('button')!.click();
                            const actual = normalize(nextClientFrame()!);
                            expect(actual).toEqual(expected);
                            acceptedCS = actual.cs as number;
                        }
                        break;
                    }
                    case 'server': {
                        FakeSocket.latest().receive(step.frame);
                        break;
                    }
                }
            }

            const text = document.querySelector('button')!.textContent;
            switch (t.name) {
                case 'join-replay':
                    expect(text).toBe('count=2');
                    expect(applySpy).toHaveBeenCalledTimes(2);
                    expect(runtime.seq).toBe(2);
                    break;
                case 'join-reset': {
                    expect(text).toBe('count=3');
                    expect(applySpy).toHaveBeenCalledTimes(3);
                    expect(runtime.seq).toBe(4);
                    expect(document.documentElement.hasAttribute('data-pond-resyncing')).toBe(false);
                    expect(document.querySelector('button')).toBe(button);
                    const before = FakeSocket.latest().frames().length;
                    document.querySelector('button')!.click();
                    const sent = FakeSocket.latest().frames().slice(before);
                    expect(sent).toHaveLength(1);
                    expect(normalize(sent[0])).toMatchObject({ t: 'h', a: 'click', i: Number(button.getAttribute('data-pi')), cs: 4 });
                    break;
                }
                case 'resync-reset':
                    expect(document.querySelector('button')).toBe(button);
                    expect(text).toBe('count=0');
                    expect(runtime.seq).toBe(1);
                    expect(document.documentElement.hasAttribute('data-pond-resyncing')).toBe(false);
                    break;
                case 'duplicate-cs':
                    expect(text).toBe('count=2');
                    expect(runtime.seq).toBe(2);
                    break;
                default:
                    throw new Error(`no final assertions for transcript ${t.name}`);
            }
        });
    }

    it('loads every committed transcript', () => {
        expect(transcripts.map((t) => t.name).sort()).toEqual(['duplicate-cs', 'join-replay', 'join-reset', 'resync-reset']);
    });
});

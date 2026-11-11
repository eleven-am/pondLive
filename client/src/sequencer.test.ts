import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Sequencer, SequencerDeps } from './sequencer';
import { EditError } from './applier';
import { Edit, Message } from './protocol';
import { Logger } from './logger';

const location = { path: '/a', query: { q: ['1', '2'] }, hash: '#h' };

function patchMsg(seq: number, value = String(seq)): Message {
    return { seq, topic: 'frame', event: 'patch', data: [['t', 1, value]] };
}

function resetMsg(seq: number, extra: Record<string, unknown> = {}): Message {
    return {
        seq,
        topic: 'frame',
        event: 'reset',
        data: {
            html: '<!DOCTYPE html><html><body>x</body></html>',
            location,
            cause: 'rejoin-gap',
            ...extra,
        },
    };
}

describe('Sequencer', () => {
    let connected: boolean;
    let order: string[];
    let applied: Edit[][];
    let deps: SequencerDeps;
    let sequencer: Sequencer;
    let applier: {
        apply: ReturnType<typeof vi.fn>;
        morph: ReturnType<typeof vi.fn>;
    };

    beforeEach(() => {
        vi.useFakeTimers();
        Logger.configure({ enabled: false });
        connected = true;
        order = [];
        applied = [];
        applier = {
            apply: vi.fn((edits: Edit[]) => {
                order.push('apply');
                applied.push(edits);
            }),
            morph: vi.fn(() => {
                order.push('morph');
            }),
        };
        deps = {
            applier,
            dispatch: vi.fn(() => {
                order.push('dispatch');
            }),
            sendAck: vi.fn(),
            sendResync: vi.fn(),
            isConnected: () => connected,
            applyLocation: vi.fn(),
            onResyncing: vi.fn(),
            onFailed: vi.fn(),
        };
        sequencer = new Sequencer(deps);
        sequencer.boot(0);
    });

    afterEach(() => {
        sequencer.destroy();
        vi.useRealTimers();
    });

    describe('ordering', () => {
        it('applies contiguous frames and advances applied', () => {
            sequencer.handle(patchMsg(1));
            sequencer.handle(patchMsg(2));

            expect(applied).toHaveLength(2);
            expect(sequencer.appliedSeq).toBe(2);
        });

        it('drops a duplicate at or below applied', () => {
            sequencer.handle(patchMsg(1));
            sequencer.handle(patchMsg(1));
            sequencer.handle(patchMsg(2));
            sequencer.handle(patchMsg(1));

            expect(applied).toHaveLength(2);
            expect(sequencer.appliedSeq).toBe(2);
            expect(deps.sendResync).not.toHaveBeenCalled();
        });

        it('starts from the boot seq', () => {
            sequencer.boot(10);

            sequencer.handle(patchMsg(10));
            sequencer.handle(patchMsg(11));

            expect(applied).toHaveLength(1);
            expect(sequencer.appliedSeq).toBe(11);
        });

        it('treats a skipped seq as a gap and asks for a resync with the applied seq', () => {
            sequencer.handle(patchMsg(1));
            sequencer.handle(patchMsg(3));

            expect(sequencer.state).toBe('resyncing');
            expect(deps.sendResync).toHaveBeenCalledTimes(1);
            expect(deps.sendResync).toHaveBeenCalledWith(1, 'gap');
            expect(applied).toHaveLength(1);
            expect(deps.onResyncing).toHaveBeenCalledWith(true);
        });

        it('routes router, dom and script messages through dispatch and advances applied', () => {
            sequencer.handle({ seq: 1, topic: 'router', event: 'push', data: {} });
            sequencer.handle({ seq: 2, topic: 'dom', event: 'call', data: {} });
            sequencer.handle({ seq: 3, topic: 'script:s1', event: 'send', data: {} });

            expect(deps.dispatch).toHaveBeenCalledTimes(3);
            expect(applier.apply).not.toHaveBeenCalled();
            expect(sequencer.appliedSeq).toBe(3);
        });

        it('survives a throwing dispatch, still commits and keeps going', () => {
            deps.dispatch = vi.fn(() => {
                throw new Error('boom');
            });
            sequencer = new Sequencer(deps);
            sequencer.boot(0);

            sequencer.handle({ seq: 1, topic: 'dom', event: 'call', data: {} });
            sequencer.handle(patchMsg(2));

            expect(sequencer.appliedSeq).toBe(2);
            expect(sequencer.state).toBe('live');
        });
    });

    describe('apply failure', () => {
        const unresolved = (): EditError =>
            new EditError('unresolved', ['t', 9, 'x'], 'path does not resolve');

        it.each([
            ['unresolved', unresolved()],
            ['apply-error', new EditError('apply-error', ['t', 1, 'x'], 'TypeError')],
            ['apply-error', new TypeError('not an edit error')],
        ])('a %s failure resyncs, acks nothing and applies nothing after it', (reason, error) => {
            sequencer.handle(patchMsg(1));
            applier.apply.mockImplementationOnce(() => {
                throw error;
            });

            sequencer.handle(patchMsg(2));
            sequencer.handle(patchMsg(3));
            sequencer.handle(patchMsg(4));
            vi.advanceTimersByTime(5000);

            expect(sequencer.state).toBe('resyncing');
            expect(sequencer.appliedSeq).toBe(1);
            expect(deps.sendResync).toHaveBeenCalledTimes(1);
            expect(deps.sendResync).toHaveBeenCalledWith(1, reason);
            expect(applier.apply).toHaveBeenCalledTimes(2);
            expect(deps.sendAck).toHaveBeenCalledTimes(0);
        });

        it('does not ack frames committed before the failure once resyncing', () => {
            sequencer.handle(patchMsg(1));
            applier.apply.mockImplementationOnce(() => {
                throw unresolved();
            });
            sequencer.handle(patchMsg(2));

            vi.advanceTimersByTime(1000);

            expect(deps.sendAck).not.toHaveBeenCalled();
        });

        it('a non-array patch body is an apply error', () => {
            sequencer.handle({ seq: 1, topic: 'frame', event: 'patch', data: { nope: true } });

            expect(deps.sendResync).toHaveBeenCalledWith(0, 'apply-error');
            expect(applier.apply).not.toHaveBeenCalled();
        });

        it('does not send a resync frame while disconnected and relies on the join flag', () => {
            connected = false;
            applier.apply.mockImplementationOnce(() => {
                throw unresolved();
            });

            sequencer.handle(patchMsg(1));

            expect(sequencer.isResyncing()).toBe(true);
            expect(deps.sendResync).not.toHaveBeenCalled();
        });

        it('drops router, dom and patch frames while resyncing', () => {
            applier.apply.mockImplementationOnce(() => {
                throw unresolved();
            });
            sequencer.handle(patchMsg(1));
            (deps.dispatch as ReturnType<typeof vi.fn>).mockClear();

            sequencer.handle({ seq: 2, topic: 'router', event: 'push', data: {} });
            sequencer.handle(patchMsg(3));

            expect(deps.dispatch).not.toHaveBeenCalled();
            expect(applier.apply).toHaveBeenCalledTimes(1);
        });
    });

    describe('reset', () => {
        function failOnce(): void {
            applier.apply.mockImplementationOnce(() => {
                throw new EditError('unresolved', ['t', 9, 'x'], 'x');
            });
            sequencer.handle(patchMsg(1));
        }

        it('morphs, restores location, acks and resumes', () => {
            failOnce();
            order.length = 0;

            sequencer.handle(resetMsg(5));

            expect(order).toEqual(['morph']);
            expect(applier.morph).toHaveBeenCalledWith('<!DOCTYPE html><html><body>x</body></html>');
            expect(deps.applyLocation).toHaveBeenCalledWith(location);
            expect(sequencer.state).toBe('live');
            expect(sequencer.appliedSeq).toBe(5);
            expect(deps.sendAck).toHaveBeenCalledWith(5);
            expect(deps.onResyncing).toHaveBeenLastCalledWith(false);
            vi.advanceTimersByTime(20000);
            expect(deps.onFailed).not.toHaveBeenCalled();

            sequencer.handle(patchMsg(6));
            expect(sequencer.appliedSeq).toBe(6);
        });

        it('applies a reset in the live state even across a seq gap', () => {
            sequencer.handle(patchMsg(1));

            sequencer.handle(resetMsg(7));

            expect(applier.morph).toHaveBeenCalledTimes(1);
            expect(sequencer.appliedSeq).toBe(7);
            expect(deps.sendResync).not.toHaveBeenCalled();
            expect(deps.sendAck).toHaveBeenCalledWith(7);
        });

        it('ignores a stale reset at or below applied', () => {
            sequencer.handle(patchMsg(1));
            sequencer.handle(patchMsg(2));

            sequencer.handle(resetMsg(2));
            sequencer.handle(resetMsg(1));

            expect(applier.morph).not.toHaveBeenCalled();
        });

        it('ignores a stale reset while resyncing', () => {
            sequencer.handle(patchMsg(1));
            sequencer.handle(patchMsg(2));
            applier.apply.mockImplementationOnce(() => {
                throw new EditError('unresolved', ['t', 9, 'x'], 'x');
            });
            sequencer.handle(patchMsg(3));

            sequencer.handle(resetMsg(2));

            expect(applier.morph).not.toHaveBeenCalled();
            expect(sequencer.state).toBe('resyncing');
        });

        it('fails over to a reload when the morph throws', () => {
            applier.morph.mockImplementationOnce(() => {
                throw new Error('morph broke');
            });

            sequencer.handle(resetMsg(1));

            expect(sequencer.state).toBe('failed');
            expect(deps.onFailed).toHaveBeenCalledTimes(1);
            expect(deps.sendAck).not.toHaveBeenCalled();
        });

        it('fails over on a malformed reset', () => {
            sequencer.handle({ seq: 1, topic: 'frame', event: 'reset', data: { nope: 1 } });

            expect(deps.onFailed).toHaveBeenCalledTimes(1);
        });

        it('a location error does not undo the reset', () => {
            (deps.applyLocation as ReturnType<typeof vi.fn>).mockImplementation(() => {
                throw new Error('bad url');
            });

            sequencer.handle(resetMsg(1));

            expect(sequencer.state).toBe('live');
            expect(sequencer.appliedSeq).toBe(1);
        });
    });

    describe('resync timeout', () => {
        it('reloads after 10 seconds without a reset and then ignores everything', () => {
            sequencer.handle(patchMsg(2));
            vi.advanceTimersByTime(9999);
            expect(deps.onFailed).not.toHaveBeenCalled();

            vi.advanceTimersByTime(1);

            expect(deps.onFailed).toHaveBeenCalledTimes(1);
            expect(sequencer.state).toBe('failed');
            sequencer.handle(resetMsg(5));
            expect(applier.morph).not.toHaveBeenCalled();
        });

        it('is cleared by a reset', () => {
            sequencer.handle(patchMsg(2));

            sequencer.handle(resetMsg(3));
            vi.advanceTimersByTime(60000);

            expect(deps.onFailed).not.toHaveBeenCalled();
        });

        it('survives a disconnect so a stuck resync still reloads', () => {
            sequencer.handle(patchMsg(2));
            sequencer.onDisconnected();

            vi.advanceTimersByTime(10000);

            expect(deps.onFailed).toHaveBeenCalledTimes(1);
        });
    });

    describe('ack scheduling', () => {
        it('acks 50 ms after the first unacked commit, once, with the latest seq', () => {
            sequencer.handle(patchMsg(1));
            vi.advanceTimersByTime(20);
            sequencer.handle(patchMsg(2));
            vi.advanceTimersByTime(29);
            expect(deps.sendAck).not.toHaveBeenCalled();

            vi.advanceTimersByTime(1);

            expect(deps.sendAck).toHaveBeenCalledTimes(1);
            expect(deps.sendAck).toHaveBeenCalledWith(2);
            vi.advanceTimersByTime(1000);
            expect(deps.sendAck).toHaveBeenCalledTimes(1);
        });

        it('acks immediately once 32 frames are unacked', () => {
            for (let i = 1; i <= 31; i++) sequencer.handle(patchMsg(i));
            expect(deps.sendAck).not.toHaveBeenCalled();

            sequencer.handle(patchMsg(32));

            expect(deps.sendAck).toHaveBeenCalledTimes(1);
            expect(deps.sendAck).toHaveBeenCalledWith(32);
            vi.advanceTimersByTime(1000);
            expect(deps.sendAck).toHaveBeenCalledTimes(1);
        });

        it('does not ack while disconnected and acks after reconnect', () => {
            connected = false;
            sequencer.handle(patchMsg(1));
            sequencer.onDisconnected();
            vi.advanceTimersByTime(1000);
            expect(deps.sendAck).not.toHaveBeenCalled();

            connected = true;
            sequencer.onConnected();
            vi.advanceTimersByTime(50);

            expect(deps.sendAck).toHaveBeenCalledWith(1);
        });

        it('does not re-ack what it has acked after a reconnect', () => {
            sequencer.handle(patchMsg(1));
            vi.advanceTimersByTime(50);
            (deps.sendAck as ReturnType<typeof vi.fn>).mockClear();

            sequencer.onDisconnected();
            sequencer.onConnected();
            vi.advanceTimersByTime(1000);

            expect(deps.sendAck).not.toHaveBeenCalled();
        });

        it('a pending ack timer is cancelled by destroy', () => {
            sequencer.handle(patchMsg(1));

            sequencer.destroy();
            vi.advanceTimersByTime(1000);

            expect(deps.sendAck).not.toHaveBeenCalled();
        });
    });
});

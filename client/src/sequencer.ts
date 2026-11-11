import { Edit, Message, ResetPayload, ResyncReason } from './protocol';
import { EditError } from './applier';
import { Logger } from './logger';

const ACK_BATCH = 32;
const ACK_DELAY_MS = 50;
const RESYNC_TIMEOUT_MS = 10000;

export type SequencerState = 'live' | 'resyncing' | 'failed';

export interface SequencerApplier {
    apply(edits: Edit[]): void;
    morph(html: string): void;
}

export interface SequencerDeps {
    applier: SequencerApplier;
    dispatch: (msg: Message) => void;
    sendAck: (seq: number) => void;
    sendResync: (seq: number, reason: ResyncReason) => void;
    isConnected: () => boolean;
    applyLocation: (location: ResetPayload['location']) => void;
    onResyncing: (resyncing: boolean) => void;
    onFailed: () => void;
}

export class Sequencer {
    private readonly deps: SequencerDeps;
    private applied = 0;
    private lastAcked = 0;
    private current: SequencerState = 'live';
    private ackTimer: ReturnType<typeof setTimeout> | null = null;
    private resyncTimer: ReturnType<typeof setTimeout> | null = null;

    constructor(deps: SequencerDeps) {
        this.deps = deps;
    }

    get state(): SequencerState {
        return this.current;
    }

    get appliedSeq(): number {
        return this.applied;
    }

    isResyncing(): boolean {
        return this.current === 'resyncing';
    }

    boot(seq: number): void {
        this.applied = seq;
        this.lastAcked = seq;
    }

    handle(msg: Message): void {
        if (this.current === 'failed') {
            return;
        }
        const isReset = msg.topic === 'frame' && msg.event === 'reset';

        if (this.current === 'resyncing') {
            if (isReset && msg.seq > this.applied) {
                this.commitReset(msg);
            }
            return;
        }

        if (msg.seq <= this.applied) {
            return;
        }
        if (isReset) {
            this.commitReset(msg);
            return;
        }
        if (msg.seq > this.applied + 1) {
            Logger.warn('Sequencer', 'Gap detected', { applied: this.applied, got: msg.seq });
            this.enterResync('gap');
            return;
        }
        this.commit(msg);
    }

    onConnected(): void {
        if (this.current === 'live' && this.applied > this.lastAcked) {
            this.scheduleAck();
        }
    }

    onDisconnected(): void {
        this.clearAckTimer();
    }

    destroy(): void {
        this.clearAckTimer();
        this.clearResyncTimer();
    }

    private commit(msg: Message): void {
        if (msg.topic === 'frame') {
            if (msg.event === 'patch') {
                if (!Array.isArray(msg.data)) {
                    this.enterResync('apply-error');
                    return;
                }
                try {
                    this.deps.applier.apply(msg.data as Edit[]);
                } catch (e) {
                    const reason: ResyncReason = e instanceof EditError ? e.reason : 'apply-error';
                    Logger.error('Sequencer', 'Patch failed', { seq: msg.seq, error: String(e) });
                    this.enterResync(reason);
                    return;
                }
            } else {
                Logger.warn('Sequencer', 'Unknown frame event', msg.event);
            }
        } else {
            try {
                this.deps.dispatch(msg);
            } catch (e) {
                Logger.error('Sequencer', 'Dispatch failed', { topic: msg.topic, event: msg.event, error: String(e) });
            }
        }
        this.applied = msg.seq;
        this.scheduleAck();
    }

    private commitReset(msg: Message): void {
        const reset = msg.data as ResetPayload;
        if (typeof reset?.html !== 'string') {
            Logger.error('Sequencer', 'Malformed reset');
            this.fail();
            return;
        }
        Logger.warn('Sequencer', 'Applying reset', { seq: msg.seq, cause: reset.cause });

        try {
            this.deps.applier.morph(reset.html);
        } catch (e) {
            Logger.error('Sequencer', 'Reset failed', { error: String(e) });
            this.fail();
            return;
        }

        try {
            if (reset.location) this.deps.applyLocation(reset.location);
        } catch (e) {
            Logger.warn('Sequencer', 'Location restore failed', { error: String(e) });
        }

        this.applied = msg.seq;
        this.leaveResync();
        this.ackNow();
    }

    private enterResync(reason: ResyncReason): void {
        if (this.current === 'resyncing') {
            return;
        }
        this.current = 'resyncing';
        this.clearAckTimer();
        this.deps.onResyncing(true);
        if (this.deps.isConnected()) {
            this.deps.sendResync(this.applied, reason);
        }
        this.clearResyncTimer();
        this.resyncTimer = setTimeout(() => {
            this.resyncTimer = null;
            if (this.current === 'resyncing') {
                Logger.error('Sequencer', 'Resync timed out');
                this.fail();
            }
        }, RESYNC_TIMEOUT_MS);
    }

    private leaveResync(): void {
        const was = this.current === 'resyncing';
        this.current = 'live';
        this.clearResyncTimer();
        if (was) {
            this.deps.onResyncing(false);
        }
    }

    private fail(): void {
        this.current = 'failed';
        this.clearAckTimer();
        this.clearResyncTimer();
        this.deps.onFailed();
    }

    private scheduleAck(): void {
        if (this.applied - this.lastAcked >= ACK_BATCH) {
            this.ackNow();
            return;
        }
        if (this.ackTimer === null) {
            this.ackTimer = setTimeout(() => {
                this.ackTimer = null;
                this.ackNow();
            }, ACK_DELAY_MS);
        }
    }

    private ackNow(): void {
        this.clearAckTimer();
        if (this.current !== 'live' || this.applied <= this.lastAcked) {
            return;
        }
        if (!this.deps.isConnected()) {
            return;
        }
        this.lastAcked = this.applied;
        this.deps.sendAck(this.applied);
    }

    private clearAckTimer(): void {
        if (this.ackTimer !== null) {
            clearTimeout(this.ackTimer);
            this.ackTimer = null;
        }
    }

    private clearResyncTimer(): void {
        if (this.resyncTimer !== null) {
            clearTimeout(this.resyncTimer);
            this.resyncTimer = null;
        }
    }
}

export { Runtime, RuntimeConfig, boot } from './runtime';
export { Logger, LogLevel, LoggerConfig } from './logger';
export { Transport, TransportConfig, ConnectionState } from './transport';
export { Bus, Subscription } from './bus';
export { Applier, ApplierCallbacks, EditError, EditFailure } from './applier';
export { Sequencer, SequencerDeps, SequencerState } from './sequencer';
export { Executor, ExecutorConfig } from './executor';
export { Scripts, ScriptsConfig, ScriptDef, define } from './scripts';

export {
    Topic,
    StaticTopic,
    Boot,
    Edit,
    EditOp,
    EventTopic,
    Location,
    Message,
    ClientEvt,
    ClientAck,
    ClientResync,
    ResyncReason,
    ResetPayload,
    ResetCause,
    JoinFrame,
    JoinedFrame,
    EvackFrame,
    DeclinedFrame,
    TokFrame,
    ServerFrame,
    isServerFrame,
    ClientConfig,
    RouterNavPayload,
    DOMCallPayload,
    DOMSetPayload,
    DOMQueryPayload,
    DOMAsyncPayload,
    DOMResponsePayload,
    isBoot,
    isMessage,
} from './protocol';

import { boot } from './runtime';
import { seal } from './scripts';

let started = false;

export function start(hash: string): void {
    if (started) return;
    started = true;
    seal();
    if (typeof window === 'undefined' || typeof document === 'undefined') return;
    if (window.location.href.endsWith('#') && window.location.hash === '') {
        history.replaceState(null, '', window.location.pathname + window.location.search);
    }
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => boot(hash));
    } else {
        boot(hash);
    }
}

export { Runtime, RuntimeConfig, boot } from './runtime';
export { Logger, LogLevel, LoggerConfig } from './logger';
export { Transport, TransportConfig, ConnectionState, JoinPayload } from './transport';
export { Bus, Subscription } from './bus';
export { Patcher, PatcherCallbacks } from './patcher';
export { Executor, ExecutorConfig } from './executor';
export { ScriptExecutor, ScriptExecutorConfig } from './scripts';

export {
    Topic,
    StaticTopic,
    ScriptTopic,
    HandlerTopic,
    Topics,
    Boot,
    Patch,
    OpKind,
    OpKinds,
    Location,
    Message,
    ClientEvt,
    ServerEvt,
    ClientAck,
    ServerAck,
    ClientConfig,
    EventOptions,
    HandlerMeta,
    ScriptMeta,
    FramePatchPayload,
    RouterNavPayload,
    DOMCallPayload,
    DOMSetPayload,
    DOMQueryPayload,
    DOMAsyncPayload,
    DOMResponsePayload,
    isBoot,
    isServerError,
    isServerEvt,
    isServerAck,
    isMessage,
} from './protocol';

import { boot } from './runtime';

if (typeof window !== 'undefined' && typeof document !== 'undefined') {
    if (window.location.href.endsWith('#') && window.location.hash === '') {
        history.replaceState(null, '', window.location.pathname + window.location.search);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => boot());
    } else {
        boot();
    }
}

import type {
    Boot,
    DeclinedFrame,
    EvackFrame,
    JoinedFrame,
    Message,
    StaticTopic,
    StaticTopicActionMap,
    TokFrame,
    Topic,
} from './wire';
export type {
    Boot,
    ClientAck,
    ClientConfig,
    ClientEvt,
    ClientResync,
    DOMAsyncPayload,
    DOMCallPayload,
    DOMClientAction,
    DOMObservePayload,
    DOMQueryPayload,
    DOMResponsePayload,
    DOMServerAction,
    DOMSetPayload,
    DOMUnobservePayload,
    DOMValuePayload,
    DeclinedFrame,
    Edit,
    EditOp,
    EvackFrame,
    Event,
    EventTopic,
    FrameServerAction,
    JoinFrame,
    JoinedFrame,
    JoinedMode,
    Location,
    Message,
    ResetCause,
    ResetPayload,
    ResyncReason,
    RouterClientAction,
    RouterNavPayload,
    RouterPopstatePayload,
    RouterRedirectPayload,
    RouterServerAction,
    StaticTopic,
    StaticTopicActionMap,
    TokFrame,
    Topic,
} from './wire';

export function urlOf(path: string, query: string, hash: string): string {
    return path + (query ? `?${query}` : '') + (hash ? (hash.startsWith('#') ? hash : `#${hash}`) : '');
}

export const SCRIPT_TOPIC = 's';

export interface ScriptMessage {
    i: number;
    d: unknown;
}

export function isScriptMessage(data: unknown): data is ScriptMessage {
    return typeof data === 'object' && data !== null && Number.isInteger((data as ScriptMessage).i);
}

export type ServerFrame = JoinedFrame | DeclinedFrame | TokFrame | EvackFrame;

export function isServerFrame(msg: unknown): msg is ServerFrame {
    if (typeof msg !== 'object' || msg === null) return false;
    const frame = msg as { t?: unknown; code?: unknown };
    if (frame.t === 'joined') return true;
    if (frame.t === 'evack') return typeof (frame as { cs?: unknown }).cs === 'number';
    if (frame.t === 'tok') return typeof (frame as { tok?: unknown }).tok === 'string';
    return frame.t === 'declined' && typeof frame.code === 'number';
}

export function isBoot(msg: unknown): msg is Boot {
    return typeof msg === 'object' && msg !== null && (msg as Boot).t === 'boot';
}

export function isMessage(msg: unknown): msg is Message {
    return (
        typeof msg === 'object' &&
        msg !== null &&
        typeof (msg as Message).seq === 'number' &&
        typeof (msg as Message).topic === 'string' &&
        typeof (msg as Message).event === 'string'
    );
}

export interface HandlerEventPayload {
    [key: string]: unknown;
}

export type ActionFor<T extends Topic> = T extends StaticTopic ? keyof StaticTopicActionMap[T] : never;

export type PayloadFor<T extends Topic, A extends string> = T extends StaticTopic
    ? A extends keyof StaticTopicActionMap[T]
        ? StaticTopicActionMap[T][A]
        : never
    : never;

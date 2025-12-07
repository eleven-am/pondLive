export type StaticTopic = 'router' | 'dom' | 'frame' | 'ack';
export type ScriptTopic = `script:${string}`;
export type HandlerTopic = `${string}:h${number}`;
export type Topic = StaticTopic | ScriptTopic | HandlerTopic;

export const Topics = {
    Router: 'router' as const,
    DOM: 'dom' as const,
    Frame: 'frame' as const,
    Ack: 'ack' as const,
} as const;

export function isScriptTopic(topic: string): topic is ScriptTopic {
    return topic.startsWith('script:');
}

export function scriptTopic(scriptId: string): ScriptTopic {
    return `script:${scriptId}`;
}

export function isHandlerTopic(topic: string): topic is HandlerTopic {
    return /^.+:h\d+$/.test(topic);
}

export function handlerTopic(handlerId: string): HandlerTopic {
    return handlerId as HandlerTopic;
}

export interface Message {
    seq: number;
    topic: string;
    event: string;
    data: unknown;
}

export interface Event {
    t: Topic;
    sid: string;
}

export interface ClientEvt extends Event {
    a: string;
    p?: unknown;
}

export interface ServerEvt extends Event {
    a: string;
    p?: unknown;
    seq?: number;
}

export interface ClientAck extends Event {
    seq: number;
}

export interface ServerAck extends Event {
    seq: number;
}

export interface ClientConfig {
    debug?: boolean;
}

export interface Boot {
    t: 'boot';
    sid: string;
    ver: number;
    seq: number;
    patch: Patch[];
    location: Location;
    client?: ClientConfig;
}

export interface ServerError {
    t: 'error';
    sid: string;
    code: string;
    message: string;
}

export interface Location {
    path: string;
    query: Record<string, string[]>;
    hash: string;
}

export interface EventOptions {
    prevent?: boolean;
    stop?: boolean;
    passive?: boolean;
    once?: boolean;
    capture?: boolean;
    debounce?: number;
    throttle?: number;
    listen?: string[];
    props?: string[];
}

export interface HandlerMeta extends EventOptions {
    event: string;
    handler: string;
}

export interface ScriptMeta {
    scriptId: string;
    script: string;
}

export interface StyleRule {
    selector: string;
    props: Record<string, string>;
}

export interface MediaBlock {
    query: string;
    rules: StyleRule[];
}

export interface Stylesheet {
    rules?: StyleRule[];
    mediaBlocks?: MediaBlock[];
    hash?: string;
}

export type OpKind =
    | 'setText'
    | 'setComment'
    | 'setAttr'
    | 'delAttr'
    | 'setStyle'
    | 'delStyle'
    | 'setStyleDecl'
    | 'delStyleDecl'
    | 'setHandlers'
    | 'setScript'
    | 'delScript'
    | 'setRef'
    | 'delRef'
    | 'replaceNode'
    | 'addChild'
    | 'delChild'
    | 'moveChild';

export const OpKinds = {
    SetText: 'setText' as OpKind,
    SetComment: 'setComment' as OpKind,
    SetAttr: 'setAttr' as OpKind,
    DelAttr: 'delAttr' as OpKind,
    SetStyle: 'setStyle' as OpKind,
    DelStyle: 'delStyle' as OpKind,
    SetStyleDecl: 'setStyleDecl' as OpKind,
    DelStyleDecl: 'delStyleDecl' as OpKind,
    SetHandlers: 'setHandlers' as OpKind,
    SetScript: 'setScript' as OpKind,
    DelScript: 'delScript' as OpKind,
    SetRef: 'setRef' as OpKind,
    DelRef: 'delRef' as OpKind,
    ReplaceNode: 'replaceNode' as OpKind,
    AddChild: 'addChild' as OpKind,
    DelChild: 'delChild' as OpKind,
    MoveChild: 'moveChild' as OpKind,
} as const;

export interface Patch {
    seq: number;
    path: number[] | null;
    op: OpKind;
    value?: unknown;
    name?: string;
    selector?: string;
    index?: number;
}

export function isBoot(msg: unknown): msg is Boot {
    return typeof msg === 'object' && msg !== null && (msg as Boot).t === 'boot';
}

export function isServerError(msg: unknown): msg is ServerError {
    return typeof msg === 'object' && msg !== null && (msg as ServerError).t === 'error';
}

export function isServerEvt(msg: unknown): msg is ServerEvt {
    return (
        typeof msg === 'object' &&
        msg !== null &&
        typeof (msg as ServerEvt).t === 'string' &&
        typeof (msg as ServerEvt).a === 'string'
    );
}

export function isServerAck(msg: unknown): msg is ServerAck {
    return (
        typeof msg === 'object' &&
        msg !== null &&
        (msg as ServerAck).t === 'ack' &&
        typeof (msg as ServerAck).seq === 'number'
    );
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

export type FrameServerAction = 'patch';

export const FrameActions = {
    Patch: 'patch' as FrameServerAction,
} as const;

export type RouterServerAction = 'push' | 'replace' | 'back' | 'forward';
export type RouterClientAction = 'popstate';

export const RouterActions = {
    Push: 'push' as RouterServerAction,
    Replace: 'replace' as RouterServerAction,
    Back: 'back' as RouterServerAction,
    Forward: 'forward' as RouterServerAction,
    Popstate: 'popstate' as RouterClientAction,
} as const;

export interface RouterNavPayload {
    path: string;
    query: string;
    hash: string;
    replace: boolean;
}

export type DOMServerAction = 'call' | 'set' | 'query' | 'async';
export type DOMClientAction = 'response';

export const DOMActions = {
    Call: 'call' as DOMServerAction,
    Set: 'set' as DOMServerAction,
    Query: 'query' as DOMServerAction,
    Async: 'async' as DOMServerAction,
    Response: 'response' as DOMClientAction,
} as const;

export interface DOMCallPayload {
    ref: string;
    method: string;
    args?: unknown[];
}

export interface DOMSetPayload {
    ref: string;
    prop: string;
    value: unknown;
}

export interface DOMQueryPayload {
    requestId: string;
    ref: string;
    selectors: string[];
}

export interface DOMAsyncPayload {
    requestId: string;
    ref: string;
    method: string;
    args?: unknown[];
}

export interface DOMResponsePayload {
    requestId: string;
    values?: Record<string, unknown>;
    result?: unknown;
    error?: string;
}

export type ScriptServerAction = 'send';
export type ScriptClientAction = 'message';

export const ScriptActions = {
    Send: 'send' as ScriptServerAction,
    Message: 'message' as ScriptClientAction,
} as const;

export interface ScriptPayload {
    scriptId: string;
    event: string;
    data?: unknown;
}

export type HandlerClientAction = 'invoke';

export const HandlerActions = {
    Invoke: 'invoke' as HandlerClientAction,
} as const;

export interface HandlerEventPayload {
    [key: string]: unknown;
}

export interface FramePatchPayload {
    seq: number;
    patches: Patch[];
}

export interface AckPayload {
    seq: number;
}

export interface RouterPopstatePayload {
    path: string;
    query: string;
    hash: string;
}

export interface StaticTopicActionMap {
    frame: {
        patch: FramePatchPayload;
    };
    router: {
        push: RouterNavPayload;
        replace: RouterNavPayload;
        back: undefined;
        forward: undefined;
        popstate: RouterPopstatePayload;
    };
    dom: {
        call: DOMCallPayload;
        set: DOMSetPayload;
        query: DOMQueryPayload;
        async: DOMAsyncPayload;
        response: DOMResponsePayload;
    };
    ack: {
        ack: AckPayload;
    };
}

export interface ScriptTopicActions {
    send: ScriptPayload;
    message: ScriptPayload;
}

export interface HandlerTopicActions {
    invoke: HandlerEventPayload;
}

export type ActionFor<T extends Topic> = T extends StaticTopic
    ? keyof StaticTopicActionMap[T]
    : T extends ScriptTopic
        ? keyof ScriptTopicActions
        : T extends HandlerTopic
            ? keyof HandlerTopicActions
            : never;

export type PayloadFor<T extends Topic, A extends string> = T extends StaticTopic
    ? A extends keyof StaticTopicActionMap[T]
        ? StaticTopicActionMap[T][A]
        : never
    : T extends ScriptTopic
        ? A extends keyof ScriptTopicActions
            ? ScriptTopicActions[A]
            : never
        : T extends HandlerTopic
            ? A extends keyof HandlerTopicActions
                ? HandlerTopicActions[A]
                : never
            : never;

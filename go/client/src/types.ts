import {DOMResponse, ServerMessage, UploadMessage, ScriptMessage, Location as ProtoLocation} from './protocol';

// ============================================================================
// Patch Types
// ============================================================================

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
    | 'setRouter'
    | 'delRouter'
    | 'setUpload'
    | 'delUpload'
    | 'setScript'
    | 'delScript'
    | 'setRef'
    | 'delRef'
    | 'replaceNode'
    | 'addChild'
    | 'delChild'
    | 'moveChild';

export interface Patch {
    seq: number;
    path: number[];
    op: OpKind;
    value?: unknown;
    name?: string;
    selector?: string;
    index?: number;
}

export interface HandlerMeta {
    event: string;
    handler: string;
    listen?: string[];
    props?: string[];
}

export interface RouterMeta {
    path: string;
    query?: string;
    hash?: string;
    replace?: boolean;
}

export interface UploadMeta {
    uploadId: string;
    multiple?: boolean;
    maxSize?: number;
    accept?: string[];
}

export interface ScriptMeta {
    scriptId: string;
    script: string;
}

export interface StructuredNode {
    tag?: string;
    text?: string;
    comment?: string;
    attrs?: Record<string, string[]>;
    style?: Record<string, string>;
    children?: StructuredNode[];
    handlers?: HandlerMeta[];
    router?: RouterMeta;
    upload?: UploadMeta;
    script?: ScriptMeta;
    refId?: string;
    unsafeHTML?: string;
}

// ============================================================================
// Patcher Types
// ============================================================================

export type EventCallback = (event: string, handler: string, data: unknown) => void;
export type RefCallback = (refId: string, el: Element) => void;
export type RefDeleteCallback = (refId: string) => void;
export type RouterCallback = (meta: RouterMeta) => void;
export type UploadCallback = (meta: UploadMeta, files: FileList) => void;
export type ScriptCallback = (meta: ScriptMeta, el: Element) => void;
export type ScriptCleanupCallback = (scriptId: string) => void;

export interface PatcherCallbacks {
    onEvent: EventCallback;
    onRef: RefCallback;
    onRefDelete: RefDeleteCallback;
    onRouter: RouterCallback;
    onUpload: UploadCallback;
    onScript: ScriptCallback;
    onScriptCleanup: ScriptCleanupCallback;
}

// ============================================================================
// Router Types
// ============================================================================

export type NavCallback = (type: 'nav' | 'pop', path: string, query: string, hash: string) => void;

// ============================================================================
// Transport Types
// ============================================================================

import type {ChannelState} from '@eleven-am/pondsocket-client';

export type MessageHandler = (msg: ServerMessage) => void;
export type StateChangeHandler = (state: ChannelState) => void;

export interface TransportConfig {
    endpoint: string;
    sessionId: string;
    version: number;
    ack: number;
    location: ProtoLocation;
}

// ============================================================================
// Uploader Types
// ============================================================================

export type UploadMessageCallback = (msg: UploadMessage) => void;

export interface UploaderConfig {
    endpoint: string;
    sessionId: string;
    onMessage: UploadMessageCallback;
}

// ============================================================================
// Effect Types
// ============================================================================

export interface DOMActionEffect {
    type: 'dom';
    kind: string;
    ref: string;
    method?: string;
    args?: unknown[];
    prop?: string;
    value?: unknown;
    class?: string;
    on?: boolean;
    behavior?: ScrollBehavior;
    block?: ScrollLogicalPosition;
    inline?: ScrollLogicalPosition;
}

export interface CookieEffect {
    type: 'cookies';
    endpoint: string;
    sid: string;
    token: string;
    method?: string;
}

export type Effect = DOMActionEffect | CookieEffect;
export type RefResolver = (refId: string) => Element | undefined;
export type DOMResponseCallback = (response: DOMResponse) => void;

export interface EffectExecutorConfig {
    sessionId: string;
    resolveRef: RefResolver;
    onDOMResponse: DOMResponseCallback;
}

// ============================================================================
// Logger Types
// ============================================================================

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface LoggerConfig {
    enabled: boolean;
    level: LogLevel;
}

// ============================================================================
// Script Types
// ============================================================================

export interface ScriptTransport {
    send(data: Record<string, unknown>): void;
    on(event: string, handler: (data: Record<string, unknown>) => void): void;
}

export interface ScriptInstance {
    cleanup?: () => void;
    eventHandlers: Map<string, (data: Record<string, unknown>) => void>;
}

export type ScriptMessageCallback = (msg: ScriptMessage) => void;

export interface ScriptExecutorConfig {
    sessionId: string;
    onMessage: ScriptMessageCallback;
}

import {Patch} from './types';

export interface Location {
    path: string;
    q: string;
    hash: string;
}

export interface ClientConfig {
    endpoint?: string;
    upload?: string;
    debug?: boolean;
}

export interface FrameMetrics {
    renderMs: number;
    ops: number;
    effectsMs?: number;
    maxEffectMs?: number;
    slowEffects?: number;
}

export interface HandlerMeta {
    event: string;
    listen?: string[];
    props?: string[];
}

export interface HandlerDelta {
    add?: Record<string, HandlerMeta>;
    del?: string[];
}

export interface RefMeta {
    tag: string;
}

export interface RefDelta {
    add?: Record<string, RefMeta>;
    del?: string[];
}

export interface NavDelta {
    push?: string;
    replace?: string;
    back?: boolean;
}

export interface ErrorDetails {
    phase?: string;
    componentId?: string;
    componentName?: string;
    hook?: string;
    hookIndex?: number;
    suggestion?: string;
    stack?: string;
    panic?: string;
    capturedAt?: string;
    metadata?: Record<string, unknown>;
}

export interface ServerError {
    t: 'error';
    sid: string;
    code: string;
    message: string;
    details?: ErrorDetails;
}

export interface Boot {
    t: 'boot';
    sid: string;
    ver: number;
    seq: number;
    patch: Patch[];
    location: Location;
    client?: ClientConfig;
    errors?: ServerError[];
}

export interface Init {
    t: 'init';
    sid: string;
    ver: number;
    seq: number;
    location: Location;
    errors?: ServerError[];
}

export interface Frame {
    t: 'frame';
    sid: string;
    seq: number;
    ver: number;
    patch: Patch[];
    effects: unknown[];
    nav?: NavDelta;
    handlers?: HandlerDelta;
    refs?: RefDelta;
    metrics: FrameMetrics;
}

export interface ResumeOK {
    t: 'resume_ok';
    sid: string;
    from: number;
    to: number;
    errors?: ServerError[];
}

export interface EventAck {
    t: 'evt_ack';
    sid: string;
    cseq: number;
}

export interface Diagnostic {
    t: 'diagnostic';
    sid: string;
    code: string;
    message: string;
    details?: ErrorDetails;
}

export interface DOMRequest {
    t: 'dom_req';
    id: string;
    ref: string;
    props?: string[];
    method?: string;
    args?: unknown[];
}

export type ServerMessage = Boot | Init | Frame | ResumeOK | EventAck | ServerError | Diagnostic | DOMRequest;

export interface Join {
    t: 'join';
    sid: string;
    ver: number;
    ack: number;
    loc: Location;
}

export interface ClientAck {
    t: 'ack';
    sid: string;
    seq: number;
}

export interface ClientEvent {
    t: 'evt';
    sid: string;
    hid: string;
    cseq: number;
    payload: Record<string, unknown>;
}

export interface NavMessage {
    t: 'nav' | 'pop';
    sid: string;
    path: string;
    q: string;
    hash: string;
}

export interface DOMResponse {
    t: 'dom_res';
    sid: string;
    id: string;
    values?: Record<string, unknown>;
    result?: unknown;
    error?: string;
}

export interface UploadMessage {
    t: 'upload';
    op: 'change' | 'progress' | 'error' | 'cancelled';
    id: string;
    meta?: { name: string; size: number; type: string };
    loaded?: number;
    total?: number;
    error?: string;
}

export type ClientMessage = Join | ClientAck | ClientEvent | NavMessage | DOMResponse | UploadMessage;

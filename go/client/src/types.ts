// Minimal protocol/type definitions for the rewritten client runtime.

export type MessageType =
  | 'boot'
  | 'init'
  | 'frame'
  | 'template'
  | 'join'
  | 'resume'
  | 'error'
  | 'diagnostic'
  | 'pubsub'
  | 'evt'
  | 'ack'
  | 'nav'
  | 'pop'
  | 'recover'
  | 'upload'
  | 'domreq'
  | 'domres';

export interface Location {
  path: string;
  q: string;
  hash: string;
}

export interface HandlerMeta {
  event: string;
  listen?: string[];
  props?: string[];
}

export type HandlerMap = Record<string, HandlerMeta>;

export interface SlotBinding {
  event: string;
  handler: string;
  listen?: string[];
  props?: string[];
}

export type BindingTable = Record<number, SlotBinding[]>;

export type DynKind = 'text' | 'attrs' | 'list';

export interface DynamicSlot {
  kind: DynKind;
  text?: string;
  attrs?: Record<string, string>;
  list?: ListRowDescriptor[];
}

export interface ListRowDescriptor {
  key: string;
  slots?: number[];
  slotPaths?: SlotPathDescriptor[];
  listPaths?: ListPathDescriptor[];
  componentPaths?: ComponentPathDescriptor[];
  rootCount?: number;
}

export type SetTextOp = ['setText', number, string];
export type SetAttrsOp = ['setAttrs', number, Record<string, string>, string[]?];
export type ListDelOp = ['del', string];
export type ListInsOp = [
  'ins',
  number,
  {
    key: string;
    html: string;
    slots?: number[];
    slotPaths?: SlotPathDescriptor[];
    listPaths?: ListPathDescriptor[];
    componentPaths?: ComponentPathDescriptor[];
    bindings?: BindingsPayload;
  },
];
export type ListMovOp = ['mov', number, number];
export type ListChildOp = ListDelOp | ListInsOp | ListMovOp;
export type ListOp = ['list', number, ...ListChildOp[]];
export type DiffOp = SetTextOp | SetAttrsOp | ListOp;

export interface SlotMeta {
  anchorId: number;
}

export interface PathSegmentDescriptor {
  kind: 'range' | 'dom';
  index: number;
}

export interface SlotPathDescriptor {
  slot: number;
  componentId: string;
  path?: PathSegmentDescriptor[];
  textChildIndex?: number;
}

export interface ListPathDescriptor {
  slot: number;
  componentId: string;
  path?: PathSegmentDescriptor[];
  atRoot?: boolean;
}

export interface ComponentPathDescriptor {
  componentId: string;
  parentId?: string;
  parentPath?: PathSegmentDescriptor[];
  firstChild?: PathSegmentDescriptor[];
  lastChild?: PathSegmentDescriptor[];
}

export interface UploadBindingDescriptor {
  componentId: string;
  path?: PathSegmentDescriptor[];
  uploadId: string;
  accept?: string[];
  multiple?: boolean;
  maxSize?: number;
}

export interface RefBindingDescriptor {
  componentId: string;
  path?: PathSegmentDescriptor[];
  refId: string;
}

export interface RouterBindingDescriptor {
  componentId: string;
  path?: PathSegmentDescriptor[];
  pathValue?: string;
  query?: string;
  hash?: string;
  replace?: string;
}

export interface RefMeta {
  tag: string;
  events?: Record<string, { handler?: string; listen?: string[]; props?: string[] }>;
}

export type RefMap = Record<string, RefMeta>;

export interface RefDelta {
  add?: RefMap;
  del?: string[];
}

export interface BindingsPayload {
  slots?: BindingTable;
  uploads?: UploadBindingDescriptor[];
  refs?: RefBindingDescriptor[];
  router?: RouterBindingDescriptor[];
}

export interface TemplatePayload {
  html?: string;
  templateHash?: string;
  s: string[];
  d: DynamicSlot[];
  slots: SlotMeta[];
  slotPaths?: SlotPathDescriptor[];
  listPaths?: ListPathDescriptor[];
  componentPaths?: ComponentPathDescriptor[];
  handlers?: HandlerMap;
  bindings?: BindingsPayload;
  refs?: RefDelta;
}

export interface BootPayload extends TemplatePayload {
  t: 'boot';
  sid: string;
  ver: number;
  seq: number;
  location: Location;
  client?: BootClientConfig;
}

export interface BootClientConfig {
  endpoint?: string;
  upload?: string;
  debug?: boolean;
}

export interface InitMessage extends TemplatePayload {
  t: 'init';
  sid: string;
  ver: number;
  location: Location;
  seq?: number;
  errors?: ErrorMessage[];
}

export interface FrameMessage {
  t: 'frame';
  sid: string;
  ver: number;
  seq?: number;
  patch?: DiffOp[];
  handlers?: HandlerDelta;
  bindings?: BindingsPayload;
  refs?: RefDelta;
  nav?: { push?: string; replace?: string; back?: boolean };
  effects?: Effect[];
}

export interface HandlerDelta {
  add?: HandlerMap;
  del?: string[];
}

export interface TemplateMessage extends TemplatePayload {
  t: 'template';
  sid: string;
  ver: number;
}

export interface JoinMessage {
  t: 'join';
  sid: string;
  ver: number;
}

export interface ResumeMessage {
  t: 'resume';
  sid: string;
  from: number;
  to: number;
  errors?: ErrorMessage[];
}

export interface ErrorMessage {
  t: 'error';
  sid: string;
  code?: string;
  message?: string;
}

export interface DiagnosticMessage {
  t: 'diagnostic';
  sid: string;
  code: string;
  message: string;
}

export interface PubsubControlMessage {
  t: 'pubsub';
  op: 'join' | 'leave';
  topic: string;
}

export interface DOMRequestMessage {
  t: 'domreq';
  id: string;
  ref: string;
  props?: string[];
  method?: string;
  args?: any[];
}

export interface DOMResponseMessage {
  t: 'domres';
  sid: string;
  id: string;
  values?: Record<string, any>;
  result?: any;
  error?: string;
}

export interface UploadMeta {
  name: string;
  size: number;
  type: string;
}

export interface MetadataTagPayload {
  key: string;
  name?: string;
  content?: string;
  property?: string;
  charset?: string;
  httpEquiv?: string;
  itemProp?: string;
  attrs?: Record<string, string>;
}

export interface LinkTagPayload {
  key: string;
  rel?: string;
  href?: string;
  type?: string;
  as?: string;
  media?: string;
  hreflang?: string;
  title?: string;
  crossorigin?: string;
  integrity?: string;
  referrerpolicy?: string;
  sizes?: string;
  attrs?: Record<string, string>;
}

export interface ScriptTagPayload {
  key: string;
  src?: string;
  type?: string;
  async?: boolean;
  defer?: boolean;
  module?: boolean;
  noModule?: boolean;
  crossorigin?: string;
  integrity?: string;
  referrerpolicy?: string;
  nonce?: string;
  attrs?: Record<string, string>;
  inner?: string;
}

export interface MetadataEffect {
  type: 'metadata';
  title?: string;
  description?: string;
  clearDescription?: boolean;
  metaAdd?: MetadataTagPayload[];
  metaRemove?: string[];
  linkAdd?: LinkTagPayload[];
  linkRemove?: string[];
  scriptAdd?: ScriptTagPayload[];
  scriptRemove?: string[];
}

export interface DOMActionEffect {
  type: 'dom';
  kind: string;
  ref: string;
  method?: string;
  args?: any[];
  prop?: string;
  value?: any;
  class?: string;
  on?: boolean;
  behavior?: string;
  block?: string;
  inline?: string;
}

export interface CookieEffect {
  type: 'cookies';
  endpoint: string;
  sid: string;
  token: string;
  method?: string;
}

export type Effect = MetadataEffect | DOMActionEffect | CookieEffect;

export type UploadClientOp = 'change' | 'progress' | 'error' | 'cancelled';

export interface UploadClientMessage {
  t: 'upload';
  sid: string;
  id: string;
  op: UploadClientOp;
  meta?: UploadMeta;
  loaded?: number;
  total?: number;
  error?: string;
}

export interface UploadControlMessage {
  t: 'upload';
  sid: string;
  op: 'ack' | 'error' | 'cancel';
  id: string;
}

export interface ClientEventMessage {
  t: 'evt';
  sid: string;
  hid: string;
  payload: EventPayload;
  cseq?: number;
}

export interface EventPayload {
  name: string;
  detail?: any;
}

export interface ClientAckMessage {
  t: 'ack';
  sid: string;
  seq: number;
}

export interface ClientNavMessage {
  t: 'nav' | 'pop';
  sid: string;
  path: string;
  q: string;
  hash?: string;
}

export interface ClientRecoverMessage {
  t: 'recover';
  sid: string;
}

export type ConnectionState =
  | { status: 'disconnected' }
  | { status: 'connecting' }
  | { status: 'connected'; sessionId: string; version: number }
  | { status: 'reconnecting'; attempt: number }
  | { status: 'error'; error: Error };

export interface LiveUIOptionOverrides {
  endpoint?: string;
  uploadEndpoint?: string;
  autoConnect?: boolean;
  debug?: boolean;
  reconnect?: boolean;
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
  boot?: BootPayload;
}

export interface LiveUIOptions extends LiveUIOptionOverrides {}

export type LiveUIEventMap = {
  init: InitMessage;
  frame: FrameMessage;
  template: TemplateMessage;
  join: JoinMessage;
  resume: ResumeMessage;
  error: ErrorMessage;
  diagnostic: DiagnosticMessage;
  pubsub: PubsubControlMessage;
  upload: UploadControlMessage;
  domreq: DOMRequestMessage;
  evt: ClientEventMessage;
  ack: ClientAckMessage;
  nav: ClientNavMessage;
  pop: ClientNavMessage;
  recover: ClientRecoverMessage;
}

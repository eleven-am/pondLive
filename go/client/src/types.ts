/**
 * Type definitions for LiveUI protocol and internal structures
 */

// Protocol message types
export type MessageType =
  | "init"
  | "frame"
  | "template"
  | "join"
  | "resume"
  | "error"
  | "diagnostic"
  | "evt"
  | "ack"
  | "nav"
  | "pop"
  | "recover"
  | "pubsub";

// Location
export interface Location {
  path: string;
  q: string;
  hash: string;
}

// Boot payload produced during SSR
export interface BindingsPayload {
  slots?: BindingTable;
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
  t: "boot";
  sid: string;
  ver: number;
  seq: number;
  location: Location;
  errors?: ErrorMessage[];
  client?: BootClientConfig;
}

export interface BootClientConfig {
  endpoint?: string;
  upload?: string;
  debug?: boolean;
  [key: string]: unknown;
}

// Handler metadata
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

// Ref metadata
export interface RefEventMeta {
  listen?: string[];
  props?: string[];
}

export interface RefMeta {
  tag: string;
  events?: Record<string, RefEventMeta>;
}

export type RefMap = Record<string, RefMeta>;

export interface RefDelta {
  add?: RefMap;
  del?: string[];
}

// Dynamic slot kinds
export type DynKind = "text" | "attrs" | "list";

export interface DynamicSlot {
  kind: DynKind;
  text?: string;
  attrs?: Record<string, string>;
  list?: ListRow[];
}

export interface ListRow {
  key: string;
  slots?: number[];
  bindings?: BindingTable;
  slotPaths?: SlotPathDescriptor[];
  listPaths?: ListPathDescriptor[];
  componentPaths?: ComponentPathDescriptor[];
}

export interface SlotMeta {
  anchorId: number;
}

export interface SlotPathDescriptor {
  slot: number;
  componentId: string;
  elementPath?: number[];
  textChildIndex: number;
}

export interface ListPathDescriptor {
  slot: number;
  componentId: string;
  elementPath?: number[];
}

export interface ComponentPathDescriptor {
  componentId: string;
  parentId?: string;
  parentPath?: number[];
  firstChild?: number[];
  lastChild?: number[];
}

// Init message
export interface InitMessage extends TemplatePayload {
  t: "init";
  sid: string;
  ver: number;
  location: Location;
  seq?: number;
  errors?: ErrorMessage[];
}

// Frame delta
export interface FrameDelta {
  statics: boolean;
  slots: any;
}

// Navigation delta
export interface NavDelta {
  push?: string;
  replace?: string;
}

// Handler delta
export interface HandlerDelta {
  add?: HandlerMap;
  del?: string[];
}

// Frame metrics
export interface FrameMetrics {
  renderMs: number;
  ops: number;
  effectsMs?: number;
  maxEffectMs?: number;
  slowEffects?: number;
}

// Diff operations
export type SetTextOp = ["setText", number, string];
export type SetAttrsOp = ["setAttrs", number, Record<string, string>, string[]];
export type ListDelOp = ["del", string];
export type ListInsOp = [
  "ins",
  number,
  {
    key: string;
    html: string;
    slots?: number[];
    bindings?: BindingTable;
  },
];
export type ListMovOp = ["mov", number, number];
export type ListChildOp = ListDelOp | ListInsOp | ListMovOp;
export type ListOp = ["list", number, ...ListChildOp[]];
export type DiffOp = SetTextOp | SetAttrsOp | ListOp;

// Frame message
export interface FrameMessage {
  t: "frame";
  sid: string;
  seq?: number;
  ver: number;
  delta: FrameDelta;
  patch: DiffOp[];
  effects: any[];
  nav?: NavDelta | null;
  handlers: HandlerDelta;
  refs: RefDelta;
  bindings?: BindingsPayload;
  metrics: FrameMetrics;
}

export interface TemplateScope {
  componentId: string;
  parentId?: string;
  parentPath?: number[];
}

export interface TemplateMessage extends TemplatePayload {
  t: "template";
  sid: string;
  ver: number;
  scope?: TemplateScope;
}

// Join message
export interface JoinMessage {
  t: "join";
  sid: string;
  ver: number;
  ack: number;
  loc: Location;
}

// Resume message
export interface ResumeMessage {
  t: "resume";
  sid: string;
  from: number;
  to: number;
  errors?: ErrorMessage[];
}

// Error message
export interface ErrorMessage {
  t: "error";
  sid: string;
  code: string;
  message: string;
  details?: ErrorDetails;
}

export interface DiagnosticMessage {
  t: "diagnostic";
  sid: string;
  code: string;
  message: string;
  details?: ErrorDetails;
}

export interface PubsubControlMessage {
  t: "pubsub";
  op: "join" | "leave";
  topic: string;
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
  metadata?: Record<string, any>;
}

// Client event message
export interface ClientEventMessage {
  t: "evt";
  sid: string;
  hid: string;
  payload: EventPayload;
  cseq?: number;
}

// Client ack message
export interface ClientAckMessage {
  t: "ack";
  sid: string;
  seq: number;
}

// Client navigation message
export interface ClientNavMessage {
  t: "nav" | "pop";
  sid: string;
  path: string;
  q: string;
  hash?: string;
}

export interface ClientRecoverMessage {
  t: "recover";
  sid: string;
}

export interface UploadMeta {
  name: string;
  size: number;
  type?: string;
}

export type UploadClientOp = "change" | "progress" | "error" | "cancelled";

export interface UploadClientMessage {
  t: "upload";
  sid: string;
  id: string;
  op: UploadClientOp;
  meta?: UploadMeta;
  loaded?: number;
  total?: number;
  error?: string;
}

export interface UploadControlMessage {
  t: "upload";
  sid: string;
  id: string;
  op: "cancel";
}

export interface DOMRequestMessage {
  t: "domreq";
  id: string;
  ref: string;
  props?: string[];
}

export interface DOMResponseMessage {
  t: "domres";
  id: string;
  values?: Record<string, any>;
  error?: string;
}

// Event payload
export interface EventPayload {
  type: string;
  value?: string;
  checked?: boolean;
  key?: string;
  keyCode?: number;
  altKey?: boolean;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
  clientX?: number;
  clientY?: number;

  [key: string]: any;
}

// Union of all server messages
export type ServerMessage =
  | InitMessage
  | FrameMessage
  | TemplateMessage
  | JoinMessage
  | ResumeMessage
  | ErrorMessage
  | DiagnosticMessage
  | PubsubControlMessage
  | UploadControlMessage
  | DOMRequestMessage;

// Union of all client messages
export type ClientMessage =
  | ClientEventMessage
  | ClientAckMessage
  | ClientNavMessage
  | ClientRecoverMessage
  | UploadClientMessage
  | DOMResponseMessage;

// PondSocket EventMap for LiveUI protocol
export interface LiveUIEventMap {
  init: InitMessage;
  frame: FrameMessage;
  template: TemplateMessage;
  join: JoinMessage;
  resume: ResumeMessage;
  error: ErrorMessage;
  diagnostic: DiagnosticMessage;
  pubsub: PubsubControlMessage;
  evt: ClientEventMessage;
  ack: ClientAckMessage;
  nav: ClientNavMessage;
  pop: ClientNavMessage;
  recover: ClientRecoverMessage;
  upload: UploadClientMessage;
  domreq: DOMRequestMessage;
  domres: DOMResponseMessage;

  [key: string]: any; // Index signature for PondEventMap compatibility
}

// List record in DOM index
export interface ListRecord {
  container: Element;
  rows: Map<string, Element>;
}

// LiveUI options
export interface LiveUIOptions {
  endpoint?: string;
  uploadEndpoint?: string;
  autoConnect?: boolean;
  debug?: boolean;
  reconnect?: boolean;
  maxReconnectAttempts?: number;
  reconnectDelay?: number;
  boot?: BootPayload;

  [key: string]: any;
}

// Connection states
export type ConnectionState =
  | { status: "disconnected" }
  | { status: "connecting" }
  | { status: "connected"; sessionId: string; version: number }
  | { status: "reconnecting"; attempt: number }
  | { status: "error"; error: Error };

// Effect types
export type EffectType =
  | "scroll"
  | "focus"
  | "alert"
  | "dispatch"
  | "custom"
  | "toast"
  | "dom"
  | "domcall"
  | "scrollTop"
  | "push"
  | "replace"
  | "metadata"
  | "cookies"
  | "Cookies"
  | "Toast"
  | "Focus"
  | "ScrollTop"
  | "DOM"
  | "DOMCall"
  | "Push"
  | "Replace"
  | "componentBoot"
  | "ComponentBoot";

export interface ScrollEffect {
  type: "scroll" | "ScrollTop";
  selector?: string;
  behavior?: ScrollBehavior;
  block?: ScrollLogicalPosition;
}

export interface FocusEffect {
  type: "focus" | "Focus";
  selector?: string;
  Selector?: string; // Go capitalized field name
}

export interface AlertEffect {
  type: "alert";
  message: string;
}

export interface ToastEffect {
  type: "toast" | "Toast";
  message?: string;
  Message?: string; // Go capitalized field name
  duration?: number;
  variant?: "info" | "success" | "warning" | "error";
}

export interface PushEffect {
  type: "push" | "Push";
  url?: string;
  URL?: string; // Go capitalized field name
}

export interface ReplaceEffect {
  type: "replace" | "Replace";
  url?: string;
  URL?: string; // Go capitalized field name
}

export interface DispatchEffect {
  type: "dispatch";
  eventName: string;
  detail?: any;
}

export interface CustomEffect {
  type: "custom";
  name: string;
  data: any;
}

export interface DOMCallEffect {
  type: "domcall" | "DOMCall";
  ref?: string;
  Ref?: string;
  method?: string;
  Method?: string;
  args?: any[];
  Args?: any[];
}

export interface DOMActionEffect {
  type: "dom" | "DOM";
  kind?: string;
  Kind?: string;
  ref?: string;
  Ref?: string;
  method?: string;
  Method?: string;
  args?: any[];
  Args?: any[];
  prop?: string;
  Prop?: string;
  value?: any;
  Value?: any;
  "class"?: string;
  Class?: string;
  on?: boolean;
  On?: boolean;
  behavior?: ScrollBehavior;
  Behavior?: ScrollBehavior;
  block?: ScrollLogicalPosition;
  Block?: ScrollLogicalPosition;
  inline?: ScrollLogicalPosition;
  Inline?: ScrollLogicalPosition;
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

export interface MetadataLinkPayload {
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

export interface MetadataScriptPayload {
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
  type: "metadata";
  title?: string;
  description?: string;
  clearDescription?: boolean;
  metaAdd?: MetadataTagPayload[];
  metaRemove?: string[];
  linkAdd?: MetadataLinkPayload[];
  linkRemove?: string[];
  scriptAdd?: MetadataScriptPayload[];
  scriptRemove?: string[];
}

export interface CookieEffect {
  type: "cookies";
  endpoint?: string;
  Endpoint?: string;
  sid?: string;
  SID?: string;
  token?: string;
  Token?: string;
  method?: string;
  Method?: string;
}

export interface ComponentBootEffect {
  type: "componentBoot" | "ComponentBoot";
  componentId: string;
  html: string;
  slots: number[];
  listSlots?: number[];
  slotPaths?: SlotPathDescriptor[];
  listPaths?: ListPathDescriptor[];
  componentPaths?: ComponentPathDescriptor[];
  bindings?: BindingTable;
}

export interface BootEffect {
  type: "boot";
  boot: BootPayload;
}

export type Effect =
  | ScrollEffect
  | FocusEffect
  | AlertEffect
  | ToastEffect
  | PushEffect
  | ReplaceEffect
  | DispatchEffect
  | CustomEffect
  | DOMActionEffect
  | DOMCallEffect
  | MetadataEffect
  | CookieEffect
  | ComponentBootEffect
  | BootEffect;

// Performance metrics
export interface PerformanceMetrics {
  patchesApplied: number;
  averagePatchTime: number;
  framesReceived: number;
  eventsProcessed: number;
  reconnections: number;
  uptime: number;
  effectsMs: number;
  maxEffectMs: number;
  slowEffects: number;
  framesBuffered: number;
  framesDropped: number;
  sequenceGaps: number;
}

// Optimistic update
export interface OptimisticUpdate {
  id: string;
  patches: DiffOp[];
  inverseOps: DiffOp[];
  timestamp: number;
}

// Lifecycle events
export interface LiveUIEvents {
  connected: { sessionId: string; version: number };
  disconnected: void;
  reconnecting: { attempt: number };
  reconnected: { sessionId: string };
  error: { error: Error; context?: string };
  diagnostic: { diagnostic: DiagnosticMessage };
  frameApplied: { operations: number; duration: number };
  stateChanged: { from: ConnectionState; to: ConnectionState };
  effect: { effect: Effect };
  metricsUpdated: PerformanceMetrics;
  rollback: { id: string; patches: DiffOp[] };
  resumed: { from: number; to: number };
}

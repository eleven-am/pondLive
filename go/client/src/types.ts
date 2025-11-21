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
  | 'setRef'
  | 'delRef'
  | 'setComponent'
  | 'replaceNode'
  | 'addChild'
  | 'delChild'
  | 'moveChild';

export interface Patch {
  path: number[];
  op: OpKind;
  value?: any;
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
  path?: string;
  query?: string;
  hash?: string;
  replace?: string;
}

export interface UploadMeta {
  uploadId: string;
  accept?: string[];
  multiple?: boolean;
  maxSize?: number;
}

export interface StructuredNode {
  componentId?: string;
  tag?: string;
  text?: string;
  comment?: string;
  fragment?: boolean;
  key?: string;
  children?: StructuredNode[];
  unsafeHtml?: string;
  attrs?: Record<string, string[]>; 
  style?: Record<string, string>;
  styles?: Record<string, Record<string, string>>;
  refId?: string;
  handlers?: HandlerMeta[];
  router?: RouterMeta;
  upload?: UploadMeta;
}


export interface ClientNode extends StructuredNode {
  el: Node | null; 
  children?: ClientNode[]; 
}

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

export interface BootPayload {
  t: 'boot';
  sid: string;
  ver: number;
  seq: number;
  json: string; 
  location: Location;
  client?: ClientConfig;
  errors?: any[];
}

export interface DOMActionEffect {
  type: string; 
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

export interface FramePayload {
  t: 'frame';
  sid: string;
  seq: number;
  ver: number;
  patch: Patch[];
  effects?: DOMActionEffect[];
  nav?: any;
  metrics?: any;
}

export interface InitPayload {
  t: 'init';
  sid: string;
  ver: number;
  location: Location;
  seq: number;
  errors?: any[];
}

export interface UploadControlMessage {
  t?: 'upload';
  op: 'cancel' | 'error' | 'change' | 'progress' | 'cancelled';
  id: string;
  error?: string;
  meta?: {
    name: string;
    size: number;
    type: string;
  };
  loaded?: number;
  total?: number;
}

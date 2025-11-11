import { registerUploadDelegate } from "./events";
import { resolveNodeInComponent } from "./manifest";
import type { ComponentRange } from "./componentRanges";
import type {
  UploadBindingDescriptor,
  UploadClientMessage,
  UploadControlMessage,
  UploadMeta,
} from "./types";

type UploadManagerOptions = {
  getSessionId: () => string | null;
  getEndpoint: () => string | null;
  send: (payload: UploadClientMessage) => void;
  isConnected: () => boolean;
};

type ActiveUpload = {
  xhr: XMLHttpRequest;
  input: HTMLInputElement | null;
};

type UploadBindingMeta = {
  id: string;
  accept?: string[];
  multiple: boolean;
  maxSize?: number;
};

const elementBindings = new WeakMap<Element, UploadBindingMeta>();
const idToElement = new Map<string, Element>();
const componentUploadIds = new Map<string, Set<string>>();

function removeUploadIdFromComponents(uploadId: string): void {
  for (const [componentId, ids] of componentUploadIds.entries()) {
    if (ids.delete(uploadId) && ids.size === 0) {
      componentUploadIds.delete(componentId);
    }
  }
}

function detachUploadBinding(uploadId: string): void {
  if (typeof uploadId !== "string" || uploadId.length === 0) {
    return;
  }
  const element = idToElement.get(uploadId);
  if (element) {
    elementBindings.delete(element);
    idToElement.delete(uploadId);
  }
  removeUploadIdFromComponents(uploadId);
}

function setInputAttributes(
  input: HTMLInputElement,
  descriptor: UploadBindingDescriptor,
): void {
  if (Array.isArray(descriptor.accept) && descriptor.accept.length > 0) {
    input.setAttribute("accept", descriptor.accept.join(","));
  } else {
    input.removeAttribute("accept");
  }
  if (descriptor.multiple) {
    input.setAttribute("multiple", "multiple");
  } else {
    input.removeAttribute("multiple");
  }
}

function attachUploadDescriptor(
  descriptor: UploadBindingDescriptor,
  overrides?: Map<string, ComponentRange>,
): boolean {
  if (!descriptor || typeof descriptor.uploadId !== "string") {
    return false;
  }
  const node = resolveNodeInComponent(
    descriptor.componentId,
    descriptor.path,
    overrides,
  );
  if (!(node instanceof HTMLInputElement)) {
    return false;
  }
  if (node.type && node.type.toLowerCase() !== "file") {
    node.type = "file";
  }
  detachUploadBinding(descriptor.uploadId);
  setInputAttributes(node, descriptor);
  const meta: UploadBindingMeta = {
    id: descriptor.uploadId,
    multiple: !!descriptor.multiple,
  };
  if (Array.isArray(descriptor.accept) && descriptor.accept.length > 0) {
    meta.accept = [...descriptor.accept];
  }
  if (typeof descriptor.maxSize === "number" && !Number.isNaN(descriptor.maxSize)) {
    meta.maxSize = descriptor.maxSize;
  }
  elementBindings.set(node, meta);
  idToElement.set(descriptor.uploadId, node);
  return true;
}

export function getUploadBinding(
  element: Element | null | undefined,
): UploadBindingMeta | undefined {
  if (!element) {
    return undefined;
  }
  return elementBindings.get(element) ?? undefined;
}

export function replaceUploadBindingsForComponent(
  componentId: string,
  descriptors: UploadBindingDescriptor[] | null | undefined,
  overrides?: Map<string, ComponentRange>,
): void {
  if (typeof componentId !== "string" || componentId.length === 0) {
    return;
  }
  const existing = componentUploadIds.get(componentId);
  if (existing) {
    existing.forEach((id) => detachUploadBinding(id));
    componentUploadIds.delete(componentId);
  }
  if (!Array.isArray(descriptors) || descriptors.length === 0) {
    return;
  }
  const next = new Set<string>();
  for (const descriptor of descriptors) {
    if (!descriptor || typeof descriptor.uploadId !== "string") {
      continue;
    }
    const normalized: UploadBindingDescriptor = {
      ...descriptor,
      componentId,
    };
    if (attachUploadDescriptor(normalized, overrides)) {
      next.add(descriptor.uploadId);
    }
  }
  if (next.size > 0) {
    componentUploadIds.set(componentId, next);
  }
}

export function applyUploadBindings(
  descriptors: UploadBindingDescriptor[] | null | undefined,
  overrides?: Map<string, ComponentRange>,
): void {
  if (!Array.isArray(descriptors) || descriptors.length === 0) {
    return;
  }
  const grouped = new Map<string, UploadBindingDescriptor[]>();
  for (const descriptor of descriptors) {
    if (!descriptor || typeof descriptor.componentId !== "string") {
      continue;
    }
    const list = grouped.get(descriptor.componentId) ?? [];
    list.push(descriptor);
    grouped.set(descriptor.componentId, list);
  }
  for (const [componentId, list] of grouped.entries()) {
    replaceUploadBindingsForComponent(componentId, list, overrides);
  }
}

export function primeUploadBindings(
  descriptors: UploadBindingDescriptor[] | null | undefined,
  overrides?: Map<string, ComponentRange>,
): void {
  componentUploadIds.forEach((ids) => {
    ids.forEach((id) => detachUploadBinding(id));
  });
  componentUploadIds.clear();
  idToElement.clear();
  if (!Array.isArray(descriptors) || descriptors.length === 0) {
    return;
  }
  applyUploadBindings(descriptors, overrides);
}

export function clearUploadBindings(): void {
  primeUploadBindings(null);
}

const enum UploadOps {
  Change = "change",
  Progress = "progress",
  Error = "error",
  Cancelled = "cancelled",
}

export class UploadManager {
  private options: UploadManagerOptions;
  private uploads: Map<string, ActiveUpload> = new Map();

  constructor(options: UploadManagerOptions) {
    this.options = options;
    registerUploadDelegate((event, type) => this.handleDomEvent(event, type));
  }

  public dispose(): void {
    registerUploadDelegate(null);
    this.abortAll();
  }

  public onDisconnect(): void {
    this.abortAll();
  }

  public handleControl(message: UploadControlMessage): void {
    if (!message || message.op !== "cancel") {
      return;
    }
    const active = this.uploads.get(message.id);
    if (active?.xhr) {
      active.xhr.abort();
    }
  }

  public handleDomEvent(event: Event, eventType: string): void {
    if (eventType !== "change") {
      return;
    }
    const target = event.target as HTMLElement | null;
    if (!target) {
      return;
    }
    const input = this.resolveInput(target);
    if (!input) {
      return;
    }
    const binding = getUploadBinding(input);
    const uploadId = binding?.id;
    if (!uploadId) {
      return;
    }
    const files = input.files;
    if (!files || files.length === 0) {
      return;
    }
    const file = files[0];
    const sessionId = this.options.getSessionId();
    if (!sessionId || !this.options.isConnected()) {
      console.warn(
        "[LiveUI] upload ignored because the session is not connected",
      );
      if (sessionId) {
        this.sendMessage({
          t: "upload",
          sid: sessionId,
          id: uploadId,
          op: UploadOps.Error,
          error: "not connected",
        });
      }
      return;
    }
    this.sendMessage({
      t: "upload",
      sid: sessionId,
      id: uploadId,
      op: UploadOps.Change,
      meta: this.buildMeta(file),
    });
    this.startUpload(sessionId, uploadId, file, input);
  }

  private startUpload(
    sessionId: string,
    uploadId: string,
    file: File,
    input: HTMLInputElement,
  ): void {
    const current = this.uploads.get(uploadId);
    if (current?.xhr) {
      current.xhr.abort();
    }
    const endpoint = this.options.getEndpoint();
    if (!endpoint) {
      console.warn("[LiveUI] upload endpoint missing; aborting file upload");
      this.sendMessage({
        t: "upload",
        sid: sessionId,
        id: uploadId,
        op: UploadOps.Error,
        error: "upload endpoint missing",
      });
      return;
    }
    const url = this.buildUploadURL(endpoint, sessionId, uploadId);
    const xhr = new XMLHttpRequest();
    xhr.open("POST", url, true);

    xhr.upload.onprogress = (evt: ProgressEvent<EventTarget>) => {
      if (!evt.lengthComputable) {
        return;
      }
      const sid = this.options.getSessionId();
      if (!sid) {
        return;
      }
      this.sendMessage({
        t: "upload",
        sid,
        id: uploadId,
        op: UploadOps.Progress,
        loaded: evt.loaded,
        total: evt.total,
      });
    };

    const finalize = () => {
      const active = this.uploads.get(uploadId);
      if (active?.xhr === xhr) {
        this.uploads.delete(uploadId);
      }
    };

    xhr.onload = () => {
      finalize();
      if (xhr.status >= 200 && xhr.status < 300) {
        return;
      }
      const sid = this.options.getSessionId();
      if (!sid) {
        return;
      }
      this.sendMessage({
        t: "upload",
        sid,
        id: uploadId,
        op: UploadOps.Error,
        error: `HTTP ${xhr.status}`,
      });
    };

    xhr.onerror = () => {
      finalize();
      const sid = this.options.getSessionId();
      if (!sid) {
        return;
      }
      this.sendMessage({
        t: "upload",
        sid,
        id: uploadId,
        op: UploadOps.Error,
        error: "network error",
      });
    };

    xhr.onabort = () => {
      finalize();
      const sid = this.options.getSessionId();
      if (!sid) {
        return;
      }
      this.sendMessage({
        t: "upload",
        sid,
        id: uploadId,
        op: UploadOps.Cancelled,
      });
    };

    const data = new FormData();
    data.append("file", file, file.name);
    xhr.send(data);

    this.uploads.set(uploadId, { xhr, input });
    input.value = "";
  }

  private buildUploadURL(base: string, sid: string, uploadId: string): string {
    const normalized = base.endsWith("/") ? base : `${base}/`;
    return `${normalized}${encodeURIComponent(sid)}/${encodeURIComponent(uploadId)}`;
  }

  private buildMeta(file: File): UploadMeta {
    return {
      name: file.name,
      size: file.size,
      type: file.type || undefined,
    };
  }

  private resolveInput(target: HTMLElement): HTMLInputElement | null {
    if (target instanceof HTMLInputElement) {
      return target;
    }
    if (target.closest) {
      const match = target.closest('input[type="file"][data-pond-upload]');
      return match instanceof HTMLInputElement ? match : null;
    }
    return null;
  }

  private sendMessage(payload: UploadClientMessage): void {
    try {
      this.options.send(payload);
    } catch (error) {
      console.error("[LiveUI] failed to send upload message", error);
    }
  }

  private abortAll(): void {
    for (const [, active] of this.uploads) {
      try {
        active.xhr.abort();
      } catch (error) {
        console.error("[LiveUI] failed to abort upload", error);
      }
    }
    this.uploads.clear();
  }
}

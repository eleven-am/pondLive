import { resolveNodeInComponent, type ComponentRange } from './manifest';
import type {
  UploadBindingDescriptor,
  UploadControlMessage,
  UploadMeta,
} from './types';
import type { LiveRuntime } from './runtime';
import { Logger } from './logger';

type BindingRecord = {
  id: string;
  element: HTMLInputElement;
  descriptor: UploadBindingDescriptor;
  changeHandler: (event: Event) => void;
};

type ActiveUpload = {
  xhr: XMLHttpRequest;
  element: HTMLInputElement;
};

export class UploadManager {
  private bindings = new Map<string, BindingRecord>();
  private elementToUpload = new WeakMap<Element, string>();
  private active = new Map<string, ActiveUpload>();
  private componentBindings = new Map<string, Set<string>>();

  constructor(private runtime: LiveRuntime) {}

  clear(): void {
    this.bindings.forEach((binding) => {
      binding.element.removeEventListener('change', binding.changeHandler);
    });
    this.bindings.clear();
    this.componentBindings.clear();
    this.active.forEach((upload) => upload.xhr.abort());
    this.active.clear();
    Logger.debug('[Uploads]', 'cleared bindings and active uploads');
  }

  prime(
    descriptors?: UploadBindingDescriptor[] | null,
    overrides?: Map<string, ComponentRange>,
  ): void {
    this.clear();
    this.registerBindings(descriptors, overrides);
    Logger.debug('[Uploads]', 'primed upload bindings', { count: descriptors?.length ?? 0 });
  }

  registerBindings(
    descriptors?: UploadBindingDescriptor[] | null,
    overrides?: Map<string, ComponentRange>,
    options?: { replace?: boolean },
  ): void {
    if (!Array.isArray(descriptors)) {
      return;
    }
    Logger.debug('[Uploads]', 'register bindings', {
      count: descriptors.length,
      replace: options?.replace !== false,
    });
    if (options?.replace === false) {
      descriptors.forEach((descriptor) => {
        if (!descriptor || typeof descriptor.uploadId !== 'string' || descriptor.uploadId.length === 0) {
          return;
        }
        this.attachDescriptor(descriptor, overrides);
      });
      return;
    }
    const grouped = new Map<string, UploadBindingDescriptor[]>();
    descriptors.forEach((descriptor) => {
      if (!descriptor || typeof descriptor.uploadId !== 'string' || descriptor.uploadId.length === 0) {
        return;
      }
      const componentId = descriptor.componentId || '__root__';
      const list = grouped.get(componentId) ?? [];
      list.push(descriptor);
      grouped.set(componentId, list);
    });
    grouped.forEach((list, componentId) => {
      this.replaceBindingsForComponent(componentId, list, overrides);
    });
  }

  replaceBindingsForComponent(
    componentId: string,
    descriptors?: UploadBindingDescriptor[] | null,
    overrides?: Map<string, ComponentRange>,
  ): void {
    const id = componentId || '__root__';
    const existing = this.componentBindings.get(id);
    if (existing) {
      existing.forEach((uploadId) => this.detachBinding(uploadId));
      this.componentBindings.delete(id);
    }
    if (!descriptors || descriptors.length === 0) {
      return;
    }
    const next = new Set<string>();
    descriptors.forEach((descriptor) => {
      if (!descriptor || typeof descriptor.uploadId !== 'string' || descriptor.uploadId.length === 0) {
        return;
      }
      this.attachDescriptor(descriptor, overrides);
      next.add(descriptor.uploadId);
    });
    if (next.size > 0) {
      this.componentBindings.set(id, next);
    }
    Logger.debug('[Uploads]', 'component bindings replaced', {
      componentId: id,
      count: descriptors?.length ?? 0,
    });
  }

  handleControl(message: UploadControlMessage): void {
    if (!message || !message.id) {
      return;
    }
    Logger.debug('[Uploads]', 'control message', { op: message.op, id: message.id });
    if (message.op === 'cancel') {
      this.abortUpload(message.id, true);
    } else if (message.op === 'error') {
      this.abortUpload(message.id, true);
    }
  }

  private attachDescriptor(
    descriptor: UploadBindingDescriptor,
    overrides?: Map<string, ComponentRange>,
  ): void {
    const node = resolveNodeInComponent(descriptor.componentId, descriptor.path, overrides);
    const element = this.resolveInput(node);
    if (!element) {
      return;
    }
    const uploadId = descriptor.uploadId;
    this.detachBinding(uploadId);
    const handler = () => this.handleInputChange(uploadId, element, descriptor);
    element.addEventListener('change', handler);
    this.syncAttributes(element, descriptor);
    this.bindings.set(uploadId, { id: uploadId, element, descriptor, changeHandler: handler });
    this.elementToUpload.set(element, uploadId);
    const componentId = descriptor.componentId || '__root__';
    const set = this.componentBindings.get(componentId) ?? new Set<string>();
    set.add(uploadId);
    this.componentBindings.set(componentId, set);
    Logger.debug('[Uploads]', 'attached upload descriptor', {
      uploadId,
      componentId,
      multiple: Boolean(descriptor.multiple),
    });
  }

  private detachBinding(uploadId: string): void {
    const binding = this.bindings.get(uploadId);
    if (!binding) {
      return;
    }
    binding.element.removeEventListener('change', binding.changeHandler);
    this.bindings.delete(uploadId);
    this.elementToUpload.delete(binding.element);
    this.abortUpload(uploadId, false);
    const componentId = binding.descriptor.componentId || '__root__';
    const set = this.componentBindings.get(componentId);
    if (set) {
      set.delete(uploadId);
      if (set.size === 0) {
        this.componentBindings.delete(componentId);
      }
    }
  }

  private resolveInput(node: Node | null): HTMLInputElement | null {
    if (!node) {
      return null;
    }
    if (node instanceof HTMLInputElement) {
      return node;
    }
    if (node instanceof Element) {
      if (node.tagName.toLowerCase() === 'input' && node.getAttribute('type') === 'file') {
        return node as HTMLInputElement;
      }
      const descendant = node.querySelector('input[type="file"]');
      if (descendant instanceof HTMLInputElement) {
        return descendant;
      }
    }
    return null;
  }

  private syncAttributes(element: HTMLInputElement, descriptor: UploadBindingDescriptor): void {
    if (Array.isArray(descriptor.accept) && descriptor.accept.length > 0) {
      element.setAttribute('accept', descriptor.accept.join(','));
    } else {
      element.removeAttribute('accept');
    }
    if (descriptor.multiple) {
      element.multiple = true;
    } else {
      element.multiple = false;
      element.removeAttribute('multiple');
    }
  }

  private handleInputChange(
    uploadId: string,
    element: HTMLInputElement,
    descriptor: UploadBindingDescriptor,
  ): void {
    const files = element.files;
    if (!files || files.length === 0) {
      this.sendUploadMessage({ op: 'cancelled', id: uploadId });
      this.abortUpload(uploadId, true);
      return;
    }
    const file =
      typeof files.item === 'function' ? files.item(0) : (files as any)[0] ?? null;
    if (!file) {
      this.sendUploadMessage({ op: 'cancelled', id: uploadId });
      return;
    }
    if (typeof descriptor.maxSize === 'number' && descriptor.maxSize > 0 && file.size > descriptor.maxSize) {
      this.sendUploadMessage({
        op: 'error',
        id: uploadId,
        error: `File exceeds maximum size (${descriptor.maxSize} bytes)`,
      });
      element.value = '';
      return;
    }
    const meta: UploadMeta = { name: file.name, size: file.size, type: file.type };
    this.sendUploadMessage({ op: 'change', id: uploadId, meta });
    this.startUpload(uploadId, file, element);
    Logger.debug('[Uploads]', 'input change processed', {
      uploadId,
      file: file.name,
      size: file.size,
    });
  }

  private startUpload(uploadId: string, file: File, element: HTMLInputElement): void {
    const sid = this.runtime.getSessionId();
    if (!sid) {
      return;
    }
    const base = this.runtime.getUploadEndpoint();
    const target = this.buildUploadURL(base, sid, uploadId);
    this.abortUpload(uploadId, false);
    const xhr = new XMLHttpRequest();
    xhr.upload.onprogress = (event: ProgressEvent<EventTarget>) => {
      const loaded = event.loaded ?? 0;
      const total = event.lengthComputable ? event.total : file.size;
      this.sendUploadMessage({ op: 'progress', id: uploadId, loaded, total });
    };
    xhr.onerror = () => {
      this.active.delete(uploadId);
      this.sendUploadMessage({ op: 'error', id: uploadId, error: 'Upload failed' });
    };
    xhr.onabort = () => {
      this.active.delete(uploadId);
      this.sendUploadMessage({ op: 'cancelled', id: uploadId });
    };
    xhr.onload = () => {
      this.active.delete(uploadId);
      if (xhr.status < 200 || xhr.status >= 300) {
        this.sendUploadMessage({
          op: 'error',
          id: uploadId,
          error: `Upload failed (${xhr.status})`,
        });
      } else {
        this.sendUploadMessage({ op: 'progress', id: uploadId, loaded: file.size, total: file.size });
        element.value = '';
      }
    };
    const form = new FormData();
    form.append('file', file);
    xhr.open('POST', target, true);
    xhr.send(form);
    this.active.set(uploadId, { xhr, element });
    Logger.debug('[Uploads]', 'upload started', { uploadId, target });
  }

  private abortUpload(uploadId: string, clearInput: boolean): void {
    const active = this.active.get(uploadId);
    if (!active) {
      return;
    }
    active.xhr.abort();
    if (clearInput) {
      active.element.value = '';
    }
    this.active.delete(uploadId);
    Logger.debug('[Uploads]', 'upload aborted', { uploadId, cleared: clearInput });
  }

  private sendUploadMessage(payload: {
    op: 'change' | 'progress' | 'error' | 'cancelled';
    id: string;
    meta?: UploadMeta;
    loaded?: number;
    total?: number;
    error?: string;
  }): void {
    if (!payload.id) {
      return;
    }
    this.runtime.sendUploadMessage(payload);
    Logger.debug('[Uploads]', 'sent upload message', payload);
  }

  private buildUploadURL(base: string | undefined, sid: string, uploadId: string): string {
    const normalized = (base && base.length > 0 ? base : '/pondlive/upload/').replace(/\/+$/, '');
    return `${normalized}/${encodeURIComponent(sid)}/${encodeURIComponent(uploadId)}`;
  }
}

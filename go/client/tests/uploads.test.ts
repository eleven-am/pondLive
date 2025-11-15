import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { UploadManager } from '../src/uploads';
import type { UploadBindingDescriptor, UploadControlMessage } from '../src/types';
import type { ComponentRange } from '../src/manifest';

class MockXHR {
  public status = 200;
  public upload = {
    onprogress: null as ((event: ProgressEvent<EventTarget>) => void) | null,
  };
  public onerror: (() => void) | null = null;
  public onload: (() => void) | null = null;
  public onabort: (() => void) | null = null;
  public openedURL: string | null = null;
  public sentData: FormData | null = null;
  public aborted = false;

  open(_method: string, url: string) {
    this.openedURL = url;
  }

  send(data: FormData) {
    this.sentData = data;
    if (this.upload.onprogress) {
      this.upload.onprogress({
        lengthComputable: true,
        loaded: 3,
        total: 3,
      } as ProgressEvent<EventTarget>);
    }
    if (this.onload) {
      this.onload();
    }
  }

  abort() {
    this.aborted = true;
    if (this.onabort) {
      this.onabort();
    }
  }
}

describe('UploadManager', () => {
  const OriginalXHR = globalThis.XMLHttpRequest;
  let lastXHR: MockXHR | null = null;

  beforeEach(() => {
    document.body.innerHTML = '';
    lastXHR = null;
    // @ts-expect-error mock xhr
    globalThis.XMLHttpRequest = class extends MockXHR {
      constructor() {
        super();
        lastXHR = this;
      }
    };
  });

  afterEach(() => {
    globalThis.XMLHttpRequest = OriginalXHR;
    vi.restoreAllMocks();
  });

  function createOverrides(container: Element): Map<string, ComponentRange> {
    return new Map<string, ComponentRange>([
      ['cmp', { container, startIndex: 0, endIndex: container.childNodes.length - 1 }],
    ]);
  }

  function createFileList(...files: File[]): FileList {
    const list: any = {
      length: files.length,
      item: (index: number) => files[index] ?? null,
    };
    files.forEach((file, index) => {
      list[index] = file;
    });
    return list as FileList;
  }

  function createRuntime() {
    return {
      getSessionId: () => 'test-session',
      getUploadEndpoint: () => '/pondlive/upload/',
      sendUploadMessage: vi.fn(),
    } as any;
  }

  describe('binding registration', () => {
    it('attaches change handler and emits upload messages', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['abc'], 'test.txt', { type: 'text/plain' })),
      });
      input.dispatchEvent(new Event('change'));

      expect(runtime.sendUploadMessage).toHaveBeenCalledWith(
        expect.objectContaining({ op: 'change', id: 'upload-1' }),
      );
      expect(runtime.sendUploadMessage).toHaveBeenCalledWith(
        expect.objectContaining({ op: 'progress', id: 'upload-1' }),
      );
    });

    it('syncs accept attribute from descriptor', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
        accept: ['image/png', 'image/jpeg'],
      };
      const root = document.getElementById('root') as HTMLElement;

      manager.registerBindings([descriptor], createOverrides(root));

      expect(input.getAttribute('accept')).toBe('image/png,image/jpeg');
    });

    it('syncs multiple attribute from descriptor', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
        multiple: true,
      };
      const root = document.getElementById('root') as HTMLElement;

      manager.registerBindings([descriptor], createOverrides(root));

      expect(input.multiple).toBe(true);
    });

    it('removes accept attribute when not in descriptor', () => {
      document.body.innerHTML = '<div id="root"><input type="file" accept="text/*" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;

      manager.registerBindings([descriptor], createOverrides(root));

      expect(input.hasAttribute('accept')).toBe(false);
    });

    it('replaces bindings for component', () => {
      document.body.innerHTML = '<div id="root"><input type="file" id="i1" /><input type="file" id="i2" /></div>';
      const input1 = document.getElementById('i1') as HTMLInputElement;
      const input2 = document.getElementById('i2') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const root = document.getElementById('root') as HTMLElement;

      manager.registerBindings(
        [{ componentId: 'cmp', path: [0], uploadId: 'upload-1' }],
        createOverrides(root),
      );

      const listeners1 = (manager as any).bindings.size;

      manager.registerBindings(
        [{ componentId: 'cmp', path: [1], uploadId: 'upload-2' }],
        createOverrides(root),
      );

      const listeners2 = (manager as any).bindings.size;
      expect(listeners2).toBe(listeners1);
      expect((manager as any).bindings.has('upload-1')).toBe(false);
      expect((manager as any).bindings.has('upload-2')).toBe(true);
    });
  });

  describe('file selection and validation', () => {
    it('rejects files above max size', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-2',
        maxSize: 1,
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['longer'], 'large.txt', { type: 'text/plain' })),
      });
      input.dispatchEvent(new Event('change'));

      expect(runtime.sendUploadMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          op: 'error',
          id: 'upload-2',
          error: expect.stringContaining('exceeds maximum size'),
        }),
      );
      expect(input.value).toBe('');
    });

    it('sends metadata when file is selected', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      const file = new File(['content'], 'doc.pdf', { type: 'application/pdf' });
      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(file),
      });
      input.dispatchEvent(new Event('change'));

      expect(runtime.sendUploadMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          op: 'change',
          id: 'upload-1',
          meta: {
            name: 'doc.pdf',
            size: 7,
            type: 'application/pdf',
          },
        }),
      );
    });

    it('sends cancelled when no file selected', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(),
      });
      input.dispatchEvent(new Event('change'));

      expect(runtime.sendUploadMessage).toHaveBeenCalledWith(
        expect.objectContaining({ op: 'cancelled', id: 'upload-1' }),
      );
    });
  });

  describe('XHR upload lifecycle', () => {
    it('posts to correct upload endpoint', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['data'], 'test.txt')),
      });
      input.dispatchEvent(new Event('change'));

      expect(lastXHR?.openedURL).toBe('/pondlive/upload/test-session/upload-1');
    });

    it('uses custom upload endpoint', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = {
        getSessionId: () => 'sid',
        getUploadEndpoint: () => '/custom/endpoint/',
        sendUploadMessage: vi.fn(),
      } as any;
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'up-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['x'], 'f.txt')),
      });
      input.dispatchEvent(new Event('change'));

      expect(lastXHR?.openedURL).toBe('/custom/endpoint/sid/up-1');
    });

    it('sends progress events', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['data'], 'test.txt')),
      });
      input.dispatchEvent(new Event('change'));

      expect(runtime.sendUploadMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          op: 'progress',
          id: 'upload-1',
          loaded: expect.any(Number),
          total: expect.any(Number),
        }),
      );
    });

    it('clears input value on successful upload', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      Object.defineProperty(input, 'value', {
        configurable: true,
        writable: true,
        value: 'C:\\fakepath\\test.txt',
      });
      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['data'], 'test.txt')),
      });

      input.dispatchEvent(new Event('change'));

      expect(input.value).toBe('');
    });

    it('handles upload error status', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      // @ts-expect-error mock xhr
      globalThis.XMLHttpRequest = class extends MockXHR {
        constructor() {
          super();
          this.status = 500;
          lastXHR = this;
        }
      };

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['data'], 'test.txt')),
      });
      input.dispatchEvent(new Event('change'));

      expect(runtime.sendUploadMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          op: 'error',
          id: 'upload-1',
          error: expect.stringContaining('500'),
        }),
      );
    });

    it('handles network error', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      // @ts-expect-error mock xhr
      globalThis.XMLHttpRequest = class extends MockXHR {
        constructor() {
          super();
          lastXHR = this;
        }
        send() {
          if (this.onerror) {
            this.onerror();
          }
        }
      };

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['data'], 'test.txt')),
      });
      input.dispatchEvent(new Event('change'));

      expect(runtime.sendUploadMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          op: 'error',
          id: 'upload-1',
          error: 'Upload failed',
        }),
      );
    });

    it('handles abort', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      // @ts-expect-error mock xhr
      globalThis.XMLHttpRequest = class extends MockXHR {
        constructor() {
          super();
          lastXHR = this;
        }
        send() {
          this.sentData = new FormData();
          // Don't call onload, simulate abort instead
        }
      };

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['data'], 'test.txt')),
      });
      input.dispatchEvent(new Event('change'));

      lastXHR?.abort();

      expect(runtime.sendUploadMessage).toHaveBeenCalledWith(
        expect.objectContaining({ op: 'cancelled', id: 'upload-1' }),
      );
    });
  });

  describe('control messages', () => {
    it('aborts upload on cancel control message', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      // @ts-expect-error mock xhr
      globalThis.XMLHttpRequest = class extends MockXHR {
        constructor() {
          super();
          lastXHR = this;
        }
        send() {
          this.sentData = new FormData();
          // Don't call onload, keep upload active
        }
      };

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['data'], 'test.txt')),
      });
      input.dispatchEvent(new Event('change'));

      const control: UploadControlMessage = {
        t: 'upload',
        sid: 'test-session',
        op: 'cancel',
        id: 'upload-1',
      };
      manager.handleControl(control);

      expect(lastXHR?.aborted).toBe(true);
    });

    it('aborts upload on error control message', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const input = document.querySelector('input') as HTMLInputElement;
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const descriptor: UploadBindingDescriptor = {
        componentId: 'cmp',
        path: [0],
        uploadId: 'upload-1',
      };
      const root = document.getElementById('root') as HTMLElement;
      manager.registerBindings([descriptor], createOverrides(root));

      // @ts-expect-error mock xhr
      globalThis.XMLHttpRequest = class extends MockXHR {
        constructor() {
          super();
          lastXHR = this;
        }
        send() {
          this.sentData = new FormData();
          // Don't call onload
        }
      };

      Object.defineProperty(input, 'files', {
        configurable: true,
        value: createFileList(new File(['data'], 'test.txt')),
      });
      input.dispatchEvent(new Event('change'));

      const control: UploadControlMessage = {
        t: 'upload',
        sid: 'test-session',
        op: 'error',
        id: 'upload-1',
      };
      manager.handleControl(control);

      expect(lastXHR?.aborted).toBe(true);
    });
  });

  describe('cleanup', () => {
    it('clears all bindings and aborts active uploads', () => {
      document.body.innerHTML = '<div id="root"><input type="file" id="i1" /><input type="file" id="i2" /></div>';
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const root = document.getElementById('root') as HTMLElement;

      manager.registerBindings(
        [
          { componentId: 'cmp', path: [0], uploadId: 'upload-1' },
          { componentId: 'cmp', path: [1], uploadId: 'upload-2' },
        ],
        createOverrides(root),
      );

      // @ts-expect-error mock xhr
      globalThis.XMLHttpRequest = class extends MockXHR {
        constructor() {
          super();
          lastXHR = this;
        }
        send() {
          this.sentData = new FormData();
        }
      };

      const input1 = document.getElementById('i1') as HTMLInputElement;
      Object.defineProperty(input1, 'files', {
        configurable: true,
        value: createFileList(new File(['a'], 'a.txt')),
      });
      input1.dispatchEvent(new Event('change'));

      manager.clear();

      expect((manager as any).bindings.size).toBe(0);
      expect((manager as any).active.size).toBe(0);
      expect(lastXHR?.aborted).toBe(true);
    });

    it('prime replaces all bindings', () => {
      document.body.innerHTML = '<div id="root"><input type="file" /></div>';
      const runtime = createRuntime();
      const manager = new UploadManager(runtime);
      const root = document.getElementById('root') as HTMLElement;

      manager.prime(
        [{ componentId: 'cmp', path: [0], uploadId: 'upload-1' }],
        createOverrides(root),
      );

      expect((manager as any).bindings.size).toBe(1);

      manager.prime(
        [{ componentId: 'cmp', path: [0], uploadId: 'upload-2' }],
        createOverrides(root),
      );

      expect((manager as any).bindings.size).toBe(1);
      expect((manager as any).bindings.has('upload-1')).toBe(false);
      expect((manager as any).bindings.has('upload-2')).toBe(true);
    });
  });
});

import { Logger } from './logger';
import { ClientNode, UploadMeta, UploadControlMessage } from './types';

type BindingRecord = {
    node: ClientNode;
    element: HTMLInputElement;
    meta: UploadMeta;
    changeHandler: (event: Event) => void;
};

type ActiveUpload = {
    xhr: XMLHttpRequest;
    element: HTMLInputElement;
};

export interface UploadRuntime {
    getSessionId(): string | undefined;
    getUploadEndpoint(): string;
    sendUploadMessage(payload: any): void;
}

export interface ClientConfig {
    endpoint?: string;
    upload?: string;
    debug?: boolean;
}

export class UploadManager {
    private bindings = new Map<string, BindingRecord>();
    private active = new Map<string, ActiveUpload>();

    constructor(private runtime: UploadRuntime) { }

    bind(node: ClientNode, meta: UploadMeta) {
        if (!node.el || !(node.el instanceof HTMLInputElement)) {
            Logger.warn('Uploads', 'Upload binding requires an input element', node);
            return;
        }
        const element = node.el;
        const uploadId = meta.uploadId;

        
        this.unbind(node);

        const handler = () => this.handleInputChange(uploadId, element, meta);
        element.addEventListener('change', handler);

        
        if (meta.accept && meta.accept.length > 0) {
            element.setAttribute('accept', meta.accept.join(','));
        } else {
            element.removeAttribute('accept');
        }

        if (meta.multiple) {
            Logger.warn('Uploads', 'Multiple file selection not supported; forcing single file');
            element.removeAttribute('multiple');
        } else {
            element.removeAttribute('multiple');
        }

        this.bindings.set(uploadId, { node, element, meta, changeHandler: handler });
        Logger.debug('Uploads', 'Bound upload', uploadId);
    }

    unbind(node: ClientNode) {
        
        
        
        
        
        
        

        for (const [id, binding] of this.bindings.entries()) {
            if (binding.node === node) {
                this.detachBinding(id);
                return;
            }
        }
    }

    private detachBinding(uploadId: string) {
        const binding = this.bindings.get(uploadId);
        if (!binding) return;

        binding.element.removeEventListener('change', binding.changeHandler);
        this.bindings.delete(uploadId);
        this.abortUpload(uploadId, false);
        Logger.debug('Uploads', 'Unbound upload', uploadId);
    }

    handleControl(message: UploadControlMessage) {
        if (!message || !message.id) return;
        Logger.debug('Uploads', 'Control message', message);
        if (message.op === 'cancel' || message.op === 'error') {
            this.abortUpload(message.id, true);
        }
    }

    private handleInputChange(uploadId: string, element: HTMLInputElement, meta: UploadMeta) {
        const files = element.files;
        if (!files || files.length === 0) {
            this.sendMessage({ op: 'cancelled', id: uploadId });
            this.abortUpload(uploadId, true);
            return;
        }

        const file = files[0]; 
        if (meta.multiple && files.length > 1) {
            this.sendMessage({
                op: 'error',
                id: uploadId,
                error: 'Multiple file uploads are not supported yet'
            });
            element.value = '';
            return;
        }

        if (!file) {
            this.sendMessage({ op: 'cancelled', id: uploadId });
            return;
        }

        if (meta.maxSize && meta.maxSize > 0 && file.size > meta.maxSize) {
            this.sendMessage({
                op: 'error',
                id: uploadId,
                error: `File exceeds maximum size (${meta.maxSize} bytes)`
            });
            element.value = '';
            return;
        }

        const fileMeta = { name: file.name, size: file.size, type: file.type };
        this.sendMessage({ op: 'change', id: uploadId, meta: fileMeta });
        this.startUpload(uploadId, file, element);
    }

    private startUpload(uploadId: string, file: File, element: HTMLInputElement) {
        const sid = this.runtime.getSessionId();
        if (!sid) return;

        const base = this.runtime.getUploadEndpoint();
        const target = `${base.replace(/\/+$/, '')}/${encodeURIComponent(sid)}/${encodeURIComponent(uploadId)}`;

        this.abortUpload(uploadId, false);

        const xhr = new XMLHttpRequest();
        xhr.upload.onprogress = (event) => {
            const loaded = event.loaded;
            const total = event.lengthComputable ? event.total : file.size;
            this.sendMessage({ op: 'progress', id: uploadId, loaded, total });
        };

        xhr.onerror = () => {
            this.active.delete(uploadId);
            this.sendMessage({ op: 'error', id: uploadId, error: 'Upload failed' });
        };

        xhr.onabort = () => {
            this.active.delete(uploadId);
            this.sendMessage({ op: 'cancelled', id: uploadId });
        };

        xhr.onload = () => {
            this.active.delete(uploadId);
            if (xhr.status < 200 || xhr.status >= 300) {
                this.sendMessage({ op: 'error', id: uploadId, error: `Upload failed (${xhr.status})` });
            } else {
                this.sendMessage({ op: 'progress', id: uploadId, loaded: file.size, total: file.size });
                element.value = '';
            }
        };

        const form = new FormData();
        form.append('file', file);
        xhr.open('POST', target, true);
        xhr.send(form);

        this.active.set(uploadId, { xhr, element });
        Logger.debug('Uploads', 'Started upload', { uploadId, target });
    }

    private abortUpload(uploadId: string, clearInput: boolean) {
        const active = this.active.get(uploadId);
        if (!active) return;

        active.xhr.abort();
        if (clearInput) {
            active.element.value = '';
        }
        this.active.delete(uploadId);
    }

    private sendMessage(payload: any) {
        this.runtime.sendUploadMessage(payload);
    }
}

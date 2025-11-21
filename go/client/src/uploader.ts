import {UploadMeta, UploaderConfig, UploadMessageCallback} from './types';
import {UploadMessage} from './protocol';

interface ActiveUpload {
    xhr: XMLHttpRequest;
    input: HTMLInputElement | null;
}

export class Uploader {
    private readonly endpoint: string;
    private readonly sessionId: string;
    private readonly onMessage: UploadMessageCallback;
    private active = new Map<string, ActiveUpload>();

    constructor(config: UploaderConfig) {
        this.endpoint = config.endpoint.replace(/\/+$/, '');
        this.sessionId = config.sessionId;
        this.onMessage = config.onMessage;
    }

    upload(meta: UploadMeta, files: FileList, input?: HTMLInputElement): void {
        const uploadId = meta.uploadId;

        if (files.length === 0) {
            this.send({t: 'upload', op: 'cancelled', id: uploadId});
            return;
        }

        const file = files[0];

        if (meta.maxSize && meta.maxSize > 0 && file.size > meta.maxSize) {
            this.send({
                t: 'upload',
                op: 'error',
                id: uploadId,
                error: `File exceeds maximum size (${meta.maxSize} bytes)`
            });
            if (input) input.value = '';
            return;
        }

        this.send({
            t: 'upload',
            op: 'change',
            id: uploadId,
            meta: {name: file.name, size: file.size, type: file.type}
        });

        this.startUpload(uploadId, file, input ?? null);
    }

    cancel(uploadId: string): void {
        const active = this.active.get(uploadId);
        if (active) {
            active.xhr.abort();
            if (active.input) active.input.value = '';
            this.active.delete(uploadId);
        }
    }

    private startUpload(uploadId: string, file: File, input: HTMLInputElement | null): void {
        this.cancel(uploadId);

        const target = `${this.endpoint}/${encodeURIComponent(this.sessionId)}/${encodeURIComponent(uploadId)}`;
        const xhr = new XMLHttpRequest();

        xhr.upload.onprogress = (event) => {
            const loaded = event.loaded;
            const total = event.lengthComputable ? event.total : file.size;
            this.send({t: 'upload', op: 'progress', id: uploadId, loaded, total});
        };

        xhr.onerror = () => {
            this.active.delete(uploadId);
            this.send({t: 'upload', op: 'error', id: uploadId, error: 'Upload failed'});
        };

        xhr.onabort = () => {
            this.active.delete(uploadId);
            this.send({t: 'upload', op: 'cancelled', id: uploadId});
        };

        xhr.onload = () => {
            this.active.delete(uploadId);
            if (xhr.status < 200 || xhr.status >= 300) {
                this.send({t: 'upload', op: 'error', id: uploadId, error: `Upload failed (${xhr.status})`});
            } else {
                this.send({t: 'upload', op: 'progress', id: uploadId, loaded: file.size, total: file.size});
                if (input) input.value = '';
            }
        };

        const form = new FormData();
        form.append('file', file);
        xhr.open('POST', target, true);
        xhr.send(form);

        this.active.set(uploadId, {xhr, input});
    }

    private send(msg: UploadMessage): void {
        this.onMessage(msg);
    }
}

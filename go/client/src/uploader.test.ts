import {describe, it, expect, beforeEach, vi, Mock} from 'vitest';
import {Uploader} from './uploader';
import {UploadMessage} from './protocol';
import {UploadMeta, UploaderConfig} from './types';

const createMockFileList = (files: File[]): FileList => {
    const fileList = {
        length: files.length,
        item: (index: number) => files[index] ?? null,
        [Symbol.iterator]: function* () {
            for (const file of files) yield file;
        }
    } as unknown as FileList;

    files.forEach((file, index) => {
        Object.defineProperty(fileList, index, {value: file, enumerable: true});
    });

    return fileList;
};

describe('Uploader', () => {
    // @ts-ignore
    let onMessage: Mock<[UploadMessage], void>;
    let uploader: Uploader;
    let mockXHR: {
        open: Mock;
        send: Mock;
        abort: Mock;
        upload: { onprogress: ((e: ProgressEvent) => void) | null };
        onload: (() => void) | null;
        onerror: (() => void) | null;
        onabort: (() => void) | null;
        status: number;
    };

    const createFile = (name: string, size: number, type: string): File => {
        const content = new Array(size).fill('a').join('');
        return new File([content], name, {type});
    };

    beforeEach(() => {
        onMessage = vi.fn();

        mockXHR = {
            open: vi.fn(),
            send: vi.fn(),
            abort: vi.fn(),
            upload: {onprogress: null},
            onload: null,
            onerror: null,
            onabort: null,
            status: 200
        };

        vi.stubGlobal('XMLHttpRequest', vi.fn(() => mockXHR));

        const config: UploaderConfig = {
            endpoint: 'http://localhost:8080/upload',
            sessionId: 'test-session',
            onMessage
        };
        uploader = new Uploader(config);
    });

    describe('upload', () => {
        it('should send cancelled message when no files selected', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const files = createMockFileList([]);

            uploader.upload(meta, files);

            expect(onMessage).toHaveBeenCalledWith({
                t: 'upload',
                op: 'cancelled',
                id: 'upload-1'
            });
        });

        it('should send error message when file exceeds max size', () => {
            const meta: UploadMeta = {uploadId: 'upload-1', maxSize: 100};
            const file = createFile('large.txt', 200, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);

            expect(onMessage).toHaveBeenCalledWith({
                t: 'upload',
                op: 'error',
                id: 'upload-1',
                error: 'File exceeds maximum size (100 bytes)'
            });
        });

        it('should send change message with file metadata', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 50, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);

            expect(onMessage).toHaveBeenCalledWith({
                t: 'upload',
                op: 'change',
                id: 'upload-1',
                meta: {name: 'test.txt', size: 50, type: 'text/plain'}
            });
        });

        it('should open XHR with correct URL', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 50, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);

            expect(mockXHR.open).toHaveBeenCalledWith(
                'POST',
                'http://localhost:8080/upload/test-session/upload-1',
                true
            );
        });

        it('should send FormData with file', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 50, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);

            expect(mockXHR.send).toHaveBeenCalled();
            const sentData = mockXHR.send.mock.calls[0][0];
            expect(sentData).toBeInstanceOf(FormData);
        });

        it('should send progress message on upload progress', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 100, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);
            onMessage.mockClear();

            const progressEvent = {loaded: 50, total: 100, lengthComputable: true} as ProgressEvent;
            mockXHR.upload.onprogress?.(progressEvent);

            expect(onMessage).toHaveBeenCalledWith({
                t: 'upload',
                op: 'progress',
                id: 'upload-1',
                loaded: 50,
                total: 100
            });
        });

        it('should send final progress message on successful upload', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 100, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);
            onMessage.mockClear();

            mockXHR.status = 200;
            mockXHR.onload?.();

            expect(onMessage).toHaveBeenCalledWith({
                t: 'upload',
                op: 'progress',
                id: 'upload-1',
                loaded: 100,
                total: 100
            });
        });

        it('should send error message on failed upload', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 100, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);
            onMessage.mockClear();

            mockXHR.status = 500;
            mockXHR.onload?.();

            expect(onMessage).toHaveBeenCalledWith({
                t: 'upload',
                op: 'error',
                id: 'upload-1',
                error: 'Upload failed (500)'
            });
        });

        it('should send error message on network error', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 100, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);
            onMessage.mockClear();

            mockXHR.onerror?.();

            expect(onMessage).toHaveBeenCalledWith({
                t: 'upload',
                op: 'error',
                id: 'upload-1',
                error: 'Upload failed'
            });
        });

        it('should send cancelled message on abort', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 100, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);
            onMessage.mockClear();

            mockXHR.onabort?.();

            expect(onMessage).toHaveBeenCalledWith({
                t: 'upload',
                op: 'cancelled',
                id: 'upload-1'
            });
        });

        it('should strip trailing slash from endpoint', () => {
            const config: UploaderConfig = {
                endpoint: 'http://localhost:8080/upload/',
                sessionId: 'test-session',
                onMessage
            };
            const uploaderWithSlash = new Uploader(config);

            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 50, 'text/plain');
            const files = createMockFileList([file]);

            uploaderWithSlash.upload(meta, files);

            expect(mockXHR.open).toHaveBeenCalledWith(
                'POST',
                'http://localhost:8080/upload/test-session/upload-1',
                true
            );
        });
    });

    describe('cancel', () => {
        it('should abort active upload', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 100, 'text/plain');
            const files = createMockFileList([file]);

            uploader.upload(meta, files);
            uploader.cancel('upload-1');

            expect(mockXHR.abort).toHaveBeenCalled();
        });

        it('should do nothing for non-existent upload', () => {
            uploader.cancel('non-existent');

            expect(mockXHR.abort).not.toHaveBeenCalled();
        });

        it('should clear input value when cancelling', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file = createFile('test.txt', 100, 'text/plain');
            const files = createMockFileList([file]);
            const input = document.createElement('input');
            input.type = 'file';

            uploader.upload(meta, files, input);
            uploader.cancel('upload-1');

            expect(input.value).toBe('');
        });
    });

    describe('replace active upload', () => {
        it('should cancel previous upload when starting new one with same id', () => {
            const meta: UploadMeta = {uploadId: 'upload-1'};
            const file1 = createFile('test1.txt', 100, 'text/plain');
            const file2 = createFile('test2.txt', 100, 'text/plain');

            uploader.upload(meta, createMockFileList([file1]));
            const firstXHR = mockXHR;

            mockXHR = {
                open: vi.fn(),
                send: vi.fn(),
                abort: vi.fn(),
                upload: {onprogress: null},
                onload: null,
                onerror: null,
                onabort: null,
                status: 200
            };
            vi.stubGlobal('XMLHttpRequest', vi.fn(() => mockXHR));

            uploader.upload(meta, createMockFileList([file2]));

            expect(firstXHR.abort).toHaveBeenCalled();
        });
    });
});

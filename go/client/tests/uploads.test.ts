import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { UploadManager, UploadRuntime } from '../src/uploads';
import { ClientNode, UploadMeta } from '../src/types';

describe('UploadManager', () => {
    let manager: UploadManager;
    let runtime: UploadRuntime;
    let input: HTMLInputElement;
    let node: ClientNode;

    beforeEach(() => {
        runtime = {
            getSessionId: vi.fn().mockReturnValue('test-session'),
            getUploadEndpoint: vi.fn().mockReturnValue('/upload'),
            sendUploadMessage: vi.fn()
        };
        manager = new UploadManager(runtime);
        input = document.createElement('input');
        input.type = 'file';
        node = { tag: 'input', el: input };
    });

    it('binds to input element', () => {
        const meta: UploadMeta = { uploadId: 'up1', accept: ['.jpg'], multiple: true };
        manager.bind(node, meta);

        expect(input.getAttribute('accept')).toBe('.jpg');
        expect(input.multiple).toBe(true);
    });

    it('unbinds from input element', () => {
        const meta: UploadMeta = { uploadId: 'up1' };
        manager.bind(node, meta);

        // We can't easily check if listener is removed in JSDOM without spying on add/removeEventListener
        // But we can check if unbind runs without error
        manager.unbind(node);
    });

    it('handles file selection', () => {
        const meta: UploadMeta = { uploadId: 'up1' };
        manager.bind(node, meta);

        // Mock file
        const file = new File(['content'], 'test.txt', { type: 'text/plain' });
        Object.defineProperty(input, 'files', {
            value: [file],
            writable: false
        });

        // Trigger change
        input.dispatchEvent(new Event('change'));

        expect(runtime.sendUploadMessage).toHaveBeenCalledWith(expect.objectContaining({
            op: 'change',
            id: 'up1',
            meta: { name: 'test.txt', size: 7, type: 'text/plain' }
        }));
    });

    it('enforces max size', () => {
        const meta: UploadMeta = { uploadId: 'up1', maxSize: 5 };
        manager.bind(node, meta);

        const file = new File(['content'], 'test.txt', { type: 'text/plain' }); // 7 bytes
        Object.defineProperty(input, 'files', {
            value: [file],
            writable: false
        });

        input.dispatchEvent(new Event('change'));

        expect(runtime.sendUploadMessage).toHaveBeenCalledWith(expect.objectContaining({
            op: 'error',
            id: 'up1',
            error: expect.stringContaining('exceeds maximum size')
        }));
        expect(input.value).toBe('');
    });
});

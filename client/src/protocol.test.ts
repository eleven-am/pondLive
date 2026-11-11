import { describe, it, expect } from 'vitest';
import {
    isScriptTopic,
    scriptTopic,
    isBoot,
    isMessage,
    Boot,
    Message,
} from './protocol';

describe('protocol', () => {
    describe('isScriptTopic', () => {
        it('should return true for script topics', () => {
            expect(isScriptTopic('script:abc123')).toBe(true);
            expect(isScriptTopic('script:my-script-id')).toBe(true);
            expect(isScriptTopic('script:')).toBe(true);
        });

        it('should return false for non-script topics', () => {
            expect(isScriptTopic('router')).toBe(false);
            expect(isScriptTopic('dom')).toBe(false);
            expect(isScriptTopic('h')).toBe(false);
        });
    });

    describe('scriptTopic', () => {
        it('should create script topic from script id', () => {
            expect(scriptTopic('abc123')).toBe('script:abc123');
            expect(scriptTopic('my-script')).toBe('script:my-script');
        });
    });

    describe('isBoot', () => {
        it('should return true for valid boot messages', () => {
            const boot: Boot = {
                t: 'boot',
                tok: 'tok',
                ws: '/live',
                sid: 'session-123',
                ver: 1,
                seq: 0,
                location: { path: '/', query: {}, hash: '' },
            };
            expect(isBoot(boot)).toBe(true);
        });

        it('should return false for non-boot messages', () => {
            expect(isBoot(null)).toBe(false);
            expect(isBoot(undefined)).toBe(false);
            expect(isBoot({})).toBe(false);
            expect(isBoot({ t: 'error' })).toBe(false);
        });
    });

    describe('isMessage', () => {
        it('should return true for valid messages', () => {
            const msg: Message = {
                seq: 1,
                topic: 'frame',
                event: 'patch',
                data: { patches: [] },
            };
            expect(isMessage(msg)).toBe(true);
        });

        it('should return false for non-message objects', () => {
            expect(isMessage(null)).toBe(false);
            expect(isMessage({ seq: 1 })).toBe(false);
        });
    });
});

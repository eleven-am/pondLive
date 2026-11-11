import { describe, it, expect } from 'vitest';
import {
    isScriptMessage,
    SCRIPT_TOPIC,
    isBoot,
    isMessage,
    Boot,
    Message,
} from './protocol';

describe('protocol', () => {
    describe('isScriptMessage', () => {
        it('uses the single-letter topic', () => {
            expect(SCRIPT_TOPIC).toBe('s');
        });

        it('requires an integer instance id', () => {
            expect(isScriptMessage({ i: 42, d: { a: 1 } })).toBe(true);
            expect(isScriptMessage({ i: 42 })).toBe(true);
            expect(isScriptMessage({ i: '42', d: 1 })).toBe(false);
            expect(isScriptMessage({ i: 1.5 })).toBe(false);
            expect(isScriptMessage({ scriptId: 's1' })).toBe(false);
            expect(isScriptMessage(null)).toBe(false);
            expect(isScriptMessage('s')).toBe(false);
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

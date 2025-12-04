import { describe, it, expect } from 'vitest';
import {
    Topics,
    OpKinds,
    isScriptTopic,
    scriptTopic,
    isHandlerTopic,
    handlerTopic,
    isBoot,
    isServerError,
    isServerEvt,
    isServerAck,
    isMessage,
    Boot,
    ServerError,
    ServerEvt,
    ServerAck,
    Message,
} from './protocol';

describe('protocol', () => {
    describe('Topics constants', () => {
        it('should have correct topic values', () => {
            expect(Topics.Router).toBe('router');
            expect(Topics.DOM).toBe('dom');
            expect(Topics.Frame).toBe('frame');
            expect(Topics.Ack).toBe('ack');
        });
    });

    describe('OpKinds constants', () => {
        it('should have correct operation kind values', () => {
            expect(OpKinds.SetText).toBe('setText');
            expect(OpKinds.SetComment).toBe('setComment');
            expect(OpKinds.SetAttr).toBe('setAttr');
            expect(OpKinds.DelAttr).toBe('delAttr');
            expect(OpKinds.SetStyle).toBe('setStyle');
            expect(OpKinds.DelStyle).toBe('delStyle');
            expect(OpKinds.SetHandlers).toBe('setHandlers');
            expect(OpKinds.SetScript).toBe('setScript');
            expect(OpKinds.DelScript).toBe('delScript');
            expect(OpKinds.SetRef).toBe('setRef');
            expect(OpKinds.DelRef).toBe('delRef');
            expect(OpKinds.ReplaceNode).toBe('replaceNode');
            expect(OpKinds.AddChild).toBe('addChild');
            expect(OpKinds.DelChild).toBe('delChild');
            expect(OpKinds.MoveChild).toBe('moveChild');
        });
    });

    describe('isScriptTopic', () => {
        it('should return true for script topics', () => {
            expect(isScriptTopic('script:abc123')).toBe(true);
            expect(isScriptTopic('script:my-script-id')).toBe(true);
            expect(isScriptTopic('script:')).toBe(true);
        });

        it('should return false for non-script topics', () => {
            expect(isScriptTopic('router')).toBe(false);
            expect(isScriptTopic('dom')).toBe(false);
            expect(isScriptTopic('c0:h0')).toBe(false);
        });
    });

    describe('scriptTopic', () => {
        it('should create script topic from script id', () => {
            expect(scriptTopic('abc123')).toBe('script:abc123');
            expect(scriptTopic('my-script')).toBe('script:my-script');
        });
    });

    describe('isHandlerTopic', () => {
        it('should return true for handler topics', () => {
            expect(isHandlerTopic('c0:h0')).toBe(true);
            expect(isHandlerTopic('c0:h123')).toBe(true);
            expect(isHandlerTopic('component:h999')).toBe(true);
        });

        it('should return false for non-handler topics', () => {
            expect(isHandlerTopic('router')).toBe(false);
            expect(isHandlerTopic('script:abc')).toBe(false);
            expect(isHandlerTopic('c0:handler0')).toBe(false);
        });
    });

    describe('handlerTopic', () => {
        it('should create handler topic from handler id', () => {
            expect(handlerTopic('c0:h0')).toBe('c0:h0');
        });
    });

    describe('isBoot', () => {
        it('should return true for valid boot messages', () => {
            const boot: Boot = {
                t: 'boot',
                sid: 'session-123',
                ver: 1,
                seq: 0,
                patch: [],
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

    describe('isServerError', () => {
        it('should return true for valid server error messages', () => {
            const error: ServerError = {
                t: 'error',
                sid: 'session-123',
                code: 'ERR001',
                message: 'Something went wrong',
            };
            expect(isServerError(error)).toBe(true);
        });

        it('should return false for non-error messages', () => {
            expect(isServerError(null)).toBe(false);
            expect(isServerError({ t: 'boot' })).toBe(false);
        });
    });

    describe('isServerEvt', () => {
        it('should return true for valid server events', () => {
            const evt: ServerEvt = {
                t: 'frame',
                sid: 'session-123',
                a: 'patch',
            };
            expect(isServerEvt(evt)).toBe(true);
        });

        it('should return false for non-event messages', () => {
            expect(isServerEvt(null)).toBe(false);
            expect(isServerEvt({ t: 'frame' })).toBe(false);
        });
    });

    describe('isServerAck', () => {
        it('should return true for valid server acks', () => {
            const ack: ServerAck = {
                t: 'ack',
                sid: 'session-123',
                seq: 10,
            };
            expect(isServerAck(ack)).toBe(true);
        });

        it('should return false for non-ack messages', () => {
            expect(isServerAck(null)).toBe(false);
            expect(isServerAck({ t: 'ack' })).toBe(false);
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

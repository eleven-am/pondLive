import {boot} from './runtime';

if (typeof window !== 'undefined') {
    window.addEventListener('DOMContentLoaded', () => {
        const runtime = boot();
        if (runtime) {
            (window as any).__LIVEUI__ = runtime;
        }
    });
}

export {Runtime, boot} from './runtime';
export {Transport} from './transport';
export {Patcher} from './patcher';
export {Router} from './router';
export {Uploader} from './uploader';
export {EffectExecutor} from './effects';
export {ScriptExecutor} from './scripts';
export {Logger} from './logger';

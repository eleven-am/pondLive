import { LiveRuntime } from './runtime';


if (typeof window !== 'undefined') {
    window.addEventListener('DOMContentLoaded', () => {
        const instance = new LiveRuntime();
        
        (window as any).__LIVEUI__ = instance;
    });
}

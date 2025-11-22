import {describe, it, expect, vi, beforeEach} from 'vitest';
import {ScriptExecutor} from './scripts';
import {ScriptMeta} from './types';

describe('ScriptExecutor', () => {
    let executor: ScriptExecutor;
    let mockOnMessage: ReturnType<typeof vi.fn>;

    beforeEach(() => {
        mockOnMessage = vi.fn();
        executor = new ScriptExecutor({
            sessionId: 'test-session',
            onMessage: mockOnMessage
        });
    });

    it('executes a simple script', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: '(element, transport) => { element.textContent = "Hello"; }'
        };

        await executor.execute(meta, element);

        expect(element.textContent).toBe('Hello');
    });

    it('sends messages to server via transport.send', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: '(element, transport) => { transport.send({ foo: "bar" }); }'
        };

        await executor.execute(meta, element);

        expect(mockOnMessage).toHaveBeenCalledWith({
            t: 'script:message',
            sid: 'test-session',
            scriptId: 'script-1',
            event: '',
            data: {foo: 'bar'}
        });
    });

    it('registers event handlers via transport.on', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                transport.on('update', (data) => {
                    element.textContent = data.text;
                });
            }`
        };

        await executor.execute(meta, element);

        executor.handleEvent('script-1', 'update', {text: 'Updated'});

        expect(element.textContent).toBe('Updated');
    });

    it('calls cleanup function when script is re-executed', async () => {
        const element = document.createElement('div');

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                return () => { window.__cleanupCalled = true; };
            }`
        };

        await executor.execute(meta, element);

        await executor.execute(meta, element);

        expect((window as any).__cleanupCalled).toBe(true);
    });

    it('executes async script', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `async (element, transport) => {
                await Promise.resolve();
                element.textContent = "Async";
            }`
        };

        await executor.execute(meta, element);

        expect(element.textContent).toBe('Async');
    });

    it('handles script execution errors gracefully', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: '(element, transport) => { throw new Error("Script error"); }'
        };

        await expect(executor.execute(meta, element)).resolves.not.toThrow();
    });

    it('handles event handler errors gracefully', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                transport.on('error', () => {
                    throw new Error("Handler error");
                });
            }`
        };

        await executor.execute(meta, element);

        expect(() => {
            executor.handleEvent('script-1', 'error', {});
        }).not.toThrow();
    });

    it('cleans up script instance', async () => {
        const element = document.createElement('div');

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                return () => { window.__cleanupCalled2 = true; };
            }`
        };

        await executor.execute(meta, element);

        executor.cleanup('script-1');

        expect((window as any).__cleanupCalled2).toBe(true);
    });

    it('supports arrow functions', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: '(element, transport) => { element.className = "test"; }'
        };

        await executor.execute(meta, element);

        expect(element.className).toBe('test');
    });

    it('supports regular functions', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: 'function(element, transport) { element.id = "test-id"; }'
        };

        await executor.execute(meta, element);

        expect(element.id).toBe('test-id');
    });

    it('handles multiple event handlers on same script', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                transport.on('event1', (data) => {
                    element.dataset.event1 = data.value;
                });
                transport.on('event2', (data) => {
                    element.dataset.event2 = data.value;
                });
            }`
        };

        await executor.execute(meta, element);

        executor.handleEvent('script-1', 'event1', {value: 'first'});
        executor.handleEvent('script-1', 'event2', {value: 'second'});

        expect(element.dataset.event1).toBe('first');
        expect(element.dataset.event2).toBe('second');
    });

    it('handles multiple messages sent to server', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                transport.send({ msg: 'first' });
                transport.send({ msg: 'second' });
                transport.send({ msg: 'third' });
            }`
        };

        await executor.execute(meta, element);

        expect(mockOnMessage).toHaveBeenCalledTimes(3);
        expect(mockOnMessage).toHaveBeenNthCalledWith(1, {
            t: 'script:message',
            sid: 'test-session',
            scriptId: 'script-1',
            event: '',
            data: {msg: 'first'}
        });
        expect(mockOnMessage).toHaveBeenNthCalledWith(2, {
            t: 'script:message',
            sid: 'test-session',
            scriptId: 'script-1',
            event: '',
            data: {msg: 'second'}
        });
        expect(mockOnMessage).toHaveBeenNthCalledWith(3, {
            t: 'script:message',
            sid: 'test-session',
            scriptId: 'script-1',
            event: '',
            data: {msg: 'third'}
        });
    });

    it('ignores events for non-existent scripts', () => {
        expect(() => {
            executor.handleEvent('non-existent', 'update', {});
        }).not.toThrow();
    });

    it('ignores events for non-registered event handlers', async () => {
        const element = document.createElement('div');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                transport.on('registered', (data) => {
                    element.textContent = 'registered';
                });
            }`
        };

        await executor.execute(meta, element);

        expect(() => {
            executor.handleEvent('script-1', 'unregistered', {});
        }).not.toThrow();

        expect(element.textContent).toBe('');
    });

    it('re-executing script with new code replaces old behavior', async () => {
        const element = document.createElement('div');

        const meta1: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                element.textContent = 'version 1';
            }`
        };

        await executor.execute(meta1, element);
        expect(element.textContent).toBe('version 1');

        const meta2: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                element.textContent = 'version 2';
            }`
        };

        await executor.execute(meta2, element);
        expect(element.textContent).toBe('version 2');
    });

    it('event handlers from previous script are not called after re-execution', async () => {
        const element = document.createElement('div');

        const meta1: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                transport.on('test', (data) => {
                    element.textContent = 'old handler';
                });
            }`
        };

        await executor.execute(meta1, element);

        const meta2: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                transport.on('test', (data) => {
                    element.textContent = 'new handler';
                });
            }`
        };

        await executor.execute(meta2, element);

        executor.handleEvent('script-1', 'test', {});
        expect(element.textContent).toBe('new handler');
    });

    it('can access and modify DOM properties', async () => {
        const element = document.createElement('input');
        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                element.value = 'test value';
                element.disabled = true;
                element.placeholder = 'test placeholder';
            }`
        };

        await executor.execute(meta, element);

        expect((element as HTMLInputElement).value).toBe('test value');
        expect((element as HTMLInputElement).disabled).toBe(true);
        expect((element as HTMLInputElement).placeholder).toBe('test placeholder');
    });

    it('can add and remove event listeners', async () => {
        const element = document.createElement('button');

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                const handler = () => {
                    window.__testClicked = true;
                };
                element.addEventListener('click', handler);
                return () => {
                    element.removeEventListener('click', handler);
                };
            }`
        };

        await executor.execute(meta, element);

        element.click();
        expect((window as any).__testClicked).toBe(true);

        (window as any).__testClicked = false;
        executor.cleanup('script-1');
        element.click();
        expect((window as any).__testClicked).toBe(false);
    });

    it('handles complex data types in messages', async () => {
        const element = document.createElement('div');
        const complexData = {
            string: 'test',
            number: 42,
            boolean: true,
            null: null,
            array: [1, 2, 3],
            nested: {
                key: 'value',
                deep: {
                    level: 3
                }
            }
        };

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                transport.send(${JSON.stringify(complexData)});
            }`
        };

        await executor.execute(meta, element);

        expect(mockOnMessage).toHaveBeenCalledWith({
            t: 'script:message',
            sid: 'test-session',
            scriptId: 'script-1',
            event: '',
            data: complexData
        });
    });

    it('handles async cleanup function', async () => {
        const element = document.createElement('div');

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `async (element, transport) => {
                return async () => {
                    window.__asyncCleanupCalled = true;
                };
            }`
        };

        await executor.execute(meta, element);
        await executor.execute(meta, element);

        expect((window as any).__asyncCleanupCalled).toBe(true);
    });

    it('handles scripts that return non-function values', async () => {
        const element = document.createElement('div');

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                element.textContent = 'test';
                return 'not a function';
            }`
        };

        await expect(executor.execute(meta, element)).resolves.not.toThrow();
        expect(element.textContent).toBe('test');
    });

    it('can interact with global window object', async () => {
        const element = document.createElement('div');
        (window as any).__testData = {counter: 0};

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                window.__testData.counter++;
                element.textContent = String(window.__testData.counter);
            }`
        };

        await executor.execute(meta, element);
        expect(element.textContent).toBe('1');
        expect((window as any).__testData.counter).toBe(1);
    });

    it('cleanup is idempotent for non-existent scripts', () => {
        expect(() => {
            executor.cleanup('non-existent');
            executor.cleanup('non-existent');
        }).not.toThrow();
    });

    it('handles script with syntax that requires parentheses', async () => {
        const element = document.createElement('div');

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => ({ result: 'object literal' })`
        };

        await executor.execute(meta, element);
    });

    it('cleanup removes script instance and prevents event handling', async () => {
        const element = document.createElement('div');

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                transport.on('test', (data) => {
                    element.textContent = 'should not be called';
                });
            }`
        };

        await executor.execute(meta, element);

        executor.cleanup('script-1');

        executor.handleEvent('script-1', 'test', {});
        expect(element.textContent).toBe('');
    });

    it('cleanup with return value calls cleanup function', async () => {
        const element = document.createElement('div');

        const meta: ScriptMeta = {
            scriptId: 'script-1',
            script: `(element, transport) => {
                return () => { window.__manualCleanupCalled = true; };
            }`
        };

        await executor.execute(meta, element);

        executor.cleanup('script-1');
        expect((window as any).__manualCleanupCalled).toBe(true);
    });
});

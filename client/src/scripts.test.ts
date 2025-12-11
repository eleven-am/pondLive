import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ScriptExecutor, ScriptExecutorConfig } from './scripts';
import { Bus } from './bus';
import { ScriptMeta } from './protocol';
import { Transport } from './transport';

describe('ScriptExecutor', () => {
    let executor: ScriptExecutor;
    let bus: Bus;
    let mockTransport: Transport;
    let sendScriptSpy: ReturnType<typeof vi.fn>;

    beforeEach(() => {
        bus = new Bus();
        sendScriptSpy = vi.fn();
        mockTransport = {
            sendScript: sendScriptSpy,
        } as unknown as Transport;
        const config: ScriptExecutorConfig = { bus, transport: mockTransport };
        executor = new ScriptExecutor(config);
    });

    describe('execute', () => {
        it('should execute a simple script', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: '(element, transport) => { element.textContent = "Hello"; }',
            };

            await executor.execute(meta, el);

            expect(el.textContent).toBe('Hello');
        });

        it('should provide element to script', async () => {
            const el = document.createElement('input');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: '(element, transport) => { element.value = "test"; }',
            };

            await executor.execute(meta, el);

            expect((el as HTMLInputElement).value).toBe('test');
        });

        it('should support async scripts', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `async (element, transport) => {
                    await Promise.resolve();
                    element.textContent = "Async Done";
                }`,
            };

            await executor.execute(meta, el);

            expect(el.textContent).toBe('Async Done');
        });

        it('should throw on script error', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: '(element, transport) => { throw new Error("Script error"); }',
            };

            await expect(executor.execute(meta, el)).rejects.toThrow('Script error');
        });
    });

    describe('transport.send', () => {
        it('should send message via transport', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: '(element, transport) => { transport.send("test", { foo: "bar" }); }',
            };

            await executor.execute(meta, el);

            expect(sendScriptSpy).toHaveBeenCalledWith('script-1', {
                scriptId: 'script-1',
                event: 'test',
                data: { foo: 'bar' },
            });
        });

        it('should send multiple messages', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `(element, transport) => {
                    transport.send("event1", { a: 1 });
                    transport.send("event2", { b: 2 });
                }`,
            };

            await executor.execute(meta, el);

            expect(sendScriptSpy).toHaveBeenCalledTimes(2);
        });
    });

    describe('transport.on', () => {
        it('should register event handler', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `(element, transport) => {
                    transport.on("update", (data) => {
                        element.textContent = data.text;
                    });
                }`,
            };

            await executor.execute(meta, el);

            bus.publishScript('script-1', 'send', {
                scriptId: 'script-1',
                event: 'update',
                data: { text: 'Updated' },
            });

            expect(el.textContent).toBe('Updated');
        });

        it('should handle multiple event types', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `(element, transport) => {
                    transport.on("event1", (data) => { element.dataset.e1 = data.v; });
                    transport.on("event2", (data) => { element.dataset.e2 = data.v; });
                }`,
            };

            await executor.execute(meta, el);

            bus.publishScript('script-1', 'send', { scriptId: 'script-1', event: 'event1', data: { v: 'one' } });
            bus.publishScript('script-1', 'send', { scriptId: 'script-1', event: 'event2', data: { v: 'two' } });

            expect(el.dataset.e1).toBe('one');
            expect(el.dataset.e2).toBe('two');
        });

        it('should swallow handler errors', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `(element, transport) => {
                    transport.on("error", () => { throw new Error("Handler error"); });
                }`,
            };

            await executor.execute(meta, el);

            expect(() => {
                bus.publishScript('script-1', 'send', { scriptId: 'script-1', event: 'error', data: {} });
            }).not.toThrow();
        });
    });

    describe('cleanup function', () => {
        it('should call cleanup on re-execute', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `(element, transport) => {
                    return () => { element.dataset.cleanupCalled = "true"; };
                }`,
            };

            await executor.execute(meta, el);
            expect(el.dataset.cleanupCalled).toBeUndefined();

            await executor.execute(meta, el);
            expect(el.dataset.cleanupCalled).toBe('true');
        });

        it('should call cleanup on manual cleanup', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `(element, transport) => {
                    return () => { element.dataset.manualCleanup = "true"; };
                }`,
            };

            await executor.execute(meta, el);
            executor.cleanup('script-1');

            expect(el.dataset.manualCleanup).toBe('true');
        });

        it('should handle async cleanup', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `async (element, transport) => {
                    return () => { element.dataset.asyncCleanup = "true"; };
                }`,
            };

            await executor.execute(meta, el);
            executor.cleanup('script-1');

            expect(el.dataset.asyncCleanup).toBe('true');
        });
    });

    describe('cleanup', () => {
        it('should be idempotent for non-existent scripts', () => {
            expect(() => {
                executor.cleanup('non-existent');
                executor.cleanup('non-existent');
            }).not.toThrow();
        });

        it('should unsubscribe from bus', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `(element, transport) => {
                    transport.on("test", () => { element.textContent = "should not appear"; });
                }`,
            };

            await executor.execute(meta, el);
            executor.cleanup('script-1');

            bus.publishScript('script-1', 'send', { scriptId: 'script-1', event: 'test', data: {} });

            expect(el.textContent).toBe('');
        });
    });

    describe('destroy', () => {
        it('should cleanup all scripts', async () => {
            const el1 = document.createElement('div');
            const el2 = document.createElement('div');

            await executor.execute(
                { scriptId: 'script-1', script: '(el, t) => () => { el.dataset.destroyed = "true"; }' },
                el1
            );
            await executor.execute(
                { scriptId: 'script-2', script: '(el, t) => () => { el.dataset.destroyed = "true"; }' },
                el2
            );

            executor.destroy();

            expect(el1.dataset.destroyed).toBe('true');
            expect(el2.dataset.destroyed).toBe('true');
        });
    });

    describe('sandbox', () => {
        it('should provide whitelisted globals', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `(element, transport) => {
                    element.dataset.hasConsole = String(typeof console !== 'undefined');
                    element.dataset.hasPromise = String(typeof Promise !== 'undefined');
                    element.dataset.hasFetch = String(typeof fetch !== 'undefined');
                    element.dataset.hasMath = String(typeof Math !== 'undefined');
                }`,
            };

            await executor.execute(meta, el);

            expect(el.dataset.hasConsole).toBe('true');
            expect(el.dataset.hasPromise).toBe('true');
            expect(el.dataset.hasFetch).toBe('true');
            expect(el.dataset.hasMath).toBe('true');
        });


        it('should allow setting local variables', async () => {
            const el = document.createElement('div');
            const meta: ScriptMeta = {
                scriptId: 'script-1',
                script: `(element, transport) => {
                    let x = 42;
                    element.textContent = String(x);
                }`,
            };

            await executor.execute(meta, el);

            expect(el.textContent).toBe('42');
        });
    });

    describe('re-execution', () => {
        it('should replace old behavior with new', async () => {
            const el = document.createElement('div');

            await executor.execute(
                { scriptId: 'script-1', script: '(el, t) => { el.textContent = "v1"; }' },
                el
            );
            expect(el.textContent).toBe('v1');

            await executor.execute(
                { scriptId: 'script-1', script: '(el, t) => { el.textContent = "v2"; }' },
                el
            );
            expect(el.textContent).toBe('v2');
        });

        it('should replace event handlers on re-execute', async () => {
            const el = document.createElement('div');

            await executor.execute(
                {
                    scriptId: 'script-1',
                    script: `(el, t) => {
                        t.on("test", () => { el.textContent = "old"; });
                    }`,
                },
                el
            );

            await executor.execute(
                {
                    scriptId: 'script-1',
                    script: `(el, t) => {
                        t.on("test", () => { el.textContent = "new"; });
                    }`,
                },
                el
            );

            bus.publishScript('script-1', 'send', { scriptId: 'script-1', event: 'test', data: {} });

            expect(el.textContent).toBe('new');
        });
    });
});

import {describe, it, expect, beforeEach, vi, Mock} from 'vitest';
import {EffectExecutor} from './effects';
import {DOMRequest, DOMResponse} from './protocol';
import {Logger} from './logger';
import {EffectExecutorConfig, DOMActionEffect, CookieEffect} from './types';

describe('EffectExecutor', () => {
    let refs: Map<string, Element>;
    // @ts-ignore
    let onDOMResponse: Mock<[DOMResponse], void>;
    let executor: EffectExecutor;

    beforeEach(() => {
        refs = new Map();
        onDOMResponse = vi.fn();
        Logger.configure({enabled: true, level: 'debug'});

        const config: EffectExecutorConfig = {
            sessionId: 'test-session',
            resolveRef: (refId) => refs.get(refId),
            onDOMResponse
        };
        executor = new EffectExecutor(config);
    });

    describe('execute DOM actions', () => {
        describe('dom.call', () => {
            it('should call method on element', () => {
                const el = document.createElement('input');
                el.focus = vi.fn();
                refs.set('input1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.call',
                    ref: 'input1',
                    method: 'focus'
                }];

                executor.execute(effects);

                expect(el.focus).toHaveBeenCalled();
            });

            it('should call method with arguments', () => {
                const el = document.createElement('div');
                el.setAttribute = vi.fn();
                refs.set('div1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.call',
                    ref: 'div1',
                    method: 'setAttribute',
                    args: ['data-foo', 'bar']
                }];

                executor.execute(effects);

                expect(el.setAttribute).toHaveBeenCalledWith('data-foo', 'bar');
            });

            it('should warn if method not found', () => {
                const el = document.createElement('div');
                refs.set('div1', el);
                const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.call',
                    ref: 'div1',
                    method: 'nonExistentMethod'
                }];

                executor.execute(effects);

                expect(warnSpy).toHaveBeenCalledWith('[LiveUI:Effects]', 'Method not found', 'nonExistentMethod');
                warnSpy.mockRestore();
            });
        });

        describe('dom.set', () => {
            it('should set property on element', () => {
                const el = document.createElement('input') as HTMLInputElement;
                refs.set('input1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.set',
                    ref: 'input1',
                    prop: 'value',
                    value: 'hello'
                }];

                executor.execute(effects);

                expect(el.value).toBe('hello');
            });

            it('should set boolean property', () => {
                const el = document.createElement('input') as HTMLInputElement;
                refs.set('input1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.set',
                    ref: 'input1',
                    prop: 'disabled',
                    value: true
                }];

                executor.execute(effects);

                expect(el.disabled).toBe(true);
            });
        });

        describe('dom.toggle', () => {
            it('should toggle boolean property from false to true', () => {
                const el = document.createElement('input') as HTMLInputElement;
                el.disabled = false;
                refs.set('input1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.toggle',
                    ref: 'input1',
                    prop: 'disabled'
                }];

                executor.execute(effects);

                expect(el.disabled).toBe(true);
            });

            it('should toggle boolean property from true to false', () => {
                const el = document.createElement('input') as HTMLInputElement;
                el.disabled = true;
                refs.set('input1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.toggle',
                    ref: 'input1',
                    prop: 'disabled'
                }];

                executor.execute(effects);

                expect(el.disabled).toBe(false);
            });
        });

        describe('dom.class', () => {
            it('should add class when on is true', () => {
                const el = document.createElement('div');
                refs.set('div1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.class',
                    ref: 'div1',
                    class: 'active',
                    on: true
                }];

                executor.execute(effects);

                expect(el.classList.contains('active')).toBe(true);
            });

            it('should remove class when on is false', () => {
                const el = document.createElement('div');
                el.classList.add('active');
                refs.set('div1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.class',
                    ref: 'div1',
                    class: 'active',
                    on: false
                }];

                executor.execute(effects);

                expect(el.classList.contains('active')).toBe(false);
            });

            it('should toggle class when on is undefined', () => {
                const el = document.createElement('div');
                refs.set('div1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.class',
                    ref: 'div1',
                    class: 'active'
                }];

                executor.execute(effects);
                expect(el.classList.contains('active')).toBe(true);

                executor.execute(effects);
                expect(el.classList.contains('active')).toBe(false);
            });
        });

        describe('dom.scroll', () => {
            it('should call scrollIntoView with options', () => {
                const el = document.createElement('div');
                el.scrollIntoView = vi.fn();
                refs.set('div1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.scroll',
                    ref: 'div1',
                    behavior: 'smooth',
                    block: 'center',
                    inline: 'nearest'
                }];

                executor.execute(effects);

                expect(el.scrollIntoView).toHaveBeenCalledWith({
                    behavior: 'smooth',
                    block: 'center',
                    inline: 'nearest'
                });
            });

            it('should call scrollIntoView with empty options if none provided', () => {
                const el = document.createElement('div');
                el.scrollIntoView = vi.fn();
                refs.set('div1', el);

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.scroll',
                    ref: 'div1'
                }];

                executor.execute(effects);

                expect(el.scrollIntoView).toHaveBeenCalledWith({});
            });
        });

        describe('ref not found', () => {
            it('should warn if ref not found', () => {
                const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

                const effects: DOMActionEffect[] = [{
                    type: 'dom',
                    kind: 'dom.call',
                    ref: 'nonexistent',
                    method: 'focus'
                }];

                executor.execute(effects);

                expect(warnSpy).toHaveBeenCalledWith('[LiveUI:Effects]', 'Ref not found', 'nonexistent');
                warnSpy.mockRestore();
            });
        });

        describe('multiple effects', () => {
            it('should execute multiple effects in order', () => {
                const el = document.createElement('input') as HTMLInputElement;
                refs.set('input1', el);

                const effects: DOMActionEffect[] = [
                    {type: 'dom', kind: 'dom.set', ref: 'input1', prop: 'value', value: 'first'},
                    {type: 'dom', kind: 'dom.set', ref: 'input1', prop: 'value', value: 'second'}
                ];

                executor.execute(effects);

                expect(el.value).toBe('second');
            });
        });
    });

    describe('execute cookie sync', () => {
        it('should fetch cookie endpoint', async () => {
            const fetchMock = vi.fn().mockResolvedValue({});
            vi.stubGlobal('fetch', fetchMock);

            const effects: CookieEffect[] = [{
                type: 'cookies',
                endpoint: 'http://localhost/cookies',
                sid: 'session-123',
                token: 'token-abc'
            }];

            executor.execute(effects);

            expect(fetchMock).toHaveBeenCalledWith(
                'http://localhost/cookies?sid=session-123&token=token-abc',
                {method: 'GET', credentials: 'include'}
            );

            vi.unstubAllGlobals();
        });

        it('should use custom method', async () => {
            const fetchMock = vi.fn().mockResolvedValue({});
            vi.stubGlobal('fetch', fetchMock);

            const effects: CookieEffect[] = [{
                type: 'cookies',
                endpoint: 'http://localhost/cookies',
                sid: 'session-123',
                token: 'token-abc',
                method: 'POST'
            }];

            executor.execute(effects);

            expect(fetchMock).toHaveBeenCalledWith(
                expect.any(String),
                {method: 'POST', credentials: 'include'}
            );

            vi.unstubAllGlobals();
        });
    });

    describe('handleDOMRequest', () => {
        describe('read props', () => {
            it('should read single property', () => {
                const el = document.createElement('input') as HTMLInputElement;
                el.value = 'test value';
                refs.set('input1', el);

                const req: DOMRequest = {
                    t: 'dom_req',
                    id: 'req-1',
                    ref: 'input1',
                    props: ['value']
                };

                executor.handleDOMRequest(req);

                expect(onDOMResponse).toHaveBeenCalledWith({
                    t: 'dom_res',
                    sid: 'test-session',
                    id: 'req-1',
                    values: {value: 'test value'}
                });
            });

            it('should read multiple properties', () => {
                const el = document.createElement('input') as HTMLInputElement;
                el.value = 'hello';
                el.type = 'text';
                refs.set('input1', el);

                const req: DOMRequest = {
                    t: 'dom_req',
                    id: 'req-1',
                    ref: 'input1',
                    props: ['value', 'type']
                };

                executor.handleDOMRequest(req);

                expect(onDOMResponse).toHaveBeenCalledWith({
                    t: 'dom_res',
                    sid: 'test-session',
                    id: 'req-1',
                    values: {value: 'hello', type: 'text'}
                });
            });

            it('should read nested property', () => {
                const el = document.createElement('div');
                el.style.color = 'red';
                refs.set('div1', el);

                const req: DOMRequest = {
                    t: 'dom_req',
                    id: 'req-1',
                    ref: 'div1',
                    props: ['style.color']
                };

                executor.handleDOMRequest(req);

                expect(onDOMResponse).toHaveBeenCalledWith({
                    t: 'dom_res',
                    sid: 'test-session',
                    id: 'req-1',
                    values: {'style.color': 'red'}
                });
            });
        });

        describe('call method', () => {
            it('should call method and return result', () => {
                const el = document.createElement('div');
                el.getAttribute = vi.fn().mockReturnValue('bar');
                refs.set('div1', el);

                const req: DOMRequest = {
                    t: 'dom_req',
                    id: 'req-1',
                    ref: 'div1',
                    method: 'getAttribute',
                    args: ['data-foo']
                };

                executor.handleDOMRequest(req);

                expect(el.getAttribute).toHaveBeenCalledWith('data-foo');
                expect(onDOMResponse).toHaveBeenCalledWith({
                    t: 'dom_res',
                    sid: 'test-session',
                    id: 'req-1',
                    result: 'bar'
                });
            });

            it('should return error if method not found', () => {
                const el = document.createElement('div');
                refs.set('div1', el);

                const req: DOMRequest = {
                    t: 'dom_req',
                    id: 'req-1',
                    ref: 'div1',
                    method: 'nonExistentMethod'
                };

                executor.handleDOMRequest(req);

                expect(onDOMResponse).toHaveBeenCalledWith({
                    t: 'dom_res',
                    sid: 'test-session',
                    id: 'req-1',
                    error: 'method not found: nonExistentMethod'
                });
            });
        });

        describe('error handling', () => {
            it('should return error if ref not found', () => {
                const req: DOMRequest = {
                    t: 'dom_req',
                    id: 'req-1',
                    ref: 'nonexistent',
                    props: ['value']
                };

                executor.handleDOMRequest(req);

                expect(onDOMResponse).toHaveBeenCalledWith({
                    t: 'dom_res',
                    sid: 'test-session',
                    id: 'req-1',
                    error: 'ref not found: nonexistent'
                });
            });

            it('should return error if no props or method specified', () => {
                const el = document.createElement('div');
                refs.set('div1', el);

                const req: DOMRequest = {
                    t: 'dom_req',
                    id: 'req-1',
                    ref: 'div1'
                };

                executor.handleDOMRequest(req);

                expect(onDOMResponse).toHaveBeenCalledWith({
                    t: 'dom_res',
                    sid: 'test-session',
                    id: 'req-1',
                    error: 'no props or method specified'
                });
            });
        });
    });
});

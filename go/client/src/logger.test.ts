import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Logger } from './logger';

describe('Logger', () => {
    const originalConsole = {
        debug: console.debug,
        info: console.info,
        warn: console.warn,
        error: console.error,
        log: console.log,
    };

    beforeEach(() => {
        console.debug = vi.fn();
        console.info = vi.fn();
        console.warn = vi.fn();
        console.error = vi.fn();
        console.log = vi.fn();
        Logger.configure({ enabled: false, level: 'debug' });
    });

    afterEach(() => {
        console.debug = originalConsole.debug;
        console.info = originalConsole.info;
        console.warn = originalConsole.warn;
        console.error = originalConsole.error;
        console.log = originalConsole.log;
    });

    describe('configure', () => {
        it('should enable logging', () => {
            Logger.configure({ enabled: true });
            Logger.info('Test', 'message');

            expect(console.info).toHaveBeenCalled();
        });

        it('should disable logging', () => {
            Logger.configure({ enabled: false });
            Logger.info('Test', 'message');

            expect(console.info).not.toHaveBeenCalled();
        });

        it('should set log level', () => {
            Logger.configure({ enabled: true, level: 'warn' });

            Logger.debug('Test', 'debug message');
            Logger.info('Test', 'info message');
            Logger.warn('Test', 'warn message');
            Logger.error('Test', 'error message');

            expect(console.debug).not.toHaveBeenCalled();
            expect(console.info).not.toHaveBeenCalled();
            expect(console.warn).toHaveBeenCalled();
            expect(console.error).toHaveBeenCalled();
        });
    });

    describe('debug', () => {
        it('should log debug messages when enabled', () => {
            Logger.configure({ enabled: true, level: 'debug' });
            Logger.debug('Runtime', 'Debug message');

            expect(console.debug).toHaveBeenCalledWith('[Pond:Runtime]', 'Debug message');
        });

        it('should log debug messages with args', () => {
            Logger.configure({ enabled: true, level: 'debug' });
            Logger.debug('Runtime', 'Debug message', { foo: 'bar' });

            expect(console.debug).toHaveBeenCalledWith('[Pond:Runtime]', 'Debug message', { foo: 'bar' });
        });

        it('should not log when level is higher', () => {
            Logger.configure({ enabled: true, level: 'info' });
            Logger.debug('Runtime', 'Debug message');

            expect(console.debug).not.toHaveBeenCalled();
        });
    });

    describe('info', () => {
        it('should log info messages when enabled', () => {
            Logger.configure({ enabled: true, level: 'info' });
            Logger.info('Runtime', 'Info message');

            expect(console.info).toHaveBeenCalledWith('[Pond:Runtime]', 'Info message');
        });

        it('should log info messages with multiple args', () => {
            Logger.configure({ enabled: true, level: 'info' });
            Logger.info('Runtime', 'Info message', 'arg1', 'arg2');

            expect(console.info).toHaveBeenCalledWith('[Pond:Runtime]', 'Info message', 'arg1', 'arg2');
        });
    });

    describe('warn', () => {
        it('should log warn messages when enabled', () => {
            Logger.configure({ enabled: true, level: 'warn' });
            Logger.warn('Runtime', 'Warning message');

            expect(console.warn).toHaveBeenCalledWith('[Pond:Runtime]', 'Warning message');
        });
    });

    describe('error', () => {
        it('should log error messages when enabled', () => {
            Logger.configure({ enabled: true, level: 'error' });
            Logger.error('Runtime', 'Error message');

            expect(console.error).toHaveBeenCalledWith('[Pond:Runtime]', 'Error message');
        });

        it('should always log errors when level is error', () => {
            Logger.configure({ enabled: true, level: 'error' });

            Logger.debug('Test', 'debug');
            Logger.info('Test', 'info');
            Logger.warn('Test', 'warn');
            Logger.error('Test', 'error');

            expect(console.debug).not.toHaveBeenCalled();
            expect(console.info).not.toHaveBeenCalled();
            expect(console.warn).not.toHaveBeenCalled();
            expect(console.error).toHaveBeenCalled();
        });
    });

    describe('disabled logging', () => {
        it('should not log when disabled', () => {
            Logger.configure({ enabled: false });

            Logger.debug('Test', 'debug');
            Logger.info('Test', 'info');
            Logger.warn('Test', 'warn');
            Logger.error('Test', 'error');

            expect(console.debug).not.toHaveBeenCalled();
            expect(console.info).not.toHaveBeenCalled();
            expect(console.warn).not.toHaveBeenCalled();
            expect(console.error).not.toHaveBeenCalled();
        });
    });
});

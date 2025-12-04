export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface LoggerConfig {
    enabled: boolean;
    level: LogLevel;
}

const levels: Record<LogLevel, number> = {
    debug: 0,
    info: 1,
    warn: 2,
    error: 3,
};

class LoggerImpl {
    private enabled = false;
    private level: LogLevel = 'info';

    configure(config: Partial<LoggerConfig>): void {
        if (config.enabled !== undefined) this.enabled = config.enabled;
        if (config.level !== undefined) this.level = config.level;
    }

    debug(tag: string, message: string, ...args: unknown[]): void {
        this.log('debug', tag, message, args);
    }

    info(tag: string, message: string, ...args: unknown[]): void {
        this.log('info', tag, message, args);
    }

    warn(tag: string, message: string, ...args: unknown[]): void {
        this.log('warn', tag, message, args);
    }

    error(tag: string, message: string, ...args: unknown[]): void {
        this.log('error', tag, message, args);
    }

    private log(level: LogLevel, tag: string, message: string, args: unknown[]): void {
        if (!this.enabled) return;
        if (levels[level] < levels[this.level]) return;

        const prefix = `[Pond:${tag}]`;
        const fn = console[level] || console.log;

        if (args.length > 0) {
            fn(prefix, message, ...args);
        } else {
            fn(prefix, message);
        }
    }
}

export const Logger = new LoggerImpl();

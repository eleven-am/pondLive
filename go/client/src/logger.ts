export interface LoggerOptions {
    debug?: boolean;
}

export class Logger {
    private static debugMode = false;

    static configure(options: LoggerOptions) {
        this.debugMode = options.debug ?? false;
    }

    static debug(tag: string, message: string, data?: any) {
        if (!this.debugMode) return;
        if (data) {
            console.debug(`[${tag}] ${message}`, data);
        } else {
            console.debug(`[${tag}] ${message}`);
        }
    }

    static warn(tag: string, message: string, error?: any) {
        if (error) {
            console.warn(`[${tag}] ${message}`, error);
        } else {
            console.warn(`[${tag}] ${message}`);
        }
    }

    static error(tag: string, message: string, error?: any) {
        if (error) {
            console.error(`[${tag}] ${message}`, error);
        } else {
            console.error(`[${tag}] ${message}`);
        }
    }
}

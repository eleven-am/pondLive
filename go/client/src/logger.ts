export interface LoggerOptions {
  debug?: boolean;
}

export class Logger {
  private static debugEnabled = false;

  static configure(options?: LoggerOptions): void {
    Logger.debugEnabled = Boolean(options?.debug);
  }

  static debug(...args: unknown[]): void {
    if (!Logger.debugEnabled) {
      return;
    }
    Logger.emit('log', 'debug', args);
  }

  static info(...args: unknown[]): void {
    Logger.emit('log', 'info', args);
  }

  static warn(...args: unknown[]): void {
    Logger.emit('warn', 'warn', args);
  }

  static error(...args: unknown[]): void {
    Logger.emit('error', 'error', args);
  }

  private static emit(method: 'log' | 'warn' | 'error', level: string, args: unknown[]): void {
    if (typeof console === 'undefined') {
      return;
    }
    const emitter = (console[method] as (...data: unknown[]) => void) ?? console.log;
    emitter(`[LiveUI][${level}]`, ...args);
  }
}

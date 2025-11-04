import { IncomingHttpHeaders, IncomingMessage } from 'http';

import { Manager } from '../context/manager';
import { parseAddress, parseCommonPrefix } from '../matcher/matcher';

type Method = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';

export class Request {
    readonly #path: URL;

    readonly #cookies: Record<string, string>;

    readonly #headers: IncomingHttpHeaders;

    readonly #userId: string;

    readonly #method: Method;

    #manager: Manager;

    constructor (manager: Manager, userId: string, req: IncomingMessage) {
        const cookies = req.headers.cookie?.split(';') ?? [];

        this.#path = new URL(`https://${req.headers.host}${req.url}`);
        this.#userId = userId;
        this.#cookies = Object.fromEntries(cookies.map((cookie) => cookie.split('=')));

        this.#headers = req.headers;
        this.#method = req.method as Method;
        this.#manager = manager;
    }

    get url (): URL {
        return this.#path;
    }

    get method (): Method {
        return this.#method;
    }

    get cookies (): Record<string, string> {
        return this.#cookies;
    }

    get headers (): IncomingHttpHeaders {
        return this.#headers;
    }

    get userId (): string {
        return this.#userId;
    }

    get params (): Record<string, string> {
        const data = parseAddress(this.#manager.path, parseCommonPrefix(this.#manager.path, this.#path.pathname) ?? '');

        if (!data) {
            return {};
        }

        return data.params;
    }

    get query (): Record<string, string> {
        return Object.fromEntries(this.#path.searchParams.entries());
    }

    static fromRequest (req: IncomingMessage, manager: Manager, userId: string): Request {
        return new Request(manager, userId, req);
    }

    getEvent (userId: string) {
        return this.#manager.context.getEvent(userId);
    }

    updateManager (manager: Manager) {
        this.#manager = manager;
    }
}


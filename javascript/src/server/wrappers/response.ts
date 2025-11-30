import { OutgoingHttpHeaders, ServerResponse } from 'http';

import { PondLiveHeaders } from '../context/context';

export interface CookieOptions {
    domain?: string;
    expires?: Date;
    httpOnly?: boolean;
    maxAge?: number;
    path?: string;
    sameSite?: 'lax' | 'strict' | 'none';
    secure?: boolean;
    signed?: boolean;
}

export function createSetCookieHeader (name: string, value: string, options?: CookieOptions): string {
    const cookieParts = [`${encodeURIComponent(name)}=${encodeURIComponent(value)}`];

    if (options) {
        if (options.expires instanceof Date) {
            cookieParts.push(`Expires=${options.expires.toUTCString()}`);
        }

        if (options.maxAge && Number.isInteger(options.maxAge)) {
            cookieParts.push(`Max-Age=${options.maxAge}`);
        }

        if (options.domain) {
            cookieParts.push(`Domain=${options.domain}`);
        }

        if (options.path) {
            cookieParts.push(`Path=${options.path}`);
        }

        if (options.secure === true) {
            cookieParts.push('Secure');
        }

        if (options.httpOnly === true) {
            cookieParts.push('HttpOnly');
        }

        if (options.sameSite && ['strict', 'lax', 'none'].includes(options.sameSite.toLowerCase())) {
            cookieParts.push(`SameSite=${options.sameSite.toLowerCase()}`);
        }
    }

    return `${cookieParts.join('; ')}`;
}


export class Response {
    readonly #response: ServerResponse;

    readonly #headers: OutgoingHttpHeaders;

    #statusCode: number;

    #responseSent: boolean;

    constructor (response: ServerResponse) {
        this.#headers = {};
        this.#statusCode = 200;
        this.#response = response;
        this.#responseSent = false;
    }

    get response (): ServerResponse {
        return this.#response;
    }

    get finished () {
        return this.#responseSent;
    }

    status (code: number): Response {
        this.#statusCode = code;

        return this;
    }

    setHeader (name: string, value: string): Response {
        if (this.#responseSent) {
            throw new Error('Cannot set headers after response is sent');
        }

        this.#headers[name] = value;

        return this;
    }

    setCookie (name: string, value: string, options?: CookieOptions): Response {
        this.setHeader('set-cookie', createSetCookieHeader(name, value, options));

        return this;
    }

    setPageTitle (title: string) {
        return this.setHeader(PondLiveHeaders.LIVE_PAGE_TITLE, title);
    }

    navigateTo (url: string) {
        this.#response.writeHead(302, {
            location: url,
        });
        this.#response.end();

        this.#responseSent = true;
    }

    html (html: string): void {
        this.setHeader('content-type', 'text/pkg');
        this.#response.writeHead(this.#statusCode, this.#headers);
        this.#response.end(html);
        this.#responseSent = true;
    }

    json (json: object): void {
        this.setHeader('content-type', 'application/json');
        this.#response.writeHead(this.#statusCode, this.#headers);
        this.#response.end(JSON.stringify(json));
        this.#responseSent = true;
    }

    replace (address: string) {
        return this.setHeader('x-router-action', `replace:${address}`);
    }

    get (header: PondLiveHeaders) {
        return this.#headers[header];
    }
}

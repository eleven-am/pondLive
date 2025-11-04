// eslint-disable-next-line import/no-unresolved
import PondClient from '@eleven-am/pondsocket/client';
import type { ClientChannel } from '@eleven-am/pondsocket/types';

import { UpdateData } from '../../server/context/context';
import { Html } from '../../server/parser/parser';
import { emitEvent } from '../events/eventEmmiter';
import { updateTheDom } from '../html/clone';
import { DomWatcher } from '../html/domWatcher';

export enum PondLiveHeaders {
    LIVE_USER_ID = 'x-pond-live-user-id',
    LIVE_ROUTER = 'x-pond-live-router',
    LIVE_PAGE_TITLE = 'x-pond-live-page-title',
    LIVE_ROUTER_ACTION = 'x-pond-live-router-action',
}

enum PondLiveActions {
    LIVE_ROUTER_NAVIGATE = 'navigate',
    LIVE_ROUTER_UPDATE = 'update',
    LIVE_ROUTER_RELOAD = 'reload',
    LIVE_ROUTER_GET_COOKIE = 'get-cookie',
}

export class ClientRouter {
    readonly #userId: string;

    readonly #channel: ClientChannel;

    #virtualDom: Html;

    constructor (channel: ClientChannel, virtualDom: Html, userId: string) {
        this.#userId = userId;
        this.#channel = channel;
        this.#virtualDom = virtualDom;
    }

    get channel () {
        return this.#channel;
    }

    public static connect (userId: string, watcher: DomWatcher, initialState: any) {
        const client = new PondClient('/live');

        client.connect();

        const channel = client.createChannel(`/${userId}`, {
            address: window.location.toString(),
        });

        channel.join();

        const virtualDom = Html.parse(initialState);

        const router = new ClientRouter(channel, virtualDom, userId);

        router.#init(watcher);

        return router;
    }

    async navigateTo (url: string, page = true) {
        const headers = {
            'Content-Type': 'application/json',
            Accept: 'application/json',
            [PondLiveHeaders.LIVE_ROUTER]: 'true',
            [PondLiveHeaders.LIVE_USER_ID]: this.#userId,
        };

        const response = await fetch(url, {
            headers,
            method: 'GET',
            redirect: 'follow',
            credentials: 'same-origin',
        });

        if (page) {
            try {
                const message: UpdateData = await response.json();

                this.#updateDom(message);
            } catch (e) {
                console.error(e);
            }

            history.pushState(null, '', url);
            emitEvent('navigate-end', { url });
        }
    }

    #init (watcher: DomWatcher) {
        this.#channel.onMessage((event, message) => {
            if (event === 'update') {
                this.#updateDom(message as UpdateData);
            } else if (event === 'live-event') {
                const { event: eventName, data } = message;

                emitEvent(eventName, data);
            }
        });

        watcher.delegateEvent('a', 'click', async (anchor, event) => {
            event.preventDefault();
            const target = anchor as HTMLAnchorElement;
            const url = target.href;

            emitEvent('navigate-start', { url });

            await this.navigateTo(url);
        });
    }

    #updateDom (message: UpdateData) {
        if (message.diff) {
            this.#virtualDom = this.#virtualDom.patch(message.diff);
            updateTheDom(this.#virtualDom.toString());
        }

        if (message[PondLiveHeaders.LIVE_PAGE_TITLE]) {
            document.title = message[PondLiveHeaders.LIVE_PAGE_TITLE];
        }

        if (message[PondLiveHeaders.LIVE_ROUTER_ACTION]) {
            switch (message[PondLiveHeaders.LIVE_ROUTER_ACTION]) {
                case PondLiveActions.LIVE_ROUTER_RELOAD:
                    window.location.reload();
                    break;
                case PondLiveActions.LIVE_ROUTER_GET_COOKIE:
                    void this.navigateTo(message.address ?? '', false);
                    break;
                case PondLiveActions.LIVE_ROUTER_UPDATE:
                    // TODO
                    break;
                case PondLiveActions.LIVE_ROUTER_NAVIGATE:
                    void this.navigateTo(message.address ?? '');
                    break;
                default:
                    break;
            }
        }
    }
}

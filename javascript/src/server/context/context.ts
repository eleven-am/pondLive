import fs from 'fs';

import type { Channel, JoinResponse } from '@eleven-am/pondsocket/types';

import { Route } from './liveContext';
import { Manager } from './manager';
import { isEmpty, uuidV4 } from '../helpers/helpers';
import { UserQueue } from '../helpers/userQueue';
import { Html } from '../parser/parser';
import { CookieOptions, createSetCookieHeader } from '../wrappers/response';
import { ServerEvent } from '../wrappers/serverEvent';


export enum PondLiveHeaders {
    LIVE_USER_ID = 'x-pond-live-user-id',
    LIVE_ROUTER = 'x-pond-live-router',
    LIVE_PAGE_TITLE = 'x-pond-live-page-title',
    LIVE_ROUTER_ACTION = 'x-pond-live-router-action',
}

export enum PondLiveActions {
    LIVE_ROUTER_NAVIGATE = 'navigate',
    LIVE_ROUTER_UPDATE = 'update',
    LIVE_ROUTER_RELOAD = 'reload',
    LIVE_ROUTER_GET_COOKIE = 'get-cookie',
}

interface ClientData {
    address: string;
    channel: Channel;
    virtualDom: Html;
}

export interface FileUpload {
    name: string;
    size: number;
    mimetype: string;
    path: string;
}

interface UploadedFile extends FileUpload {
    stream: () => NodeJS.ReadableStream;
}

export interface UpdateData {
    diff: Record<string, any> | null;
    [PondLiveHeaders.LIVE_ROUTER_ACTION]?: PondLiveActions;
    [PondLiveHeaders.LIVE_PAGE_TITLE]?: string;
    address?: string;
}

export type HookFunction = (event: ServerEvent) => void;
export type UploadFunction = (event: ServerEvent, files: UploadedFile[]) => void | Promise<void>;

export class Context {
    readonly queues: UserQueue;

    #clients: Map<string, ClientData>;

    readonly #managers: Manager[];

    #upgrading: Map<string, Html>;

    #dynamicRoutes: Map<string, string>;

    readonly #functionMap: Map<string, HookFunction>;

    readonly #uploadMap: Map<string, UploadFunction>;

    constructor () {
        this.#clients = new Map();
        this.#upgrading = new Map();
        this.#dynamicRoutes = new Map();
        this.#functionMap = new Map();
        this.#managers = [];
        this.queues = new UserQueue();
        this.#uploadMap = new Map();
    }

    upgradeUserOnJoin (userId: string, channel: Channel, response: JoinResponse, address: string, pondSocketId: string) {
        const html = this.#upgrading.get(userId);
        const event = new ServerEvent(userId, channel, this, {
            address,
            action: 'upgrade',
            value: null,
            dataId: null,
        });

        if (!html) {
            response.accept();
            event.reloadPage();

            channel.evictUser(pondSocketId, 'No upgrade available');

            return;
        }

        response.accept({
            userId,
        });

        const client: ClientData = {
            channel,
            address,
            virtualDom: html,
        };

        this.#clients.set(userId, client);
        this.#upgrading.delete(userId);

        return this.upgradeUser(event);
    }

    addEntryPoint (route: Route) {
        const absolutePath = `/${route.path}`
            .replace(/\/$/g, '')
            .replace(/\/+/g, '/');
        const existingManager = this.#managers.find((manager) => manager.path === absolutePath);

        if (existingManager) {
            return existingManager;
        }

        const manager = new Manager(this, route.component, absolutePath);

        manager.render('*', 'server');
        manager.doneBuilding();

        this.#managers.push(manager);

        return manager;
    }

    addUpgradingUser (userId: string, html: Html) {
        this.#upgrading.set(userId, html);
    }

    getClient (userId: string) {
        return this.#clients.get(userId);
    }

    upgradeUser (event: ServerEvent) {
        const promises = this.#managers.map((manager) => manager.upgradeUser(event));

        return Promise.all(promises);
    }

    unmountUser (userId: string, newPath: string | null) {
        const event = this.getEvent(userId);

        if (!event) {
            return;
        }

        const promises = this.#managers.map((manager) => manager.unmountUser(event, newPath));

        return Promise.all(promises);
    }

    getEvent (userId: string) {
        const client = this.#clients.get(userId);

        if (!client) {
            return;
        }

        return new ServerEvent(userId, client.channel, this, {
            address: client.address,
            action: 'update',
            value: null,
            dataId: null,
        });
    }

    addUploadPath (moveTo: string, managerId: string): string {
        const uploadPath = uuidV4();
        const pathWithToken = `/upload/${uploadPath}`;

        this.#dynamicRoutes.set(pathWithToken, JSON.stringify({
            moveTo,
            managerId,
        }));

        return pathWithToken;
    }

    addCookiePath (name: string, value: string, options: CookieOptions = {}): string {
        const pathWithToken = `/cookie/${uuidV4()}`;

        this.#dynamicRoutes.set(pathWithToken, createSetCookieHeader(name, value, options));

        return pathWithToken;
    }

    getUploadPath (uploadPath: string): { moveTo: string, managerId: string } | undefined {
        if (!uploadPath.startsWith('/upload/')) {
            return;
        }

        const data = this.#dynamicRoutes.get(uploadPath);

        this.#dynamicRoutes.delete(uploadPath);

        if (!data) {
            return;
        }

        try {
            return JSON.parse(data);
            // eslint-disable-next-line no-inline-comments
        } catch { /* empty */ }
    }

    getCookiePath (cookiePath: string): string | undefined {
        if (!cookiePath.startsWith('/cookie/')) {
            return;
        }

        const data = this.#dynamicRoutes.get(cookiePath);

        this.#dynamicRoutes.delete(cookiePath);

        return data;
    }

    upSertHook (name: string, fn: HookFunction) {
        this.#functionMap.set(name, fn);
    }

    performAction (event: ServerEvent) {
        const fn = this.#functionMap.get(event.action);

        if (!fn) {
            return;
        }

        fn(event);
    }

    reload (userId: string) {
        void this.queues.add(userId, () => this.#reload(userId));
    }

    addTask (userId: string, task: () => void | Promise<void>) {
        void this.queues.add(userId, task);
    }

    addUploadFunction (managerId: string, fn: UploadFunction) {
        this.#uploadMap.set(managerId, fn);
    }

    triggerUpload (managerId: string, userId: string, files: FileUpload[]) {
        const uploadFiles = files.map((file): UploadedFile => ({
            ...file,
            stream: () => fs.createReadStream(file.path),
        }));

        const fn = this.#uploadMap.get(managerId);

        if (!fn) {
            return;
        }

        const client = this.#clients.get(userId);

        if (!client) {
            return;
        }

        const event = new ServerEvent(userId, client.channel, this, {
            address: client.address,
            action: 'upload',
            value: null,
            dataId: null,
        });

        fn(event, uploadFiles);
    }

    #reload (userId: string) {
        const client = this.#clients.get(userId);

        if (!client) {
            return;
        }

        const event = new ServerEvent(userId, client.channel, this, {
            address: client.address,
            action: 'reload',
            value: null,
            dataId: null,
        });

        const manager = this.#managers.find((m) => m.canRender(event.path));

        if (!manager) {
            return;
        }

        const html = manager.render(event.path, userId);
        const diff = client.virtualDom.differentiate(html);

        if (isEmpty(diff)) {
            return;
        }

        event.sendDiff(diff);
        client.virtualDom = html;
    }
}

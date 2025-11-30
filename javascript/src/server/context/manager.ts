import fs from 'fs';
import { ServerResponse, IncomingMessage } from 'http';
import path from 'path';

import { Context, UpdateData, PondLiveHeaders, HookFunction, UploadFunction } from './context';
import { Component, LiveContext, Route } from './liveContext';
import { getMimeType } from './router';
import { uuidV4, deepCompare, fileExists, isEmpty } from '../helpers/helpers';
import { parseCommonPrefix, parseAddress } from '../matcher/matcher';
import { NextFunction } from '../middleware/middleware';
import { html } from '../parser/parser';
import { Request } from '../wrappers/request';
import { Response } from '../wrappers/response';
import { ServerEvent } from '../wrappers/serverEvent';


export type MountFunction = (req: Request, res: Response) => void | Promise<void>;
export type UpgradeFunction = (event: ServerEvent) => void | Promise<void>;
export type UnmountFunction = (event: ServerEvent) => void | Promise<void>;

type Hook<T> = {
    state: Map<string, T>;
    args: Record<string, T | ((state: T, event: ServerEvent) => T | Promise<T>)>;
}

export class Manager {
    readonly id: string;

    readonly #hooks: Map<string, Hook<any>>;

    readonly #mountFunctions: MountFunction[];

    readonly #upgradeFunctions: UpgradeFunction[];

    readonly #unmountFunctions: UnmountFunction[];

    readonly #component: Component;

    readonly #children: Map<string, Manager>;

    readonly #mountedUsers: Map<string, string>;

    readonly #upgradedUsers: Map<string, string>;

    readonly #cleanups: Map<string, (() => void)[]>;

    readonly #absolutePath: string;

    readonly #context: Context;

    #isBuilt: boolean;

    constructor (context: Context, component: Component, absolutePath: string) {
        this.#mountFunctions = [];
        this.#upgradeFunctions = [];
        this.#unmountFunctions = [];
        this.#component = component;
        this.#isBuilt = false;
        this.#children = new Map();
        this.#cleanups = new Map();
        this.#mountedUsers = new Map();
        this.#upgradedUsers = new Map();
        this.id = Math.random()
            .toString(36)
            .substring(7);
        this.#hooks = new Map();
        this.#absolutePath = absolutePath;
        this.#context = context;
    }

    get component () {
        return this.#component;
    }

    get isBuilt () {
        return this.#isBuilt;
    }

    get path () {
        return this.#absolutePath;
    }

    get context () {
        return this.#context;
    }

    doneBuilding () {
        this.#isBuilt = true;
    }

    onMount (fn: MountFunction) {
        if (this.#isBuilt) {
            return;
        }

        this.#mountFunctions.push(fn);
    }

    onUpgrade (fn: UpgradeFunction) {
        if (this.#isBuilt) {
            return;
        }

        this.#upgradeFunctions.push(fn);
    }

    onUnmount (fn: UnmountFunction) {
        if (this.#isBuilt) {
            return;
        }

        this.#unmountFunctions.push(fn);
    }

    onUpload (fn: UploadFunction) {
        if (this.#isBuilt) {
            return;
        }

        this.#context.addUploadFunction(this.id, fn);
    }

    setUpHook (hookCount: number) {
        const hookKey = `hook-${this.id}-${hookCount}`;
        let hookFunctions = this.#hooks.get(hookKey);

        if (!hookFunctions) {
            if (this.#isBuilt) {
                throw new Error('Cannot add hook after building phase');
            }

            hookFunctions = {
                args: {},
                state: new Map(),
            };
            this.#hooks.set(hookKey, hookFunctions);
        }

        return hookKey;
    }

    addHookFunction (hookKey: string, arg: unknown, fn: HookFunction): string {
        const hook = this.#hooks.get(hookKey);

        if (!hook) {
            throw new Error('Cannot add hook function to non existing hook');
        }

        const argKey = Object.keys(hook.args).find((key) => deepCompare(hook.args[key], arg));

        if (argKey) {
            this.#context.upSertHook(argKey, fn);

            return argKey;
        }

        const newArgKey = `${hookKey}-${Math.random()
            .toString(36)
            .substring(7)}`;

        hook.args[newArgKey] = arg;
        this.#context.upSertHook(newArgKey, fn);

        return newArgKey;
    }

    getHookState<T> (hookKey: string, initialState: T, userId: string): T {
        const hook = this.#hooks.get(hookKey);

        if (!hook) {
            throw new Error('Cannot get hook state from non existing hook');
        }

        if (!hook.state.has(userId)) {
            hook.state.set(userId, initialState);
        }

        return hook.state.get(userId) as T;
    }

    setHookState<T> (hookKey: string, newState: T, userId: string, address: string) {
        const hook = this.#hooks.get(hookKey);

        if (!hook) {
            throw new Error('Cannot set hook state from non existing hook');
        }

        if (!this.isMounted(userId)) {
            return;
        }

        const currentState = hook.state.get(userId);

        if (deepCompare(currentState, newState)) {
            return;
        }

        hook.state.set(userId, newState);
        this.#context.reload(userId);
    }

    unMountHook (userId: string, fn: () => void) {
        const cleanups = this.#cleanups.get(userId) ?? [];

        cleanups.push(fn);
        this.#cleanups.set(userId, cleanups);
    }

    deleteHookState (hookKey: string, userId: string) {
        const hook = this.#hooks.get(hookKey);

        if (!hook) {
            throw new Error('Cannot delete hook state from non existing hook');
        }

        hook.state.delete(userId);
    }

    render (address: string, userId: string) {
        const liveContext = new LiveContext(userId, address, this);

        const htmlData = this.#component(liveContext);
        const styles = liveContext.styles;

        return html`${styles}${htmlData}`;
    }

    handleHttpRequest (req: IncomingMessage, res: ServerResponse, next: NextFunction, publicDir: string[]) {
        if (!this.canRender(req.url ?? '')) {
            return next();
        }

        let userId = req.headers[PondLiveHeaders.LIVE_USER_ID] as string;

        if (!userId) {
            userId = uuidV4();
            const request = Request.fromRequest(req, this, userId);
            const response = new Response(res);

            return this.#handleFirstHttpRequest(request, response, publicDir);
        }

        const request = Request.fromRequest(req, this, userId);
        const response = new Response(res);

        return this.#handleSubsequentHttpRequest(request, response, next);
    }

    canRender (address: string): boolean {
        if (!this.#isBuilt) {
            throw new Error('Cannot check if component can render before the building phase');
        }

        const children = Array.from(this.#children.values());

        if (parseAddress(this.#absolutePath, address) !== null) {
            return true;
        }

        return children.some((manager) => manager.canRender(address));
    }

    createContext (address: string, userId: string) {
        return new LiveContext(userId, address, this);
    }

    initRoute (route: Route) {
        if (this.#isBuilt) {
            throw new Error('Cannot add route after the building phase');
        }

        const absolutePath = `${this.#absolutePath}/${route.path}`.replace(/\/+/g, '/');
        const manager = new Manager(this.#context, route.component, absolutePath);

        this.#children.set(route.path, manager);

        return manager;
    }

    getContext (path: string, userId: string, address: string) {
        if (!this.#isBuilt) {
            throw new Error('Cannot retrieve route before the building phase');
        }

        if (!this.#children.has(path)) {
            throw new Error(`Cannot retrieve non existing route ${path}`);
        }

        const manager = this.#children.get(path) as Manager;

        return new LiveContext(userId, address, manager);
    }

    isMounted (userId: string) {
        return this.#mountedUsers.has(userId);
    }

    async unmountUser (event: ServerEvent, newPath: string | null) {
        for (const [_, manager] of this.#children) {
            await manager.unmountUser(event, newPath);
        }

        const canRender = this.canRender(event.path);
        const canRenderNew = newPath ? this.canRender(newPath) : false;

        if (canRenderNew) {
            return;
        }

        if (canRender && this.#mountedUsers.has(event.userId) && this.#upgradedUsers.has(event.userId)) {
            await this.#unmount(event);
        }
    }

    async upgradeUser (event: ServerEvent) {
        await this.#performRenderAction(event.path, async (manager) => {
            await manager.#upgrade(event);
        });
    }

    #upgrade (event: ServerEvent) {
        if (!this.#isBuilt) {
            throw new Error('Cannot upgrade component before the building phase');
        }

        const prefix = parseCommonPrefix(this.#absolutePath, event.url.pathname);
        const previous = this.#upgradedUsers.get(event.userId);

        if ((previous && previous === prefix) || prefix === null) {
            return;
        }

        this.#upgradedUsers.set(event.userId, prefix);
        const promises = this.#upgradeFunctions.map((fn) => fn(event));

        return Promise.all(promises);
    }

    #unmount (event: ServerEvent) {
        if (!this.#isBuilt) {
            throw new Error('Cannot unmount component before the building phase');
        }

        this.#mountedUsers.delete(event.userId);
        this.#upgradedUsers.delete(event.userId);
        (this.#cleanups.get(event.userId) ?? []).forEach((fn) => fn());
        this.#cleanups.delete(event.userId);

        const promises = this.#unmountFunctions.map((fn) => fn(event));

        return Promise.all(promises);
    }

    #mount (req: Request, res: Response) {
        if (!this.#isBuilt) {
            throw new Error('Cannot mount component before the building phase');
        }

        if (res.finished) {
            return;
        }

        const prefix = parseCommonPrefix(this.#absolutePath, req.url.pathname);
        const previous = this.#mountedUsers.get(req.userId);

        if ((previous && previous === prefix) || prefix === null) {
            return;
        }

        req.updateManager(this);
        this.#mountedUsers.set(req.userId, prefix);
        const promises = this.#mountFunctions.map((fn) => fn(req, res));

        return Promise.all(promises);
    }

    #serveFile (filePath: string, res: ServerResponse, callback: (data: string) => string) {
        fs.readFile(filePath, 'utf-8', (err, data) => {
            if (err) {
                res.writeHead(500);
                res.end('Error loading file');
            } else {
                // eslint-disable-next-line callback-return
                const result = callback(data);

                res.writeHead(200, { 'Content-Type': getMimeType(filePath) });
                res.end(result);
            }
        });
    }

    async #mountUser (req: Request, res: Response) {
        await this.#performRenderAction(req.url.pathname, async (manager) => {
            await manager.#mount(req, res);
        });
    }

    async #handleFirstHttpRequest (req: Request, res: Response, publicDir: string[]) {
        await this.#mountUser(req, res);

        if (res.finished) {
            return;
        }

        const html = this.render(req.url.pathname, req.userId);

        this.#context.addUpgradingUser(req.userId, html);

        const title = res.get(PondLiveHeaders.LIVE_PAGE_TITLE) ?? 'Pond Live';

        const store = `
                <script>
                    window.__USER_ID__ = '${req.userId}';
                    window.__STATE__ = ${JSON.stringify(html.getParts())};
                </script>
            `;

        const promises = publicDir.map((dir) => fileExists(path.join(dir, 'index.pkg')));
        const folder = (await Promise.all(promises))
            .map((exists, index) => ({
                exists,
                folder: publicDir[index],
            }))
            .find((folder) => folder.exists)?.folder;

        if (!folder) {
            const index = `
                <!DOCTYPE html>
                <html lang="en">
                <head>
                    <meta charset="UTF-8">
                    <title>${title}</title>
                    ${store}
                </head>
                <body>
                    <div id="app">${html.toString().trim()}</div>
                    <script src="/pondLive.js" defer></script>
                </body>
                </html>
            `.trim();

            res.html(index);

            return;
        }

        this.#serveFile(path.join(folder, 'index.pkg'), res.response, (data) => data
            .replace(/<head>/, `<head>${store}`)
            .replace(/<body>/, `<body><div id="app">${html.toString()
                .trim()}</div><script src="/pondLive.js" defer></script>`)
            .replace(/<title>(.*?)<\/title>/, `<title>${title}</title>`));
    }

    #handleSubsequentHttpRequest (req: Request, res: Response, next: NextFunction) {
        this.#context.addTask(req.userId, async () => {
            await this.#context.unmountUser(req.userId, req.url.pathname);
            const client = this.#context.getClient(req.userId);

            if (req.headers[PondLiveHeaders.LIVE_ROUTER] !== 'true') {
                return next();
            }

            if (!client) {
                res.status(500).json({ error: 'Client not found' });

                return;
            }

            const event = new ServerEvent(req.userId, client.channel, this.#context, {
                action: 'navigate',
                address: req.url.toString(),
                value: null,
                dataId: null,
            });

            await this.#mountUser(req, res);

            if (res.finished) {
                return;
            }

            await this.upgradeUser(event);

            const html = this.render(req.url.pathname, req.userId);
            const diff = client.virtualDom.differentiate(html);

            client.virtualDom = html;
            client.address = req.url.toString();
            const title = res.get(PondLiveHeaders.LIVE_PAGE_TITLE) ?? 'Pond Live';

            const data: UpdateData = {
                diff: isEmpty(diff) ? null : diff,
                [PondLiveHeaders.LIVE_PAGE_TITLE]: title as string,
            };

            res.json(data);
        });
    }

    async #performRenderAction (address: string, fn: (manager: Manager) => Promise<void>) {
        const canRender = this.canRender(address);

        if (!canRender) {
            return;
        }

        await fn(this);

        for (const [_, manager] of this.#children) {
            await manager.#performRenderAction(address, fn);
        }
    }
}

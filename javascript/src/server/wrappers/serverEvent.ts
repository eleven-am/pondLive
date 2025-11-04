import type { Channel } from '@eleven-am/pondsocket/types';

import { CookieOptions } from './response';
import { PondUploadResponse } from '../../client/actors/uploader';
import { LiveEvent, DragData, PondFile, UploadList, FileMetaData } from '../../client/types';
import { Context, UpdateData, PondLiveHeaders, PondLiveActions } from '../context/context';
import { parseAddress } from '../matcher/matcher';


interface PondEvent {
    value: string | null;
    dataId: string | null;
    dragData?: DragData;
    formData?: Record<string, string>;
}

interface UploadEvent extends FileMetaData {
    accept: (saveTo: string) => void;
    reject: (message: string) => void;
}

interface UploadEventList {
    files: UploadEvent[];
    accept: (saveTo: string) => void;
    reject: (message: string) => void;
}

export class ServerEvent {
    #uploadList: UploadEventList | null;

    #managerId: string;

    readonly #userId: string;

    readonly #client: Channel;

    readonly #data: LiveEvent;

    readonly #url: URL;

    readonly #context: Context;

    constructor (userId: string, client: Channel, context: Context, data: LiveEvent) {
        this.#userId = userId;
        this.#client = client;
        this.#data = data;
        this.#url = new URL(data.address);
        this.#context = context;
        this.#uploadList = null;
        this.#managerId = '';
    }

    get path () {
        return this.#url.pathname;
    }

    get url () {
        return this.#url;
    }

    get userId () {
        return this.#userId;
    }

    get action () {
        return this.#data.action;
    }

    set action (action: string) {
        this.#data.action = action;
    }

    get data (): PondEvent {
        return {
            value: this.#data.value,
            dataId: this.#data.dataId,
            dragData: this.#data.dragData,
            formData: this.#data.formData,
        };
    }

    get files (): UploadEventList | null {
        if (!this.#data.files) {
            return null;
        }

        if (this.#uploadList) {
            return this.#uploadList;
        }

        this.#uploadList = this.#buildFileList(this.#data.files);

        return this.#uploadList;
    }

    get managerId (): string {
        return this.#managerId;
    }

    set managerId (managerId: string) {
        this.#managerId = managerId;
    }

    emit (event: string, data: any) {
        if (this.action === 'unmount') {
            return;
        }

        this.#client.broadcastMessage('live-event', {
            event,
            data,
        });
    }

    matches (path: string): boolean {
        return Boolean(parseAddress(`${path}/*`.replace(/\/+/g, '/'), this.#url.pathname));
    }

    navigateTo (path: string) {
        this.#sendMessage({
            [PondLiveHeaders.LIVE_ROUTER_ACTION]: PondLiveActions.LIVE_ROUTER_NAVIGATE,
            address: path,
            diff: null,
        });
    }

    setCookie (name: string, value: string, options?: CookieOptions) {
        const path = this.#context.addCookiePath(name, value, options);

        this.#sendMessage({
            diff: null,
            address: path,
            [PondLiveHeaders.LIVE_ROUTER_ACTION]: PondLiveActions.LIVE_ROUTER_GET_COOKIE,
        });
    }

    setPageTitle (title: string) {
        this.#sendMessage({
            [PondLiveHeaders.LIVE_PAGE_TITLE]: title,
            diff: null,
        });
    }

    reloadPage () {
        this.#sendMessage({
            [PondLiveHeaders.LIVE_ROUTER_ACTION]: PondLiveActions.LIVE_ROUTER_RELOAD,
            diff: null,
        });
    }

    sendDiff (diff: Record<string, any>) {
        this.#sendMessage({
            diff,
        });
    }

    getEvent (userId: string) {
        return this.#context.getEvent(userId);
    }

    #buildFileList (files: UploadList): UploadEventList {
        const filesList = files.files.map((file) => this.#buildFile(file));

        const accept = this.#acceptFile.bind(this, files.identifier);
        const reject = this.#rejectFile.bind(this, files.identifier);

        return {
            files: filesList,
            accept,
            reject,
        };
    }

    #acceptFile (identifier: string, moveTo: string) {
        const path = this.#context.addUploadPath(moveTo, this.#managerId);
        const event: PondUploadResponse = {
            accepted: true,
            route: path,
        };

        this.#client.broadcastMessage(identifier, event);
    }

    #rejectFile (identifier: string, message: string) {
        const event: PondUploadResponse = {
            accepted: false,
            message,
        };

        this.#client.broadcastMessage(identifier, event);
    }

    #buildFile (file: PondFile): UploadEvent {
        const { identifier, lastModifiedDate, ...rest } = file;

        const accept = this.#acceptFile.bind(this, identifier);
        const reject = this.#rejectFile.bind(this, identifier);

        return {
            lastModifiedDate: new Date(lastModifiedDate),
            ...rest,
            accept,
            reject,
        };
    }

    #sendMessage (message: UpdateData) {
        this.#client.broadcastMessage('update', message);
    }
}

import { UploadFunction } from './context';
import { Manager, MountFunction, UpgradeFunction, UnmountFunction } from './manager';
import { Html } from '../parser/parser';

export type Component = (props: LiveContext) => Html;

export interface Route {
    path: string;
    component: Component;
}

export class LiveContext {
    readonly #manager: Manager;

    readonly #userId: string;

    readonly #address: string;

    #hookCount: number;

    #styles: Html[];

    constructor (userId: string, address: string, manager: Manager) {
        this.#manager = manager;
        this.#userId = userId;
        this.#hookCount = -1;
        this.#address = address;
        this.#styles = [];
    }

    get userId (): string {
        return this.#userId;
    }

    get isBuilt (): boolean {
        return this.#manager.isBuilt;
    }

    get manager (): Manager {
        return this.#manager;
    }

    get address (): string {
        return this.#address;
    }

    get styles () {
        return this.#styles;
    }

    onMount (fn: MountFunction) {
        this.#manager.onMount(fn);
    }

    onUpgrade (fn: UpgradeFunction) {
        this.#manager.onUpgrade(fn);
    }

    onUnmount (fn: UnmountFunction) {
        this.#manager.onUnmount(fn);
    }

    onUpload (fn: UploadFunction) {
        this.#manager.onUpload(fn);
    }

    setUpHook () {
        this.#hookCount += 1;

        const hookKey = this.#manager.setUpHook(this.#hookCount);
        const onUnMount = this.#manager.unMountHook.bind(this.#manager);
        const deleteState = this.#manager.deleteHookState.bind(this.#manager, hookKey);
        const isMounted = this.#manager.isMounted.bind(this.#manager);
        const getEvent = this.getEvent.bind(this);

        return {
            hookKey,
            onUnMount,
            deleteState,
            isMounted,
            getEvent,
        };
    }

    setUpStateHook <T> (initialState: T, _debugValue: string) {
        const { hookKey, deleteState, onUnMount, isMounted, getEvent } = this.setUpHook();
        const getState = this.#manager.getHookState.bind(this.#manager, hookKey, initialState) as (userId: string) => T;
        const setState = this.#manager.setHookState.bind(this.#manager, hookKey) as (state: T, userId: string) => void;
        const addDispatcher = this.#manager.addHookFunction.bind(this.#manager, hookKey);

        return {
            getEvent,
            getState,
            setState,
            onUnMount,
            isMounted,
            deleteState,
            addDispatcher,
            managerId: this.#manager.id,
        };
    }

    reload () {
        this.#manager.context.reload(this.#userId);
    }

    getContext (route: Route): LiveContext {
        if (!this.isBuilt) {
            const manager = this.#manager.initRoute(route);

            manager.render('*', 'server');
            manager.doneBuilding();

            return manager.createContext(this.#address, this.#userId);
        }

        return this.#manager.getContext(route.path, this.#userId, this.#address);
    }

    canRender (address: string): boolean {
        return this.#manager.canRender(address);
    }

    addStyle (css: Html) {
        this.#styles = [...this.#styles, css];
    }

    getEvent (userId: string) {
        return this.#manager.context.getEvent(userId);
    }
}

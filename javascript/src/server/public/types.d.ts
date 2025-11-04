import { IncomingHttpHeaders, Server } from 'http';

import { PondSocketExpressApp } from '@eleven-am/pondsocket/types';
import { Express } from 'express';

export interface DragData {
    top: number;
    left: number;
    width: number;
    height: number;
}

interface PondEvent {
    value: string | null;
    dataId: string | null;
    dragData?: DragData;
    formData?: Record<string, string>;
}

interface FileUpload {
    name: string;
    size: number;
    mimetype: string;
    path: string;
}

interface UploadedFile extends FileUpload {
    stream: () => NodeJS.ReadableStream;
}

export interface Route {
    // The path the component will be mounted on
    path: string;
    // The component to be mounted
    component: Component;
}

export type Method = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
export type Component = (props: LiveContext) => Html;
export type HookContext = LiveContext | ServerEvent | Request;
export type MountFunction = (req: Request, res: Response) => void | Promise<void>;
export type UpgradeFunction = (event: ServerEvent) => void | Promise<void>;
export type UnmountFunction = (event: ServerEvent) => void | Promise<void>
export type UploadFunction = (event: ServerEvent, files: UploadedFile[]) => void | Promise<void>;
export type SetState<T> = (state: (T | ((state: T, event: ServerEvent) => T | Promise<T>))) => string;
export type SetOnServer<T> = (context: HookContext, state: (T | ((state: T) => T | Promise<T>))) => void;
export type CreatedState<T> = [T, SetState<T>, SetOnServer<T>];
export type CSSProperties = Record<string, string | number>;
export type CSSClasses = Record<string, CSSProperties>;
export type CSSGenerator = (props: any) => CSSClasses;
export type Action<T> = Record<string, (event: ServerEvent, prev: T) => T | Promise<T | void> | void>;
export type RunAction<T> = Record<keyof T, string>;
export type CreatedAction<T, A extends Action<T>> = [T, RunAction<A>, SetOnServer<T>];
export type Effect<T> = (change: T, event: ServerEvent) => (() => void) | Promise<(() => void)> | void | Promise<void>;
export type CreatedInfo<T> = [T, (context: HookContext, newState: Partial<T>) => void, (effect: Effect<T>) => void];

interface CookieOptions {
    domain?: string;
    expires?: Date;
    httpOnly?: boolean;
    maxAge?: number;
    path?: string;
    sameSite?: boolean | 'lax' | 'strict' | 'none';
    secure?: boolean;
    signed?: boolean;
}

export declare class UploadEvent {
    name: string;

    size: number;

    mimeType: string;

    path: string;

    lastModified: number;

    lastModifiedDate: Date;

    accept: (saveTo: string) => void;

    reject: (message: string) => void;
}

export declare class UploadEventList {
    files: UploadEvent[];

    accept: (saveTo: string) => void;

    reject: (message: string) => void;
}

export declare class ServerEvent {
    // The user id of the user that triggered the event
    userId: string;

    // the payload of the event
    data: PondEvent;

    // if the event was triggered by a file upload, this will be the file
    files: UploadEventList | null;

    /**
     * Emit an event to the client
     * @param event - The event to emit
     * @param data - The data to send with the event
     */
    emit(event: string, data: any): void;

    /**
     * Navigate to a new page
     * @param path - The path to navigate to
     */
    navigateTo(path: string): void;

    /**
     * Set a cookie on the client
     * @param name - The name of the cookie
     * @param value - The value of the cookie
     * @param options - The options of the cookie
     */
    setCookie(name: string, value: string, options?: CookieOptions): void;

    /**
     * Set the page title of the current page
     * @param title - The title of the page
     */
    setPageTitle(title: string): void;
}

export declare class Request {
    // The url of the request
    url: URL;

    // The method of the request
    method: Method;

    // The cookies sent with the request
    cookies: Record<string, string>;

    // The query parameters sent with the request
    query: Record<string, string>;

    // The parameters sent with the request
    params: Record<string, string>;

    // The headers sent with the request
    headers: IncomingHttpHeaders;

    // The id of the user that made the request
    userId: string;
}

export declare class Response {
    /**
     * Set the status code of the response
     * @param code - The status code to set
     */
    status(code: number): Response;

    /**
     * Set the header of the response
     * @param name - The name of the header
     * @param value - The value of the header
     */
    setHeader(name: string, value: string): Response;

    /**
     * Set the cookie of the response
     * @param name - The name of the cookie
     * @param value - The value of the cookie
     * @param options - The options of the cookie
     */
    setCookie(name: string, value: string, options?: CookieOptions): Response;

    /**
     * Set the Page Title of the current page
     * @param title - The title of the page
     */
    setPageTitle(title: string): Response;

    /**
     * Navigate to a new page
     * @param url - The url to navigate to
     */
    navigateTo(url: string): Response;

    /**
     * Modify the browser history without navigating to a new page
     * @param address - The address to replace the current page with
     */
    replace(address: string): Response;
}

export declare class LiveContext {
    // The id of the user that made the request
    userId: string;

    /**
     * Called when a user visits the component for the first time
     * @param fn - The function to be called
     */
    onMount(fn: MountFunction): void;

    /**
     * Called when a user upgrades from HTTP to WebSockets
     * @param fn - The function to be called
     */
    onUpgrade(fn: UpgradeFunction): void;

    /**
     * Called when a user leaves the component
     * @param fn - The function to be called
     */
    onUnmount(fn: UnmountFunction): void;

    /**
     * Called when a user uploads a file | files
     * @param fn - The function to be called
     */
    onUpload(fn: UploadFunction): void;
}

export declare class ServerInfo<T> {
    /**
     * Assign a new state to the serverInfo object
     * @param context - The context of the hook
     * @param newState - The new state to assign
     */
    assign (context: HookContext, newState: Partial<T>): void;

    /**
     * Get the current state of the serverInfo object
     */
    getState (): T;

    /**
     * Set the state of the serverContext object
     * @param context - The context of the hook
     * @param setter - The function that sets the state
     */
    setState(context: HookContext, setter: ((state: T) => T) | T): void;
}

export declare class ServerContext<T> {
    /**
     * Assign a new state to the serverInfo object
     * @param context - The context of the hook
     * @param newState - The new state to assign
     */
    assign (context: HookContext, newState: Partial<T>): void;

    /**
     * Get the current state of the serverContext object
     * @param context - The context of the hook
     */
    getState (context: HookContext): T;

    /**
     * Destroy the serverContext object for a user
     * @param context - The context of the hook
     */
    destroy (context: HookContext): void;

    /**
     * Set the state of the serverContext object
     * @param context - The context of the hook
     * @param setter - The function that sets the state
     */
    setState(context: HookContext, setter: ((state: T) => T) | T): void;
}

export class Html {}

export declare class Router {
    /**
     * Add a route to the router
     * @param path - The path of the route, where the component will be rendered
     * @param component - The component to render
     */
    mount(path: string, component: Component): void;

    /**
     * Add a directory that should be served statically
     * @param dir - The directory to serve
     */
    addStaticRoute(dir: string): void;

    /**
     * Activates the router to start serving components
     * @param args
     */
    serve(...args: any[]): Server;

    /**
     * Activates the router to start serving components using express
     * @param entryPoint - The entry point of the application
     * @param app - The express app to use
     */
    serveWithExpress(entryPoint: string, app: Express): PondSocketExpressApp;
}

/**
 * The createServerInfo function creates a ServerInfo object that holds a single state object that can be accessed by all users
 * @param initialState - The initial state of the serverInfo object
 */
export declare function createServerInfo<T>(initialState: T): ServerInfo<T>;

/**
 * The createServerContext function creates a ServerContext object that holds a state object for each user
 * @param initialState - The initial state of the serverContext object
 */
export declare function createClientContext<T>(initialState: T): ServerContext<T>;

/**
 * The useServerInfo hook returns the current state of the serverInfo | serverContext object, and a function to set the state
 * @param context - The LiveContext object
 * @param serverInfo - The ServerInfo | ServerContext object
 */
export declare function useServerInfo<T>(context: LiveContext, serverInfo: ServerInfo<T> | ServerContext<T>): CreatedInfo<T>;

/**
 * The useState hook returns the current state of the component, and a function to set the state
 * @param context - The LiveContext object
 * @param initialState - The initial state of the component
 */
export declare function useState<T>(context: LiveContext, initialState: T): CreatedState<T>;

/**
 * The makeStyle hook returns a function that takes a CSSObject and returns a hook that returns the styles in aa component
 * @param classes - The CSS class object to use
 */
export declare function makeStyles<A extends CSSClasses>(classes: A): (context: LiveContext) => {
    [K in keyof A]: string;
};

/**
 * The makeStyle hook returns a function that takes a CSSObject and returns a hook that returns the styles in aa component
 * @param classes - The CSS class object to use
 */
export declare function makeStyles<B extends CSSGenerator>(classes: B): (context: LiveContext, props: Parameters<B>[0]) => {
    [K in keyof ReturnType<B>]: string;
};

/**
 * The useAction is uses to generate actions for a component when an event is triggered. the returned value for the hook is the result of the action
 * @param context - The LiveContext object
 * @param initialState - The initial state of the component, which us used when no action has been triggered
 * @param actions - The actions to use
 */
export declare function useAction<T, A extends Action<T>>(context: LiveContext, initialState: T, actions: A): CreatedAction<T, A>;

/**
 * The useRouter hook takes in multiple sub routes that can be rendered within a component, it returns the correct route to render
 * @param routes - The routes to use
 */
export declare function useRouter(routes: Route[]): Component;

export declare function html(statics: TemplateStringsArray, ...dynamics: unknown[]): Html;


import { HookContext } from './useServerInfo';
import { LiveContext } from '../context/liveContext';
import { ServerEvent } from '../wrappers/serverEvent';

export type SetState<T> = (state: (T | ((state: T, event: ServerEvent) => T | Promise<T>))) => string;
export type SetOnServer<T> = (event: HookContext, state: (T | ((state: T) => T | Promise<T>))) => void;
export type CreatedState<T> = [T, SetState<T>, SetOnServer<T>];

export function useState<T> (context: LiveContext, initialState: T): CreatedState<T> {
    const { getState, setState, addDispatcher, onUnMount, deleteState, managerId } = context.setUpStateHook(initialState, 'useState');

    const state = getState(context.userId);

    const setStateFn = async (state: (T | ((state: T, event: ServerEvent) => T | Promise<T>)), event: HookContext) => {
        let newState: T;

        if (event instanceof ServerEvent) {
            event.managerId = managerId;
        }

        if (typeof state === 'function') {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            newState = await state(getState(event.userId), event);
        } else {
            newState = state;
        }

        setState(newState, event.userId);
    };

    onUnMount(context.userId, () => deleteState(context.userId));

    const setOnServer = (event: HookContext, state: (T | ((state: T) => T | Promise<T>))) => setStateFn(state, event);

    const mutate = (state: (T | ((state: T, event: ServerEvent) => T | Promise<T>))) => addDispatcher(state, (event) => setStateFn(state, event));

    return [state, mutate, setOnServer];
}

import { HookContext } from './useServerInfo';
import { LiveContext } from '../context/liveContext';
import { ServerEvent } from '../wrappers/serverEvent';

type Action<T> = Record<string, (event: ServerEvent, prev: T) => T | void | Promise<T> | Promise<void>>;
type RunAction<T> = Record<keyof T, string>;
type SetOnServer<T> = (context: HookContext, state: (T | ((state: T) => T | Promise<T>))) => void;
type InitialState<A extends Action<any>> = A extends Action<infer T> ? T : never;
type CreatedAction<A extends Action<any>> = [InitialState<A>, RunAction<A>, SetOnServer<InitialState<A>>];

export function useAction<A extends Action<any>> (context: LiveContext, initialState: InitialState<A>, actions: A): CreatedAction<A> {
    const { setState, getState, addDispatcher, deleteState, onUnMount, managerId } = context.setUpStateHook(initialState, 'useAction');

    async function performAction (type: keyof A, event: ServerEvent) {
        event.managerId = managerId;
        const action = actions[type];
        const state = getState(event.userId);

        if (!action) {
            return;
        }

        const newState = await action(event, state);

        if (newState) {
            setState(newState, event.userId);
        }
    }

    onUnMount(context.userId, () => deleteState(context.userId));

    const setOnServer = (event: HookContext, state: (InitialState<A> | ((state: InitialState<A>) => InitialState<A> | Promise<InitialState<A>>))) => {
        let newState: InitialState<A>;

        if (typeof state === 'function') {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            newState = state(getState(event.userId));
        } else {
            newState = state;
        }

        setState(newState, event.userId);
    };

    const runAction = Object.keys(actions).reduce((acc, key) => {
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        acc[key] = addDispatcher(key, (event) => performAction(key, event));

        return acc;
    }, {} as RunAction<A>);

    return [getState(context.userId), runAction, setOnServer];
}

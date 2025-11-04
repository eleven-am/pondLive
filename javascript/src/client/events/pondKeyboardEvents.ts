import { HandlerFunction } from './handler';

const keyDownEventHandler = (handlerFunction: HandlerFunction<KeyboardEvent>) => {
    handlerFunction('[pond-keydown]', 'keydown', (event) => {
        const input = event.target as HTMLInputElement;

        return {
            value: input.value,
        };
    });
};

const keyUpEventHandler = (handlerFunction: HandlerFunction<KeyboardEvent>) => {
    handlerFunction('[pond-keyup]', 'keyup', (event) => {
        const input = event.target as HTMLInputElement;

        return {
            value: input.value,
        };
    });
};

const keyPressEventHandler = (handlerFunction: HandlerFunction<KeyboardEvent>) => {
    handlerFunction('[pond-keypress]', 'keypress', (event) => {
        const input = event.target as HTMLInputElement;

        return {
            value: input.value,
        };
    });
};

export const pondKeyboardEvents = (handlerFunction: HandlerFunction<KeyboardEvent>) => {
    keyDownEventHandler(handlerFunction);
    keyUpEventHandler(handlerFunction);
    keyPressEventHandler(handlerFunction);
};

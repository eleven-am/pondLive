import { Context } from './context';
import { Component } from './liveContext';
import { Manager } from './manager';
import { html } from '../parser/parser';
import { Request } from '../wrappers/request';
import { Response } from '../wrappers/response';
import { ServerEvent } from '../wrappers/serverEvent';

const createEvent = (userId: string, context: Context, action = 'upgrade', address = '/example') => {
    const client = {
        broadcastMessage: jest.fn(),
        banUser: jest.fn(),
    } as any;

    const event = new ServerEvent(userId, client, context, {
        address: `http://localhost:3000${address}`,
        dataId: null,
        value: null,
        action,
    });

    return {
        event,
        client,
    };
};

const createMocks = (options: any, manager: Manager, userId = 'user123') => {
    const request = {
        method: 'GET',
        url: '/',
        ...options,
    };

    const response = {
        ...options,
    };

    const req = Request.fromRequest(request, manager, userId);
    const res = new Response(response as any);

    return {
        req,
        res,
    };
};

describe('Context', () => {
    let context: Context;
    const ExampleComponent: Component = () => html`<h1>Hello World</h1>`;

    beforeEach(() => {
        context = new Context();
    });

    describe('addEntryPoint', () => {
        test('should initialize a new route manager', () => {
            const route = {
                path: '/example',
                component: ExampleComponent,
            };
            const manager = context.addEntryPoint(route);

            expect(manager.path).toBe('/example');
            expect(manager.component).toBe(ExampleComponent);
        });

        test('should throw an error if the route already exists', () => {
            const route = {
                path: '/example',
                component: ExampleComponent,
            };

            const manager = context.addEntryPoint(route);

            expect(context.addEntryPoint(route)).toBe(manager);
        });
    });

    describe('performAction', () => {
        test('should perform an action on all managers', async () => {
            const event = createEvent('user123', context, 'action2').event;
            const route = {
                path: '/example',
                component: ExampleComponent,
            };

            route.path = '/another';

            const action1 = jest.fn();
            const action2 = jest.fn();

            context.upSertHook('action1', action1);
            context.upSertHook('action2', action2);
            await context.performAction(event);

            expect(action1).not.toHaveBeenCalledWith(event);
            expect(action2).toHaveBeenCalledWith(event);
        });
    });
});


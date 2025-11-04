import { Route, Component } from '../context/liveContext';
import { html } from '../parser/parser';

export function useRouter (routes: Route[]): Component {
    return (context) => {
        if (!context.isBuilt) {
            routes.forEach((route) => context.getContext(route));

            return html``;
        }

        const newContext = routes.map((route) => context.getContext(route))
            .find((context) => context.canRender(context.address));

        if (!newContext) {
            return html``;
        }

        const data = newContext.manager.component(newContext);

        newContext.styles.forEach((style) => context.addStyle(style));

        return data;
    };
}

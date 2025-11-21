import { DOMActionEffect, ClientNode } from './types';
import { Logger } from './logger';

export class DOMActionExecutor {
    constructor(private refs: Map<string, ClientNode>) { }

    execute(effects: DOMActionEffect[]) {
        if (!effects || effects.length === 0) return;

        for (const effect of effects) {
            this.executeOne(effect);
        }
    }

    private executeOne(effect: DOMActionEffect) {
        const node = this.refs.get(effect.ref);
        if (!node || !node.el) {
            Logger.warn('DOMAction', 'Ref not found', { ref: effect.ref });
            return;
        }

        const el = node.el as any;

        try {
            switch (effect.kind) {
                case 'dom.call':
                    if (effect.method && typeof el[effect.method] === 'function') {
                        el[effect.method](...(effect.args || []));
                    } else {
                        Logger.warn('DOMAction', 'Method not found', { method: effect.method });
                    }
                    break;

                case 'dom.set':
                    if (effect.prop) {
                        el[effect.prop] = effect.value;
                    }
                    break;

                case 'dom.toggle':
                    if (effect.prop) {
                        el[effect.prop] = !el[effect.prop];
                    }
                    break;

                case 'dom.class':
                    if (effect.class) {
                        if (effect.on === true) {
                            el.classList.add(effect.class);
                        } else if (effect.on === false) {
                            el.classList.remove(effect.class);
                        } else {
                            el.classList.toggle(effect.class);
                        }
                    }
                    break;

                case 'dom.scroll':
                    if (el.scrollIntoView) {
                        const opts: ScrollIntoViewOptions = {};
                        if (effect.behavior) opts.behavior = effect.behavior as ScrollBehavior;
                        if (effect.block) opts.block = effect.block as ScrollLogicalPosition;
                        if (effect.inline) opts.inline = effect.inline as ScrollLogicalPosition;
                        el.scrollIntoView(opts);
                    }
                    break;

                default:
                    Logger.warn('DOMAction', 'Unknown action kind', { kind: effect.kind });
            }
        } catch (e) {
            Logger.error('DOMAction', 'Execution failed', e);
        }
    }
}

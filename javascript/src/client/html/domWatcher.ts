type Watcher = (element: HTMLElement, oldValue: string | null, currentValue: string | null) => void;

type TagName = keyof HTMLElementTagNameMap;

type ModificationWatcher = {
    onAdd?: Watcher;
    onUpdated?: Watcher;
    onRemove?: Watcher;
}

export class DomWatcher {
    private readonly _observer: MutationObserver;

    private readonly _modifiers: {[p: string]: ModificationWatcher[]} = {};

    constructor () {
        const observer = new MutationObserver((mutations) => {
            for (let i = 0; i < mutations.length; i++) {
                const mutation = mutations[i];

                switch (mutation.type) {
                    case 'childList':
                        for (let j = 0; j < mutation.removedNodes.length; j++) {
                            const node = mutation.removedNodes[j];

                            if (node instanceof HTMLElement) {
                                const element = node as HTMLElement;

                                this.#removeElement(element);
                            }
                        }

                        for (let j = 0; j < mutation.addedNodes.length; j++) {
                            const node = mutation.addedNodes[j];

                            if (node instanceof HTMLElement) {
                                const element = node as HTMLElement;

                                this.#addNewElement(element);
                            }
                        }
                        break;

                    case 'attributes':
                        const element = mutation.target as HTMLElement;
                        const name = `[${mutation.attributeName}]`;

                        for (const selector in this._modifiers) {
                            if (name === selector) {
                                if (mutation.oldValue === null) {
                                    const modifier = this._modifiers[selector];
                                    const current = element.getAttribute(mutation.attributeName!);

                                    modifier.forEach((object) => {
                                        if (object.onAdd) {
                                            object.onAdd(element, null, current);
                                        }
                                    });
                                } else {
                                    const modifier = this._modifiers[selector];
                                    const valuePresent = element.getAttribute(mutation.attributeName!);

                                    modifier.forEach((object) => {
                                        if (valuePresent && object.onUpdated) {
                                            object.onUpdated(element, mutation.oldValue, valuePresent);
                                        } else if (!valuePresent && object.onRemove) {
                                            object.onRemove(element, mutation.oldValue, null);
                                        }
                                    });
                                }
                            }
                        }
                        break;

                    default:
                        break;
                }
            }
        });

        observer.observe(document.body, {
            childList: true,
            subtree: true,
            attributes: true,
            attributeOldValue: true,
        });
        this._observer = observer;
    }

    public watch (selector: string, object: ModificationWatcher): void {
        const elements = document.querySelectorAll(selector);

        if (object.onAdd) {
            for (let i = 0; i < elements.length; i++) {
                const element = elements[i] as HTMLElement;
                const attribute = selector.replace(/[\[\].#]/g, '');
                const value = element.getAttribute(attribute);

                object.onAdd(element, null, value);
            }
        }

        const modifier = this._modifiers[selector] || [];

        modifier.push(object);
        this._modifiers[selector] = modifier;
    }

    public addEventListener<S extends Event> (selector: string, event: string, callback: (node: HTMLElement, event: S) => void): void {
        this.watch(selector, {
            onAdd: (element) => {
                element.addEventListener(event, (event) => {
                    callback(element, event as S);
                }, { capture: true });
            },
            onRemove: (element) => {
                element.removeEventListener(event, (event) => {
                    callback(element, event as S);
                }, { capture: true });
            },
        });
    }

    public delegateEvent<S extends Event> (tagName: TagName, event: string, callback: (node: HTMLElement, event: S) => void): void {
        document.body.addEventListener(event, (event) => {
            const element = event.target as HTMLElement;

            if (element.tagName.toLowerCase() === tagName.toLowerCase()) {
                return callback(element, event as S);
            }

            const parent = element.parentElement as HTMLElement;

            if (parent && parent.tagName.toLowerCase() === tagName.toLowerCase()) {
                return callback(parent, event as S);
            }

            const firstChild = element.firstElementChild as HTMLElement;

            if (firstChild && firstChild.tagName.toLowerCase() === tagName.toLowerCase()) {
                return callback(firstChild, event as S);
            }
        });
    }

    public delegateSelectorEvent<S extends Event> (selector: string, event: string, callback: (node: HTMLElement, event: S) => void) {
        document.body.addEventListener(event, (event) => {
            const element = event.target as HTMLElement;
            const closest = element.closest(selector) as HTMLElement;

            if (closest) {
                return callback(closest, event as S);
            }
        });
    }

    public onAttributeChange (selector: string, callback: (node: HTMLElement, oldValue: string | null, currentValue: string | null) => void): void {
        this.watch(selector, {
            onAdd: callback,
            onUpdated: callback,
            onRemove: callback,
        });
    }

    #addNewElement (node: HTMLElement): void {
        for (const selector in this._modifiers) {
            if (node.matches(selector)) {
                const modifier = this._modifiers[selector];

                modifier.forEach((object) => {
                    const attribute = selector.replace(/[\[\].#]/g, '');
                    const value = node.getAttribute(attribute);

                    if (object.onAdd) {
                        object.onAdd(node, null, value);
                    }
                });
            }
        }

        for (let i = 0; i < node.children.length; i++) {
            this.#addNewElement(node.children[i] as HTMLElement);
        }
    }

    #removeElement (node: HTMLElement): void {
        for (const selector in this._modifiers) {
            if (node.matches(selector)) {
                const modifier = this._modifiers[selector];

                modifier.forEach((object) => {
                    const attribute = selector.replace(/[\[\].#]/g, '');
                    const value = node.getAttribute(attribute);

                    if (object.onRemove) {
                        object.onRemove(node, value, null);
                    }
                });
            }
        }

        for (let i = 0; i < node.children.length; i++) {
            this.#removeElement(node.children[i] as HTMLElement);
        }
    }
}

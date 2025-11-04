import { emitEvent } from '../events/eventEmmiter';
import { DomWatcher } from '../html/domWatcher';

const pondValue = (watcher: DomWatcher) => {
    watcher.onAttributeChange('[pond-value]', (element, _, currentValue) => {
        const input = element as HTMLInputElement;

        input.value = currentValue || '';
    });
};

const pondChecked = (watcher: DomWatcher) => {
    watcher.onAttributeChange('[pond-checked]', (element, _, currentValue) => {
        const input = element as HTMLInputElement;

        input.checked = currentValue === 'true';
    });
};

const pondDisabled = (watcher: DomWatcher) => {
    watcher.onAttributeChange('[pond-disabled]', (element, _, currentValue) => {
        const input = element as HTMLInputElement;

        input.disabled = currentValue === 'true';
    });
};

const pondHidden = (watcher: DomWatcher) => {
    watcher.onAttributeChange('[pond-hidden]', (element, _, currentValue) => {
        const input = element as HTMLInputElement;

        input.hidden = currentValue === 'true';
    });
};

const pondReadonly = (watcher: DomWatcher) => {
    watcher.onAttributeChange('[pond-readonly]', (element, _, currentValue) => {
        const input = element as HTMLInputElement;

        input.readOnly = currentValue === 'true';
    });
};

const pondRequired = (watcher: DomWatcher) => {
    watcher.onAttributeChange('[pond-required]', (element, _, currentValue) => {
        const input = element as HTMLInputElement;

        input.required = currentValue === 'true';
    });
};

const pondFocused = (watcher: DomWatcher) => {
    watcher.onAttributeChange('[pond-focused]', (element, _, currentValue) => {
        const input = element as HTMLInputElement;

        if (currentValue === 'true') {
            input.focus();
        } else {
            input.blur();
        }
    });
};

const pondCopy = (watcher: DomWatcher) => {
    watcher.onAttributeChange('[pond-copy]', (element, _, currentValue) => {
        const input = element as HTMLInputElement;

        input.value = currentValue || '';
        navigator.clipboard.writeText(input.value)
            .catch(console.error);
    });
};

const pondEmit = (watcher: DomWatcher) => {
    watcher.onAttributeChange('[pond-emit]', (element, _, currentValue) => {
        try {
            const json = JSON.parse(currentValue || '{}');

            emitEvent('pond-emit', json, element);
        } catch (e) {
            emitEvent('pond-emit', currentValue, element);
        }
    });
};

export const initPondActors = (watcher: DomWatcher) => {
    pondValue(watcher);
    pondChecked(watcher);
    pondDisabled(watcher);
    pondHidden(watcher);
    pondReadonly(watcher);
    pondRequired(watcher);
    pondFocused(watcher);
    pondEmit(watcher);
    pondCopy(watcher);
};

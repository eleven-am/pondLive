import type { ComputedField, FileWire, TouchWire } from './wire';

type Any = any;

const COMPUTED: Record<ComputedField, (e: Any, t: Any) => unknown> = {
    value: (_, t) => (t.isContentEditable ? t.innerText : t.value),
    selected: (_, t) => t.selectedOptions && Array.from(t.selectedOptions, (o: Any) => o.value),
    files: (_, t) => t.files && Array.from(t.files, (f: Any): FileWire => ({ n: f.name, s: f.size, t: f.type, m: f.lastModified })),
    touches: (e) => e.changedTouches && Array.from(e.changedTouches, (x: Any): TouchWire => [x.identifier, x.clientX, x.clientY]),
    text: (e) => (e.clipboardData || e.dataTransfer)?.getData('text/plain'),
    fullscreen: (e) => document.fullscreenElement === e.target,
    returnValue: (_, t) => t.returnValue,
};

export function readFields(e: Event, names: string): Record<string, unknown> {
    const out: Record<string, unknown> = {};
    const t: Any = e.target;
    for (const n of names.split(' ')) {
        const c = COMPUTED[n as ComputedField];
        const v = c ? t && c(e, t) : n in e ? (e as Any)[n] : t?.[n];
        if (v != null) out[n] = v;
    }
    return out;
}

export function domEvent(el: Element, name: string): string {
    const t: Any = el;
    const live = t.isContentEditable || (t.value !== undefined && !/^(checkbox|radio|file|select)/.test(t.type));
    return name === '~change' ? (live ? 'input' : 'change') : name;
}

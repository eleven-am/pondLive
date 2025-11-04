import { differ, getChanges, mergeObjects } from './differ';

const ENTITIES: {
    [key: string]: string;
} = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    '\'': '&#39;',
    '/': '&#x2F;',
    '`': '&#x60;',
    '=': '&#x3D;',
};

const ENT_REGEX = new RegExp(Object.keys(ENTITIES).join('|'), 'g');

type StaticParsed = Record<string, string[]>;
type DynamicParsed = Record<string, unknown>;

export type ParsedHTML = StaticParsed | DynamicParsed;

export class Html {
    private readonly statics: readonly string[];

    private readonly dynamics: unknown[];

    constructor (statics: readonly string[], dynamics: unknown[]) {
        this.statics = statics;
        this.dynamics = dynamics;
    }

    static parse (parts: ParsedHTML): Html {
        const data = parts as StaticParsed;
        const statics = data.s || [''];

        delete data.s;
        const dynamics = Object.values(data).filter((d) => d !== undefined && d !== null);

        return new Html(statics, dynamics);
    }

    public toString () {
        let result = this.statics[0];

        for (let i = 0; i < this.dynamics.length; i++) {
            if (this.dynamics[i] instanceof Html) {
                result += (this.dynamics[i] as Html).toString() + this.statics[i + 1];
            } else if (Array.isArray(this.dynamics[i])) {
                // eslint-disable-next-line @typescript-eslint/no-use-before-define
                result += join(this.dynamics[i] as any[], '').toString() + this.statics[i + 1];
            } else if (typeof this.dynamics[i] === 'object') {
                result += this.parsedHtmlToString(this.dynamics[i] as ParsedHTML) + this.statics[i + 1];
            } else {
                // eslint-disable-next-line @typescript-eslint/no-use-before-define
                result += escapeHTML(this.dynamics[i]) + this.statics[i + 1];
            }
        }

        return result;
    }

    public getParts (): ParsedHTML {
        const result: ParsedHTML = {
            s: this.statics,
        };

        for (let i = 0; i < this.dynamics.length; i++) {
            if (this.dynamics[i] instanceof Html) {
                result[i] = (this.dynamics[i] as Html).getParts();
            } else if (Array.isArray(this.dynamics[i])) {
                // eslint-disable-next-line @typescript-eslint/no-use-before-define
                result[i] = join(this.dynamics[i] as any[], '').getParts();
            } else if (this.dynamics[i] === undefined) {
                result[i] = null;
            } else {
                result[i] = this.dynamics[i];
            }
        }

        return result;
    }

    public parsedHtmlToString (parsed: ParsedHTML): string {
        const data = parsed as StaticParsed;
        let result = '';

        if (Array.isArray(data)) {
            // eslint-disable-next-line @typescript-eslint/no-use-before-define
            return join(data, '').toString();
        }
        if (data?.s?.length > 0) {
            const stat = data.s.filter((s) => s !== undefined && s !== null);

            result = data.s[0];
            for (let i = 0; i < stat.length - 1; i++) {
                if (typeof data[i] === 'object') {
                    result += this.parsedHtmlToString(parsed[i] as ParsedHTML) + data.s[i + 1];
                } else {
                    // eslint-disable-next-line @typescript-eslint/no-use-before-define
                    result += escapeHTML(parsed[i]) + data.s[i + 1];
                }
            }
        }

        return result;
    }

    public differentiate (parsed: Html): Record<string, any> {
        const newParsed = parsed.getParts();
        const oldParsed = this.getParts();

        const mapped = differ(oldParsed, newParsed);

        return getChanges(mapped);
    }

    public patch (changes: Record<string, any>): Html {
        const data = mergeObjects(this.getParts(), changes);

        return Html.parse(data);
    }
}

export function html (statics: TemplateStringsArray, ...dynamics: unknown[]) {
    return new Html(statics, dynamics);
}

export function join (array: (string | Html)[], separator: string | Html) {
    if (array.length <= 0) {
        return new Html([''], []);
    }

    return new Html(['', ...Array(array.length - 1).fill(separator), ''], array);
}

function escapeHTML (unsafe: unknown): string {
    if (unsafe === undefined || unsafe === null) {
        return '';
    }
    if (unsafe instanceof Html) {
        return unsafe.toString();
    }
    if (Array.isArray(unsafe)) {
        return join(unsafe, '').toString();
    }

    return String(unsafe).replace(ENT_REGEX, (char) => ENTITIES[char]);
}

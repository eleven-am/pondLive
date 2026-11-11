interface HTMLPolicy {
    createHTML(input: string): unknown;
}

let policy: HTMLPolicy | null | undefined;

export function toHTML(input: string): string {
    if (policy === undefined) {
        policy = null;
        const factory = (globalThis as { trustedTypes?: { createPolicy(name: string, rules: { createHTML(s: string): string }): HTMLPolicy } }).trustedTypes;
        if (factory) {
            try {
                policy = factory.createPolicy('pondlive', { createHTML: (s) => s });
            } catch {
                policy = null;
            }
        }
    }
    return (policy ? policy.createHTML(input) : input) as string;
}

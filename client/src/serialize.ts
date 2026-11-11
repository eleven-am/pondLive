export function serializeValue(value: unknown): unknown {
    if (value instanceof Node) return undefined;
    try {
        return JSON.parse(JSON.stringify(value ?? null));
    } catch {
        return undefined;
    }
}

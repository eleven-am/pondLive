interface ExtractOptions {
  refElement?: Element;
}

export function extractEventDetail(event: Event, props?: string[] | null, options?: ExtractOptions): Record<string, unknown> | undefined {
  if (!Array.isArray(props) || props.length === 0) {
    return undefined;
  }
  const detail: Record<string, unknown> = {};
  props.forEach((path) => {
    if (typeof path !== 'string' || path.length === 0) {
      return;
    }
    const value = resolvePath(path, event, options);
    if (value !== undefined) {
      detail[path] = value;
    }
  });
  return Object.keys(detail).length > 0 ? detail : undefined;
}

function resolvePath(path: string, event: Event, options?: ExtractOptions): unknown {
  const segments = path.split('.').map((segment) => segment.trim()).filter(Boolean);
  if (segments.length === 0) {
    return undefined;
  }
  const root = segments.shift()!;
  let current: any;
  switch (root) {
    case 'event':
      current = event;
      break;
    case 'target':
      current = event.target ?? null;
      break;
    case 'currentTarget':
      current = event.currentTarget ?? null;
      break;
    case 'element':
    case 'ref':
      current = options?.refElement ?? (event.currentTarget instanceof Element ? event.currentTarget : null);
      break;
    default:
      return undefined;
  }
  for (const segment of segments) {
    if (current == null) {
      return undefined;
    }
    try {
      current = current[segment as keyof typeof current];
    } catch {
      return undefined;
    }
  }
  return serializeValue(current);
}

function serializeValue(value: unknown): unknown {
  if (value === null || value === undefined) {
    return null;
  }
  const type = typeof value;
  if (type === 'string' || type === 'number' || type === 'boolean') {
    return value;
  }
  if (Array.isArray(value)) {
    const mapped = value.map(serializeValue).filter((entry) => entry !== undefined);
    return mapped.length > 0 ? mapped : null;
  }
  if (value instanceof Date) {
    return value.toISOString();
  }
  if (value instanceof DOMTokenList) {
    return Array.from(value);
  }
  if (value instanceof Node) {
    return undefined;
  }
  try {
    return JSON.parse(JSON.stringify(value));
  } catch {
    return undefined;
  }
}

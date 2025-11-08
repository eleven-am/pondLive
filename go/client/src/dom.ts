import { getRefElement } from "./refs";

export interface DOMGetContext {
  event?: Event | null;
  target?: Element | null;
  handlerElement?: Element | null;
  refElement?: Element | null;
}

function pickSource(scope: string, ctx: DOMGetContext): any {
  switch (scope) {
    case "event":
      return ctx.event ?? undefined;
    case "target":
      return ctx.target ?? undefined;
    case "currentTarget":
      if (ctx.handlerElement) {
        return ctx.handlerElement;
      }
      return ctx.event && "currentTarget" in ctx.event
        ? (ctx.event as Event).currentTarget ?? undefined
        : undefined;
    case "element":
    case "ref":
      return ctx.refElement ?? ctx.handlerElement ?? ctx.target ?? undefined;
    default:
      return undefined;
  }
}

function resolvePath(source: any, parts: string[]): any {
  let value = source;
  for (const part of parts) {
    if (!part) {
      continue;
    }
    if (value == null) {
      return undefined;
    }
    try {
      value = value[part as keyof typeof value];
    } catch (_error) {
      return undefined;
    }
  }
  return value;
}

export function resolvePropertySelector(
  selector: string,
  ctx: DOMGetContext,
): any {
  if (typeof selector !== "string") {
    return undefined;
  }
  const trimmed = selector.trim();
  if (!trimmed) {
    return undefined;
  }
  const parts = trimmed.split(".");
  if (parts.length === 0) {
    return undefined;
  }
  const scope = parts.shift();
  if (!scope) {
    return undefined;
  }
  const source = pickSource(scope, ctx);
  if (source === undefined) {
    return undefined;
  }
  return resolvePath(source, parts);
}

export function domGetSync(
  selectors: string[] | null | undefined,
  ctx: DOMGetContext,
): Record<string, any> | null {
  if (!Array.isArray(selectors) || selectors.length === 0) {
    return null;
  }
  const result: Record<string, any> = {};
  for (const selector of selectors) {
    if (typeof selector !== "string" || selector.length === 0) {
      continue;
    }
    const raw = resolvePropertySelector(selector, ctx);
    if (raw === undefined) {
      continue;
    }
    const normalized = normalizePropertyValue(raw);
    if (normalized !== undefined) {
      result[selector] = normalized;
    }
  }
  return Object.keys(result).length > 0 ? result : null;
}

export function normalizePropertyValue(value: any): any {
  if (value === null) {
    return null;
  }
  const type = typeof value;
  if (type === "string" || type === "number" || type === "boolean") {
    return value;
  }
  if (value instanceof Date) {
    return value.toISOString();
  }
  if (typeof FileList !== "undefined" && value instanceof FileList) {
    return serializeFileList(value);
  }
  if (typeof File !== "undefined" && value instanceof File) {
    return { name: value.name, size: value.size, type: value.type };
  }
  if (typeof DOMTokenList !== "undefined" && value instanceof DOMTokenList) {
    return Array.from(value);
  }
  if (typeof TimeRanges !== "undefined" && value instanceof TimeRanges) {
    const ranges: Array<{ start: number; end: number }> = [];
    for (let i = 0; i < value.length; i++) {
      try {
        ranges.push({ start: value.start(i), end: value.end(i) });
      } catch (_error) {
        // Ignore invalid ranges
      }
    }
    return ranges;
  }
  if (Array.isArray(value)) {
    return value
      .map((item) => normalizePropertyValue(item))
      .filter((item) => item !== undefined);
  }
  if (value && typeof value === "object") {
    try {
      return JSON.parse(JSON.stringify(value));
    } catch (_error) {
      return undefined;
    }
  }
  return undefined;
}

function serializeFileList(list: FileList): Array<{
  name: string;
  size: number;
  type: string;
}> {
  const files: Array<{ name: string; size: number; type: string }> = [];
  for (let i = 0; i < list.length; i++) {
    const file = list.item(i);
    if (file) {
      files.push({ name: file.name, size: file.size, type: file.type });
    }
  }
  return files;
}

export interface DOMCallOptions {
  allowMissing?: boolean;
}

export function callElementMethod(
  refId: string,
  method: string,
  args: any[] = [],
  options?: DOMCallOptions,
): unknown {
  const element = refId ? getRefElement(refId) : null;
  if (!element) {
    if (!options?.allowMissing) {
      console.warn("[LiveUI] DOMCall target missing", { refId, method });
    }
    return undefined;
  }
  const fn: any = (element as any)[method];
  if (typeof fn !== "function") {
    console.warn("[LiveUI] DOMCall method missing", { refId, method });
    return undefined;
  }
  try {
    const result = fn.apply(element, Array.isArray(args) ? args : []);
    if (result && typeof (result as Promise<unknown>).then === "function") {
      (result as Promise<unknown>).catch((error) => {
        console.warn("[LiveUI] DOMCall promise rejected", {
          refId,
          method,
          error,
        });
      });
    }
    return result;
  } catch (error) {
    console.error("[LiveUI] DOMCall failed", { refId, method, error });
    return undefined;
  }
}

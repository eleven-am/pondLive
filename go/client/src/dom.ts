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

export interface DOMActionResult {
  ok: boolean;
  reason?: string;
  error?: unknown;
}

export function callElementMethod(
  refId: string,
  method: string,
  args: any[] = [],
  options?: DOMCallOptions,
): DOMActionResult {
  const element = refId ? getRefElement(refId) : null;
  if (!element) {
    if (!options?.allowMissing) {
      console.warn("[LiveUI] DOMCall target missing", { refId, method });
    }
    return { ok: false, reason: "missing_element" };
  }
  const fn: any = (element as any)[method];
  if (typeof fn !== "function") {
    console.warn("[LiveUI] DOMCall method missing", { refId, method });
    return { ok: false, reason: "missing_method" };
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
    return { ok: true };
  } catch (error) {
    console.error("[LiveUI] DOMCall failed", { refId, method, error });
    return { ok: false, reason: "invoke_failed", error };
  }
}

export function setElementProperty(
  refId: string,
  prop: string,
  value: unknown,
): DOMActionResult {
  const element = refId ? getRefElement(refId) : null;
  if (!element) {
    console.warn("[LiveUI] DOMSet target missing", { refId, prop });
    return { ok: false, reason: "missing_element" };
  }
  const name = typeof prop === "string" ? prop.trim() : "";
  if (!name) {
    return { ok: false, reason: "missing_property" };
  }
  try {
    (element as any)[name] = value;
    return { ok: true };
  } catch (error) {
    console.error("[LiveUI] DOMSet failed", { refId, prop: name, error });
    return { ok: false, reason: "set_failed", error };
  }
}

export function setBooleanProperty(
  refId: string,
  prop: string,
  value: boolean,
): DOMActionResult {
  if (typeof value !== "boolean") {
    return { ok: false, reason: "invalid_value" };
  }
  return setElementProperty(refId, prop, value);
}

export function toggleElementClass(
  refId: string,
  className: string,
  on?: boolean,
): DOMActionResult {
  const element = refId ? getRefElement(refId) : null;
  if (!element) {
    console.warn("[LiveUI] DOMClass target missing", { refId, className });
    return { ok: false, reason: "missing_element" };
  }
  const token = typeof className === "string" ? className.trim() : "";
  if (!token) {
    return { ok: false, reason: "missing_class" };
  }
  if (!("classList" in (element as Element))) {
    return { ok: false, reason: "classlist_missing" };
  }
  try {
    if (on === undefined) {
      (element as Element).classList.toggle(token);
    } else {
      (element as Element).classList.toggle(token, on);
    }
    return { ok: true };
  } catch (error) {
    console.error("[LiveUI] DOMClass failed", {
      refId,
      className: token,
      error,
    });
    return { ok: false, reason: "class_toggle_failed", error };
  }
}

export function scrollElementIntoView(
  refId: string,
  options?: ScrollIntoViewOptions | null,
): DOMActionResult {
  const element = refId ? getRefElement(refId) : null;
  if (!element) {
    console.warn("[LiveUI] DOMScroll target missing", { refId });
    return { ok: false, reason: "missing_element" };
  }
  const scrollFn = (element as Element).scrollIntoView;
  if (typeof scrollFn !== "function") {
    return { ok: false, reason: "missing_scroll" };
  }
  try {
    if (options && Object.keys(options).length > 0) {
      scrollFn.call(element, options);
    } else {
      scrollFn.call(element);
    }
    return { ok: true };
  } catch (error) {
    console.error("[LiveUI] DOMScroll failed", { refId, error });
    return { ok: false, reason: "scroll_failed", error };
  }
}

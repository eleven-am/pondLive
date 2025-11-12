import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  callElementMethod,
  domGetSync,
  scrollElementIntoView,
  setElementProperty,
  toggleElementClass,
} from "../src/dom";
import {
  attachRefToElement,
  clearRefs,
  getRefElement,
  registerRefs,
} from "../src/refs";
import LiveUI from "../src/index";

vi.mock("@eleven-am/pondsocket-client", () => ({
  PondClient: vi.fn().mockImplementation(() => ({
    createChannel: vi.fn().mockReturnValue({
      onJoin: vi.fn(),
      onMessage: vi.fn(),
      onLeave: vi.fn(),
      join: vi.fn(),
      leave: vi.fn(),
      sendMessage: vi.fn(),
    }),
    connect: vi.fn(),
    disconnect: vi.fn(),
  })),
}));

describe("dom utilities", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    clearRefs();
  });

  afterEach(() => {
    document.body.innerHTML = "";
    clearRefs();
  });

  it("invokes element methods via callElementMethod", () => {
    const button = document.createElement("button");
    document.body.appendChild(button);
    const handler = vi.fn();
    (button as any).focus = handler;

    registerRefs({ btn: { tag: "button" } });
    attachRefToElement("btn", button);

    expect(getRefElement("btn")).toBe(button);
    const result = callElementMethod("btn", "focus");
    expect(result.ok).toBe(true);
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it("sets element properties via setElementProperty", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);

    registerRefs({ field: { tag: "input" } });
    attachRefToElement("field", input);

    const result = setElementProperty("field", "value", "hello");
    expect(result.ok).toBe(true);
    expect(input.value).toBe("hello");
  });

  it("toggles classes via toggleElementClass", () => {
    const div = document.createElement("div");
    div.className = "foo";
    document.body.appendChild(div);

    registerRefs({ box: { tag: "div" } });
    attachRefToElement("box", div);

    const initial = toggleElementClass("box", "active", true);
    expect(initial.ok).toBe(true);
    expect(div.classList.contains("active")).toBe(true);

    const second = toggleElementClass("box", "active", false);
    expect(second.ok).toBe(true);
    expect(div.classList.contains("active")).toBe(false);
  });

  it("scrolls elements into view via scrollElementIntoView", () => {
    const element = document.createElement("div");
    document.body.appendChild(element);
    (element as any).scrollIntoView = vi.fn();

    registerRefs({ panel: { tag: "div" } });
    attachRefToElement("panel", element);

    const result = scrollElementIntoView("panel", { behavior: "smooth" });
    expect(result.ok).toBe(true);
    expect((element.scrollIntoView as any)).toHaveBeenCalledWith({ behavior: "smooth" });
  });

  it("collects selector values synchronously with domGetSync", () => {
    const div = document.createElement("div");
    div.dataset.value = "42";
    document.body.appendChild(div);

    const result = domGetSync(["element.dataset.value"], {
      event: null,
      target: div,
      handlerElement: div,
      refElement: div,
    });

    expect(result).toEqual({ "element.dataset.value": "42" });
  });

  it("responds to dom requests with captured values", () => {
    const div = document.createElement("div");
    div.id = "alpha";
    document.body.appendChild(div);
    registerRefs({ foo: { tag: "div" } });
    attachRefToElement("foo", div);

    const live = new LiveUI({ autoConnect: false });
    const sendMessage = vi.fn();
    (live as any).channel = { sendMessage };

    (live as any).handleDOMRequest({
      t: "domreq",
      id: "req-1",
      ref: "foo",
      props: ["element.id"],
    });

    expect(sendMessage).toHaveBeenCalledWith("domres", {
      t: "domres",
      id: "req-1",
      values: { "element.id": "alpha" },
    });
  });

  it("responds with error when ref is missing", () => {
    const live = new LiveUI({ autoConnect: false });
    const sendMessage = vi.fn();
    (live as any).channel = { sendMessage };

    (live as any).handleDOMRequest({
      t: "domreq",
      id: "req-2",
      ref: "missing",
      props: ["element.id"],
    });

    expect(sendMessage).toHaveBeenCalledWith("domres", {
      t: "domres",
      id: "req-2",
      error: "not_found",
    });
  });
});

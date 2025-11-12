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

  it("handles getBoundingClientRect method call", () => {
    const div = document.createElement("div");
    document.body.appendChild(div);
    registerRefs({ box: { tag: "div" } });
    attachRefToElement("box", div);

    const mockRect = {
      x: 10,
      y: 20,
      width: 100,
      height: 50,
      top: 20,
      left: 10,
      right: 110,
      bottom: 70,
      toJSON: () => mockRect,
    };
    div.getBoundingClientRect = vi.fn().mockReturnValue(mockRect);

    const live = new LiveUI({ autoConnect: false });
    const sendMessage = vi.fn();
    (live as any).channel = { sendMessage };

    (live as any).handleDOMRequest({
      t: "domreq",
      id: "req-3",
      ref: "box",
      method: "getBoundingClientRect",
    });

    expect(sendMessage).toHaveBeenCalledWith("domres", {
      t: "domres",
      id: "req-3",
      result: mockRect,
    });
  });

  it("handles getScrollMetrics custom method", () => {
    const div = document.createElement("div");
    Object.defineProperty(div, "scrollTop", { value: 100, writable: true });
    Object.defineProperty(div, "scrollLeft", { value: 50, writable: true });
    Object.defineProperty(div, "scrollHeight", { value: 500, writable: true });
    Object.defineProperty(div, "scrollWidth", { value: 300, writable: true });
    Object.defineProperty(div, "clientHeight", { value: 200, writable: true });
    Object.defineProperty(div, "clientWidth", { value: 150, writable: true });
    document.body.appendChild(div);
    registerRefs({ scroller: { tag: "div" } });
    attachRefToElement("scroller", div);

    const live = new LiveUI({ autoConnect: false });
    const sendMessage = vi.fn();
    (live as any).channel = { sendMessage };

    (live as any).handleDOMRequest({
      t: "domreq",
      id: "req-4",
      ref: "scroller",
      method: "getScrollMetrics",
    });

    expect(sendMessage).toHaveBeenCalledWith("domres", {
      t: "domres",
      id: "req-4",
      result: {
        scrollTop: 100,
        scrollLeft: 50,
        scrollHeight: 500,
        scrollWidth: 300,
        clientHeight: 200,
        clientWidth: 150,
      },
    });
  });

  it("handles getComputedStyle with specific properties", () => {
    const div = document.createElement("div");
    div.style.color = "red";
    div.style.fontSize = "16px";
    document.body.appendChild(div);
    registerRefs({ styled: { tag: "div" } });
    attachRefToElement("styled", div);

    const live = new LiveUI({ autoConnect: false });
    const sendMessage = vi.fn();
    (live as any).channel = { sendMessage };

    (live as any).handleDOMRequest({
      t: "domreq",
      id: "req-5",
      ref: "styled",
      method: "getComputedStyle",
      args: [["color", "font-size"]],
    });

    expect(sendMessage).toHaveBeenCalled();
    const call = sendMessage.mock.calls[0];
    expect(call[0]).toBe("domres");
    expect(call[1].t).toBe("domres");
    expect(call[1].id).toBe("req-5");
    expect(call[1].result).toHaveProperty("color");
    expect(call[1].result).toHaveProperty("font-size");
  });

  it("handles getComputedStyle with default properties", () => {
    const div = document.createElement("div");
    document.body.appendChild(div);
    registerRefs({ styled: { tag: "div" } });
    attachRefToElement("styled", div);

    const live = new LiveUI({ autoConnect: false });
    const sendMessage = vi.fn();
    (live as any).channel = { sendMessage };

    (live as any).handleDOMRequest({
      t: "domreq",
      id: "req-6",
      ref: "styled",
      method: "getComputedStyle",
    });

    expect(sendMessage).toHaveBeenCalled();
    const call = sendMessage.mock.calls[0];
    expect(call[0]).toBe("domres");
    expect(call[1].t).toBe("domres");
    expect(call[1].id).toBe("req-6");
    expect(call[1].result).toHaveProperty("display");
    expect(call[1].result).toHaveProperty("position");
    expect(call[1].result).toHaveProperty("color");
    expect(call[1].result).toHaveProperty("fontSize");
  });

  it("handles matches method call", () => {
    const div = document.createElement("div");
    div.className = "active featured";
    document.body.appendChild(div);
    registerRefs({ item: { tag: "div" } });
    attachRefToElement("item", div);

    const live = new LiveUI({ autoConnect: false });
    const sendMessage = vi.fn();
    (live as any).channel = { sendMessage };

    (live as any).handleDOMRequest({
      t: "domreq",
      id: "req-7",
      ref: "item",
      method: "matches",
      args: [".active"],
    });

    expect(sendMessage).toHaveBeenCalledWith("domres", {
      t: "domres",
      id: "req-7",
      result: true,
    });
  });

  it("handles checkVisibility method call", () => {
    const div = document.createElement("div");
    document.body.appendChild(div);
    registerRefs({ visible: { tag: "div" } });
    attachRefToElement("visible", div);

    // Mock checkVisibility if not available in test environment
    if (typeof (div as any).checkVisibility !== "function") {
      (div as any).checkVisibility = vi.fn().mockReturnValue(true);
    }

    const live = new LiveUI({ autoConnect: false });
    const sendMessage = vi.fn();
    (live as any).channel = { sendMessage };

    (live as any).handleDOMRequest({
      t: "domreq",
      id: "req-8",
      ref: "visible",
      method: "checkVisibility",
    });

    expect(sendMessage).toHaveBeenCalled();
    const call = sendMessage.mock.calls[0];
    expect(call[0]).toBe("domres");
    expect(call[1].t).toBe("domres");
    expect(call[1].id).toBe("req-8");
    expect(typeof call[1].result).toBe("boolean");
  });
});

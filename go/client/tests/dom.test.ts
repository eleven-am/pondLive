import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { callElementMethod, domGetSync } from "../src/dom";
import {
  clearRefs,
  registerRefs,
  bindRefsInTree,
  getRefElement,
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
    document.body.innerHTML = '<button data-live-ref="btn"></button>';
    const button = document.querySelector("[data-live-ref]") as HTMLButtonElement;
    const handler = vi.fn();
    (button as any).focus = handler;

    registerRefs({ btn: { tag: "button" } });
    bindRefsInTree(document.body);

    expect(getRefElement("btn")).toBe(button);
    callElementMethod("btn", "focus");
    expect(handler).toHaveBeenCalledTimes(1);
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
    document.body.innerHTML = '<div data-live-ref="foo" id="alpha"></div>';
    registerRefs({ foo: { tag: "div" } });
    bindRefsInTree(document.body);

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

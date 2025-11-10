import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  registerSlot,
  getSlot,
  unregisterSlot,
  reset,
  initLists,
  ensureList,
  registerList,
  getRow,
  setRow,
  deleteRow,
  unregisterList,
} from "../src/dom-index";
import * as events from "../src/events";

function createListContainer(slot: number, rows: string[] = []): HTMLElement {
  const container = document.createElement("div");
  container.setAttribute("data-list-slot", String(slot));
  for (const key of rows) {
    const row = document.createElement("div");
    row.setAttribute("data-row-key", key);
    container.appendChild(row);
  }
  document.body.appendChild(container);
  return container;
}

describe("dom index", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    reset();
  });

  it("registers and retrieves slot nodes", () => {
    const spy = vi.spyOn(events, "onSlotRegistered");
    const node = document.createElement("div");

    registerSlot(7, node);

    expect(getSlot(7)).toBe(node);
    expect(spy).toHaveBeenCalledWith(7, node);
  });

  it("re-registers slots with new nodes and replays hooks", () => {
    const spy = vi.spyOn(events, "onSlotRegistered");
    const first = document.createElement("div");
    const second = document.createElement("div");

    registerSlot(4, first);
    registerSlot(4, second);

    expect(getSlot(4)).toBe(second);
    expect(spy).toHaveBeenNthCalledWith(1, 4, first);
    expect(spy).toHaveBeenNthCalledWith(2, 4, second);
  });

  it("ignores empty slot registration", () => {
    const spy = vi.spyOn(events, "onSlotRegistered");

    registerSlot(9, null as unknown as Node);

    expect(getSlot(9)).toBeNull();
    expect(spy).not.toHaveBeenCalled();
  });

  it("unregisters slots and notifies events", () => {
    const registerSpy = vi.spyOn(events, "onSlotRegistered");
    const unregisterSpy = vi.spyOn(events, "onSlotUnregistered");
    const node = document.createElement("div");

    registerSlot(3, node);
    expect(registerSpy).toHaveBeenCalledWith(3, node);

    unregisterSlot(3);

    expect(unregisterSpy).toHaveBeenCalledWith(3);
    expect(getSlot(3)).toBeNull();
  });

  it("resets all slots and lists", () => {
    const unregisterSpy = vi.spyOn(events, "onSlotUnregistered");
    const nodeA = document.createElement("div");
    const nodeB = document.createElement("div");

    registerSlot(1, nodeA);
    registerSlot(2, nodeB);
    createListContainer(5, ["first"]);
    initLists([5]);

    reset();
    document.body.innerHTML = "";

    expect(unregisterSpy).toHaveBeenCalledTimes(2);
    expect(getSlot(1)).toBeNull();
    expect(getSlot(2)).toBeNull();
    expect(() => ensureList(5)).toThrowError(/list slot 5/);
  });

  it("throws when ensuring a missing list", () => {
    expect(() => ensureList(404)).toThrowError(/list slot 404/);
  });

  it("initializes list containers and tracks rows", () => {
    createListContainer(6, ["alpha", "beta"]);

    initLists([6]);

    const alpha = getRow(6, "alpha");
    const beta = getRow(6, "beta");

    expect(alpha).toBeInstanceOf(Element);
    expect(beta).toBeInstanceOf(Element);
    expect(alpha?.getAttribute("data-row-key")).toBe("alpha");
    expect(beta?.getAttribute("data-row-key")).toBe("beta");
  });

  it("prefers provided row maps when registering lists", () => {
    const container = createListContainer(8);
    const rows = new Map<string, Element>();
    const sentinel = document.createElement("span");
    rows.set("sentinel", sentinel);

    registerList(8, container, rows);

    expect(getRow(8, "sentinel")).toBe(sentinel);
  });

  it("initLists does not overwrite custom row bookkeeping", () => {
    const container = createListContainer(13, ["dom"]);
    const rows = new Map<string, Element>();
    const sentinel = document.createElement("span");
    rows.set("preserve", sentinel);

    registerList(13, container, rows);

    initLists([13]);

    expect(getRow(13, "preserve")).toBe(sentinel);
    expect(getRow(13, "dom")).toBeNull();
  });

  it("updates row bookkeeping for set and delete operations", () => {
    const container = createListContainer(11);

    registerList(11, container);

    const row = document.createElement("div");
    row.setAttribute("data-row-key", "fresh");

    setRow(11, "fresh", row);
    expect(getRow(11, "fresh")).toBe(row);

    deleteRow(11, "fresh");
    expect(getRow(11, "fresh")).toBeNull();
  });

  it("rebuilds list records after unregistering", () => {
    createListContainer(12, ["first"]);
    initLists([12]);

    unregisterList(12);

    document.body.innerHTML = "";
    const refreshed = createListContainer(12, ["second"]);

    expect(getRow(12, "second")).toBe(refreshed.querySelector('[data-row-key="second"]'));
  });
});

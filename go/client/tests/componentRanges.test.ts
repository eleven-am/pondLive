import { beforeEach, describe, expect, it } from "vitest";
import { applyComponentRanges, resolveSlotAnchors } from "../src/manifest";
import { getComponentRange, registerComponentRange, resetComponentRanges } from "../src/componentRanges";
import type { ComponentPathDescriptor, SlotPathDescriptor } from "../src/types";

describe("component range registration", () => {
  beforeEach(() => {
    document.body.replaceChildren();
    resetComponentRanges();
  });

  it("registers component ranges and resolves slot anchors", () => {
    const host = document.createElement("div");
    const textNode = document.createTextNode("value");
    host.appendChild(textNode);
    document.body.append(host);

    const descriptors: ComponentPathDescriptor[] = [
      { componentId: "comp", firstChild: [0], lastChild: [0] },
    ];

    applyComponentRanges(descriptors, { root: document });

    const range = getComponentRange("comp");
    expect(range).not.toBeNull();
    expect(range?.container).toBe(document.body);
    expect(range?.startIndex).toBe(0);
    expect(range?.endIndex).toBe(0);

    const slots: SlotPathDescriptor[] = [
      { slot: 1, componentId: "comp", elementPath: [], textChildIndex: 0 },
    ];
    const anchors = resolveSlotAnchors(slots);
    expect(anchors.get(1)).toBe(textNode);
  });

  it("normalizes empty component registrations", () => {
    const host = document.createElement("div");
    document.body.append(host);

    const normalized = registerComponentRange("empty", {
      container: document.body,
      startIndex: 5,
      endIndex: 1,
    });

    expect(normalized).not.toBeNull();
    expect(normalized?.startIndex).toBe(5);
    expect(normalized?.endIndex).toBe(4);
    expect(getComponentRange("empty")).toEqual(normalized);
  });
});

import { beforeEach, describe, expect, it } from "vitest";
import {
  getComponentBounds,
  initializeComponentMarkers,
} from "../src/componentMarkers";

describe('component marker indexing', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });

  it('captures component boundaries and clears markers', () => {
    document.body.innerHTML = '<div id="inner"></div>';

    initializeComponentMarkers(
      {
        c1: { start: 0, end: 1 },
      },
      document.body,
    );

    const bounds = getComponentBounds('c1');
    expect(bounds).not.toBeNull();
    expect(bounds?.container).toBe(document.body);
    expect(bounds?.start).toBe(0);
    expect(bounds?.end).toBe(1);

    initializeComponentMarkers(null, document.body);
    expect(getComponentBounds('c1')).toBeNull();
  });
});

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
    document.body.innerHTML = `
      <!---->
      <div id="inner"></div>
      <!---->
    `;

    initializeComponentMarkers({
      c1: { start: 0, end: 1 },
    }, document);

    const bounds = getComponentBounds('c1');
    expect(bounds).not.toBeNull();
    expect(bounds?.start.data).toBe('');
    expect(bounds?.end.data).toBe('');
    expect(bounds?.start.isConnected).toBe(true);
    expect(bounds?.end.isConnected).toBe(true);
  });
});

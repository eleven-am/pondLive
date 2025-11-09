import { beforeEach, describe, expect, it } from "vitest";
import {
  getComponentBounds,
  initializeComponentMarkers,
  registerComponentMarkers,
} from "../src/componentMarkers";

describe('component marker indexing', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });

  it('captures component boundaries and clears markers', () => {
    document.body.innerHTML = '<div id="outer"><div id="inner"></div></div>';

    initializeComponentMarkers(
      {
        c1: { parentPath: [0], start: 0, end: 1 },
      },
      document.body,
    );

    const bounds = getComponentBounds('c1');
    expect(bounds).not.toBeNull();
    expect(bounds?.start.data).toBe('');
    expect(bounds?.end.data).toBe('');
    expect(bounds?.start.isConnected).toBe(true);
    expect(bounds?.end.isConnected).toBe(true);
    const parent = document.getElementById('inner')?.parentNode;
    expect(parent?.childNodes.item(0)).toBe(bounds?.start);
    expect(parent?.childNodes.item(2)).toBe(bounds?.end);
  });

  it('ignores existing marker comments when locating children', () => {
    const host = document.createElement('div');
    document.body.appendChild(host);
    host.innerHTML = '<div id="first"></div><div id="second"></div>';

    initializeComponentMarkers(
      {
        first: { start: 0, end: 1 },
      },
      host,
    );

    registerComponentMarkers(
      {
        second: { start: 1, end: 2 },
      },
      host,
    );

    const bounds = getComponentBounds('second');
    expect(bounds).not.toBeNull();
    expect(bounds?.start.isConnected).toBe(true);
    expect(bounds?.end.isConnected).toBe(true);
    const second = host.querySelector('#second');
    expect(bounds?.start.nextSibling).toBe(second);
  });

  it('updates component markers after subtree replacement', () => {
    document.body.innerHTML = '<div id="root"><div id="old">Old</div></div>';
    const root = document.getElementById('root') as HTMLElement;

    initializeComponentMarkers(
      {
        comp: { start: 0, end: 1 },
      },
      root,
    );

    const original = getComponentBounds('comp');
    expect(original).not.toBeNull();

    const template = document.createElement('template');
    template.innerHTML =
      '<div id="parent"><button id="btn" data-onclick="h1">Click</button><div id="child">Child</div></div>';
    const fragment = template.content.cloneNode(true);

    registerComponentMarkers(
      {
        comp: { start: 0, end: 1 },
        child: { parentPath: [0], start: 1, end: 2 },
      },
      fragment,
    );

    const range = document.createRange();
    range.setStartBefore(original!.start);
    range.setEndAfter(original!.end);
    range.deleteContents();
    range.insertNode(fragment);
    range.detach();

    const refreshed = getComponentBounds('comp');
    expect(refreshed).not.toBeNull();
    expect(refreshed?.start.isConnected).toBe(true);
    expect(refreshed?.end.isConnected).toBe(true);
    expect(refreshed?.start.nextSibling).toBe(root.querySelector('#parent'));

    const childBounds = getComponentBounds('child');
    expect(childBounds).not.toBeNull();
    expect(childBounds?.start.isConnected).toBe(true);
    expect(childBounds?.end.isConnected).toBe(true);
    expect(childBounds?.start.nextSibling).toBe(
      root.querySelector('#child'),
    );
  });
});

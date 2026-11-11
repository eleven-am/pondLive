import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, it, expect, vi } from 'vitest';
import { Applier } from '../../src/applier';
import type { Edit } from '../../src/protocol';
import { Logger } from '../../src/logger';

interface Step {
    edits: Edit[];
    expected: string;
}

interface Case {
    ssr: string;
    steps: Step[];
}

const cases: Case[] = JSON.parse(readFileSync(join(dirname(fileURLToPath(import.meta.url)), '..', 'fixtures', 'applier', 'eproto.json'), 'utf-8'));

function normalize(el: Element): string {
    const clone = el.cloneNode(true) as Element;
    const walker = document.createTreeWalker(clone, NodeFilter.SHOW_ALL);
    const drop: Node[] = [];
    let n: Node | null = walker.currentNode;
    while (n) {
        if (n.nodeType === Node.COMMENT_NODE || (n.nodeType === Node.TEXT_NODE && n.nodeValue === '')) drop.push(n);
        if (n.nodeType === Node.ELEMENT_NODE) (n as Element).removeAttribute('data-pi');
        n = walker.nextNode();
    }
    drop.forEach((d) => d.parentNode?.removeChild(d));
    clone.normalize();
    return clone.outerHTML;
}

describe('migrated study fixtures (insert, move, remove, text over keyed lists)', () => {
    it('applies every edit stream strictly and reaches the expected DOM', () => {
        Logger.configure({ enabled: false });
        const callbacks = { onEvent: vi.fn(), onScript: vi.fn(), onScriptCleanup: vi.fn() };
        let steps = 0;
        let edits = 0;
        for (const c of cases) {
            const host = document.createElement('div');
            host.innerHTML = c.ssr;
            document.body.replaceChildren(host);
            const applier = new Applier(host, callbacks);
            applier.index();
            for (const step of c.steps) {
                applier.apply(step.edits);
                edits += step.edits.length;
                const expected = document.createElement('div');
                expected.innerHTML = step.expected;
                expect(normalize(host)).toBe(normalize(expected));
                steps++;
            }
        }
        expect(cases.length).toBe(20);
        expect(steps).toBeGreaterThan(100);
        expect(edits).toBeGreaterThan(300);
    });

    it('keeps the identity of every row that survives a step', () => {
        const callbacks = { onEvent: vi.fn(), onScript: vi.fn(), onScriptCleanup: vi.fn() };
        for (const c of cases.slice(0, 10)) {
            const host = document.createElement('div');
            host.innerHTML = c.ssr;
            document.body.replaceChildren(host);
            const applier = new Applier(host, callbacks);
            applier.index();
            for (const step of c.steps) {
                const before = new Map<number, Node>();
                for (const el of Array.from(host.querySelectorAll('li[data-pi]'))) before.set(Number(el.getAttribute('data-pi')), el);
                const removed = new Set(step.edits.filter((e) => e[0] === 'rm').map((e) => e[1] as number));
                applier.apply(step.edits);
                for (const [id, node] of before) {
                    if (removed.has(id)) {
                        expect(applier.node(id)).toBeUndefined();
                    } else {
                        expect(applier.node(id)).toBe(node);
                    }
                }
            }
        }
    });
});

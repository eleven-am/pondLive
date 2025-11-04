(function (global) {
  const LiveUI = (global.LiveUI = global.LiveUI || {});
  const dom = LiveUI.domIndex || {
    registerSlot() {},
    getSlot() { return null; },
    unregisterSlot() {},
    ensureList() { throw new Error('liveui: dom index missing'); },
    setRow() {},
    getRow() { return null; },
    deleteRow() {},
  };

  function ensureTextNode(node, slotIndex) {
    if (!node) {
      throw new Error(`liveui: slot ${slotIndex} not registered`);
    }
    return node;
  }

  function applySetText(slotIndex, text) {
    const node = ensureTextNode(dom.getSlot(slotIndex), slotIndex);
    node.textContent = text;
  }

  function applySetAttrs(slotIndex, upsert, remove) {
    const node = dom.getSlot(slotIndex);
    if (!(node instanceof Element)) {
      throw new Error(`liveui: slot ${slotIndex} is not an element`);
    }
    if (upsert) {
      for (const [k, v] of Object.entries(upsert)) {
        if (v === undefined || v === null) continue;
        node.setAttribute(k, String(v));
      }
    }
    if (remove) {
      for (const key of remove) {
        node.removeAttribute(key);
      }
    }
  }

  function createFragment(html) {
    const template = document.createElement('template');
    template.innerHTML = html;
    return template.content;
  }

  function registerRowSlots(slotIndexes, fragment) {
    if (!Array.isArray(slotIndexes) || slotIndexes.length === 0) {
      return;
    }
    const placeholders = fragment.querySelectorAll
      ? fragment.querySelectorAll('[data-slot-index]')
      : [];
    placeholders.forEach((el) => {
      const index = Number(el.getAttribute('data-slot-index'));
      if (!Number.isNaN(index)) {
        dom.registerSlot(index, el);
      }
    });
    slotIndexes.forEach((idx) => {
      if (!dom.getSlot(idx)) {
        console.warn(`liveui: slot ${idx} not resolved in inserted row`);
      }
    });
  }

  function applyList(slotIndex, childOps) {
    if (!Array.isArray(childOps) || childOps.length === 0) return;
    const record = dom.ensureList(slotIndex);
    const container = record.container;
    for (const op of childOps) {
      if (!op || !op.length) continue;
      const kind = op[0];
      switch (kind) {
        case 'del': {
          const key = op[1];
          const row = dom.getRow(slotIndex, key);
          if (row && row.parentNode === container) {
            container.removeChild(row);
          }
          dom.deleteRow(slotIndex, key);
          break;
        }
        case 'ins': {
          const pos = op[1];
          const payload = op[2] || {};
          const fragment = createFragment(payload.html || '');
          const nodes = Array.from(fragment.childNodes);
          if (nodes.length === 0) {
            console.warn('liveui: insertion payload missing nodes for key', payload.key);
            break;
          }
          container.insertBefore(fragment, container.children[pos] || null);
          const root = nodes[0];
          if (root instanceof HTMLElement) {
            dom.setRow(slotIndex, payload.key, root);
            registerRowSlots(payload.slots || [], root);
          } else {
            console.warn('liveui: row root is not an element for key', payload.key);
          }
          break;
        }
        case 'mov': {
          const from = op[1];
          const to = op[2];
          if (from === to) break;
          const child = container.children[from];
          if (child) {
            container.insertBefore(child, container.children[to] || null);
          }
          break;
        }
        default:
          console.warn('liveui: unknown list child op', op);
      }
    }
  }

  function applyOps(ops) {
    if (!Array.isArray(ops)) return;
    for (const op of ops) {
      if (!op || op.length === 0) continue;
      const kind = op[0];
      switch (kind) {
        case 'setText':
          applySetText(op[1], op[2]);
          break;
        case 'setAttrs':
          applySetAttrs(op[1], op[2] || {}, op[3] || []);
          break;
        case 'list':
          applyList(op[1], op.slice(2));
          break;
        default:
          console.warn('liveui: unknown op', op);
      }
    }
  }

  LiveUI.apply = applyOps;
})(typeof window !== 'undefined' ? window : globalThis);

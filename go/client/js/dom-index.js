(function (global) {
  const slotMap = new Map();
  const listMap = new Map();

  function registerSlot(index, node) {
    if (node) {
      slotMap.set(index, node);
    }
  }

  function getSlot(index) {
    return slotMap.get(index) || null;
  }

  function unregisterSlot(index) {
    slotMap.delete(index);
  }

  function ensureList(slotIndex) {
    if (!listMap.has(slotIndex)) {
      const container = document.querySelector(`[data-list-slot="${slotIndex}"]`);
      if (!container) {
        throw new Error(`liveui: list slot ${slotIndex} not registered`);
      }
      listMap.set(slotIndex, { container, rows: new Map() });
    }
    return listMap.get(slotIndex);
  }

  function registerList(slotIndex, container) {
    if (!container) return;
    listMap.set(slotIndex, { container, rows: new Map() });
  }

  function setRow(slotIndex, key, root) {
    const list = ensureList(slotIndex);
    list.rows.set(key, root);
  }

  function getRow(slotIndex, key) {
    const list = ensureList(slotIndex);
    return list.rows.get(key) || null;
  }

  function deleteRow(slotIndex, key) {
    const list = ensureList(slotIndex);
    list.rows.delete(key);
  }

  global.LiveUI = global.LiveUI || {};
  global.LiveUI.domIndex = {
    registerSlot,
    getSlot,
    unregisterSlot,
    registerList,
    setRow,
    getRow,
    deleteRow,
    ensureList,
  };
})(typeof window !== 'undefined' ? window : globalThis);

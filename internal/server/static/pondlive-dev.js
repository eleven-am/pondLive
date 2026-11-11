// LiveUI Client v1.0.0 - Built with esbuild
"use strict";
var LiveUIModule = (() => {
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

  // src/index.ts
  var index_exports = {};
  __export(index_exports, {
    Applier: () => Applier,
    Bus: () => Bus,
    EditError: () => EditError,
    Executor: () => Executor,
    Logger: () => Logger,
    Runtime: () => Runtime,
    ScriptExecutor: () => ScriptExecutor,
    Sequencer: () => Sequencer,
    Transport: () => Transport,
    boot: () => boot,
    isBoot: () => isBoot,
    isMessage: () => isMessage,
    isServerFrame: () => isServerFrame
  });

  // src/wire.ts
  var REF_KINDS = {
    CanvasRef: { method: " toDataURL ", set: " height width ", query: " height width " },
    DetailsRef: { set: " open ", query: " open " },
    DialogRef: { method: " close show showModal ", query: " open returnValue " },
    ElementRef: { method: " blur checkVisibility click focus getComputedStyle matches scrollIntoView ", set: " scrollLeft scrollTop ", observe: " focused rect scroll " },
    FormRef: { method: " checkValidity reportValidity requestSubmit reset submit " },
    InputRef: { method: " select setRangeText setSelectionRange ", query: " selectionDirection selectionEnd selectionStart " },
    MediaRef: { method: " load pause play ", set: " currentTime muted playbackRate volume ", query: " networkState playbackRate readyState ", observe: " currentTime duration ended muted paused volume " }
  };
  function refAllows(kind, op, name) {
    const key = ` ${name} `;
    const own = REF_KINDS[kind ?? ""];
    return !!own && [own, REF_KINDS.ElementRef].some((k) => !!k[op]?.includes(key));
  }
  var STATIC_TOPICS = ["frame", "router", "dom"];
  var SCRIPT_TOPIC_PREFIX = "script:";
  var SIGNATURES = {
    t: "is",
    a: "iss",
    ra: "is",
    st: "iss",
    h: "is",
    ins: "irs",
    mv: "iir",
    rm: "i",
    clr: "i"
  };
  var ATTR_ID = "data-pi";
  var ATTR_HANDLERS = "data-pon";
  var ATTR_SCRIPT = "data-pscript";
  var ATTR_RAW = "data-praw";
  var ATTR_SELECTION = "data-psel";
  var ATTR_RESYNCING = "data-pond-resyncing";
  var MARKER = /^(t|r|\/r):(\d+)$/;
  var OBSERVED_EVENTS = {
    currentTime: ["timeupdate", "seeked"],
    duration: ["durationchange", "loadedmetadata"],
    paused: ["play", "pause", "ended"],
    ended: ["play", "pause", "ended", "seeked"],
    volume: ["volumechange"],
    muted: ["volumechange"],
    focused: ["focus", "blur"],
    scroll: ["scroll"],
    rect: []
  };
  var UNSAFE_URL = /[\u0000-\u0020\\\u007f-\u00a0\u1680\u2000-\u200a\u2028-\u2029\u202f\u205f\u3000\ufeff]/;
  var SAFE_PROTOCOL = /^(http|https):$/;

  // src/protocol.ts
  function urlOf(path, query, hash) {
    return path + (query ? `?${query}` : "") + (hash ? hash.startsWith("#") ? hash : `#${hash}` : "");
  }
  function isScriptTopic(topic) {
    return topic.startsWith(SCRIPT_TOPIC_PREFIX);
  }
  function scriptTopic(scriptId) {
    return `${SCRIPT_TOPIC_PREFIX}${scriptId}`;
  }
  function isServerFrame(msg) {
    if (typeof msg !== "object" || msg === null) return false;
    const frame = msg;
    if (frame.t === "joined") return true;
    if (frame.t === "evack") return typeof frame.cs === "number";
    if (frame.t === "tok") return typeof frame.tok === "string";
    return frame.t === "declined" && typeof frame.code === "number";
  }
  function isBoot(msg) {
    return typeof msg === "object" && msg !== null && msg.t === "boot";
  }
  function isMessage(msg) {
    return typeof msg === "object" && msg !== null && typeof msg.seq === "number" && typeof msg.topic === "string" && typeof msg.event === "string";
  }

  // src/logger.ts
  var levels = {
    debug: 0,
    info: 1,
    warn: 2,
    error: 3
  };
  var LoggerImpl = class {
    constructor() {
      this.enabled = false;
      this.level = "info";
    }
    configure(config) {
      if (config.enabled !== void 0) this.enabled = config.enabled;
      if (config.level !== void 0) this.level = config.level;
    }
    debug(tag, message, ...args) {
      this.log("debug", tag, message, args);
    }
    info(tag, message, ...args) {
      this.log("info", tag, message, args);
    }
    warn(tag, message, ...args) {
      this.log("warn", tag, message, args);
    }
    error(tag, message, ...args) {
      this.log("error", tag, message, args);
    }
    log(level, tag, message, args) {
      if (!this.enabled && level !== "warn" && level !== "error") return;
      if (levels[level] < levels[this.level]) return;
      const prefix = `[Pond:${tag}]`;
      const fn = console[level] || console.log;
      if (args.length > 0) {
        fn(prefix, message, ...args);
      } else {
        fn(prefix, message);
      }
    }
  };
  var Logger = new LoggerImpl();

  // src/bus.ts
  var Bus = class {
    constructor() {
      this.subscribers = /* @__PURE__ */ new Map();
      this.wildcardSubscribers = [];
      this.nextSubId = 0;
    }
    subscribe(topic, action, callback) {
      const key = `${topic}:${String(action)}`;
      const subId = ++this.nextSubId;
      const sub = {
        id: subId,
        callback
      };
      const subs = this.subscribers.get(key) ?? [];
      subs.push(sub);
      this.subscribers.set(key, subs);
      return {
        unsubscribe: () => this.unsubscribe(key, subId)
      };
    }
    upsert(topic, action, callback) {
      const key = `${topic}:${String(action)}`;
      const subs = this.subscribers.get(key);
      if (subs && subs.length > 0) {
        const first = subs[0];
        first.callback = callback;
        if (subs.length > 1) {
          this.subscribers.set(key, [first]);
        }
        const subId2 = first.id;
        return {
          unsubscribe: () => this.unsubscribe(key, subId2)
        };
      }
      const subId = ++this.nextSubId;
      const sub = {
        id: subId,
        callback
      };
      this.subscribers.set(key, [sub]);
      return {
        unsubscribe: () => this.unsubscribe(key, subId)
      };
    }
    subscribeAll(callback) {
      const subId = ++this.nextSubId;
      const sub = { id: subId, callback };
      this.wildcardSubscribers.push(sub);
      return {
        unsubscribe: () => this.unsubscribeWildcard(subId)
      };
    }
    publish(topic, action, payload) {
      this.deliver(topic, String(action), payload);
    }
    subscriberCount(topic, action) {
      const key = `${topic}:${String(action)}`;
      return this.subscribers.get(key)?.length ?? 0;
    }
    subscribeScript(scriptId, action, callback) {
      return this.subscribe(scriptTopic(scriptId), action, callback);
    }
    publishScript(scriptId, action, payload) {
      this.deliver(scriptTopic(scriptId), String(action), payload);
    }
    clear() {
      this.subscribers.clear();
      this.wildcardSubscribers = [];
    }
    deliver(topic, action, payload) {
      const key = `${topic}:${action}`;
      const subs = this.subscribers.get(key) ?? [];
      for (const sub of subs) {
        try {
          sub.callback(payload);
        } catch (error) {
          Logger.error("Bus", "Subscriber failed", { topic, action, error });
        }
      }
      for (const sub of this.wildcardSubscribers) {
        try {
          sub.callback(topic, action, payload);
        } catch (error) {
          Logger.error("Bus", "Subscriber failed", { topic, action, error });
        }
      }
    }
    unsubscribe(key, subId) {
      const subs = this.subscribers.get(key);
      if (!subs) return;
      const idx = subs.findIndex((s) => s.id === subId);
      if (idx !== -1) {
        subs.splice(idx, 1);
        if (subs.length === 0) {
          this.subscribers.delete(key);
        }
      }
    }
    unsubscribeWildcard(subId) {
      const idx = this.wildcardSubscribers.findIndex((s) => s.id === subId);
      if (idx !== -1) {
        this.wildcardSubscribers.splice(idx, 1);
      }
    }
  };

  // src/transport.ts
  var BACKOFF_BASE_MS = 1e3;
  var BACKOFF_CAP_MS = 3e4;
  var HANDSHAKE_TIMEOUT_MS = 1e4;
  var QUEUE_MAX_MESSAGES = 100;
  var QUEUE_MAX_BYTES = 1024 * 1024;
  var DECLINED_CLOSE_MIN = 4400;
  var DECLINED_CLOSE_MAX = 4499;
  var utf8 = new TextEncoder();
  var Transport = class {
    constructor(config) {
      this.state = "disconnected";
      this.stateListeners = [];
      this.socket = null;
      this.joined = false;
      this.attempts = 0;
      this.reconnectTimer = null;
      this.handshakeTimer = null;
      this.unconfirmed = [];
      this.unconfirmedBytes = 0;
      this.overflowLogged = false;
      this.nextCS = 0;
      this.endpoint = config.endpoint;
      this.token = config.token;
      this.sessionId = config.sessionId;
      this.version = config.version;
      this.applied = config.applied;
      this.wantsReset = config.wantsReset;
      this.onMessage = config.onMessage;
    }
    get sid() {
      return this.sessionId;
    }
    get connectionState() {
      return this.state;
    }
    connect() {
      if (this.socket !== null || this.reconnectTimer !== null) {
        return;
      }
      this.open();
    }
    disconnect() {
      this.clearTimers();
      const socket = this.socket;
      this.socket = null;
      this.joined = false;
      this.attempts = 0;
      if (socket) {
        socket.close();
      }
      this.setState("disconnected");
    }
    onStateChange(listener) {
      this.stateListeners.push(listener);
      return () => {
        const idx = this.stateListeners.indexOf(listener);
        if (idx !== -1) {
          this.stateListeners.splice(idx, 1);
        }
      };
    }
    send(topic, action, payload) {
      this.sendEvent(topic, String(action), payload);
    }
    sendAck(seq) {
      const ack = {
        t: "ack",
        sid: this.sessionId,
        seq
      };
      this.sendControl(ack);
    }
    sendResync(seq, reason) {
      const resync = {
        t: "resync",
        sid: this.sessionId,
        seq,
        reason
      };
      this.sendControl(resync);
    }
    sendHandler(id, event, payload) {
      this.sendEvent("h", event, payload, id);
    }
    sendScript(scriptId, payload) {
      this.sendEvent(scriptTopic(scriptId), "message", payload);
    }
    handleMessage(payload) {
      if (!isMessage(payload)) {
        return;
      }
      if (!this.isValidTopic(payload.topic)) {
        return;
      }
      this.onMessage(payload);
    }
    isValidTopic(topic) {
      return STATIC_TOPICS.includes(topic) || isScriptTopic(topic);
    }
    open() {
      this.setState("connecting");
      let socket;
      try {
        socket = new WebSocket(this.url());
      } catch (err) {
        Logger.error("TRANSPORT", "Failed to open socket", err);
        this.scheduleReconnect();
        return;
      }
      this.socket = socket;
      this.joined = false;
      this.handshakeTimer = setTimeout(() => {
        this.handshakeTimer = null;
        if (this.socket === socket && !this.joined) {
          socket.close();
        }
      }, HANDSHAKE_TIMEOUT_MS);
      socket.onopen = () => {
        if (this.socket !== socket) return;
        const join = {
          t: "join",
          sid: this.sessionId,
          tok: this.token,
          ver: this.version,
          bs: this.applied(),
          resync: this.wantsReset()
        };
        socket.send(JSON.stringify(join));
      };
      socket.onmessage = (event) => {
        if (this.socket !== socket) return;
        this.handleFrame(event.data);
      };
      socket.onclose = (event) => {
        if (this.socket !== socket) return;
        this.handleClose(event.code);
      };
    }
    url() {
      const url = new URL(this.endpoint, window.location.href);
      url.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      return url.toString();
    }
    handleFrame(raw) {
      if (typeof raw !== "string") {
        Logger.warn("TRANSPORT", "Ignoring non-text frame");
        return;
      }
      let parsed;
      try {
        parsed = JSON.parse(raw);
      } catch (err) {
        Logger.warn("TRANSPORT", "Ignoring malformed frame", err);
        return;
      }
      Logger.info("TRANSPORT", "Transport received message:", parsed);
      if (isServerFrame(parsed)) {
        if (parsed.t === "joined") {
          this.handleJoined(parsed.tok, parsed.cs);
        } else if (parsed.t === "evack") {
          this.confirmThrough(parsed.cs);
        } else if (parsed.t === "tok") {
          this.token = parsed.tok;
        } else {
          this.handleDeclined(parsed.code, parsed.reason);
        }
        return;
      }
      if (!this.joined) {
        Logger.warn("TRANSPORT", "Ignoring frame received before joined");
        return;
      }
      this.handleMessage(parsed);
    }
    handleJoined(token, cs) {
      if (this.joined) return;
      if (typeof token === "string" && token !== "") {
        this.token = token;
      }
      this.joined = true;
      this.attempts = 0;
      this.clearHandshakeTimer();
      this.setState("connected");
      if (typeof cs === "number") {
        this.confirmThrough(cs);
      }
      this.resendUnconfirmed();
    }
    handleDeclined(code, reason) {
      Logger.warn("TRANSPORT", "Join declined", code, reason);
      const socket = this.socket;
      this.socket = null;
      this.terminateDeclined();
      if (socket) {
        socket.close();
      }
    }
    handleClose(code) {
      this.socket = null;
      this.joined = false;
      this.clearHandshakeTimer();
      if (code >= DECLINED_CLOSE_MIN && code <= DECLINED_CLOSE_MAX) {
        this.terminateDeclined();
        return;
      }
      this.scheduleReconnect();
    }
    terminateDeclined() {
      this.clearTimers();
      this.joined = false;
      this.clearUnconfirmed();
      this.setState("declined");
    }
    scheduleReconnect() {
      this.setState("stalled");
      const ceiling = Math.min(BACKOFF_CAP_MS, BACKOFF_BASE_MS * 2 ** this.attempts);
      const delay = Math.random() * ceiling;
      this.attempts++;
      this.reconnectTimer = setTimeout(() => {
        this.reconnectTimer = null;
        this.open();
      }, delay);
    }
    clearHandshakeTimer() {
      if (this.handshakeTimer !== null) {
        clearTimeout(this.handshakeTimer);
        this.handshakeTimer = null;
      }
    }
    clearTimers() {
      this.clearHandshakeTimer();
      if (this.reconnectTimer !== null) {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }
    }
    confirmThrough(cs) {
      let drop = 0;
      while (drop < this.unconfirmed.length && this.unconfirmed[drop].cs <= cs) {
        this.unconfirmedBytes -= this.unconfirmed[drop].bytes;
        drop++;
      }
      if (drop > 0) {
        this.unconfirmed.splice(0, drop);
      }
    }
    remember(event) {
      this.unconfirmed.push(event);
      this.unconfirmedBytes += event.bytes;
      while (this.unconfirmed.length > QUEUE_MAX_MESSAGES || this.unconfirmedBytes > QUEUE_MAX_BYTES) {
        const dropped = this.unconfirmed.shift();
        if (!dropped) break;
        this.unconfirmedBytes -= dropped.bytes;
        if (!this.overflowLogged) {
          this.overflowLogged = true;
          Logger.warn("TRANSPORT", "Resend buffer full, dropping oldest events");
        }
      }
    }
    resendUnconfirmed() {
      const socket = this.socket;
      if (!socket) return;
      for (const event of this.unconfirmed) {
        socket.send(event.data);
      }
    }
    clearUnconfirmed() {
      this.unconfirmed = [];
      this.unconfirmedBytes = 0;
      this.overflowLogged = false;
    }
    setState(state) {
      if (this.state === state) return;
      this.state = state;
      for (const listener of [...this.stateListeners]) {
        try {
          listener(state);
        } catch (err) {
          Logger.error("TRANSPORT", "State listener failed", err);
        }
      }
    }
    sendEvent(topic, action, payload, id) {
      const evt = {
        t: topic,
        sid: this.sessionId,
        a: action,
        p: payload,
        cs: ++this.nextCS
      };
      if (id !== void 0) evt.i = id;
      Logger.info("TRANSPORT", "Transport sending event:", evt);
      const data = JSON.stringify(evt);
      this.remember({ cs: evt.cs, data, bytes: utf8.encode(data).length });
      if (this.isOpen()) {
        this.socket.send(data);
      }
    }
    isOpen() {
      return this.joined && !!this.socket && this.socket.readyState === 1;
    }
    sendControl(frame) {
      if (!this.isOpen()) return;
      Logger.info("TRANSPORT", "Transport sending control:", frame);
      this.socket.send(JSON.stringify(frame));
    }
  };

  // src/fields.ts
  var COMPUTED = {
    value: (_, t) => t.isContentEditable ? t.innerText : t.value,
    selected: (_, t) => t.selectedOptions && Array.from(t.selectedOptions, (o) => o.value),
    files: (_, t) => t.files && Array.from(t.files, (f) => ({ n: f.name, s: f.size, t: f.type, m: f.lastModified })),
    touches: (e) => e.changedTouches && Array.from(e.changedTouches, (x) => [x.identifier, x.clientX, x.clientY]),
    text: (e) => (e.clipboardData || e.dataTransfer)?.getData("text/plain"),
    fullscreen: (e) => document.fullscreenElement === e.target,
    returnValue: (_, t) => t.returnValue
  };
  function readFields(e, names) {
    const out = {};
    const t = e.target;
    for (const n of names.split(" ")) {
      const c = COMPUTED[n];
      const v = c ? t && c(e, t) : n in e ? e[n] : t?.[n];
      if (v != null) out[n] = v;
    }
    return out;
  }
  function domEvent(el, name) {
    const t = el;
    const live = t.isContentEditable || t.value !== void 0 && !/^(checkbox|radio|file|select)/.test(t.type);
    return name === "~change" ? live ? "input" : "change" : name;
  }

  // src/rate.ts
  var queued = /* @__PURE__ */ new Set();
  function flush(upTo) {
    for (const s of [...queued]) {
      queued.delete(s);
      s.run();
      if (s === upTo) break;
    }
  }
  function limit(db, th, send) {
    if (db === void 0 && th === void 0) {
      return (d) => {
        flush();
        send(d);
      };
    }
    let data;
    let timer;
    let busy = false;
    const frame = th === 0;
    const slot = {
      run: () => {
        send(data);
        if (db === void 0) arm();
      }
    };
    const end = () => {
      busy = false;
      if (queued.has(slot)) flush(slot);
    };
    const arm = () => {
      frame ? cancelAnimationFrame(timer) : clearTimeout(timer);
      busy = true;
      timer = frame ? requestAnimationFrame(end) : setTimeout(end, th);
    };
    const later = () => {
      queued.delete(slot);
      queued.add(slot);
    };
    return (d) => {
      data = d;
      if (db !== void 0) {
        later();
        clearTimeout(timer);
        timer = setTimeout(() => flush(slot), db);
      } else if (busy) {
        later();
      } else {
        flush();
        send(d);
        arm();
      }
    };
  }

  // node_modules/morphdom/dist/morphdom-esm.js
  var DOCUMENT_FRAGMENT_NODE = 11;
  function morphAttrs(fromNode, toNode) {
    var toNodeAttrs = toNode.attributes;
    var attr;
    var attrName;
    var attrNamespaceURI;
    var attrValue;
    var fromValue;
    if (toNode.nodeType === DOCUMENT_FRAGMENT_NODE || fromNode.nodeType === DOCUMENT_FRAGMENT_NODE) {
      return;
    }
    for (var i = toNodeAttrs.length - 1; i >= 0; i--) {
      attr = toNodeAttrs[i];
      attrName = attr.name;
      attrNamespaceURI = attr.namespaceURI;
      attrValue = attr.value;
      if (attrNamespaceURI) {
        attrName = attr.localName || attrName;
        fromValue = fromNode.getAttributeNS(attrNamespaceURI, attrName);
        if (fromValue !== attrValue) {
          if (attr.prefix === "xmlns") {
            attrName = attr.name;
          }
          fromNode.setAttributeNS(attrNamespaceURI, attrName, attrValue);
        }
      } else {
        fromValue = fromNode.getAttribute(attrName);
        if (fromValue !== attrValue) {
          fromNode.setAttribute(attrName, attrValue);
        }
      }
    }
    var fromNodeAttrs = fromNode.attributes;
    for (var d = fromNodeAttrs.length - 1; d >= 0; d--) {
      attr = fromNodeAttrs[d];
      attrName = attr.name;
      attrNamespaceURI = attr.namespaceURI;
      if (attrNamespaceURI) {
        attrName = attr.localName || attrName;
        if (!toNode.hasAttributeNS(attrNamespaceURI, attrName)) {
          fromNode.removeAttributeNS(attrNamespaceURI, attrName);
        }
      } else {
        if (!toNode.hasAttribute(attrName)) {
          fromNode.removeAttribute(attrName);
        }
      }
    }
  }
  var range;
  var NS_XHTML = "http://www.w3.org/1999/xhtml";
  var doc = typeof document === "undefined" ? void 0 : document;
  var HAS_TEMPLATE_SUPPORT = !!doc && "content" in doc.createElement("template");
  var HAS_RANGE_SUPPORT = !!doc && doc.createRange && "createContextualFragment" in doc.createRange();
  function createFragmentFromTemplate(str) {
    var template = doc.createElement("template");
    template.innerHTML = str;
    return template.content.childNodes[0];
  }
  function createFragmentFromRange(str) {
    if (!range) {
      range = doc.createRange();
      range.selectNode(doc.body);
    }
    var fragment = range.createContextualFragment(str);
    return fragment.childNodes[0];
  }
  function createFragmentFromWrap(str) {
    var fragment = doc.createElement("body");
    fragment.innerHTML = str;
    return fragment.childNodes[0];
  }
  function toElement(str) {
    str = str.trim();
    if (HAS_TEMPLATE_SUPPORT) {
      return createFragmentFromTemplate(str);
    } else if (HAS_RANGE_SUPPORT) {
      return createFragmentFromRange(str);
    }
    return createFragmentFromWrap(str);
  }
  function compareNodeNames(fromEl, toEl) {
    var fromNodeName = fromEl.nodeName;
    var toNodeName = toEl.nodeName;
    var fromCodeStart, toCodeStart;
    if (fromNodeName === toNodeName) {
      return true;
    }
    fromCodeStart = fromNodeName.charCodeAt(0);
    toCodeStart = toNodeName.charCodeAt(0);
    if (fromCodeStart <= 90 && toCodeStart >= 97) {
      return fromNodeName === toNodeName.toUpperCase();
    } else if (toCodeStart <= 90 && fromCodeStart >= 97) {
      return toNodeName === fromNodeName.toUpperCase();
    } else {
      return false;
    }
  }
  function createElementNS(name, namespaceURI) {
    return !namespaceURI || namespaceURI === NS_XHTML ? doc.createElement(name) : doc.createElementNS(namespaceURI, name);
  }
  function moveChildren(fromEl, toEl) {
    var curChild = fromEl.firstChild;
    while (curChild) {
      var nextChild = curChild.nextSibling;
      toEl.appendChild(curChild);
      curChild = nextChild;
    }
    return toEl;
  }
  function syncBooleanAttrProp(fromEl, toEl, name) {
    if (fromEl[name] !== toEl[name]) {
      fromEl[name] = toEl[name];
      if (fromEl[name]) {
        fromEl.setAttribute(name, "");
      } else {
        fromEl.removeAttribute(name);
      }
    }
  }
  var specialElHandlers = {
    OPTION: function(fromEl, toEl) {
      var parentNode = fromEl.parentNode;
      if (parentNode) {
        var parentName = parentNode.nodeName.toUpperCase();
        if (parentName === "OPTGROUP") {
          parentNode = parentNode.parentNode;
          parentName = parentNode && parentNode.nodeName.toUpperCase();
        }
        if (parentName === "SELECT" && !parentNode.hasAttribute("multiple")) {
          if (fromEl.hasAttribute("selected") && !toEl.selected) {
            fromEl.setAttribute("selected", "selected");
            fromEl.removeAttribute("selected");
          }
          parentNode.selectedIndex = -1;
        }
      }
      syncBooleanAttrProp(fromEl, toEl, "selected");
    },
    /**
     * The "value" attribute is special for the <input> element since it sets
     * the initial value. Changing the "value" attribute without changing the
     * "value" property will have no effect since it is only used to the set the
     * initial value.  Similar for the "checked" attribute, and "disabled".
     */
    INPUT: function(fromEl, toEl) {
      syncBooleanAttrProp(fromEl, toEl, "checked");
      syncBooleanAttrProp(fromEl, toEl, "disabled");
      if (fromEl.value !== toEl.value) {
        fromEl.value = toEl.value;
      }
      if (!toEl.hasAttribute("value")) {
        fromEl.removeAttribute("value");
      }
    },
    TEXTAREA: function(fromEl, toEl) {
      var newValue = toEl.value;
      if (fromEl.value !== newValue) {
        fromEl.value = newValue;
      }
      var firstChild = fromEl.firstChild;
      if (firstChild) {
        var oldValue = firstChild.nodeValue;
        if (oldValue == newValue || !newValue && oldValue == fromEl.placeholder) {
          return;
        }
        firstChild.nodeValue = newValue;
      }
    },
    SELECT: function(fromEl, toEl) {
      if (!toEl.hasAttribute("multiple")) {
        var selectedIndex = -1;
        var i = 0;
        var curChild = fromEl.firstChild;
        var optgroup;
        var nodeName;
        while (curChild) {
          nodeName = curChild.nodeName && curChild.nodeName.toUpperCase();
          if (nodeName === "OPTGROUP") {
            optgroup = curChild;
            curChild = optgroup.firstChild;
            if (!curChild) {
              curChild = optgroup.nextSibling;
              optgroup = null;
            }
          } else {
            if (nodeName === "OPTION") {
              if (curChild.hasAttribute("selected")) {
                selectedIndex = i;
                break;
              }
              i++;
            }
            curChild = curChild.nextSibling;
            if (!curChild && optgroup) {
              curChild = optgroup.nextSibling;
              optgroup = null;
            }
          }
        }
        fromEl.selectedIndex = selectedIndex;
      }
    }
  };
  var ELEMENT_NODE = 1;
  var DOCUMENT_FRAGMENT_NODE$1 = 11;
  var TEXT_NODE = 3;
  var COMMENT_NODE = 8;
  function noop() {
  }
  function defaultGetNodeKey(node) {
    if (node) {
      return node.getAttribute && node.getAttribute("id") || node.id;
    }
  }
  function morphdomFactory(morphAttrs2) {
    return function morphdom2(fromNode, toNode, options) {
      if (!options) {
        options = {};
      }
      if (typeof toNode === "string") {
        if (fromNode.nodeName === "#document" || fromNode.nodeName === "HTML") {
          var toNodeHtml = toNode;
          toNode = doc.createElement("html");
          toNode.innerHTML = toNodeHtml;
        } else if (fromNode.nodeName === "BODY") {
          var toNodeBody = toNode;
          toNode = doc.createElement("html");
          toNode.innerHTML = toNodeBody;
          var bodyElement = toNode.querySelector("body");
          if (bodyElement) {
            toNode = bodyElement;
          }
        } else {
          toNode = toElement(toNode);
        }
      } else if (toNode.nodeType === DOCUMENT_FRAGMENT_NODE$1) {
        toNode = toNode.firstElementChild;
      }
      var getNodeKey = options.getNodeKey || defaultGetNodeKey;
      var onBeforeNodeAdded = options.onBeforeNodeAdded || noop;
      var onNodeAdded = options.onNodeAdded || noop;
      var onBeforeElUpdated = options.onBeforeElUpdated || noop;
      var onElUpdated = options.onElUpdated || noop;
      var onBeforeNodeDiscarded = options.onBeforeNodeDiscarded || noop;
      var onNodeDiscarded = options.onNodeDiscarded || noop;
      var onBeforeElChildrenUpdated = options.onBeforeElChildrenUpdated || noop;
      var skipFromChildren = options.skipFromChildren || noop;
      var addChild = options.addChild || function(parent, child) {
        return parent.appendChild(child);
      };
      var childrenOnly = options.childrenOnly === true;
      var fromNodesLookup = /* @__PURE__ */ Object.create(null);
      var keyedRemovalList = [];
      function addKeyedRemoval(key) {
        keyedRemovalList.push(key);
      }
      function walkDiscardedChildNodes(node, skipKeyedNodes) {
        if (node.nodeType === ELEMENT_NODE) {
          var curChild = node.firstChild;
          while (curChild) {
            var key = void 0;
            if (skipKeyedNodes && (key = getNodeKey(curChild))) {
              addKeyedRemoval(key);
            } else {
              onNodeDiscarded(curChild);
              if (curChild.firstChild) {
                walkDiscardedChildNodes(curChild, skipKeyedNodes);
              }
            }
            curChild = curChild.nextSibling;
          }
        }
      }
      function removeNode(node, parentNode, skipKeyedNodes) {
        if (onBeforeNodeDiscarded(node) === false) {
          return;
        }
        if (parentNode) {
          parentNode.removeChild(node);
        }
        onNodeDiscarded(node);
        walkDiscardedChildNodes(node, skipKeyedNodes);
      }
      function indexTree(node) {
        if (node.nodeType === ELEMENT_NODE || node.nodeType === DOCUMENT_FRAGMENT_NODE$1) {
          var curChild = node.firstChild;
          while (curChild) {
            var key = getNodeKey(curChild);
            if (key) {
              fromNodesLookup[key] = curChild;
            }
            indexTree(curChild);
            curChild = curChild.nextSibling;
          }
        }
      }
      indexTree(fromNode);
      function handleNodeAdded(el) {
        onNodeAdded(el);
        var curChild = el.firstChild;
        while (curChild) {
          var nextSibling = curChild.nextSibling;
          var key = getNodeKey(curChild);
          if (key) {
            var unmatchedFromEl = fromNodesLookup[key];
            if (unmatchedFromEl && compareNodeNames(curChild, unmatchedFromEl)) {
              curChild.parentNode.replaceChild(unmatchedFromEl, curChild);
              morphEl(unmatchedFromEl, curChild);
            } else {
              handleNodeAdded(curChild);
            }
          } else {
            handleNodeAdded(curChild);
          }
          curChild = nextSibling;
        }
      }
      function cleanupFromEl(fromEl, curFromNodeChild, curFromNodeKey) {
        while (curFromNodeChild) {
          var fromNextSibling = curFromNodeChild.nextSibling;
          if (curFromNodeKey = getNodeKey(curFromNodeChild)) {
            addKeyedRemoval(curFromNodeKey);
          } else {
            removeNode(
              curFromNodeChild,
              fromEl,
              true
              /* skip keyed nodes */
            );
          }
          curFromNodeChild = fromNextSibling;
        }
      }
      function morphEl(fromEl, toEl, childrenOnly2) {
        var toElKey = getNodeKey(toEl);
        if (toElKey) {
          delete fromNodesLookup[toElKey];
        }
        if (!childrenOnly2) {
          var beforeUpdateResult = onBeforeElUpdated(fromEl, toEl);
          if (beforeUpdateResult === false) {
            return;
          } else if (beforeUpdateResult instanceof HTMLElement) {
            fromEl = beforeUpdateResult;
            indexTree(fromEl);
          }
          morphAttrs2(fromEl, toEl);
          onElUpdated(fromEl);
          if (onBeforeElChildrenUpdated(fromEl, toEl) === false) {
            return;
          }
        }
        if (fromEl.nodeName !== "TEXTAREA") {
          morphChildren(fromEl, toEl);
        } else {
          specialElHandlers.TEXTAREA(fromEl, toEl);
        }
      }
      function morphChildren(fromEl, toEl) {
        var skipFrom = skipFromChildren(fromEl, toEl);
        var curToNodeChild = toEl.firstChild;
        var curFromNodeChild = fromEl.firstChild;
        var curToNodeKey;
        var curFromNodeKey;
        var fromNextSibling;
        var toNextSibling;
        var matchingFromEl;
        outer: while (curToNodeChild) {
          toNextSibling = curToNodeChild.nextSibling;
          curToNodeKey = getNodeKey(curToNodeChild);
          while (!skipFrom && curFromNodeChild) {
            fromNextSibling = curFromNodeChild.nextSibling;
            if (curToNodeChild.isSameNode && curToNodeChild.isSameNode(curFromNodeChild)) {
              curToNodeChild = toNextSibling;
              curFromNodeChild = fromNextSibling;
              continue outer;
            }
            curFromNodeKey = getNodeKey(curFromNodeChild);
            var curFromNodeType = curFromNodeChild.nodeType;
            var isCompatible = void 0;
            if (curFromNodeType === curToNodeChild.nodeType) {
              if (curFromNodeType === ELEMENT_NODE) {
                if (curToNodeKey) {
                  if (curToNodeKey !== curFromNodeKey) {
                    if (matchingFromEl = fromNodesLookup[curToNodeKey]) {
                      if (fromNextSibling === matchingFromEl) {
                        isCompatible = false;
                      } else {
                        fromEl.insertBefore(matchingFromEl, curFromNodeChild);
                        if (curFromNodeKey) {
                          addKeyedRemoval(curFromNodeKey);
                        } else {
                          removeNode(
                            curFromNodeChild,
                            fromEl,
                            true
                            /* skip keyed nodes */
                          );
                        }
                        curFromNodeChild = matchingFromEl;
                        curFromNodeKey = getNodeKey(curFromNodeChild);
                      }
                    } else {
                      isCompatible = false;
                    }
                  }
                } else if (curFromNodeKey) {
                  isCompatible = false;
                }
                isCompatible = isCompatible !== false && compareNodeNames(curFromNodeChild, curToNodeChild);
                if (isCompatible) {
                  morphEl(curFromNodeChild, curToNodeChild);
                }
              } else if (curFromNodeType === TEXT_NODE || curFromNodeType == COMMENT_NODE) {
                isCompatible = true;
                if (curFromNodeChild.nodeValue !== curToNodeChild.nodeValue) {
                  curFromNodeChild.nodeValue = curToNodeChild.nodeValue;
                }
              }
            }
            if (isCompatible) {
              curToNodeChild = toNextSibling;
              curFromNodeChild = fromNextSibling;
              continue outer;
            }
            if (curFromNodeKey) {
              addKeyedRemoval(curFromNodeKey);
            } else {
              removeNode(
                curFromNodeChild,
                fromEl,
                true
                /* skip keyed nodes */
              );
            }
            curFromNodeChild = fromNextSibling;
          }
          if (curToNodeKey && (matchingFromEl = fromNodesLookup[curToNodeKey]) && compareNodeNames(matchingFromEl, curToNodeChild)) {
            if (!skipFrom) {
              addChild(fromEl, matchingFromEl);
            }
            morphEl(matchingFromEl, curToNodeChild);
          } else {
            var onBeforeNodeAddedResult = onBeforeNodeAdded(curToNodeChild);
            if (onBeforeNodeAddedResult !== false) {
              if (onBeforeNodeAddedResult) {
                curToNodeChild = onBeforeNodeAddedResult;
              }
              if (curToNodeChild.actualize) {
                curToNodeChild = curToNodeChild.actualize(fromEl.ownerDocument || doc);
              }
              addChild(fromEl, curToNodeChild);
              handleNodeAdded(curToNodeChild);
            }
          }
          curToNodeChild = toNextSibling;
          curFromNodeChild = fromNextSibling;
        }
        cleanupFromEl(fromEl, curFromNodeChild, curFromNodeKey);
        var specialElHandler = specialElHandlers[fromEl.nodeName];
        if (specialElHandler) {
          specialElHandler(fromEl, toEl);
        }
      }
      var morphedNode = fromNode;
      var morphedNodeType = morphedNode.nodeType;
      var toNodeType = toNode.nodeType;
      if (!childrenOnly) {
        if (morphedNodeType === ELEMENT_NODE) {
          if (toNodeType === ELEMENT_NODE) {
            if (!compareNodeNames(fromNode, toNode)) {
              onNodeDiscarded(fromNode);
              morphedNode = moveChildren(fromNode, createElementNS(toNode.nodeName, toNode.namespaceURI));
            }
          } else {
            morphedNode = toNode;
          }
        } else if (morphedNodeType === TEXT_NODE || morphedNodeType === COMMENT_NODE) {
          if (toNodeType === morphedNodeType) {
            if (morphedNode.nodeValue !== toNode.nodeValue) {
              morphedNode.nodeValue = toNode.nodeValue;
            }
            return morphedNode;
          } else {
            morphedNode = toNode;
          }
        }
      }
      if (morphedNode === toNode) {
        onNodeDiscarded(fromNode);
      } else {
        if (toNode.isSameNode && toNode.isSameNode(morphedNode)) {
          return;
        }
        morphEl(morphedNode, toNode, childrenOnly);
        if (keyedRemovalList) {
          for (var i = 0, len = keyedRemovalList.length; i < len; i++) {
            var elToRemove = fromNodesLookup[keyedRemovalList[i]];
            if (elToRemove) {
              removeNode(elToRemove, elToRemove.parentNode, false);
            }
          }
        }
      }
      if (!childrenOnly && morphedNode !== fromNode && fromNode.parentNode) {
        if (morphedNode.actualize) {
          morphedNode = morphedNode.actualize(fromNode.ownerDocument || doc);
        }
        fromNode.parentNode.replaceChild(morphedNode, fromNode);
      }
      return morphedNode;
    };
  }
  var morphdom = morphdomFactory(morphAttrs);
  var morphdom_esm_default = morphdom;

  // src/applier.ts
  var EditError = class extends Error {
    constructor(reason, edit, detail, options) {
      super(`${reason}:${Array.isArray(edit) ? String(edit[0]) : typeof edit} ${detail}`, options);
      this.reason = reason;
      this.edit = edit;
      this.name = "EditError";
    }
  };
  var MAX_ID = 4294967295;
  var SHOW_ELEMENT_AND_COMMENT = 129;
  var SVG_NS = "http://www.w3.org/2000/svg";
  var MATH_NS = "http://www.w3.org/1998/Math/MathML";
  var FOLLOWING = 4;
  function isUint(value, min) {
    return typeof value === "number" && Number.isInteger(value) && value >= min && value <= MAX_ID;
  }
  var Applier = class {
    constructor(root, callbacks) {
      this.root = root;
      this.callbacks = callbacks;
      this.nodes = /* @__PURE__ */ new Map();
      this.ends = /* @__PURE__ */ new Map();
      this.bound = /* @__PURE__ */ new WeakSet();
      this.synth = /* @__PURE__ */ new WeakMap();
      this.scripts = /* @__PURE__ */ new Map();
      this.pending = [];
    }
    index() {
      this.nodes.clear();
      this.ends.clear();
      this.scan(this.root, false);
      this.flush();
    }
    node(id) {
      return this.nodes.get(id);
    }
    element(id) {
      const node = this.nodes.get(id);
      return node?.nodeType === Node.ELEMENT_NODE ? node : void 0;
    }
    destroy() {
      for (const scriptId of this.scripts.values()) {
        this.callbacks.onScriptCleanup(scriptId);
      }
      this.scripts.clear();
    }
    apply(edits) {
      for (const edit of edits) {
        try {
          this.validate(edit);
          this.run(edit);
        } catch (e) {
          this.pending = [];
          if (e instanceof EditError) throw e;
          throw new EditError("apply-error", edit, e instanceof Error ? e.name : typeof e, { cause: e });
        }
      }
    }
    morph(html) {
      const root = this.root;
      const next = new DOMParser().parseFromString(html, "text/html").documentElement;
      const scrollX = window.scrollX;
      const scrollY = window.scrollY;
      const preservedValues = /* @__PURE__ */ new Map();
      const priorSelects = /* @__PURE__ */ new Map();
      morphdom_esm_default(root, next, {
        getNodeKey: (node) => node.nodeType === Node.ELEMENT_NODE ? node.getAttribute(ATTR_ID) ?? void 0 : void 0,
        onBeforeElUpdated: (from, to) => {
          if (from instanceof HTMLInputElement && to instanceof HTMLInputElement) {
            if (!to.hasAttribute("value") && from.value !== from.defaultValue && !["checkbox", "radio", "file"].includes(from.type)) {
              let start = null;
              let end = null;
              try {
                start = from.selectionStart;
                end = from.selectionEnd;
              } catch {
              }
              preservedValues.set(from, { value: from.value, start, end });
            }
          } else if (from instanceof HTMLTextAreaElement && to instanceof HTMLTextAreaElement) {
            if (to.value === "" && from.value !== from.defaultValue) {
              preservedValues.set(from, { value: from.value, start: from.selectionStart, end: from.selectionEnd });
            }
          } else if (from instanceof HTMLSelectElement && !from.multiple) {
            priorSelects.set(from, { value: from.value, explicit: to.querySelector("option[selected]") !== null });
          }
          return true;
        }
      });
      for (const [el, saved] of preservedValues) {
        const field = el;
        if (field.value !== saved.value) field.value = saved.value;
        if (saved.start !== null && saved.end !== null && document.activeElement === field) {
          try {
            field.setSelectionRange(saved.start, saved.end);
          } catch {
          }
        }
      }
      for (const [select, prior] of priorSelects) {
        this.settleSelect(select, prior.value, prior.explicit);
      }
      for (const select of Array.from(root.querySelectorAll("select"))) {
        if (!priorSelects.has(select)) {
          this.settleSelect(select, null, Array.from(select.options).some((o) => o.hasAttribute("selected")));
        }
      }
      for (const [el, scriptId] of this.scripts) {
        if (!el.isConnected) {
          this.scripts.delete(el);
          this.callbacks.onScriptCleanup(scriptId);
        }
      }
      this.index();
      if (window.scrollX !== scrollX || window.scrollY !== scrollY) {
        window.scrollTo(scrollX, scrollY);
      }
    }
    settleSelect(select, prior, explicit) {
      if (select.multiple || explicit) return;
      const options = Array.from(select.options);
      for (const option of options) {
        option.removeAttribute("selected");
      }
      const chosen = prior === null ? void 0 : options.find((o) => o.value === prior);
      if (chosen) {
        select.selectedIndex = chosen.index;
      } else if (options.length > 0) {
        select.selectedIndex = 0;
      }
    }
    validate(edit) {
      if (!Array.isArray(edit) || typeof edit[0] !== "string") {
        throw new EditError("apply-error", edit, "edit is not a tuple");
      }
      const sig = Object.prototype.hasOwnProperty.call(SIGNATURES, edit[0]) ? SIGNATURES[edit[0]] : void 0;
      if (sig === void 0) {
        throw new EditError("apply-error", edit, "unknown op");
      }
      if (edit.length !== sig.length + 1) {
        throw new EditError("apply-error", edit, `expected ${sig.length} arguments`);
      }
      for (let i = 0; i < sig.length; i++) {
        const arg = edit[i + 1];
        const ok = sig[i] === "s" ? typeof arg === "string" : isUint(arg, sig[i] === "i" ? 1 : 0);
        if (!ok) {
          throw new EditError("apply-error", edit, `argument ${i + 1} is invalid`);
        }
      }
    }
    run(edit) {
      switch (edit[0]) {
        case "t": {
          const node = this.require(edit[1], edit);
          if (node.nodeType === Node.TEXT_NODE) {
            node.nodeValue = edit[2];
          } else if (node.nodeType === Node.ELEMENT_NODE) {
            node.textContent = edit[2];
          } else {
            throw new EditError("unresolved", edit, "target is not a text slot or element");
          }
          break;
        }
        case "a": {
          const el = this.requireElement(edit[1], edit);
          const [, , name, value] = edit;
          el.setAttribute(name, value);
          if (name === "src" || name === "srcset") this.synth.delete(el);
          if (el instanceof HTMLSelectElement && (name === ATTR_SELECTION || name === "value")) {
            const wanted = new Set(name === "value" ? [value] : JSON.parse(value));
            for (const option of Array.from(el.options)) {
              option.selected = wanted.has(option.value);
              option.toggleAttribute("selected", option.selected);
            }
          } else if (name === "value" && /^(input|textarea|select)$/.test(el.localName)) {
            if (el instanceof HTMLTextAreaElement) el.defaultValue = value;
            const field = el;
            if (field.value !== value) field.value = value;
          } else if (name === "checked" && el instanceof HTMLInputElement) {
            if (!el.checked) el.checked = true;
          } else if (name === "selected" && el instanceof HTMLOptionElement) {
            if (!el.selected) el.selected = true;
          }
          break;
        }
        case "ra": {
          const el = this.requireElement(edit[1], edit);
          const name = edit[2];
          el.removeAttribute(name);
          if (name === "src" || name === "srcset") this.synth.delete(el);
          if (name === "checked" && el instanceof HTMLInputElement) {
            if (el.checked) el.checked = false;
          } else if (name === "selected" && el instanceof HTMLOptionElement) {
            if (el.selected) el.selected = false;
          }
          break;
        }
        case "st": {
          const style = this.requireElement(edit[1], edit).style;
          if (!style) {
            throw new EditError("unresolved", edit, "target has no style");
          }
          if (edit[3] === "") {
            style.removeProperty(edit[2]);
          } else {
            style.setProperty(edit[2], edit[3]);
          }
          break;
        }
        case "h": {
          const el = this.requireElement(edit[1], edit);
          if (!el.hasAttribute(ATTR_RAW)) {
            throw new EditError("unresolved", edit, "target is not a raw host");
          }
          el.innerHTML = edit[2];
          break;
        }
        case "ins":
          this.insert(edit[1], edit[2], edit[3], edit);
          break;
        case "mv":
          this.move(edit[1], edit[2], edit[3], edit);
          break;
        case "rm":
          this.remove(this.unit(edit[1], edit));
          break;
        case "clr": {
          const [start, end] = this.region(edit[1], edit);
          const doomed = [];
          for (let n = start.nextSibling; n && n !== end; n = n.nextSibling) {
            doomed.push(n);
          }
          this.remove(doomed);
          break;
        }
      }
    }
    require(id, edit) {
      const node = this.nodes.get(id);
      if (!node) {
        throw new EditError("unresolved", edit, `id ${id} does not resolve`);
      }
      return node;
    }
    requireElement(id, edit) {
      const node = this.require(id, edit);
      if (node.nodeType !== Node.ELEMENT_NODE) {
        throw new EditError("unresolved", edit, `id ${id} is not an element`);
      }
      return node;
    }
    region(id, edit) {
      const start = this.nodes.get(id);
      const end = this.ends.get(id);
      if (!start || start.nodeType !== Node.COMMENT_NODE || !end || start.parentNode !== end.parentNode) {
        throw new EditError("unresolved", edit, `id ${id} is not a region`);
      }
      return [start, end];
    }
    unit(id, edit) {
      const start = this.require(id, edit);
      if (start.nodeType === Node.ELEMENT_NODE) return [start];
      const [, end] = this.region(id, edit);
      const nodes = [];
      for (let n = start; n; n = n.nextSibling) {
        nodes.push(n);
        if (n === end) return nodes;
      }
      throw new EditError("unresolved", edit, `region ${id} is not closed`);
    }
    anchor(region, ref, edit) {
      const [start, end] = this.region(region, edit);
      if (ref === 0) return end;
      const before = this.unit(ref, edit)[0];
      this.requireInside(start, end, before, ref, edit);
      return before;
    }
    requireInside(start, end, node, id, edit) {
      if (node.parentNode !== start.parentNode || !(start.compareDocumentPosition(node) & FOLLOWING) || !(node.compareDocumentPosition(end) & FOLLOWING)) {
        throw new EditError("unresolved", edit, `unit ${id} is not inside its region`);
      }
    }
    insert(region, ref, html, edit) {
      const before = this.anchor(region, ref, edit);
      const parent = before.parentNode;
      const fragment = this.parse(html, parent);
      this.scan(fragment, true);
      parent.insertBefore(fragment, before);
      this.flush();
    }
    move(unit, region, ref, edit) {
      const nodes = this.unit(unit, edit);
      const [start, end] = this.region(region, edit);
      this.requireInside(start, end, nodes[0], unit, edit);
      const before = this.anchor(region, ref, edit);
      if (nodes.includes(before)) {
        throw new EditError("unresolved", edit, "unit cannot move before itself");
      }
      if (nodes.length === 1) {
        before.parentNode.insertBefore(nodes[0], before);
        return;
      }
      const fragment = document.createDocumentFragment();
      fragment.append(...nodes);
      before.parentNode.insertBefore(fragment, before);
    }
    remove(nodes) {
      for (const node of nodes) {
        this.walk(node, (n) => this.forget(n));
        node.parentNode?.removeChild(node);
      }
    }
    parse(html, parent) {
      const ns = parent.namespaceURI;
      const wrap = ns === SVG_NS && parent.localName !== "foreignObject" ? "svg" : ns === MATH_NS ? "math" : "";
      const template = document.createElement("template");
      template.innerHTML = wrap ? `<${wrap}>${html}</${wrap}>` : html;
      const fragment = template.content;
      if (wrap) {
        fragment.replaceChildren(...Array.from(fragment.firstChild.childNodes));
      }
      for (const old of Array.from(fragment.querySelectorAll("script"))) {
        if (old.closest(`[${ATTR_RAW}]`)) continue;
        const live = document.createElementNS(old.namespaceURI, "script");
        for (const attr of Array.from(old.attributes)) {
          live.setAttribute(attr.name, attr.value);
        }
        live.textContent = old.textContent;
        old.replaceWith(live);
      }
      return fragment;
    }
    walk(scope, fn) {
      const walker = document.createTreeWalker(scope, SHOW_ELEMENT_AND_COMMENT);
      const visible = scope.nodeType === Node.ELEMENT_NODE || scope.nodeType === Node.COMMENT_NODE;
      let n = visible ? scope : walker.nextNode();
      while (n) {
        fn(n);
        n = walker.nextNode();
      }
    }
    scan(scope, strict) {
      this.walk(scope, (n) => this.visit(n, strict));
    }
    fault(detail, strict) {
      if (strict) {
        throw new EditError("apply-error", "index", detail);
      }
      Logger.warn("Applier", detail);
    }
    register(id, node, strict) {
      if (this.nodes.has(id)) {
        this.fault(`duplicate id ${id}`, strict);
        return;
      }
      this.nodes.set(id, node);
    }
    visit(n, strict) {
      if (n.nodeType === Node.ELEMENT_NODE) {
        const el = n;
        const pi = el.getAttribute(ATTR_ID);
        if (pi !== null) {
          if (/^[1-9]\d*$/.test(pi) && isUint(+pi, 1)) {
            this.register(+pi, el, strict);
          } else {
            this.fault(`invalid data-pi ${pi}`, strict);
          }
        }
        if (el.hasAttribute(ATTR_HANDLERS)) this.bind(el, strict);
        if (el.hasAttribute(ATTR_SCRIPT) && !this.scripts.has(el)) this.pending.push(el);
        return;
      }
      const match = MARKER.exec(n.nodeValue ?? "");
      if (!match) return;
      const id = +match[2];
      if (match[1] === "t") {
        let slot = n.nextSibling;
        if (!slot || slot.nodeType !== Node.TEXT_NODE) {
          slot = document.createTextNode("");
          n.parentNode.insertBefore(slot, n.nextSibling);
        }
        this.register(id, slot, strict);
      } else if (match[1] === "r") {
        this.register(id, n, strict);
      } else if (this.ends.has(id)) {
        this.fault(`duplicate region end ${id}`, strict);
      } else {
        this.ends.set(id, n);
      }
    }
    forget(n) {
      if (n.nodeType === Node.ELEMENT_NODE) {
        const el = n;
        const pi = el.getAttribute(ATTR_ID);
        if (pi !== null && this.nodes.get(+pi) === el) this.nodes.delete(+pi);
        const scriptId = this.scripts.get(el);
        if (scriptId !== void 0) {
          this.scripts.delete(el);
          this.callbacks.onScriptCleanup(scriptId);
        }
        return;
      }
      const match = MARKER.exec(n.nodeValue ?? "");
      if (!match) return;
      const id = +match[2];
      if (match[1] === "/r") {
        this.ends.delete(id);
      } else if (match[1] === "t" || this.nodes.get(id) === n) {
        this.nodes.delete(id);
      }
    }
    flush() {
      const queue = this.pending;
      this.pending = [];
      for (const el of queue) {
        if (this.scripts.has(el)) continue;
        let spec;
        try {
          spec = JSON.parse(el.getAttribute(ATTR_SCRIPT));
          if (typeof spec.i !== "string" || typeof spec.s !== "string") throw new TypeError("invalid script spec");
        } catch (e) {
          Logger.warn("Applier", "invalid data-pscript", String(e));
          continue;
        }
        this.scripts.set(el, spec.i);
        this.callbacks.onScript({ scriptId: spec.i, script: spec.s }, el);
      }
    }
    bind(el, strict) {
      if (this.bound.has(el)) return;
      const id = Number(el.getAttribute(ATTR_ID));
      let specs;
      try {
        specs = JSON.parse(el.getAttribute(ATTR_HANDLERS));
        if (!isUint(id, 1) || !Array.isArray(specs) || specs.some((s) => typeof s?.e !== "string")) {
          throw new TypeError("invalid handler spec");
        }
      } catch (e) {
        this.fault(`invalid data-pon: ${String(e)}`, strict);
        return;
      }
      this.bound.add(el);
      for (const spec of specs) {
        this.listen(el, id, spec);
      }
    }
    listen(el, id, spec) {
      const send = limit(spec.db, spec.th, (data) => this.callbacks.onEvent(id, spec.e, data));
      const once = new AbortController();
      el.addEventListener(
        domEvent(el, spec.e),
        (e) => {
          const marked = this.synth.get(el);
          if (marked !== void 0 && marked === el.currentSrc) {
            this.synth.delete(el);
            return;
          }
          if (spec.k && !spec.k.includes(e.key)) return;
          if (spec.o) once.abort();
          if (spec.pv && e.cancelable) e.preventDefault();
          if (spec.sp) e.stopPropagation();
          if (spec.cp) el.setPointerCapture(e.pointerId);
          send(spec.p ? readFields(e, spec.p) : {});
        },
        { capture: !!spec.c, passive: !!spec.ps, signal: once.signal }
      );
      if (spec.e === "load" && el instanceof HTMLImageElement && el.complete && el.naturalWidth) {
        queueMicrotask(() => {
          el.dispatchEvent(new Event("load"));
          this.synth.set(el, el.currentSrc);
        });
      }
    }
  };

  // src/sequencer.ts
  var ACK_BATCH = 32;
  var ACK_DELAY_MS = 50;
  var RESYNC_TIMEOUT_MS = 1e4;
  var Sequencer = class {
    constructor(deps) {
      this.applied = 0;
      this.lastAcked = 0;
      this.current = "live";
      this.ackTimer = null;
      this.resyncTimer = null;
      this.deps = deps;
    }
    get state() {
      return this.current;
    }
    get appliedSeq() {
      return this.applied;
    }
    isResyncing() {
      return this.current === "resyncing";
    }
    boot(seq) {
      this.applied = seq;
      this.lastAcked = seq;
    }
    handle(msg) {
      if (this.current === "failed") {
        return;
      }
      const isReset = msg.topic === "frame" && msg.event === "reset";
      if (this.current === "resyncing") {
        if (isReset && msg.seq > this.applied) {
          this.commitReset(msg);
        }
        return;
      }
      if (msg.seq <= this.applied) {
        return;
      }
      if (isReset) {
        this.commitReset(msg);
        return;
      }
      if (msg.seq > this.applied + 1) {
        Logger.warn("Sequencer", "Gap detected", { applied: this.applied, got: msg.seq });
        this.enterResync("gap");
        return;
      }
      this.commit(msg);
    }
    onConnected() {
      if (this.current === "live" && this.applied > this.lastAcked) {
        this.scheduleAck();
      }
    }
    onDisconnected() {
      this.clearAckTimer();
    }
    destroy() {
      this.clearAckTimer();
      this.clearResyncTimer();
    }
    commit(msg) {
      if (msg.topic === "frame") {
        if (msg.event === "patch") {
          if (!Array.isArray(msg.data)) {
            this.enterResync("apply-error");
            return;
          }
          try {
            this.deps.applier.apply(msg.data);
          } catch (e) {
            const reason = e instanceof EditError ? e.reason : "apply-error";
            Logger.error("Sequencer", "Patch failed", { seq: msg.seq, error: String(e) });
            this.enterResync(reason);
            return;
          }
        } else {
          Logger.warn("Sequencer", "Unknown frame event", msg.event);
        }
      } else {
        try {
          this.deps.dispatch(msg);
        } catch (e) {
          Logger.error("Sequencer", "Dispatch failed", { topic: msg.topic, event: msg.event, error: String(e) });
        }
      }
      this.applied = msg.seq;
      this.scheduleAck();
    }
    commitReset(msg) {
      const reset = msg.data;
      if (typeof reset?.html !== "string") {
        Logger.error("Sequencer", "Malformed reset");
        this.fail();
        return;
      }
      Logger.warn("Sequencer", "Applying reset", { seq: msg.seq, cause: reset.cause });
      try {
        this.deps.applier.morph(reset.html);
      } catch (e) {
        Logger.error("Sequencer", "Reset failed", { error: String(e) });
        this.fail();
        return;
      }
      try {
        if (reset.location) this.deps.applyLocation(reset.location);
      } catch (e) {
        Logger.warn("Sequencer", "Location restore failed", { error: String(e) });
      }
      this.applied = msg.seq;
      this.leaveResync();
      this.ackNow();
    }
    enterResync(reason) {
      if (this.current === "resyncing") {
        return;
      }
      this.current = "resyncing";
      this.clearAckTimer();
      this.deps.onResyncing(true);
      if (this.deps.isConnected()) {
        this.deps.sendResync(this.applied, reason);
      }
      this.clearResyncTimer();
      this.resyncTimer = setTimeout(() => {
        this.resyncTimer = null;
        if (this.current === "resyncing") {
          Logger.error("Sequencer", "Resync timed out");
          this.fail();
        }
      }, RESYNC_TIMEOUT_MS);
    }
    leaveResync() {
      const was = this.current === "resyncing";
      this.current = "live";
      this.clearResyncTimer();
      if (was) {
        this.deps.onResyncing(false);
      }
    }
    fail() {
      this.current = "failed";
      this.clearAckTimer();
      this.clearResyncTimer();
      this.deps.onFailed();
    }
    scheduleAck() {
      if (this.applied - this.lastAcked >= ACK_BATCH) {
        this.ackNow();
        return;
      }
      if (this.ackTimer === null) {
        this.ackTimer = setTimeout(() => {
          this.ackTimer = null;
          this.ackNow();
        }, ACK_DELAY_MS);
      }
    }
    ackNow() {
      this.clearAckTimer();
      if (this.current !== "live" || this.applied <= this.lastAcked) {
        return;
      }
      if (!this.deps.isConnected()) {
        return;
      }
      this.lastAcked = this.applied;
      this.deps.sendAck(this.applied);
    }
    clearAckTimer() {
      if (this.ackTimer !== null) {
        clearTimeout(this.ackTimer);
        this.ackTimer = null;
      }
    }
    clearResyncTimer() {
      if (this.resyncTimer !== null) {
        clearTimeout(this.resyncTimer);
        this.resyncTimer = null;
      }
    }
  };

  // src/serialize.ts
  function serializeValue(value) {
    if (value instanceof Node) return void 0;
    try {
      return JSON.parse(JSON.stringify(value ?? null));
    } catch {
      return void 0;
    }
  }

  // src/executor.ts
  var rectOf = (el) => {
    const r = el.getBoundingClientRect();
    return { x: r.x, y: r.y, width: r.width, height: r.height, top: r.top, right: r.right, bottom: r.bottom, left: r.left };
  };
  var READ = {
    currentTime: (e) => e.currentTime,
    duration: (e) => e.duration,
    paused: (e) => e.paused,
    ended: (e) => e.ended,
    volume: (e) => e.volume,
    muted: (e) => e.muted,
    focused: (e) => document.activeElement === e,
    scroll: (e) => ({ scrollTop: e.scrollTop, scrollLeft: e.scrollLeft, scrollHeight: e.scrollHeight, scrollWidth: e.scrollWidth, clientHeight: e.clientHeight, clientWidth: e.clientWidth }),
    rect: rectOf
  };
  var Executor = class {
    constructor(config) {
      this.subscriptions = [];
      this.observers = /* @__PURE__ */ new Map();
      this.popstateHandler = null;
      this.bus = config.bus;
      this.transport = config.transport;
      this.resolveRef = config.resolveRef;
      this.setupDOMSubscriptions();
      this.setupRouterSubscriptions();
      this.setupPopstateListener();
    }
    destroy() {
      for (const sub of this.subscriptions) {
        sub.unsubscribe();
      }
      this.subscriptions.length = 0;
      this.observers.forEach((o) => o.stop());
      this.observers.clear();
      if (this.popstateHandler) {
        window.removeEventListener("popstate", this.popstateHandler);
        this.popstateHandler = null;
      }
    }
    setupDOMSubscriptions() {
      this.subscriptions.push(
        this.bus.subscribe("dom", "call", (payload) => this.handleCall(payload))
      );
      this.subscriptions.push(
        this.bus.subscribe("dom", "set", (payload) => this.handleSet(payload))
      );
      this.subscriptions.push(
        this.bus.subscribe("dom", "query", (payload) => this.handleQuery(payload))
      );
      this.subscriptions.push(
        this.bus.subscribe("dom", "async", (payload) => this.handleAsync(payload))
      );
      this.subscriptions.push(
        this.bus.subscribe("dom", "observe", (payload) => this.handleObserve(payload))
      );
      this.subscriptions.push(
        this.bus.subscribe("dom", "unobserve", (payload) => this.handleUnobserve(payload))
      );
    }
    setupRouterSubscriptions() {
      this.subscriptions.push(
        this.bus.subscribe("router", "push", (payload) => this.handlePush(payload))
      );
      this.subscriptions.push(
        this.bus.subscribe("router", "replace", (payload) => this.handleReplace(payload))
      );
      this.subscriptions.push(
        this.bus.subscribe("router", "back", () => this.handleBack())
      );
      this.subscriptions.push(
        this.bus.subscribe("router", "forward", () => this.handleForward())
      );
      this.subscriptions.push(
        this.bus.subscribe("router", "redirect", (payload) => this.handleRedirect(payload))
      );
    }
    setupPopstateListener() {
      this.popstateHandler = () => {
        const payload = {
          path: window.location.pathname,
          query: window.location.search.replace(/^\?/, ""),
          hash: window.location.hash.replace(/^#/, "")
        };
        this.transport.send("router", "popstate", payload);
      };
      window.addEventListener("popstate", this.popstateHandler);
    }
    handleCall(payload) {
      const el = refAllows(payload.k, "method", payload.method) && this.resolveRef(payload.id);
      if (el && typeof el[payload.method] === "function") {
        el[payload.method](...payload.args ?? []);
        this.pushObserved(payload.id);
      }
    }
    handleSet(payload) {
      const el = refAllows(payload.k, "set", payload.prop) && this.resolveRef(payload.id);
      if (el) {
        el[payload.prop] = payload.value;
        this.pushObserved(payload.id);
      }
    }
    pushObserved(id) {
      this.observers.forEach((o) => {
        if (o.id === id) o.push();
      });
    }
    handleObserve(payload) {
      const key = this.handleUnobserve(payload);
      const el = this.resolveRef(payload.id);
      const prop = payload.prop;
      const read = READ[prop];
      if (!el || !read || !refAllows(payload.k, "observe", prop)) return;
      const events = OBSERVED_EVENTS[prop];
      let last;
      const push = () => {
        const value = read(el);
        const text = JSON.stringify(value);
        if (text === last) return;
        last = text;
        this.transport.send("dom", "value", { id: payload.id, prop: payload.prop, value });
      };
      const fire = limit(void 0, payload.ms < 0 ? void 0 : payload.ms, push);
      let stop;
      if (prop === "rect") {
        const ro = new ResizeObserver(fire);
        ro.observe(el);
        stop = () => ro.disconnect();
      } else {
        events.forEach((name) => el.addEventListener(name, fire));
        stop = () => events.forEach((name) => el.removeEventListener(name, fire));
      }
      this.observers.set(key, { id: payload.id, push, stop });
      push();
    }
    handleUnobserve(payload) {
      const key = payload.id + ":" + payload.prop;
      this.observers.get(key)?.stop();
      this.observers.delete(key);
      return key;
    }
    reply(requestId, rest) {
      this.transport.send("dom", "response", { requestId, ...rest });
    }
    handleQuery(payload) {
      const { requestId, id, selectors } = payload;
      const el = this.resolveRef(id);
      if (!selectors.every((name) => refAllows(payload.k, "query", name))) return this.reply(requestId, { error: "property not allowed" });
      if (!el) return this.reply(requestId, { error: `ref not found: ${id}` });
      const values = {};
      for (const name of selectors) values[name] = serializeValue(el[name]);
      this.reply(requestId, { values });
    }
    handleAsync(payload) {
      const { requestId, id, method: name } = payload;
      const el = this.resolveRef(id);
      if (!refAllows(payload.k, "method", name)) return this.reply(requestId, { error: `method not allowed: ${name}` });
      if (!el) return this.reply(requestId, { error: `ref not found: ${id}` });
      if (typeof el[name] !== "function") return this.reply(requestId, { error: `method not found: ${name}` });
      Promise.resolve(el[name](...payload.args ?? [])).then((result) => this.reply(requestId, { result: serializeValue(result) })).catch((err) => this.reply(requestId, { error: err instanceof Error ? err.message : String(err) }));
    }
    handlePush(payload) {
      const url = urlOf(payload.path, payload.query, payload.hash);
      window.history.pushState({}, "", url);
    }
    handleReplace(payload) {
      const url = urlOf(payload.path, payload.query, payload.hash);
      window.history.replaceState({}, "", url);
    }
    handleBack() {
      window.history.back();
    }
    handleForward() {
      window.history.forward();
    }
    handleRedirect(payload) {
      const url = payload.url;
      if (!url || UNSAFE_URL.test(url)) return;
      let parsed;
      try {
        parsed = new URL(url, window.location.href);
      } catch {
        return;
      }
      if (!SAFE_PROTOCOL.test(parsed.protocol)) return;
      window.location.assign(url);
    }
  };

  // src/scripts.ts
  var ScriptExecutor = class {
    constructor(config) {
      this.scripts = /* @__PURE__ */ new Map();
      this.bus = config.bus;
      this.transport = config.transport;
    }
    async execute(meta, element) {
      const { scriptId, script } = meta;
      Logger.debug("Script", "execute called", { scriptId, element: element.tagName, script: script.substring(0, 100) + "..." });
      this.cleanup(scriptId);
      const instance = {
        eventHandlers: /* @__PURE__ */ new Map(),
        subscription: this.bus.subscribeScript(scriptId, "send", (payload) => {
          Logger.debug("Script", "server message received", { scriptId, event: payload.event, data: payload.data });
          this.handleServerMessage(scriptId, payload.event, payload.data);
        })
      };
      const transport = {
        send: (event, data) => {
          Logger.debug("Script", "transport.send called", { scriptId, event, data });
          const payload = {
            scriptId,
            event,
            data
          };
          this.transport.sendScript(scriptId, payload);
        },
        on: (event, handler) => {
          Logger.debug("Script", "transport.on registered", { scriptId, event });
          instance.eventHandlers.set(event, handler);
        }
      };
      try {
        Logger.debug("Script", "creating function", { scriptId });
        const fn = new Function("element", "transport", `return (${script})(element, transport);`);
        Logger.debug("Script", "executing function", { scriptId });
        const cleanup = await fn(element, transport);
        if (typeof cleanup === "function") {
          Logger.debug("Script", "cleanup function returned", { scriptId });
          instance.cleanup = cleanup;
        }
        this.scripts.set(scriptId, instance);
        Logger.debug("Script", "execute complete", { scriptId, handlers: Array.from(instance.eventHandlers.keys()) });
      } catch (err) {
        Logger.error("Script", "execute failed", { scriptId, error: String(err) });
        instance.subscription.unsubscribe();
        throw err;
      }
    }
    handleServerMessage(scriptId, event, data) {
      Logger.debug("Script", "handleServerMessage", { scriptId, event, data });
      const instance = this.scripts.get(scriptId);
      if (!instance) {
        Logger.warn("Script", "no instance found", { scriptId });
        return;
      }
      const handler = instance.eventHandlers.get(event);
      if (!handler) {
        Logger.warn("Script", "no handler found", { scriptId, event, availableHandlers: Array.from(instance.eventHandlers.keys()) });
        return;
      }
      try {
        Logger.debug("Script", "invoking handler", { scriptId, event });
        handler(data);
      } catch (err) {
        Logger.error("Script", "handler error", { scriptId, event, error: String(err) });
      }
    }
    cleanup(scriptId) {
      Logger.debug("Script", "cleanup called", { scriptId });
      const instance = this.scripts.get(scriptId);
      if (!instance) {
        Logger.debug("Script", "cleanup: no instance found", { scriptId });
        return;
      }
      if (instance.cleanup) {
        try {
          Logger.debug("Script", "running cleanup function", { scriptId });
          instance.cleanup();
        } catch (err) {
          Logger.error("Script", "cleanup function error", { scriptId, error: String(err) });
        }
      }
      instance.subscription.unsubscribe();
      this.scripts.delete(scriptId);
      Logger.debug("Script", "cleanup complete", { scriptId });
    }
    destroy() {
      for (const scriptId of this.scripts.keys()) {
        this.cleanup(scriptId);
      }
    }
  };

  // src/runtime.ts
  var RELOAD_JITTER_MIN = 1e3;
  var RELOAD_JITTER_MAX = 1e4;
  var MAX_RELOADS = 10;
  var RELOAD_TRACKING_KEY = "pond_reload_count";
  var RELOAD_TIMESTAMP_KEY = "pond_reload_timestamp";
  var RELOAD_WINDOW_MS = 6e4;
  var Runtime = class {
    constructor(config) {
      this.connectedState = false;
      this.root = config.root;
      Logger.configure({ enabled: config.debug ?? false, level: "debug" });
      Logger.info("Runtime", "Initializing", { sid: config.sessionId, ver: config.version });
      this.bus = new Bus();
      this.transport = new Transport({
        endpoint: config.endpoint,
        token: config.token,
        sessionId: config.sessionId,
        version: config.version,
        applied: () => this.sequencer.appliedSeq,
        wantsReset: () => this.sequencer.isResyncing(),
        onMessage: (msg) => this.sequencer.handle(msg)
      });
      this.applier = new Applier(config.root, {
        onEvent: (id, event, data) => this.handleEvent(id, event, data),
        onScript: (meta, el) => this.handleScript(meta, el),
        onScriptCleanup: (scriptId) => this.handleScriptCleanup(scriptId)
      });
      this.executor = new Executor({
        bus: this.bus,
        transport: this.transport,
        resolveRef: (id) => this.applier.element(id)
      });
      this.scripts = new ScriptExecutor({ bus: this.bus, transport: this.transport });
      this.sequencer = new Sequencer({
        applier: this.applier,
        dispatch: (msg) => this.dispatchMessage(msg),
        sendAck: (seq) => this.transport.sendAck(seq),
        sendResync: (seq, reason) => this.transport.sendResync(seq, reason),
        isConnected: () => this.connectedState,
        applyLocation: (location) => this.applyLocation(location),
        onResyncing: (resyncing) => this.markResyncing(resyncing),
        onFailed: () => this.reloadWithJitter()
      });
      this.sequencer.boot(config.seq);
      this.transport.onStateChange((state) => this.handleStateChange(state));
      window.__POND_RUNTIME__ = this;
    }
    connect() {
      Logger.info("Runtime", "Connecting");
      this.transport.connect();
    }
    disconnect() {
      Logger.info("Runtime", "Disconnecting");
      this.transport.disconnect();
      this.sequencer.destroy();
      this.executor.destroy();
      this.scripts.destroy();
      this.bus.clear();
      this.applier.destroy();
    }
    connected() {
      return this.connectedState;
    }
    get seq() {
      return this.sequencer.appliedSeq;
    }
    handleBoot(boot2) {
      Logger.info("Runtime", "Boot received", { ver: boot2.ver, seq: boot2.seq });
      this.applier.index();
      this.sequencer.boot(boot2.seq);
    }
    handleEvent(id, event, data) {
      if (this.sequencer.isResyncing()) {
        Logger.warn("Runtime", "Dropping event while resyncing", { id, event });
        return;
      }
      Logger.debug("Runtime", "Event", { id, event });
      this.transport.sendHandler(id, event, data);
    }
    dispatchMessage({ topic, event, data }) {
      if (isScriptTopic(topic)) {
        if (event === "send") this.bus.publishScript(data.scriptId, "send", data);
        return;
      }
      this.bus.publish(topic, event, data);
    }
    applyLocation(location) {
      const params = new URLSearchParams();
      for (const [key, values] of Object.entries(location.query ?? {})) {
        for (const value of values) params.append(key, value);
      }
      history.replaceState(history.state, "", urlOf(location.path, params.toString(), location.hash));
    }
    markResyncing(resyncing) {
      const el = this.root;
      if (typeof el.toggleAttribute !== "function") return;
      el.toggleAttribute(ATTR_RESYNCING, resyncing);
    }
    handleScript(meta, el) {
      Logger.debug("Runtime", "Script execute", meta.scriptId);
      this.scripts.execute(meta, el).catch((err) => {
        Logger.error("Runtime", "Script error", { scriptId: meta.scriptId, error: String(err) });
      });
    }
    handleScriptCleanup(scriptId) {
      Logger.debug("Runtime", "Script cleanup", scriptId);
      this.scripts.cleanup(scriptId);
    }
    handleStateChange(state) {
      Logger.debug("Runtime", "Connection state", state);
      const wasConnected = this.connectedState;
      this.connectedState = state === "connected";
      if (!wasConnected && this.connectedState) {
        Logger.info("Runtime", "Connected");
        this.clearReloadTracking();
        this.sequencer.onConnected();
      } else if (wasConnected && !this.connectedState) {
        Logger.warn("Runtime", "Disconnected");
        this.sequencer.onDisconnected();
      }
      if (state === "declined") {
        Logger.warn("Runtime", "Session declined - session expired or not found");
        this.reloadWithJitter();
      }
    }
    reloadWithJitter() {
      if (this.shouldEnterFailsafeMode()) {
        Logger.error("Runtime", "Entering failsafe mode - too many consecutive reloads");
        this.enterFailsafeMode();
        return;
      }
      this.incrementReloadCount();
      const jitter = Math.floor(Math.random() * (RELOAD_JITTER_MAX - RELOAD_JITTER_MIN)) + RELOAD_JITTER_MIN;
      Logger.info("Runtime", `Reloading in ${jitter}ms`);
      setTimeout(() => {
        window.location.reload();
      }, jitter);
    }
    shouldEnterFailsafeMode() {
      try {
        const lastTimestamp = parseInt(sessionStorage.getItem(RELOAD_TIMESTAMP_KEY) || "0", 10);
        const reloadCount = parseInt(sessionStorage.getItem(RELOAD_TRACKING_KEY) || "0", 10);
        const now = Date.now();
        if (now - lastTimestamp > RELOAD_WINDOW_MS) {
          return false;
        }
        return reloadCount >= MAX_RELOADS;
      } catch {
        return false;
      }
    }
    incrementReloadCount() {
      try {
        const lastTimestamp = parseInt(sessionStorage.getItem(RELOAD_TIMESTAMP_KEY) || "0", 10);
        const now = Date.now();
        if (now - lastTimestamp > RELOAD_WINDOW_MS) {
          sessionStorage.setItem(RELOAD_TRACKING_KEY, "1");
        } else {
          const count = parseInt(sessionStorage.getItem(RELOAD_TRACKING_KEY) || "0", 10);
          sessionStorage.setItem(RELOAD_TRACKING_KEY, String(count + 1));
        }
        sessionStorage.setItem(RELOAD_TIMESTAMP_KEY, String(now));
      } catch {
      }
    }
    clearReloadTracking() {
      try {
        sessionStorage.removeItem(RELOAD_TRACKING_KEY);
        sessionStorage.removeItem(RELOAD_TIMESTAMP_KEY);
      } catch {
      }
    }
    enterFailsafeMode() {
      try {
        sessionStorage.removeItem(RELOAD_TRACKING_KEY);
        sessionStorage.removeItem(RELOAD_TIMESTAMP_KEY);
      } catch {
      }
    }
  };
  function boot() {
    if (typeof window === "undefined") return null;
    const script = document.getElementById("live-boot");
    let bootData = null;
    if (script?.textContent) {
      try {
        bootData = JSON.parse(script.textContent);
      } catch {
        Logger.error("Runtime", "Failed to parse boot payload");
      }
    }
    if (!bootData) {
      bootData = window.__LIVEUI_BOOT__ ?? null;
    }
    if (!bootData || !isBoot(bootData)) {
      Logger.error("Runtime", "No boot payload found");
      return null;
    }
    const config = {
      root: document.documentElement,
      sessionId: bootData.sid,
      version: bootData.ver,
      seq: bootData.seq,
      endpoint: bootData.ws,
      token: bootData.tok,
      debug: bootData.client?.debug
    };
    const runtime = new Runtime(config);
    runtime.handleBoot(bootData);
    runtime.connect();
    return runtime;
  }

  // src/index.ts
  if (typeof window !== "undefined" && typeof document !== "undefined") {
    if (window.location.href.endsWith("#") && window.location.hash === "") {
      history.replaceState(null, "", window.location.pathname + window.location.search);
    }
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", () => boot());
    } else {
      boot();
    }
  }
  return __toCommonJS(index_exports);
})();
//# sourceMappingURL=pondlive-dev.js.map

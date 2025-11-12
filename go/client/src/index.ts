/**
 * LiveUI Client
 *
 * Main entry point that connects to the server via PondSocket,
 * applies patches, and manages event delegation.
 */

import { PondClient } from "@eleven-am/pondsocket-client";
import * as dom from "./dom-index";
import {
  callElementMethod,
  domGetSync,
  scrollElementIntoView,
  setBooleanProperty,
  setElementProperty,
  toggleElementClass,
} from "./dom";
import {
  applyOps,
  clearPatcherCaches,
  configurePatcher,
  getPatcherConfig,
  getPatcherStats,
  morphElement,
} from "./patcher";
import {
  UploadManager,
  applyUploadBindings,
  primeUploadBindings,
  replaceUploadBindingsForComponent,
} from "./uploads";
import {
  clearHandlers,
  primeSlotBindings,
  registerBindingsForSlot,
  registerHandlers,
  registerNavigationHandler,
  setupEventDelegation,
  syncEventListeners,
  teardownEventDelegation,
  unregisterHandlers,
  unregisterNavigationHandler,
} from "./events";
import {
  clearRefs as clearRefRegistry,
  getRefElement,
  registerRefs as registerRefMetadata,
  unbindRefsInTree,
  unregisterRefs as unregisterRefMetadata,
} from "./refs";
import { ComputedSignal, Signal } from "./reactive";
import { EventEmitter } from "./emitter";
import { OptimisticUpdateManager } from "./optimistic";
import { BootHandler } from "./boot";
import { resolveListContainers, resolveSlotAnchors, applyComponentRanges } from "./manifest";
import { getComponentRange, registerComponentRange, resetComponentRanges } from "./componentRanges";
import { applyRefBindings, applyRouterBindings } from "./bindings";
import type {
  AlertEffect,
  ComponentBootEffect,
  BootPayload,
  ConnectionState,
  DiffOp,
  DispatchEffect,
  DOMActionEffect,
  DOMCallEffect,
  DOMRequestMessage,
  DOMResponseMessage,
  Effect,
  ErrorDetails,
  ErrorMessage,
  EventPayload,
  FocusEffect,
  FrameMessage,
  InitMessage,
  JoinMessage,
  LiveUIEventMap,
  LiveUIEvents,
  LiveUIOptions,
  PerformanceMetrics,
  PubsubControlMessage,
  PushEffect,
  ReplaceEffect,
  ResumeMessage,
  ScrollEffect,
  ServerMessage,
  TemplateMessage,
  DiagnosticMessage,
  ToastEffect,
  MetadataEffect,
  MetadataTagPayload,
  MetadataLinkPayload,
  MetadataScriptPayload,
  CookieEffect,
  UploadClientMessage,
  UploadControlMessage,
  UploadBindingDescriptor,
  TemplatePayload,
  TemplateScope,
} from "./types";

interface RuntimeDiagnosticEntry {
  key: string;
  code: string;
  message: string;
  details?: ErrorDetails;
  timestamp: number;
}

type PondChannel = ReturnType<PondClient["createChannel"]>;

type PubsubSubscription = {
  channel: PondChannel;
  dispose: () => void;
};

const isServerMessage = (value: unknown): value is ServerMessage => {
  if (value === null || typeof value !== "object") {
    return false;
  }
  const candidate = value as { t?: unknown };
  return typeof candidate.t === "string";
};

const DEFAULT_COOKIE_ENDPOINT = "/pondlive/cookie";

class LiveUI extends EventEmitter<LiveUIEvents> {
  private options: Required<Omit<LiveUIOptions, "boot">> & {
    boot?: BootPayload;
  };
  private client: PondClient | null = null;
  private channel: PondChannel | null = null;
  private pubsubChannels: Map<string, PubsubSubscription> = new Map();

  // Reactive state
  private connectionState = new Signal<ConnectionState>({
    status: "disconnected",
  });
  private sessionId = new Signal<string | null>(null);
  private version = new Signal<number>(0);
  private bootHandler: BootHandler;
  private lastAck: number = 0;
  private hasBootPayload = false;
  private autoConnectCleanup: (() => void) | null = null;

  // Frame sequence validation
  private expectedSeq: number | null = null;
  private frameBuffer: Map<number, FrameMessage> = new Map();
  private readonly MAX_FRAME_BUFFER_SIZE = 50;

  private eventQueue: Array<{ hid: string; payload: EventPayload }> = [];

  // Frame batching for patch operations
  private patchQueue: DiffOp[] = [];
  private batchScheduled: boolean = false;
  private rafHandle: number | ReturnType<typeof setTimeout> | null = null;
  private rafUsesTimeoutFallback = false;
  private pendingUploadBindings: UploadBindingDescriptor[][] = [];

  private cookieRequests = new Set<string>();

  private templateCache: Map<string, string> = new Map();
  private readonly MAX_TEMPLATE_CACHE_SIZE = 100;

  // Reconnection
  private reconnectAttempts: number = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private isReconnecting: boolean = false;

  // Runtime diagnostics
  private diagnostics: RuntimeDiagnosticEntry[] = [];
  private errorOverlay: HTMLElement | null = null;
  private errorListEl: HTMLElement | null = null;
  private errorOverlayVisible = false;
  private readonly handleDebugKeydown = (event: KeyboardEvent) =>
    this.onDebugKeydown(event);

  // Performance metrics
  private metrics: PerformanceMetrics = {
    patchesApplied: 0,
    averagePatchTime: 0,
    framesReceived: 0,
    eventsProcessed: 0,
    reconnections: 0,
    uptime: 0,
    effectsMs: 0,
    maxEffectMs: 0,
    slowEffects: 0,
    framesBuffered: 0,
    framesDropped: 0,
    sequenceGaps: 0,
  };
  private startTime: number = 0;
  private patchTimes: number[] = [];
  private patchTimeTotal = 0;

  // Optimistic updates
  private optimistic: OptimisticUpdateManager;
  private uploads: UploadManager | null = null;

  // Navigation tracking (to prevent double pushState)
  private lastOptimisticNavTime: number = 0;
  private readonly OPTIMISTIC_NAV_WINDOW = 100; // ms

  // Event debouncing
  private eventDebouncer = new Map<string, ReturnType<typeof setTimeout>>();

  constructor(options: LiveUIOptions = {}) {
    super();

    // Initialize boot handler
    this.bootHandler = new BootHandler({ debug: options.debug || false });

    // Initialize optimistic update manager
    this.optimistic = new OptimisticUpdateManager({
      onRollback: (id, patches) => this.emit("rollback", { id, patches }),
      onError: (error, context) => this.emit("error", { error, context }),
      debug: options.debug || false,
    });

    this.uploads = new UploadManager({
      getSessionId: () => this.sessionId.get(),
      getEndpoint: () => this.options.uploadEndpoint ?? null,
      send: (payload) => this.sendUploadMessage(payload),
      isConnected: () =>
        this.connectionState.get().status === "connected" &&
        this.channel !== null,
    });

    const baseOptions = { ...(options ?? {}) } as LiveUIOptions;
    const providedBoot = baseOptions.boot ?? null;
    delete (baseOptions as any).boot;

    this.options = {
      endpoint: baseOptions.endpoint ?? "/live",
      uploadEndpoint: baseOptions.uploadEndpoint ?? "/pondlive/upload/",
      autoConnect: baseOptions.autoConnect !== false,
      debug: baseOptions.debug ?? false,
      reconnect: baseOptions.reconnect !== false,
      maxReconnectAttempts: baseOptions.maxReconnectAttempts ?? 5,
      reconnectDelay: baseOptions.reconnectDelay ?? 1000,
      ...baseOptions,
    } as Required<Omit<LiveUIOptions, "boot">> & { boot?: BootPayload };

    if (providedBoot) {
      this.options.boot = providedBoot;
    }

    // Load boot payload
    const boot = this.bootHandler.load(providedBoot);
    this.hasBootPayload = !!boot;
    if (boot) {
      if (boot.client?.endpoint && typeof boot.client.endpoint === "string") {
        this.options.endpoint = boot.client.endpoint;
      }
      if (boot.client?.upload && typeof boot.client.upload === "string") {
        this.options.uploadEndpoint = boot.client.upload;
      }
      if (typeof boot.client?.debug === "boolean") {
        this.options.debug = boot.client.debug;
      }
      this.sessionId.set(boot.sid);
      this.version.set(boot.ver ?? 0);
      this.lastAck = boot.seq ?? 0;
      if (boot.errors && boot.errors.length > 0) {
        for (const err of boot.errors) {
          this.recordDiagnostic(err);
        }
      }
    }

    // Subscribe to connection state changes
    this.connectionState.subscribe((state) => {
      this.log("Connection state changed:", state);
    });

    if (this.options.debug && typeof window !== "undefined") {
      window.addEventListener("keydown", this.handleDebugKeydown);
    }

    this.setupAutoConnect();
  }

  /**
   * Initialize and connect to the server
   */
  async connect(): Promise<void> {
    this.clearAutoConnect();
    const currentState = this.connectionState.get();
    if (
      currentState.status === "connected" ||
      currentState.status === "connecting"
    ) {
      this.log("Already connected or connecting");
      return;
    }

    let boot: BootPayload;
    try {
      boot = this.bootHandler.ensure();
    } catch (error) {
      this.log("Boot payload error:", error);
      this.setState({ status: "error", error: error as Error });
      this.emit("error", { error: error as Error, context: "boot" });
      return;
    }

    if (boot.client?.endpoint && typeof boot.client.endpoint === "string") {
      this.options.endpoint = boot.client.endpoint;
    }
    if (boot.client?.upload && typeof boot.client.upload === "string") {
      this.options.uploadEndpoint = boot.client.upload;
    }

    this.setState({ status: "connecting" });
    this.startTime = Date.now();

    try {
      this.log("Connecting to", this.options.endpoint);
      const sid = boot.sid;
      const joinLocation = this.bootHandler.getJoinLocation();
      const joinPayload = {
        sid,
        ver: boot.ver ?? 0,
        ack: this.lastAck ?? boot.seq ?? 0,
        loc: {
          path: joinLocation.path,
          q: joinLocation.q,
          hash: joinLocation.hash,
        },
      };

      // Create PondSocket client
      this.client = new PondClient(this.options.endpoint);

      // Set up event delegation
      if (typeof document !== "undefined") {
        setupEventDelegation((event) => this.sendEvent(event));
      }

      // Set up navigation interception
      if (typeof document !== "undefined") {
        registerNavigationHandler((path, query, hash) =>
          this.sendNavigation(path, query, hash),
        );
      }

      // Set up popstate handler for browser back/forward
      if (typeof window !== "undefined") {
        window.addEventListener("popstate", this.handlePopState);
      }

      // Create channel
      const topic = `live/${sid}`;
      this.channel = this.client.createChannel<LiveUIEventMap>(
        topic,
        joinPayload,
      );

      this.channel.onChannelStateChange((state) => {
        this.log("Channel state changed:", state);
        if (state === "JOINED") {
          this.onConnected();
        }
      });

      // Handle messages
      this.channel.onMessage((event, message) => {
        if (isServerMessage(message)) {
          this.handleMessage(message);
        } else {
          this.log("Ignoring non-server payload", event, message);
        }
      });

      // Handle leave
      this.channel.onLeave(() => {
        this.log("Channel left");
        this.onDisconnected();
      });

      // Connect and join
      this.client.connect();
      this.channel.join();
    } catch (error) {
      this.log("Connection error:", error);
      this.setState({ status: "error", error: error as Error });
      this.emit("error", { error: error as Error, context: "connect" });

      if (this.options.reconnect && !this.isReconnecting) {
        this.scheduleReconnect();
      }
    }
  }

  /**
   * Disconnect from the server
   */
  disconnect(): void {
    this.clearReconnectTimer();
    this.isReconnecting = false;

    // Cancel any pending batches
    this.cancelScheduledBatch();
    this.patchQueue = [];
    this.pendingUploadBindings = [];

    for (const entry of this.pubsubChannels.values()) {
      try {
        entry.dispose();
        entry.channel.leave();
      } catch (err) {
        this.log("Error leaving pubsub channel", err);
      }
    }
    this.pubsubChannels.clear();

    if (this.channel) {
      this.channel.leave();
      this.channel = null;
    }
    if (this.client) {
      this.client.disconnect();
      this.client = null;
    }

    // Remove popstate handler
    if (typeof window !== "undefined") {
      window.removeEventListener("popstate", this.handlePopState);
    }

    this.setState({ status: "disconnected" });
    this.sessionId.set(null);
    this.version.set(0);
    clearHandlers();
    clearRefRegistry();
    teardownEventDelegation();
    unregisterNavigationHandler();
    this.uploads?.onDisconnect();
    this.emit("disconnected", undefined);
  }

  private setupAutoConnect(): void {
    if (!this.options.autoConnect) {
      return;
    }

    if (!this.hasBootPayload) {
      this.log("Auto-connect skipped: no boot payload detected");
      return;
    }

    const connectSafely = () => {
      this.clearAutoConnect();
      this.log("Auto-connecting after DOM ready");
      this.connect().catch((error) => {
        this.log("Auto-connect failed:", error);
      });
    };

    if (typeof document === "undefined") {
      connectSafely();
      return;
    }

    if (document.readyState === "loading") {
      const onReady = () => {
        document.removeEventListener("DOMContentLoaded", onReady);
        connectSafely();
      };
      document.addEventListener("DOMContentLoaded", onReady);
      this.autoConnectCleanup = () => {
        document.removeEventListener("DOMContentLoaded", onReady);
      };
      return;
    }

    if (typeof queueMicrotask === "function") {
      queueMicrotask(connectSafely);
    } else {
      Promise.resolve().then(connectSafely);
    }
  }

  private clearAutoConnect(): void {
    if (this.autoConnectCleanup) {
      this.autoConnectCleanup();
      this.autoConnectCleanup = null;
    }
  }

  /**
   * Get current connection state
   */
  getConnectionState(): ConnectionState {
    return this.connectionState.get();
  }

  /**
   * Get current metrics
   */
  getMetrics(): PerformanceMetrics {
    return {
      ...this.metrics,
      uptime: this.startTime ? Date.now() - this.startTime : 0,
    };
  }

  /**
   * Subscribe to connection state changes
   */
  onStateChange(callback: (state: ConnectionState) => void): () => void {
    return this.connectionState.subscribe(callback);
  }

  /**
   * Subscribe to session ID changes
   */
  onSessionIdChange(callback: (sessionId: string | null) => void): () => void {
    return this.sessionId.subscribe(callback);
  }

  /**
   * Subscribe to version changes
   */
  onVersionChange(callback: (version: number) => void): () => void {
    return this.version.subscribe(callback);
  }

  /**
   * Send event with optional debouncing
   */
  sendEventDebounced(
    event: { hid: string; payload: EventPayload },
    delay: number = 300,
  ): void {
    const existing = this.eventDebouncer.get(event.hid);
    if (existing) {
      clearTimeout(existing);
    }

    const timer = setTimeout(() => {
      this.sendEvent(event);
      this.eventDebouncer.delete(event.hid);
    }, delay);

    this.eventDebouncer.set(event.hid, timer);
  }

  /**
   * Apply optimistic update
   */
  applyOptimistic(patches: DiffOp[]): string {
    return this.optimistic.apply(patches);
  }

  /**
   * Commit optimistic update (server confirmed)
   */
  commitOptimistic(id: string): void {
    this.optimistic.commit(id);
  }

  /**
   * Rollback optimistic update (server rejected)
   */
  rollbackOptimistic(id: string): void {
    this.optimistic.rollback(id);
  }

  // ========================================================================
  // Private methods
  // ========================================================================

  private setState(newState: ConnectionState): void {
    const oldState = this.connectionState.get();
    if (!this.hasStateChanged(oldState, newState)) {
      return;
    }

    this.connectionState.set(newState);
    this.emit("stateChanged", { from: oldState, to: newState });
  }

  private onConnected(): void {
    const sid = this.sessionId.get();
    const ver = this.version.get();

    if (sid) {
      if (this.isReconnecting) {
        this.isReconnecting = false;
        this.reconnectAttempts = 0;
        this.emit("reconnected", { sessionId: sid });
      }

      this.setState({ status: "connected", sessionId: sid, version: ver });
      this.emit("connected", { sessionId: sid, version: ver });
      this.flushEventQueue();
    }
  }

  private onDisconnected(): void {
    this.setState({ status: "disconnected" });

    if (this.options.reconnect && !this.isReconnecting) {
      this.scheduleReconnect();
    }
    this.uploads?.onDisconnect();
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      this.log("Max reconnect attempts reached");
      this.emit("error", {
        error: new Error("Max reconnect attempts reached"),
        context: "reconnect",
      });
      return;
    }

    this.isReconnecting = true;
    this.reconnectAttempts++;

    const delay =
      this.options.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    this.log(
      `Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.options.maxReconnectAttempts})`,
    );

    this.setState({ status: "reconnecting", attempt: this.reconnectAttempts });
    this.emit("reconnecting", { attempt: this.reconnectAttempts });

    this.reconnectTimer = setTimeout(() => {
      this.metrics.reconnections++;
      this.connect();
    }, delay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  /**
   * Handle incoming messages from the server
   */
  private handleMessage(msg: ServerMessage): void {
    if (!msg || !msg.t) {
      this.log("Invalid message", msg);
      return;
    }

    this.log("Received message:", msg.t, msg);

    switch (msg.t) {
      case "init":
        this.handleInit(msg as InitMessage);
        break;
      case "frame":
        this.handleFrame(msg as FrameMessage);
        break;
      case "template":
        this.handleTemplate(msg as TemplateMessage);
        break;
      case "join":
        this.handleJoin(msg as JoinMessage);
        break;
      case "resume":
        this.handleResume(msg as ResumeMessage);
        break;
      case "error":
        this.handleError(msg as ErrorMessage);
        break;
      case "diagnostic":
        this.handleDiagnostic(msg as DiagnosticMessage);
        break;
      case "pubsub":
        this.handlePubsub(msg as PubsubControlMessage);
        break;
      case "upload":
        this.uploads?.handleControl(msg as UploadControlMessage);
        break;
      case "domreq":
        this.handleDOMRequest(msg as DOMRequestMessage);
        break;
      default:
        this.log("Unknown message type:", (msg as any).t);
    }
  }

  /**
   * Handle init message
   */
  private handleInit(msg: InitMessage): void {
    this.sessionId.set(msg.sid);
    this.version.set(msg.ver);

    // Reset sequence tracking for new session
    this.expectedSeq = msg.seq !== undefined ? msg.seq + 1 : null;
    this.frameBuffer.clear();

    if (Array.isArray(msg.errors) && msg.errors.length > 0) {
      for (const err of msg.errors) {
        this.recordDiagnostic(err);
      }
    }

    // Register handlers
    if (msg.handlers) {
      registerHandlers(msg.handlers);
      syncEventListeners();
    }

    primeSlotBindings(msg.bindings?.slots ?? null);

    clearRefRegistry();
    if (msg.refs?.add) {
      registerRefMetadata(msg.refs.add);
    }
    let initOverrides: Map<string, import("./componentRanges").ComponentRange> | undefined;
    let initRootRange: import("./componentRanges").ComponentRange | null = null;
    if (typeof document !== "undefined") {
      const rootContainer: ParentNode = (document.body ?? document) as ParentNode;
      const childCount = rootContainer.childNodes.length;
      initRootRange = {
        container: rootContainer,
        startIndex: 0,
        endIndex: childCount > 0 ? childCount - 1 : -1,
      };
      resetComponentRanges();
      initOverrides = applyComponentRanges(msg.componentPaths, { root: document });
    }

    const initBindingOptions = { overrides: initOverrides, fallbackRange: initRootRange };
    applyRouterBindings(msg.bindings?.router ?? null, initBindingOptions);
    applyRefBindings(msg.bindings?.refs ?? null, initBindingOptions);

    primeUploadBindings(msg.bindings?.uploads ?? null, initOverrides);

    // Acknowledge if needed
    if (msg.seq !== undefined) {
      this.lastAck = msg.seq;
      this.sendAck(msg.seq);
    }

    this.log("Session initialized:", msg.sid, "version:", msg.ver);
    this.onConnected();
  }

  /**
   * Handle frame message with sequence validation
   */
  private handleFrame(msg: FrameMessage): void {
    const sid = this.sessionId.get();
    if (msg.sid !== sid) {
      this.log("Session mismatch, ignoring frame");
      return;
    }

    this.metrics.framesReceived++;

    // Sequence validation and buffering
    if (msg.seq !== undefined) {
      // First frame - establish expected sequence
      if (this.expectedSeq === null) {
        this.expectedSeq = msg.seq + 1;
        this.applyFrame(msg);
        this.drainFrameBuffer();
        return;
      }

      // Frame arrives in expected order - process immediately
      if (msg.seq === this.expectedSeq) {
        this.expectedSeq = msg.seq + 1;
        this.applyFrame(msg);
        this.drainFrameBuffer();
        return;
      }

      // Out-of-order frame (arrived early) - buffer it
      if (msg.seq > this.expectedSeq) {
        this.metrics.sequenceGaps++;
        this.log("Frame arrived out of order", {
          expected: this.expectedSeq,
          received: msg.seq,
          gap: msg.seq - this.expectedSeq,
        });

        // Check buffer size limit
        if (this.frameBuffer.size >= this.MAX_FRAME_BUFFER_SIZE) {
          this.metrics.framesDropped++;
          this.log("Frame buffer full, dropping oldest frame");
          // Drop the oldest buffered frame
          const oldestSeq = Math.min(...this.frameBuffer.keys());
          this.frameBuffer.delete(oldestSeq);
        }

        this.frameBuffer.set(msg.seq, msg);
        this.metrics.framesBuffered++;
        return;
      }

      // Duplicate or late frame (already processed) - drop it
      if (msg.seq < this.expectedSeq) {
        this.metrics.framesDropped++;
        this.log("Dropping duplicate/late frame", {
          expected: this.expectedSeq,
          received: msg.seq,
        });
        return;
      }
    } else {
      // Frame without sequence number - process immediately (shouldn't happen)
      this.applyFrame(msg);
    }
  }

  /**
   * Apply a frame's contents (extracted for reuse)
   */
  private applyFrame(msg: FrameMessage): void {
    this.version.set(msg.ver);

    // Queue patches for batched application
    if (msg.patch && msg.patch.length > 0) {
      this.log("Queueing", msg.patch.length, "operations");
      this.patchQueue.push(...msg.patch);
      this.scheduleBatch();
    }

    // Update handlers
    if (msg.handlers) {
      if (msg.handlers.add) {
        registerHandlers(msg.handlers.add);
      }
      if (msg.handlers.del) {
        unregisterHandlers(msg.handlers.del);
      }
      // Sync event listeners based on new handlers
      syncEventListeners();
    }

    if (msg.bindings?.slots) {
      for (const [slotKey, specs] of Object.entries(msg.bindings.slots)) {
        const slotId = Number(slotKey);
        if (!Number.isInteger(slotId) || slotId < 0) {
          continue;
        }
        if (Array.isArray(specs)) {
          registerBindingsForSlot(slotId, specs);
        } else {
          registerBindingsForSlot(slotId, []);
        }
      }
    }

    if (msg.bindings?.uploads) {
      applyUploadBindings(msg.bindings.uploads);
    }

    if (msg.bindings?.router) {
      applyRouterBindings(msg.bindings.router);
    }

    if (msg.bindings?.refs) {
      applyRefBindings(msg.bindings.refs);
    }

    if (Array.isArray(msg.bindings?.uploads)) {
      if (this.batchScheduled || this.patchQueue.length > 0) {
        this.pendingUploadBindings.push(msg.bindings.uploads);
      } else {
        applyUploadBindings(msg.bindings.uploads);
      }
    }

    if (msg.refs) {
      if (msg.refs.del) {
        unregisterRefMetadata(msg.refs.del);
      }
      if (msg.refs.add) {
        registerRefMetadata(msg.refs.add);
      }
    }

    // Handle navigation
    if (msg.nav) {
      const now = Date.now();
      const wasOptimistic =
        now - this.lastOptimisticNavTime < this.OPTIMISTIC_NAV_WINDOW;

      if (msg.nav.push) {
        // If we just did an optimistic navigation, use replaceState to avoid duplicate history entries
        if (wasOptimistic) {
          window.history.replaceState({}, "", msg.nav.push);
        } else {
          window.history.pushState({}, "", msg.nav.push);
        }
      } else if (msg.nav.replace) {
        window.history.replaceState({}, "", msg.nav.replace);
      }

      // Reset optimistic nav tracking
      this.lastOptimisticNavTime = 0;
    }

    // Apply effects
    if (msg.effects && msg.effects.length > 0) {
      this.applyEffects(msg.effects);
    }

    // Acknowledge if needed
    if (msg.seq !== undefined) {
      this.sendAck(msg.seq);
    }

    if (msg.metrics) {
      this.log("Frame metrics:", msg.metrics);
      if (typeof msg.metrics.effectsMs === "number") {
        this.metrics.effectsMs = msg.metrics.effectsMs;
      }
      if (typeof msg.metrics.maxEffectMs === "number") {
        this.metrics.maxEffectMs = msg.metrics.maxEffectMs;
      }
      if (typeof msg.metrics.slowEffects === "number") {
        this.metrics.slowEffects = msg.metrics.slowEffects;
      }
    }
  }

  private handleTemplate(msg: TemplateMessage): void {
    if (!msg) {
      return;
    }
    const sid = this.sessionId.get();
    if (msg.sid && sid && msg.sid !== sid) {
      this.log("Template message for different session", msg.sid);
      return;
    }
    if (!msg.scope) {
      this.applyRootTemplatePayload(msg);
      return;
    }
    this.applyComponentTemplatePayload(msg, msg.scope);
  }

  private handleDiagnostic(msg: DiagnosticMessage): void {
    if (!msg) {
      return;
    }
    this.recordDiagnostic(msg);
  }

  /**
   * Drain buffered frames that are now in sequence
   */
  private drainFrameBuffer(): void {
    while (
      this.expectedSeq !== null &&
      this.frameBuffer.has(this.expectedSeq)
    ) {
      const bufferedFrame = this.frameBuffer.get(this.expectedSeq)!;
      this.frameBuffer.delete(this.expectedSeq);
      this.expectedSeq = bufferedFrame.seq! + 1;

      this.log("Draining buffered frame", { seq: bufferedFrame.seq });
      this.applyFrame(bufferedFrame);
    }
  }

  /**
   * Apply effects from server
   */
  private applyEffects(effects: Effect[]): void {
    if (!effects || effects.length === 0) {
      return;
    }

    if (this.patchQueue.length > 0 || this.batchScheduled) {
      if (this.batchScheduled) {
        this.cancelScheduledBatch();
      }
      this.flushBatch();
    }

    for (const effect of effects) {
      try {
        this.log("Applying effect:", effect);
        this.emit("effect", { effect });

        const effectType =
          typeof effect.type === "string" ? effect.type.toLowerCase() : "";
        switch (effectType) {
          case "boot":
            this.applyBootEffect(effect as any);
            break;
          case "componentboot":
            this.applyComponentBootEffect(effect as ComponentBootEffect);
            break;
          case "dom":
            this.applyDOMActionEffect(effect as DOMActionEffect);
            break;
          case "domcall":
            this.applyLegacyDOMCallEffect(effect as DOMCallEffect);
            break;
          case "scroll":
          case "scrolltop":
            this.applyScrollEffect(effect as ScrollEffect);
            break;
          case "focus":
            this.applyFocusEffect(effect as FocusEffect);
            break;
          case "alert":
            window.alert((effect as AlertEffect).message);
            break;
          case "toast":
            this.applyToastEffect(effect as ToastEffect);
            break;
          case "push":
            this.applyPushEffect(effect as PushEffect);
            break;
          case "replace":
            this.applyReplaceEffect(effect as ReplaceEffect);
            break;
          case "dispatch":
            this.dispatchCustomEvent(
              (effect as DispatchEffect).eventName,
              (effect as DispatchEffect).detail,
            );
            break;
          case "metadata":
            this.applyMetadataEffect(effect as MetadataEffect);
            break;
          case "cookies":
            this.handleCookieEffect(effect as CookieEffect);
            break;
          case "custom":
            // Custom effects are handled via the 'effect' event
            break;
          default:
            // Unknown effect type - emit for custom handling
            this.log("Unknown effect type:", effect.type);
            break;
        }
      } catch (error) {
        this.log("Error applying effect:", effect, error);
        this.emit("error", { error: error as Error, context: "effect" });
      }
    }
  }

  private applyRootTemplatePayload(payload: TemplatePayload): void {
    if (typeof document === "undefined") {
      return;
    }

    clearRefRegistry();

    const markup = this.getTemplateMarkup(payload, "__root__");

    clearPatcherCaches();
    dom.reset();

    if (document.body) {
      document.body.innerHTML = markup;
    } else if (document.documentElement) {
      document.documentElement.innerHTML = markup;
    }

    const rootContainer: ParentNode = (document.body ?? document) as ParentNode;
    clearHandlers();
    if (payload.handlers && Object.keys(payload.handlers).length > 0) {
      registerHandlers(payload.handlers);
    }
    syncEventListeners();

    primeSlotBindings(payload.bindings?.slots ?? null);

    if (payload.refs?.del) {
      unregisterRefMetadata(payload.refs.del);
    }
    if (payload.refs?.add) {
      registerRefMetadata(payload.refs.add);
    }

    resetComponentRanges();
    const childCount = rootContainer.childNodes.length;
    const range: import("./componentRanges").ComponentRange = {
      container: rootContainer,
      startIndex: 0,
      endIndex: childCount > 0 ? childCount - 1 : -1,
    };
    const overrides = applyComponentRanges(payload.componentPaths, { root: document });
    this.registerTemplateAnchors(range, payload, overrides);
    const bindingOptions = { overrides, fallbackRange: range };
    applyRouterBindings(payload.bindings?.router ?? null, bindingOptions);
    applyRefBindings(payload.bindings?.refs ?? null, bindingOptions);
    primeUploadBindings(payload.bindings?.uploads ?? null, overrides);
  }

  private applyComponentTemplatePayload(
    payload: TemplatePayload,
    scope: TemplateScope,
  ): void {
    if (typeof document === "undefined") {
      return;
    }
    if (!scope || !scope.componentId) {
      return;
    }

    const range = this.findComponentRange(scope.componentId);
    if (!range) {
      if (this.options.debug) {
        console.warn(
          `liveui: component ${scope.componentId} range not found for template payload`,
        );
      }
      return;
    }

    const slotIds = this.extractSlotIds(payload.slots);
    for (const slot of slotIds) {
      dom.unregisterSlot(slot);
    }

    const listSlotIds = this.extractListSlotIds(payload.listPaths);
    for (const slot of listSlotIds) {
      dom.unregisterList(slot);
    }

    clearPatcherCaches();

    const template = document.createElement("template");
    const markup = this.getTemplateMarkup(payload, scope.componentId);
    template.innerHTML = markup;
    const fragment = template.content.cloneNode(true);

    const insertedNodes: Node[] = [];
    for (let child = fragment.firstChild; child; child = child.nextSibling) {
      insertedNodes.push(child);
    }

    const { container } = range;
    const startIndex = range.startIndex;
    const endIndex = range.endIndex;
    const referenceNode =
      endIndex + 1 < container.childNodes.length
        ? container.childNodes.item(endIndex + 1)
        : null;

    for (let index = endIndex; index >= startIndex; index--) {
      const node = container.childNodes.item(index);
      if (node) {
        if (node instanceof Element || node instanceof DocumentFragment) {
          unbindRefsInTree(node);
        }
        container.removeChild(node);
      }
    }

    container.insertBefore(fragment, referenceNode);

    const insertedCount = insertedNodes.length;
    let refreshed = registerComponentRange(scope.componentId, {
      container,
      startIndex,
      endIndex: insertedCount > 0 ? startIndex + insertedCount - 1 : startIndex - 1,
    });
    if (!refreshed) {
      refreshed = {
        container,
        startIndex,
        endIndex: insertedCount > 0 ? startIndex + insertedCount - 1 : startIndex - 1,
      };
    }

    const overrides = applyComponentRanges(payload.componentPaths, {
      baseId: scope.componentId,
      baseRange: refreshed,
    });

    this.registerTemplateAnchors(refreshed, payload, overrides);
    const bindingOptions = { overrides, fallbackRange: refreshed };
    applyRouterBindings(payload.bindings?.router ?? null, bindingOptions);
    applyRefBindings(payload.bindings?.refs ?? null, bindingOptions);
    replaceUploadBindingsForComponent(scope.componentId, payload.bindings?.uploads ?? null, overrides);

    if (payload.handlers && Object.keys(payload.handlers).length > 0) {
      registerHandlers(payload.handlers);
      syncEventListeners();
    }

    if (payload.bindings?.slots) {
      for (const [slotKey, specs] of Object.entries(payload.bindings.slots)) {
        const slotId = Number(slotKey);
        if (!Number.isInteger(slotId) || slotId < 0) {
          continue;
        }
        registerBindingsForSlot(slotId, Array.isArray(specs) ? specs : []);
      }
    }

    if (payload.refs?.del) {
      unregisterRefMetadata(payload.refs.del);
    }
    if (payload.refs?.add) {
      registerRefMetadata(payload.refs.add);
    }

  }

  private handleDOMRequest(msg: DOMRequestMessage): void {
    if (!msg || !msg.id) {
      return;
    }

    const refId = msg.ref ?? "";
    const selectors = Array.isArray(msg.props) ? msg.props : [];
    const response: DOMResponseMessage = { t: "domres", id: msg.id };

    if (!refId) {
      response.error = "missing_ref";
      this.sendDOMResponse(response);
      return;
    }

    const element = getRefElement(refId);
    if (!element) {
      response.error = "not_found";
      this.sendDOMResponse(response);
      return;
    }

    // Handle method call if specified
    if (msg.method) {
      try {
        const method = msg.method.trim();
        if (!method) {
          response.error = "empty_method";
          this.sendDOMResponse(response);
          return;
        }

        // Get the method from the element
        const fn = (element as any)[method];
        if (typeof fn !== "function") {
          response.error = "method_not_found";
          this.sendDOMResponse(response);
          return;
        }

        // Call the method with provided arguments
        const args = Array.isArray(msg.args) ? msg.args : [];
        const result = fn.apply(element, args);
        response.result = result;
      } catch (error) {
        response.error = error instanceof Error ? error.message : String(error);
        this.sendDOMResponse(response);
        return;
      }
    }

    // Handle property selectors if specified
    if (selectors.length > 0) {
      try {
        const values = domGetSync(selectors, {
          event: null,
          target: element,
          handlerElement: element,
          refElement: element,
        });
        if (values && Object.keys(values).length > 0) {
          response.values = values;
        } else {
          response.values = {};
        }
      } catch (error) {
        response.error = error instanceof Error ? error.message : String(error);
        this.sendDOMResponse(response);
        return;
      }
    }

    this.sendDOMResponse(response);
  }

  private applyDOMActionEffect(effect: DOMActionEffect): void {
    if (typeof document === "undefined") {
      return;
    }

    const kind = (effect.kind ?? effect.Kind ?? "").toLowerCase();
    const refId = (effect.ref ?? effect.Ref ?? "").trim();

    if (!kind) {
      this.reportDOMActionFailure(effect, "missing_kind");
      return;
    }
    if (!refId) {
      this.reportDOMActionFailure(effect, "missing_ref");
      return;
    }

    switch (kind) {
      case "dom.call": {
        const method = (effect.method ?? effect.Method ?? "").trim();
        if (!method) {
          this.reportDOMActionFailure(effect, "missing_method");
          return;
        }
        const args = this.normalizeDOMActionArgs(effect);
        const result = callElementMethod(refId, method, args);
        if (!result.ok) {
          this.reportDOMActionFailure(
            effect,
            result.reason ?? "call_failed",
            { method, argsLength: args.length },
            result.error,
          );
        }
        break;
      }
      case "dom.set": {
        const prop = (effect.prop ?? effect.Prop ?? "").trim();
        if (!prop) {
          this.reportDOMActionFailure(effect, "missing_property");
          return;
        }
        const value = effect.value ?? effect.Value;
        const result = setElementProperty(refId, prop, value);
        if (!result.ok) {
          this.reportDOMActionFailure(
            effect,
            result.reason ?? "set_failed",
            { prop },
            result.error,
          );
        }
        break;
      }
      case "dom.toggle": {
        const prop = (effect.prop ?? effect.Prop ?? "").trim();
        if (!prop) {
          this.reportDOMActionFailure(effect, "missing_property");
          return;
        }
        const value = effect.value ?? effect.Value;
        if (typeof value !== "boolean") {
          this.reportDOMActionFailure(effect, "invalid_toggle_value", { prop, value });
          return;
        }
        const result = setBooleanProperty(refId, prop, value);
        if (!result.ok) {
          this.reportDOMActionFailure(
            effect,
            result.reason ?? "toggle_failed",
            { prop, value },
            result.error,
          );
        }
        break;
      }
      case "dom.class": {
        const className = (effect["class"] ?? effect.Class ?? "").trim();
        if (!className) {
          this.reportDOMActionFailure(effect, "missing_class");
          return;
        }
        const toggle = effect.on ?? effect.On;
        const result = toggleElementClass(refId, className, toggle);
        if (!result.ok) {
          this.reportDOMActionFailure(
            effect,
            result.reason ?? "class_failed",
            { className, toggle },
            result.error,
          );
        }
        break;
      }
      case "dom.scroll": {
        const options: ScrollIntoViewOptions = {};
        const behavior = effect.behavior ?? effect.Behavior;
        const block = effect.block ?? effect.Block;
        const inline = effect.inline ?? effect.Inline;
        if (behavior) {
          options.behavior = behavior;
        }
        if (block) {
          options.block = block;
        }
        if (inline) {
          options.inline = inline;
        }
        const result = scrollElementIntoView(
          refId,
          Object.keys(options).length > 0 ? options : null,
        );
        if (!result.ok) {
          this.reportDOMActionFailure(
            effect,
            result.reason ?? "scroll_failed",
            undefined,
            result.error,
          );
        }
        break;
      }
      default:
        this.reportDOMActionFailure(effect, "unknown_kind", { kind });
    }
  }

  private applyLegacyDOMCallEffect(effect: DOMCallEffect): void {
    const normalizedArgs = Array.isArray(effect.args)
      ? effect.args
      : Array.isArray(effect.Args)
      ? effect.Args
      : effect.args !== undefined
      ? [effect.args]
      : effect.Args !== undefined
      ? [effect.Args]
      : undefined;

    const normalized: DOMActionEffect = {
      type: "dom",
      kind: "dom.call",
      ref: effect.ref ?? effect.Ref ?? undefined,
      method: effect.method ?? effect.Method ?? undefined,
      args: normalizedArgs,
    };

    this.applyDOMActionEffect(normalized);
  }

  private normalizeDOMActionArgs(effect: DOMActionEffect): any[] {
    if (Array.isArray(effect.args)) {
      return effect.args;
    }
    if (Array.isArray(effect.Args)) {
      return effect.Args;
    }
    if (effect.args !== undefined) {
      return [effect.args];
    }
    if (effect.Args !== undefined) {
      return [effect.Args];
    }
    return [];
  }

  private reportDOMActionFailure(
    effect: DOMActionEffect,
    reason: string,
    metadata?: Record<string, unknown>,
    error?: unknown,
  ): void {
    const kind = effect.kind ?? effect.Kind ?? "";
    const refId = (effect.ref ?? effect.Ref ?? "") ?? "";
    const message = `DOM action ${kind || "unknown"} failed: ${reason}`;

    const detailsMetadata: Record<string, unknown> = {
      kind,
      ref: refId,
    };
    if (metadata) {
      Object.assign(detailsMetadata, metadata);
    }
    if (error instanceof Error) {
      detailsMetadata.error = error.message;
      if (error.stack) {
        detailsMetadata.stack = error.stack;
      }
    } else if (error !== undefined) {
      detailsMetadata.error = error;
    }

    console.warn("[LiveUI]", message, { effect, metadata: detailsMetadata });

    const diagnostic: DiagnosticMessage = {
      t: "diagnostic",
      sid: this.sessionId.get() ?? "",
      code: "dom_action_failed",
      message,
      details: {
        phase: "dom_action",
        metadata: detailsMetadata,
      },
    };

    this.recordDiagnostic(diagnostic);
  }

  private sendDOMResponse(payload: DOMResponseMessage): void {
    if (!this.channel) {
      this.log("Cannot send DOM response without channel", payload);
      return;
    }
    try {
      this.channel.sendMessage("domres", payload);
    } catch (error) {
      this.log("Failed to send DOM response", error);
    }
  }

  private applyScrollEffect(effect: ScrollEffect): void {
    const behavior = effect.behavior || "smooth";
    const selector = effect.selector;

    if (!selector || effect.type === "ScrollTop") {
      window.scrollTo({ top: 0, behavior });
      return;
    }

    const element = document.querySelector(selector);
    if (element instanceof Element) {
      element.scrollIntoView({
        behavior,
        block: effect.block || "start",
      });
    }
  }

  private applyMetadataEffect(effect: MetadataEffect): void {
    if (typeof document === "undefined") {
      return;
    }

    if (
      Object.prototype.hasOwnProperty.call(effect, "title") &&
      effect.title !== undefined
    ) {
      document.title = effect.title;
    }

    const head = document.head;
    if (!head) {
      return;
    }

    const keyAttr = "data-live-key";
    const typeAttr = "data-live-head";

    const removeByKeys = (keys?: string[]) => {
      if (!Array.isArray(keys) || keys.length === 0) {
        return;
      }
      keys.forEach((key) => {
        head.querySelectorAll(`[${keyAttr}="${key}"]`).forEach((node) => {
          node.parentNode?.removeChild(node);
        });
      });
    };

    const resetAttributes = (el: Element, preserve: string[]) => {
      const keep = new Set(preserve);
      Array.from(el.attributes).forEach((attr) => {
        if (!keep.has(attr.name)) {
          el.removeAttribute(attr.name);
        }
      });
    };

    const upsertMeta = (payload: MetadataTagPayload, typeValue: string) => {
      if (!payload || !payload.key) {
        return;
      }
      let element = head.querySelector(
        `meta[${keyAttr}="${payload.key}"]`,
      ) as HTMLMetaElement | null;
      if (!element) {
        element = document.createElement("meta");
        head.appendChild(element);
      }
      resetAttributes(element, [keyAttr]);
      element.setAttribute(keyAttr, payload.key);
      element.setAttribute(typeAttr, typeValue);
      if (payload.name !== undefined)
        element.setAttribute("name", payload.name);
      if (payload.content !== undefined)
        element.setAttribute("content", payload.content);
      if (payload.property !== undefined)
        element.setAttribute("property", payload.property);
      if (payload.charset !== undefined)
        element.setAttribute("charset", payload.charset);
      if (payload.httpEquiv !== undefined)
        element.setAttribute("http-equiv", payload.httpEquiv);
      if (payload.itemProp !== undefined)
        element.setAttribute("itemprop", payload.itemProp);
      if (payload.attrs) {
        for (const [k, v] of Object.entries(payload.attrs)) {
          if (v != null) {
            element.setAttribute(k, String(v));
          }
        }
      }
    };

    const upsertLink = (payload: MetadataLinkPayload) => {
      if (!payload || !payload.key) {
        return;
      }
      let element = head.querySelector(
        `link[${keyAttr}="${payload.key}"]`,
      ) as HTMLLinkElement | null;
      if (!element) {
        element = document.createElement("link");
        head.appendChild(element);
      }
      resetAttributes(element, [keyAttr]);
      element.setAttribute(keyAttr, payload.key);
      element.setAttribute(typeAttr, "link");
      if (payload.rel !== undefined) element.setAttribute("rel", payload.rel);
      if (payload.href !== undefined)
        element.setAttribute("href", payload.href);
      if (payload.type !== undefined)
        element.setAttribute("type", payload.type);
      if (payload.as !== undefined) element.setAttribute("as", payload.as);
      if (payload.media !== undefined)
        element.setAttribute("media", payload.media);
      if (payload.hreflang !== undefined)
        element.setAttribute("hreflang", payload.hreflang);
      if (payload.title !== undefined)
        element.setAttribute("title", payload.title);
      if (payload.crossorigin !== undefined)
        element.setAttribute("crossorigin", payload.crossorigin);
      if (payload.integrity !== undefined)
        element.setAttribute("integrity", payload.integrity);
      if (payload.referrerpolicy !== undefined)
        element.setAttribute("referrerpolicy", payload.referrerpolicy);
      if (payload.sizes !== undefined)
        element.setAttribute("sizes", payload.sizes);
      if (payload.attrs) {
        for (const [k, v] of Object.entries(payload.attrs)) {
          if (v != null) {
            element.setAttribute(k, String(v));
          }
        }
      }
    };

    const upsertScript = (payload: MetadataScriptPayload) => {
      if (!payload || !payload.key) {
        return;
      }
      let element = head.querySelector(
        `script[${keyAttr}="${payload.key}"]`,
      ) as HTMLScriptElement | null;
      if (!element) {
        element = document.createElement("script");
        head.appendChild(element);
      }
      resetAttributes(element, [keyAttr]);
      element.setAttribute(keyAttr, payload.key);
      element.setAttribute(typeAttr, "script");

      if (payload.module) {
        element.setAttribute("type", "module");
      } else if (payload.type !== undefined) {
        element.setAttribute("type", payload.type);
      }
      if (payload.src !== undefined) element.setAttribute("src", payload.src);
      if (payload.async) element.setAttribute("async", "async");
      if (payload.defer) element.setAttribute("defer", "defer");
      if (payload.noModule) element.setAttribute("nomodule", "nomodule");
      if (payload.crossorigin !== undefined)
        element.setAttribute("crossorigin", payload.crossorigin);
      if (payload.integrity !== undefined)
        element.setAttribute("integrity", payload.integrity);
      if (payload.referrerpolicy !== undefined)
        element.setAttribute("referrerpolicy", payload.referrerpolicy);
      if (payload.nonce !== undefined)
        element.setAttribute("nonce", payload.nonce);
      if (payload.attrs) {
        for (const [k, v] of Object.entries(payload.attrs)) {
          if (v != null) {
            element.setAttribute(k, String(v));
          }
        }
      }
      if (payload.inner !== undefined) {
        element.textContent = payload.inner;
      }
    };

    if (effect.clearDescription) {
      removeByKeys(["description"]);
    }
    if (
      Object.prototype.hasOwnProperty.call(effect, "description") &&
      effect.description !== undefined
    ) {
      const content = effect.description;
      if (content.trim().length > 0) {
        upsertMeta(
          { key: "description", name: "description", content },
          "description",
        );
      } else {
        removeByKeys(["description"]);
      }
    }

    if (Array.isArray(effect.metaRemove)) {
      removeByKeys(effect.metaRemove);
    }
    if (Array.isArray(effect.metaAdd)) {
      for (const payload of effect.metaAdd) {
        upsertMeta(payload, "meta");
      }
    }

    if (Array.isArray(effect.linkRemove)) {
      removeByKeys(effect.linkRemove);
    }
    if (Array.isArray(effect.linkAdd)) {
      for (const payload of effect.linkAdd) {
        upsertLink(payload);
      }
    }

    if (Array.isArray(effect.scriptRemove)) {
      removeByKeys(effect.scriptRemove);
    }
    if (Array.isArray(effect.scriptAdd)) {
      for (const payload of effect.scriptAdd) {
        upsertScript(payload);
      }
    }
  }

  private handleCookieEffect(effect: CookieEffect): void {
    void this.performCookieSync(effect);
  }

  private async performCookieSync(effect: CookieEffect): Promise<void> {
    const token = effect.token ?? effect.Token;
    if (!token) {
      this.log("Cookie effect missing token", effect);
      return;
    }
    if (this.cookieRequests.has(token)) {
      return;
    }

    const endpoint = effect.endpoint ?? effect.Endpoint ?? DEFAULT_COOKIE_ENDPOINT;
    if (!endpoint) {
      this.log("Cookie effect missing endpoint", effect);
      return;
    }

    const sid = effect.sid ?? effect.SID ?? this.sessionId.get();
    if (!sid) {
      this.log("Cookie effect missing session identifier", effect);
      return;
    }

    const method = (effect.method ?? effect.Method ?? "POST").toUpperCase();

    this.cookieRequests.add(token);
    try {
      const response = await fetch(endpoint, {
        method,
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify({ sid, token }),
      });

      if (!response.ok && response.status !== 204) {
        this.log("Cookie negotiation failed", {
          endpoint,
          status: response.status,
          statusText: response.statusText,
        });
      }
    } catch (error) {
      this.log("Cookie negotiation error", error);
      this.emit("error", { error: error as Error, context: "cookies" });
    } finally {
      this.cookieRequests.delete(token);
    }
  }

  private applyFocusEffect(effect: FocusEffect): void {
    // Support both lowercase and capitalized field names
    const selector = effect.selector || effect.Selector;
    if (!selector) return;

    const element = document.querySelector(selector);
    if (
      element &&
      (element instanceof HTMLElement || element instanceof SVGElement) &&
      typeof element.focus === "function"
    ) {
      element.focus();
    }
  }

  private applyToastEffect(effect: ToastEffect): void {
    // Support both lowercase and capitalized field names
    const message = effect.message || effect.Message;
    const duration = effect.duration || 3000;
    const variant = effect.variant || "info";

    // Emit toast event for custom toast implementations
    this.emit("effect", {
      effect: {
        type: "toast",
        message,
        duration,
        variant,
      },
    });

    // Fallback to console if no toast handler
    if (!this.listenerCount("effect")) {
      console.info(`[Toast ${variant}] ${message}`);
    }
  }

  private applyPushEffect(effect: PushEffect): void {
    // Support both lowercase and capitalized field names
    const url = effect.url || effect.URL;
    if (url) {
      window.history.pushState({}, "", url);
    }
  }

  private applyReplaceEffect(effect: ReplaceEffect): void {
    // Support both lowercase and capitalized field names
    const url = effect.url || effect.URL;
    if (url) {
      window.history.replaceState({}, "", url);
    }
  }

  private applyBootEffect(effect: any): void {
    const boot = effect?.boot as BootPayload | undefined;
    if (!boot) {
      return;
    }
    this.applyRootTemplatePayload(boot);
    this.bootHandler.load(boot);
  }

  private applyComponentBootEffect(effect: ComponentBootEffect): void {
    if (typeof document === "undefined") return;
    if (!effect || !effect.componentId) return;

    const { componentId, html, slots, listSlots, bindings, slotPaths, listPaths } = effect;
    const bounds = this.findComponentRange(componentId);
    if (!bounds) {
      if (this.options.debug) {
        console.warn(`liveui: component ${componentId} bounds not found for componentBoot effect`);
      }
      return;
    }

    if (Array.isArray(slots)) {
      for (const slot of slots) {
        dom.unregisterSlot(slot);
      }
    }
    if (Array.isArray(listSlots)) {
      for (const slot of listSlots) {
        dom.unregisterList(slot);
      }
    }

    // Reset any cached patcher state tied to the replaced component subtree so
    // future diffs don't reference stale DOM nodes from before the boot
    // replacement.
    clearPatcherCaches();

    const template = document.createElement("template");
    template.innerHTML = html || "";
    const fragment = template.content.cloneNode(true);

    const insertedNodes: Node[] = [];
    for (let child = fragment.firstChild; child; child = child.nextSibling) {
      insertedNodes.push(child);
    }

    const container = bounds.container;
    const afterIndex = bounds.endIndex + 1;
    const referenceNode =
      afterIndex < container.childNodes.length ? container.childNodes.item(afterIndex) : null;

    for (let index = bounds.endIndex; index >= bounds.startIndex; index--) {
      const node = container.childNodes.item(index);
      if (node) {
        container.removeChild(node);
      }
    }

    container.insertBefore(fragment, referenceNode ?? null);

    const insertedCount = insertedNodes.length;
    let refreshed: import("./componentRanges").ComponentRange | null = null;
    if (insertedCount > 0) {
      const endIndex = bounds.startIndex + insertedCount - 1;
      refreshed = registerComponentRange(componentId, {
        container,
        startIndex: bounds.startIndex,
        endIndex,
      });
    } else {
      refreshed = registerComponentRange(componentId, {
        container,
        startIndex: bounds.startIndex,
        endIndex: bounds.startIndex - 1,
      });
    }
    if (!refreshed) {
      refreshed = { container, startIndex: bounds.startIndex, endIndex: bounds.startIndex - 1 };
    }

    const rangeOverrides = applyComponentRanges(effect.componentPaths, {
      baseId: componentId,
      baseRange: refreshed,
    });

    const slotBindingTable = (bindings as any)?.slots ?? bindings;
    if (slotBindingTable && typeof slotBindingTable === "object") {
      for (const [slotKey, specs] of Object.entries(slotBindingTable)) {
        const slotId = Number(slotKey);
        if (Number.isNaN(slotId)) {
          continue;
        }
        registerBindingsForSlot(slotId, Array.isArray(specs) ? specs : []);
      }
    }

    this.registerComponentAnchors(
      refreshed,
      Array.isArray(slots) ? slots : [],
      Array.isArray(listSlots) ? listSlots : [],
      Array.isArray(slotPaths) ? slotPaths : undefined,
      Array.isArray(listPaths) ? listPaths : undefined,
      rangeOverrides,
    );

    const uploadBindings = (bindings as any)?.uploads ?? bindings?.uploads ?? null;
    replaceUploadBindingsForComponent(componentId, uploadBindings, rangeOverrides);

    if (Array.isArray(slotPaths) && slotPaths.length > 0) {
      const anchors = resolveSlotAnchors(slotPaths, rangeOverrides);
      anchors.forEach((node, slotId) => dom.registerSlot(slotId, node));
    }
  }

  private registerTemplateAnchors(
    range: import("./componentRanges").ComponentRange | null,
    payload: TemplatePayload,
    overrides?: Map<string, import("./componentRanges").ComponentRange>,
  ): void {
    if (!range) {
      return;
    }
    const slotIds = this.extractSlotIds(payload.slots);
    const listSlotIds = this.extractListSlotIds(payload.listPaths);
    this.registerComponentAnchors(
      range,
      slotIds,
      listSlotIds,
      payload.slotPaths,
      payload.listPaths,
      overrides,
    );
  }

  private extractSlotIds(slots?: TemplatePayload["slots"] | null): number[] {
    if (!Array.isArray(slots)) {
      return [];
    }
    const ids: number[] = [];
    for (const meta of slots) {
      const id = Number(meta?.anchorId);
      if (Number.isInteger(id) && id >= 0) {
        ids.push(id);
      }
    }
    return ids;
  }

  private extractListSlotIds(
    descriptors?: TemplatePayload["listPaths"] | null,
  ): number[] {
    if (!Array.isArray(descriptors)) {
      return [];
    }
    const seen = new Set<number>();
    const ids: number[] = [];
    for (const descriptor of descriptors) {
      const id = Number(descriptor?.slot);
      if (!Number.isInteger(id) || id < 0 || seen.has(id)) {
        continue;
      }
      seen.add(id);
      ids.push(id);
    }
    return ids;
  }

  private getTemplateMarkup(payload: TemplatePayload, scopeKey?: string): string {
    const scope = scopeKey && scopeKey.length > 0 ? scopeKey : "__root__";
    const templateHash = typeof payload.templateHash === "string" ? payload.templateHash : "";

    if (typeof payload.html === "string") {
      this.cacheTemplateMarkup(scope, templateHash, payload.html);
      return payload.html;
    }

    if (templateHash) {
      const cached = this.lookupTemplateMarkup(scope, templateHash);
      if (cached !== null) {
        return cached;
      }
    }

    if (Array.isArray(payload.s) && payload.s.length > 0) {
      const markup = payload.s.join("");
      this.cacheTemplateMarkup(scope, templateHash, markup);
      return markup;
    }

    this.cacheTemplateMarkup(scope, templateHash, "");
    return "";
  }

  private buildTemplateCacheKey(scope: string, hash: string): string {
    return `${scope}::${hash}`;
  }

  private cacheTemplateMarkup(scope: string, hash: string, markup: string): void {
    if (!hash) {
      return;
    }
    const key = this.buildTemplateCacheKey(scope, hash);
    if (this.templateCache.has(key)) {
      this.templateCache.delete(key);
    }
    this.templateCache.set(key, markup);
    if (this.templateCache.size > this.MAX_TEMPLATE_CACHE_SIZE) {
      const oldest = this.templateCache.keys().next();
      if (!oldest.done) {
        this.templateCache.delete(oldest.value);
      }
    }
  }

  private lookupTemplateMarkup(scope: string, hash: string): string | null {
    if (!hash) {
      return null;
    }
    const key = this.buildTemplateCacheKey(scope, hash);
    if (!this.templateCache.has(key)) {
      return null;
    }
    const cached = this.templateCache.get(key);
    return cached !== undefined ? cached : null;
  }

  private findComponentRange(id: string): import("./componentRanges").ComponentRange | null {
    if (typeof document === "undefined" || !id) return null;
    return getComponentRange(id);
  }

  private registerComponentAnchors(
    range: import("./componentRanges").ComponentRange,
    slots: number[],
    listSlots: number[],
    slotDescriptors?: import("./types").SlotPathDescriptor[],
    listDescriptors?: import("./types").ListPathDescriptor[],
    overrides?: Map<string, import("./componentRanges").ComponentRange>,
  ): void {
    if (!range || range.endIndex < range.startIndex) {
      return;
    }
    const requestedSlots = Array.isArray(slots)
      ? slots.filter((value) => Number.isInteger(value) && value >= 0)
      : [];
    const requestedLists = Array.isArray(listSlots)
      ? listSlots.filter((value) => Number.isInteger(value) && value >= 0)
      : [];

    const pendingSlots = new Set<number>(requestedSlots);
    const pendingLists = new Set<number>(requestedLists);

    const slotAnchors = resolveSlotAnchors(slotDescriptors, overrides);
    if (slotAnchors.size > 0) {
      if (pendingSlots.size > 0) {
        for (const slotId of Array.from(pendingSlots)) {
          const node = slotAnchors.get(slotId);
          if (node) {
            dom.registerSlot(slotId, node);
            pendingSlots.delete(slotId);
          }
        }
      } else {
        for (const [slotId, node] of slotAnchors.entries()) {
          dom.registerSlot(slotId, node);
        }
      }
    }

    const listAnchors = resolveListContainers(listDescriptors, overrides);
    if (listAnchors.size > 0) {
      if (pendingLists.size > 0) {
        for (const slotId of Array.from(pendingLists)) {
          const element = listAnchors.get(slotId);
          if (element) {
            dom.registerList(slotId, element);
            pendingLists.delete(slotId);
          }
        }
      } else {
        for (const [slotId, element] of listAnchors.entries()) {
          dom.registerList(slotId, element);
        }
      }
    }
  }
  private dispatchCustomEvent(eventName: string, detail?: unknown): void {
    const event = new CustomEvent(eventName, {
      detail,
      bubbles: true,
      cancelable: true,
    });
    document.dispatchEvent(event);
  }

  /**
   * Schedule a batch to be applied on the next animation frame
   */
  private scheduleBatch(): void {
    if (this.batchScheduled) return;

    this.batchScheduled = true;

    if (typeof requestAnimationFrame === "function") {
      this.rafUsesTimeoutFallback = false;
      this.rafHandle = requestAnimationFrame(() => {
        this.rafHandle = null;
        this.flushBatch();
      });
      return;
    }

    this.rafUsesTimeoutFallback = true;
    const handle = setTimeout(() => {
      this.rafHandle = null;
      this.flushBatch();
    }, 16);
    this.rafHandle = handle;
  }

  /**
   * Apply all queued patches in a single batch
   */
  private flushBatch(): void {
    if (this.patchQueue.length === 0) {
      this.batchScheduled = false;
      this.rafHandle = null;
      this.rafUsesTimeoutFallback = false;
      if (this.pendingUploadBindings.length > 0) {
        this.flushPendingUploadBindings();
      }
      return;
    }

    const startTime = performance.now();

    this.log("Applying batched", this.patchQueue.length, "operations");
    const patches = this.patchQueue;
    this.patchQueue = [];
    this.batchScheduled = false;
    this.rafHandle = null;
    this.rafUsesTimeoutFallback = false;

    applyOps(patches);

    this.flushPendingUploadBindings();

    const duration = performance.now() - startTime;

    // Update metrics
    this.metrics.patchesApplied += patches.length;
    this.patchTimes.push(duration);
    this.patchTimeTotal += duration;
    if (this.patchTimes.length > 100) {
      const removed = this.patchTimes.shift();
      if (removed !== undefined) {
        this.patchTimeTotal -= removed;
      }
    }
    const windowLength = this.patchTimes.length;
    this.metrics.averagePatchTime = windowLength
      ? this.patchTimeTotal / windowLength
      : 0;

    this.emit("frameApplied", { operations: patches.length, duration });
    this.emit("metricsUpdated", this.getMetrics());
  }

  private cancelScheduledBatch(): void {
    if (this.rafHandle === null) {
      return;
    }

    if (this.rafUsesTimeoutFallback) {
      clearTimeout(this.rafHandle as ReturnType<typeof setTimeout>);
    } else if (typeof cancelAnimationFrame === "function") {
      cancelAnimationFrame(this.rafHandle as number);
    }

    this.rafHandle = null;
    this.batchScheduled = false;
    this.rafUsesTimeoutFallback = false;
  }

  private flushPendingUploadBindings(): void {
    if (this.pendingUploadBindings.length === 0) {
      return;
    }

    const pending = this.pendingUploadBindings;
    this.pendingUploadBindings = [];

    const descriptors: UploadBindingDescriptor[] = [];
    for (const group of pending) {
      if (!Array.isArray(group) || group.length === 0) {
        continue;
      }
      descriptors.push(...group);
    }

    if (descriptors.length > 0) {
      applyUploadBindings(descriptors);
    }
  }

  private hasStateChanged(
    oldState: ConnectionState,
    newState: ConnectionState,
  ): boolean {
    if (oldState.status !== newState.status) {
      return true;
    }

    switch (newState.status) {
      case "connected":
        return (
          oldState.status !== "connected" ||
          oldState.sessionId !== newState.sessionId ||
          oldState.version !== newState.version
        );
      case "reconnecting":
        return (
          oldState.status !== "reconnecting" ||
          oldState.attempt !== newState.attempt
        );
      case "error":
        return oldState.status !== "error" || oldState.error !== newState.error;
      default:
        return false;
    }
  }

  /**
   * Handle join message
   */
  private handleJoin(msg: JoinMessage): void {
    this.sessionId.set(msg.sid);
    this.version.set(msg.ver);
    if (typeof msg.ack === "number" && msg.ack > this.lastAck) {
      this.lastAck = msg.ack;
    }
    this.log("Joined session:", msg.sid, "ack:", msg.ack);
  }

  /**
   * Handle resume message
   */
  private handleResume(msg: ResumeMessage): void {
    this.log("Resume from", msg.from, "to", msg.to);
    const ack = msg.from > 0 ? msg.from - 1 : 0;
    if (ack > this.lastAck) {
      this.lastAck = ack;
    }

    // Reset sequence tracking - expect frames starting from msg.from
    this.expectedSeq = msg.from;
    this.frameBuffer.clear();

    if (Array.isArray(msg.errors) && msg.errors.length > 0) {
      for (const err of msg.errors) {
        this.recordDiagnostic(err);
      }
    }
    this.emit("resumed", { from: msg.from, to: msg.to });
    if (this.isReconnecting) {
      this.flushEventQueue();
    }
  }

  /**
   * Handle error message
   */
  private handleError(msg: ErrorMessage): void {
    const error = new Error(msg.message);
    console.error("LiveUI error:", msg.code, msg.message);
    this.emit("error", { error, context: msg.code });
    this.recordDiagnostic(msg);
  }

  private handlePubsub(msg: PubsubControlMessage): void {
    if (!msg.topic) {
      return;
    }
    switch (msg.op) {
      case "join":
        void this.joinPubsubTopic(msg.topic);
        break;
      case "leave":
        void this.leavePubsubTopic(msg.topic);
        break;
      default:
        this.log("Unknown pubsub op:", msg);
    }
  }

  private normalizeDiagnosticMessage(
    msg: ErrorMessage | DiagnosticMessage,
  ): DiagnosticMessage {
    if (!msg) {
      return {
        t: "diagnostic",
        sid: this.sessionId.get() ?? "",
        code: "runtime_panic",
        message: "Runtime diagnostic",
      };
    }
    if (msg.t === "diagnostic") {
      return msg as DiagnosticMessage;
    }
    return {
      t: "diagnostic",
      sid: msg.sid,
      code: msg.code,
      message: msg.message,
      details: msg.details,
    };
  }

  private recordDiagnostic(msg: ErrorMessage | DiagnosticMessage): void {
    const diagnostic = this.normalizeDiagnosticMessage(msg);
    this.emit("diagnostic", { diagnostic });
    if (msg.t === "diagnostic") {
      console.warn(
        "[LiveUI][diagnostic]",
        diagnostic.code,
        diagnostic.message,
        diagnostic.details,
      );
    }
    if (!this.options.debug) {
      return;
    }
    const key = this.buildDiagnosticKey(diagnostic);
    const entry: RuntimeDiagnosticEntry = {
      key,
      code: diagnostic.code ?? "runtime_panic",
      message: diagnostic.message ?? "Runtime panic recovered",
      details: diagnostic.details,
      timestamp: Date.now(),
    };
    const existingIndex = this.diagnostics.findIndex(
      (item) => item.key === key,
    );
    if (existingIndex >= 0) {
      this.diagnostics[existingIndex] = entry;
    } else {
      this.diagnostics.push(entry);
      if (this.diagnostics.length > 20) {
        this.diagnostics.shift();
      }
    }
    if (typeof document !== "undefined") {
      this.renderErrorOverlay();
    }
  }

  private buildDiagnosticKey(msg: ErrorMessage | DiagnosticMessage): string {
    const details = msg.details;
    const component = details?.componentId ?? details?.componentName ?? "";
    const hook = details?.hook ?? "";
    const phase = details?.phase ?? "";
    const panic = details?.panic ?? "";
    return [
      msg.code ?? "",
      msg.message ?? "",
      component,
      hook,
      phase,
      panic,
    ].join("|");
  }

  private joinPubsubTopic(topic: string): void {
    if (!topic || !this.client) {
      return;
    }
    if (this.pubsubChannels.has(topic)) {
      return;
    }

    const channel = this.client.createChannel<LiveUIEventMap>(
      `pubsub/${topic}`,
      {
        sid: this.sessionId.get() ?? undefined,
      },
    );

    const dispose = channel.onLeave(() => {
      this.pubsubChannels.delete(topic);
    });

    this.pubsubChannels.set(topic, { channel, dispose });

    try {
      channel.join();
      this.log("Joined pubsub topic", topic);
    } catch (error) {
      this.pubsubChannels.delete(topic);
      dispose();
      this.log("Failed to join pubsub topic", topic, error);
    }
  }

  private leavePubsubTopic(topic: string): void {
    const entry = this.pubsubChannels.get(topic);
    if (!entry) {
      return;
    }
    this.pubsubChannels.delete(topic);
    try {
      entry.dispose();
      entry.channel.leave();
      this.log("Left pubsub topic", topic);
    } catch (error) {
      this.log("Failed to leave pubsub topic", topic, error);
    }
  }

  private ensureErrorOverlayElements(): void {
    if (!this.options.debug || typeof document === "undefined") {
      return;
    }
    if (this.errorOverlay) {
      return;
    }
    const overlay = document.createElement("div");
    overlay.id = "live-error-overlay";
    overlay.setAttribute("role", "alertdialog");
    overlay.setAttribute("aria-live", "assertive");
    overlay.setAttribute("aria-modal", "true");
    overlay.style.cssText = [
      "position:fixed",
      "inset:0",
      "background:rgba(15,23,42,0.82)",
      "color:#e2e8f0",
      "z-index:2147483646",
      "display:none",
      "align-items:flex-start",
      "justify-content:center",
      "overflow-y:auto",
      "padding:48px 24px",
      "box-sizing:border-box",
      "backdrop-filter:blur(6px)",
    ].join(";");
    overlay.addEventListener("click", (event) => {
      if (event.target === overlay) {
        this.hideErrorOverlay();
      }
    });

    const panel = document.createElement("div");
    panel.style.cssText = [
      "width:min(960px,100%)",
      "background:rgba(15,23,42,0.95)",
      "border-radius:16px",
      "border:1px solid rgba(148,163,184,0.35)",
      "box-shadow:0 25px 50px -12px rgba(15,23,42,0.8)",
      "display:flex",
      "flex-direction:column",
      "overflow:hidden",
      'font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif',
    ].join(";");

    const header = document.createElement("header");
    header.style.cssText = [
      "display:flex",
      "align-items:center",
      "justify-content:space-between",
      "gap:12px",
      "padding:20px 24px",
      "background:rgba(15,23,42,0.88)",
      "border-bottom:1px solid rgba(148,163,184,0.28)",
      "position:sticky",
      "top:0",
      "z-index:1",
    ].join(";");

    const title = document.createElement("h2");
    title.textContent = "LiveUI Runtime Errors";
    title.style.cssText =
      "margin:0;font-size:18px;font-weight:700;color:#f8fafc;";

    const headerMeta = document.createElement("div");
    headerMeta.style.cssText = "display:flex;flex-direction:column;gap:4px;";
    headerMeta.appendChild(title);

    const hint = document.createElement("span");
    hint.textContent = "⇧⌘E / ⇧Ctrl+E to toggle • Esc to dismiss";
    hint.style.cssText = "font-size:12px;color:#94a3b8;";
    headerMeta.appendChild(hint);

    const controls = document.createElement("div");
    controls.style.cssText = "display:flex;align-items:center;gap:8px;";

    const retryButton = document.createElement("button");
    retryButton.type = "button";
    retryButton.textContent = "Retry render";
    retryButton.style.cssText = [
      "padding:6px 12px",
      "border-radius:6px",
      "border:1px solid rgba(59,130,246,0.4)",
      "background:rgba(37,99,235,0.18)",
      "color:#bfdbfe",
      "font-size:12px",
      "font-weight:600",
      "cursor:pointer",
    ].join(";");
    retryButton.addEventListener("click", () => this.requestRecovery());

    const clearButton = document.createElement("button");
    clearButton.type = "button";
    clearButton.textContent = "Clear";
    clearButton.style.cssText = [
      "padding:6px 12px",
      "border-radius:6px",
      "border:1px solid rgba(34,197,94,0.45)",
      "background:rgba(34,197,94,0.18)",
      "color:#bbf7d0",
      "font-size:12px",
      "font-weight:600",
      "cursor:pointer",
    ].join(";");
    clearButton.addEventListener("click", () => this.clearDiagnostics());

    const closeButton = document.createElement("button");
    closeButton.type = "button";
    closeButton.textContent = "Close";
    closeButton.style.cssText = [
      "padding:6px 12px",
      "border-radius:6px",
      "border:1px solid rgba(248,113,113,0.4)",
      "background:rgba(248,113,113,0.18)",
      "color:#fecaca",
      "font-size:12px",
      "font-weight:600",
      "cursor:pointer",
    ].join(";");
    closeButton.addEventListener("click", () => this.hideErrorOverlay());

    controls.appendChild(retryButton);
    controls.appendChild(clearButton);
    controls.appendChild(closeButton);

    header.appendChild(headerMeta);
    header.appendChild(controls);

    const list = document.createElement("div");
    list.style.cssText =
      "padding:16px 24px 24px;display:flex;flex-direction:column;gap:16px;";

    panel.appendChild(header);
    panel.appendChild(list);
    overlay.appendChild(panel);

    document.body.appendChild(overlay);

    this.errorOverlay = overlay;
    this.errorListEl = list;
  }

  private renderErrorOverlay(): void {
    if (!this.options.debug || typeof document === "undefined") {
      return;
    }
    this.ensureErrorOverlayElements();
    if (!this.errorOverlay || !this.errorListEl) {
      return;
    }
    const list = this.errorListEl;
    list.innerHTML = "";
    if (this.diagnostics.length === 0) {
      const empty = document.createElement("div");
      empty.textContent = "No runtime errors captured yet.";
      empty.style.cssText = "font-size:14px;color:#94a3b8;";
      list.appendChild(empty);
    } else {
      const ordered = [...this.diagnostics].sort(
        (a, b) => b.timestamp - a.timestamp,
      );
      for (const entry of ordered) {
        list.appendChild(this.createDiagnosticElement(entry));
      }
    }
    this.errorOverlay.style.display = "flex";
    this.errorOverlayVisible = true;
  }

  private createDiagnosticElement(entry: RuntimeDiagnosticEntry): HTMLElement {
    const container = document.createElement("article");
    container.style.cssText = [
      "background:rgba(15,23,42,0.65)",
      "border:1px solid rgba(148,163,184,0.25)",
      "border-radius:12px",
      "padding:18px 20px",
      "display:flex",
      "flex-direction:column",
      "gap:12px",
    ].join(";");

    const header = document.createElement("div");
    header.style.cssText =
      "display:flex;flex-wrap:wrap;align-items:center;justify-content:space-between;gap:12px;";

    const title = document.createElement("h3");
    title.textContent = entry.message || "Runtime panic recovered";
    title.style.cssText =
      "margin:0;font-size:16px;font-weight:700;color:#f8fafc;";

    const badgeGroup = document.createElement("div");
    badgeGroup.style.cssText = "display:flex;align-items:center;gap:8px;";

    const badge = document.createElement("span");
    badge.textContent = entry.code || "runtime_panic";
    badge.style.cssText = [
      "font-size:11px",
      "font-weight:700",
      "letter-spacing:0.08em",
      "text-transform:uppercase",
      "background:rgba(37,99,235,0.18)",
      "color:#bfdbfe",
      "border:1px solid rgba(37,99,235,0.35)",
      "border-radius:9999px",
      "padding:4px 10px",
    ].join(";");

    const timeEl = document.createElement("time");
    const captured = entry.details?.capturedAt
      ? new Date(entry.details.capturedAt)
      : new Date(entry.timestamp);
    if (!Number.isNaN(captured.getTime())) {
      timeEl.dateTime = captured.toISOString();
      timeEl.textContent = captured.toLocaleString();
    } else {
      timeEl.textContent = new Date(entry.timestamp).toLocaleString();
    }
    timeEl.style.cssText = "font-size:12px;color:#94a3b8;";

    badgeGroup.appendChild(badge);
    badgeGroup.appendChild(timeEl);

    header.appendChild(title);
    header.appendChild(badgeGroup);
    container.appendChild(header);

    const details = entry.details;
    const metaContainer = document.createElement("div");
    metaContainer.style.cssText =
      "display:flex;flex-direction:column;gap:4px;font-size:12px;color:#cbd5f5;";

    const componentLabel =
      details?.componentName && details.componentId
        ? `${details.componentName} (${details.componentId})`
        : (details?.componentName ?? details?.componentId ?? "");
    this.appendMeta(metaContainer, "Component", componentLabel);
    this.appendMeta(metaContainer, "Phase", details?.phase);
    if (details?.hook) {
      const hookInfo =
        details.hookIndex != null
          ? `${details.hook} (#${details.hookIndex})`
          : details.hook;
      this.appendMeta(metaContainer, "Hook", hookInfo);
    }
    this.appendMeta(metaContainer, "Panic", details?.panic);

    if (metaContainer.childNodes.length > 0) {
      container.appendChild(metaContainer);
    }

    if (details?.suggestion) {
      const suggestion = document.createElement("p");
      suggestion.textContent = details.suggestion;
      suggestion.style.cssText =
        "margin:0;padding:12px 14px;border-radius:8px;background:rgba(34,197,94,0.12);color:#bbf7d0;font-size:13px;border-left:3px solid rgba(34,197,94,0.45);";
      container.appendChild(suggestion);
    }

    const metadata = details?.metadata;
    if (metadata && Object.keys(metadata).length > 0) {
      const dl = document.createElement("dl");
      dl.style.cssText =
        "display:grid;grid-template-columns:max-content 1fr;gap:4px 12px;padding:12px;border-radius:8px;background:rgba(15,23,42,0.5);";
      for (const [key, value] of Object.entries(metadata)) {
        const dt = document.createElement("dt");
        dt.textContent = key;
        dt.style.cssText = "font-weight:600;color:#e2e8f0;";
        const dd = document.createElement("dd");
        dd.textContent = this.formatMetadataValue(value);
        dd.style.cssText = "margin:0;white-space:pre-wrap;color:#cbd5f5;";
        dl.appendChild(dt);
        dl.appendChild(dd);
      }
      container.appendChild(dl);
    }

    const componentId =
      typeof details?.componentId === "string"
        ? details.componentId.trim()
        : "";
    if (componentId) {
      const actions = document.createElement("div");
      actions.style.cssText =
        "display:flex;justify-content:flex-end;margin-top:4px;";
      const resetButton = document.createElement("button");
      resetButton.type = "button";
      resetButton.textContent = "Reset component";
      resetButton.title = `Request router reset for ${componentId}`;
      resetButton.style.cssText = [
        "padding:6px 12px",
        "border-radius:6px",
        "border:1px solid rgba(59,130,246,0.45)",
        "background:rgba(37,99,235,0.2)",
        "color:#dbeafe",
        "font-size:12px",
        "font-weight:600",
        "cursor:pointer",
      ].join(";");
      resetButton.addEventListener("click", () => {
        this.requestRouterReset(componentId);
      });
      actions.appendChild(resetButton);
      container.appendChild(actions);
    }

    if (details?.stack) {
      const stack = document.createElement("pre");
      stack.textContent = details.stack.trim();
      stack.style.cssText =
        "margin:0;padding:12px;border-radius:8px;background:#020617;color:#f1f5f9;max-height:220px;overflow:auto;font-size:12px;line-height:1.45;";
      container.appendChild(stack);
    }

    return container;
  }

  private appendMeta(
    container: HTMLElement,
    label: string,
    value?: string | null,
  ): void {
    if (!value) {
      return;
    }
    const row = document.createElement("div");
    row.style.cssText = "display:flex;gap:6px;align-items:flex-start;";
    const strong = document.createElement("span");
    strong.textContent = `${label}:`;
    strong.style.cssText = "font-weight:600;min-width:88px;color:#e2e8f0;";
    const span = document.createElement("span");
    span.textContent = value;
    span.style.cssText = "flex:1;";
    row.appendChild(strong);
    row.appendChild(span);
    container.appendChild(row);
  }

  private formatMetadataValue(value: unknown): string {
    if (value === null || value === undefined) {
      return "null";
    }
    if (typeof value === "string") {
      return value;
    }
    if (typeof value === "number" || typeof value === "boolean") {
      return String(value);
    }
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value);
    }
  }

  private hideErrorOverlay(): void {
    if (!this.errorOverlay) {
      return;
    }
    this.errorOverlay.style.display = "none";
    this.errorOverlayVisible = false;
  }

  private clearDiagnostics(): void {
    this.diagnostics = [];
    if (this.errorListEl) {
      this.errorListEl.innerHTML = "";
    }
    this.hideErrorOverlay();
  }

  private requestRecovery(): void {
    const sid = this.sessionId.get();
    if (!sid || !this.channel) {
      this.log("Cannot request recovery without active session/channel");
      return;
    }
    this.log("Requesting runtime recovery");
    this.channel.sendMessage("recover", { t: "recover", sid });
  }

  private onDebugKeydown(event: KeyboardEvent): void {
    if (!this.options.debug) {
      return;
    }
    if (event.key === "Escape" && this.errorOverlayVisible) {
      this.hideErrorOverlay();
      return;
    }
    if (
      (event.key === "e" || event.key === "E") &&
      event.shiftKey &&
      (event.metaKey || event.ctrlKey)
    ) {
      event.preventDefault();
      this.renderErrorOverlay();
    }
  }

  /**
   * Send event to the server
   */
  private sendEvent(event: { hid: string; payload: EventPayload }): void {
    const state = this.connectionState.get();
    if (state.status !== "connected" || !this.channel) {
      this.log("Not connected, queueing event");
      this.eventQueue.push(event);
      return;
    }

    const msg = {
      t: "evt" as const,
      sid: this.sessionId.get()!,
      hid: event.hid,
      payload: event.payload,
    };

    this.log("Sending event:", msg);
    this.channel.sendMessage("evt", msg);
    this.metrics.eventsProcessed++;
  }

  private sendUploadMessage(payload: UploadClientMessage): void {
    const state = this.connectionState.get();
    if (state.status !== "connected" || !this.channel) {
      this.log("Not connected, skipping upload message");
      return;
    }

    this.channel.sendMessage("upload", payload);
  }

  /**
   * Send navigation message to the server
   * Returns true if navigation was sent, false if not connected
   */
  private sendNavigation(path: string, query: string, hash: string): boolean {
    const state = this.connectionState.get();
    if (state.status !== "connected" || !this.channel) {
      this.log("Not connected, cannot navigate");
      return false;
    }

    const msg = {
      t: "nav" as const,
      sid: this.sessionId.get()!,
      path: path,
      q: query,
      hash: hash,
    };

    this.log("Sending navigation:", msg);
    this.channel.sendMessage("nav", msg);

    // Optimistically update the URL
    const queryPart = query ? `?${query}` : "";
    const hashPart = hash ? `#${hash}` : "";
    window.history.pushState({}, "", `${path}${queryPart}${hashPart}`);

    // Track optimistic navigation time to prevent double pushState
    this.lastOptimisticNavTime = Date.now();

    return true;
  }

  /**
   * Handle browser back/forward navigation
   */
  private handlePopState = (): void => {
    const state = this.connectionState.get();
    if (state.status !== "connected" || !this.channel) {
      this.log("Not connected, cannot handle popstate");
      return;
    }

    const path = window.location.pathname;
    const query = window.location.search.substring(1); // Remove leading '?'
    const hash = window.location.hash.startsWith("#")
      ? window.location.hash.substring(1)
      : window.location.hash;

    const msg = {
      t: "pop" as const,
      sid: this.sessionId.get()!,
      path: path,
      q: query,
      hash: hash,
    };

    this.log("Sending popstate navigation:", msg);
    this.channel.sendMessage("pop", msg);
  };

  public requestRouterReset(componentId: string): void {
    const sid = this.sessionId.get();
    if (!sid || !this.channel) {
      this.log("Cannot request router reset without active session/channel");
      return;
    }

    const trimmed = typeof componentId === "string" ? componentId.trim() : "";
    if (!trimmed) {
      this.log("Cannot request router reset without component id");
      return;
    }

    const payload = {
      t: "routerReset" as const,
      sid,
      componentId: trimmed,
    };

    this.log("Requesting router reset:", payload);
    this.channel.sendMessage("routerReset", payload);
  }

  /**
   * Send acknowledgement to the server
   */
  private sendAck(seq: number): void {
    if (!this.channel) return;

    const msg = {
      t: "ack" as const,
      sid: this.sessionId.get()!,
      seq: seq,
    };

    this.channel.sendMessage("ack", msg);
    if (typeof seq === "number" && seq > this.lastAck) {
      this.lastAck = seq;
    }
  }

  /**
   * Flush queued events
   */
  private flushEventQueue(): void {
    if (this.eventQueue.length === 0) return;

    this.log("Flushing", this.eventQueue.length, "queued events");
    const queue = this.eventQueue;
    this.eventQueue = [];

    for (const event of queue) {
      this.sendEvent(event);
    }
  }

  /**
   * Log helper
   */
  private log(...args: unknown[]): void {
    if (this.options.debug) {
      console.log("[LiveUI]", ...args);
    }
  }
}

export default LiveUI;

export {
  configurePatcher,
  getPatcherConfig,
  clearPatcherCaches,
  getPatcherStats,
  morphElement,
  dom,
  applyOps,
  Signal,
  ComputedSignal,
};

export {
  getRefElement,
  getRefMeta,
  getRefPayload,
  getRefPayloads,
} from "./refs";

export type { BootPayload } from "./types";

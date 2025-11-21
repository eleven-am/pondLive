// LiveUI Client v1.0.0 - Built with esbuild
"use strict";
var LiveUIModule = (() => {
  var __create = Object.create;
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __getProtoOf = Object.getPrototypeOf;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __commonJS = (cb, mod) => function __require() {
    return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
    // If the importer is in node compatibility mode or this is not an ESM
    // file that has been converted to a CommonJS file using a Babel-
    // compatible transform (i.e. "__esModule" has not been set), then set
    // "default" to the CommonJS "module.exports" for node compatibility.
    isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
    mod
  ));

  // node_modules/@eleven-am/pondsocket-common/subjects/subject.js
  var require_subject = __commonJS({
    "node_modules/@eleven-am/pondsocket-common/subjects/subject.js"(exports) {
      "use strict";
      var __classPrivateFieldSet = exports && exports.__classPrivateFieldSet || function(receiver, state, value, kind, f) {
        if (kind === "m") throw new TypeError("Private method is not writable");
        if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
        if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
        return kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value), value;
      };
      var __classPrivateFieldGet = exports && exports.__classPrivateFieldGet || function(receiver, state, kind, f) {
        if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
        if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
        return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
      };
      var _Subject_isClosed;
      var _Subject_observers;
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.Subject = void 0;
      var Subject = class {
        constructor() {
          _Subject_isClosed.set(this, void 0);
          _Subject_observers.set(this, void 0);
          __classPrivateFieldSet(this, _Subject_isClosed, false, "f");
          __classPrivateFieldSet(this, _Subject_observers, /* @__PURE__ */ new Set(), "f");
        }
        /**
         * @desc Returns the number of subscribers
         */
        get size() {
          return __classPrivateFieldGet(this, _Subject_observers, "f").size;
        }
        /**
         * @desc Subscribes to a subject
         * @param observer - The observer to subscribe
         */
        subscribe(observer) {
          if (__classPrivateFieldGet(this, _Subject_isClosed, "f")) {
            throw new Error("Cannot subscribe to a closed subject");
          }
          __classPrivateFieldGet(this, _Subject_observers, "f").add(observer);
          return () => __classPrivateFieldGet(this, _Subject_observers, "f").delete(observer);
        }
        /**
         * @desc Publishes a message to all subscribers
         * @param message - The message to publish
         */
        publish(message) {
          __classPrivateFieldGet(this, _Subject_observers, "f").forEach((observer) => observer(message));
        }
        /**
         * @desc Closes the subject
         */
        close() {
          __classPrivateFieldGet(this, _Subject_observers, "f").clear();
          __classPrivateFieldSet(this, _Subject_isClosed, true, "f");
        }
      };
      exports.Subject = Subject;
      _Subject_isClosed = /* @__PURE__ */ new WeakMap(), _Subject_observers = /* @__PURE__ */ new WeakMap();
    }
  });

  // node_modules/@eleven-am/pondsocket-common/subjects/behaviorSubject.js
  var require_behaviorSubject = __commonJS({
    "node_modules/@eleven-am/pondsocket-common/subjects/behaviorSubject.js"(exports) {
      "use strict";
      var __classPrivateFieldSet = exports && exports.__classPrivateFieldSet || function(receiver, state, value, kind, f) {
        if (kind === "m") throw new TypeError("Private method is not writable");
        if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
        if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
        return kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value), value;
      };
      var __classPrivateFieldGet = exports && exports.__classPrivateFieldGet || function(receiver, state, kind, f) {
        if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
        if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
        return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
      };
      var _BehaviorSubject_lastMessage;
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.BehaviorSubject = void 0;
      var subject_1 = require_subject();
      var BehaviorSubject = class extends subject_1.Subject {
        constructor(initialValue) {
          super();
          _BehaviorSubject_lastMessage.set(this, void 0);
          __classPrivateFieldSet(this, _BehaviorSubject_lastMessage, initialValue, "f");
        }
        /**
         * @desc Returns the last message published
         */
        get value() {
          return __classPrivateFieldGet(this, _BehaviorSubject_lastMessage, "f");
        }
        /**
         * @desc Publishes a message to all subscribers
         * @param message - The message to publish
         */
        publish(message) {
          __classPrivateFieldSet(this, _BehaviorSubject_lastMessage, message, "f");
          super.publish(message);
        }
        /**
         * @desc Subscribes to a subject
         * @param observer - The observer to subscribe
         */
        subscribe(observer) {
          if (__classPrivateFieldGet(this, _BehaviorSubject_lastMessage, "f")) {
            observer(__classPrivateFieldGet(this, _BehaviorSubject_lastMessage, "f"));
          }
          return super.subscribe(observer);
        }
      };
      exports.BehaviorSubject = BehaviorSubject;
      _BehaviorSubject_lastMessage = /* @__PURE__ */ new WeakMap();
    }
  });

  // node_modules/@eleven-am/pondsocket-common/subjects/index.js
  var require_subjects = __commonJS({
    "node_modules/@eleven-am/pondsocket-common/subjects/index.js"(exports) {
      "use strict";
      var __createBinding = exports && exports.__createBinding || (Object.create ? (function(o, m, k, k2) {
        if (k2 === void 0) k2 = k;
        var desc = Object.getOwnPropertyDescriptor(m, k);
        if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
          desc = { enumerable: true, get: function() {
            return m[k];
          } };
        }
        Object.defineProperty(o, k2, desc);
      }) : (function(o, m, k, k2) {
        if (k2 === void 0) k2 = k;
        o[k2] = m[k];
      }));
      var __exportStar = exports && exports.__exportStar || function(m, exports2) {
        for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports2, p)) __createBinding(exports2, m, p);
      };
      Object.defineProperty(exports, "__esModule", { value: true });
      __exportStar(require_behaviorSubject(), exports);
      __exportStar(require_subject(), exports);
    }
  });

  // node_modules/@eleven-am/pondsocket-common/enums.js
  var require_enums = __commonJS({
    "node_modules/@eleven-am/pondsocket-common/enums.js"(exports) {
      "use strict";
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.PubSubEvents = exports.Events = exports.ChannelReceiver = exports.SystemSender = exports.ErrorTypes = exports.ChannelState = exports.ClientActions = exports.ServerActions = exports.PresenceEventTypes = void 0;
      var PresenceEventTypes;
      (function(PresenceEventTypes2) {
        PresenceEventTypes2["JOIN"] = "JOIN";
        PresenceEventTypes2["LEAVE"] = "LEAVE";
        PresenceEventTypes2["UPDATE"] = "UPDATE";
      })(PresenceEventTypes || (exports.PresenceEventTypes = PresenceEventTypes = {}));
      var ServerActions;
      (function(ServerActions2) {
        ServerActions2["PRESENCE"] = "PRESENCE";
        ServerActions2["SYSTEM"] = "SYSTEM";
        ServerActions2["BROADCAST"] = "BROADCAST";
        ServerActions2["ERROR"] = "ERROR";
        ServerActions2["CONNECT"] = "CONNECT";
      })(ServerActions || (exports.ServerActions = ServerActions = {}));
      var ClientActions;
      (function(ClientActions2) {
        ClientActions2["JOIN_CHANNEL"] = "JOIN_CHANNEL";
        ClientActions2["LEAVE_CHANNEL"] = "LEAVE_CHANNEL";
        ClientActions2["BROADCAST"] = "BROADCAST";
      })(ClientActions || (exports.ClientActions = ClientActions = {}));
      var ChannelState2;
      (function(ChannelState3) {
        ChannelState3["IDLE"] = "IDLE";
        ChannelState3["JOINING"] = "JOINING";
        ChannelState3["JOINED"] = "JOINED";
        ChannelState3["STALLED"] = "STALLED";
        ChannelState3["CLOSED"] = "CLOSED";
      })(ChannelState2 || (exports.ChannelState = ChannelState2 = {}));
      var ErrorTypes;
      (function(ErrorTypes2) {
        ErrorTypes2["UNAUTHORIZED_CONNECTION"] = "UNAUTHORIZED_CONNECTION";
        ErrorTypes2["UNAUTHORIZED_JOIN_REQUEST"] = "UNAUTHORIZED_JOIN_REQUEST";
        ErrorTypes2["UNAUTHORIZED_BROADCAST"] = "UNAUTHORIZED_BROADCAST";
        ErrorTypes2["INVALID_MESSAGE"] = "INVALID_MESSAGE";
        ErrorTypes2["HANDLER_NOT_FOUND"] = "HANDLER_NOT_FOUND";
        ErrorTypes2["PRESENCE_ERROR"] = "PRESENCE_ERROR";
        ErrorTypes2["CHANNEL_ERROR"] = "CHANNEL_ERROR";
        ErrorTypes2["ENDPOINT_ERROR"] = "ENDPOINT_ERROR";
        ErrorTypes2["INTERNAL_SERVER_ERROR"] = "INTERNAL_SERVER_ERROR";
      })(ErrorTypes || (exports.ErrorTypes = ErrorTypes = {}));
      var SystemSender;
      (function(SystemSender2) {
        SystemSender2["ENDPOINT"] = "ENDPOINT";
        SystemSender2["CHANNEL"] = "CHANNEL";
      })(SystemSender || (exports.SystemSender = SystemSender = {}));
      var ChannelReceiver;
      (function(ChannelReceiver2) {
        ChannelReceiver2["ALL_USERS"] = "ALL_USERS";
        ChannelReceiver2["ALL_EXCEPT_SENDER"] = "ALL_EXCEPT_SENDER";
      })(ChannelReceiver || (exports.ChannelReceiver = ChannelReceiver = {}));
      var Events;
      (function(Events2) {
        Events2["ACKNOWLEDGE"] = "ACKNOWLEDGE";
        Events2["CONNECTION"] = "CONNECTION";
      })(Events || (exports.Events = Events = {}));
      var PubSubEvents;
      (function(PubSubEvents2) {
        PubSubEvents2["MESSAGE"] = "MESSAGE";
        PubSubEvents2["PRESENCE"] = "PRESENCE";
        PubSubEvents2["GET_PRESENCE"] = "GET_PRESENCE";
      })(PubSubEvents || (exports.PubSubEvents = PubSubEvents = {}));
    }
  });

  // node_modules/@eleven-am/pondsocket-common/misc/uuid.js
  var require_uuid = __commonJS({
    "node_modules/@eleven-am/pondsocket-common/misc/uuid.js"(exports) {
      "use strict";
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.uuid = uuid;
      function uuid() {
        return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
          const r = Math.random() * 16 | 0;
          const v = c === "x" ? r : r & 3 | 8;
          return v.toString(16);
        });
      }
    }
  });

  // node_modules/@eleven-am/pondsocket-common/misc/types.js
  var require_types = __commonJS({
    "node_modules/@eleven-am/pondsocket-common/misc/types.js"(exports) {
      "use strict";
      Object.defineProperty(exports, "__esModule", { value: true });
    }
  });

  // node_modules/@eleven-am/pondsocket-common/misc/index.js
  var require_misc = __commonJS({
    "node_modules/@eleven-am/pondsocket-common/misc/index.js"(exports) {
      "use strict";
      var __createBinding = exports && exports.__createBinding || (Object.create ? (function(o, m, k, k2) {
        if (k2 === void 0) k2 = k;
        var desc = Object.getOwnPropertyDescriptor(m, k);
        if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
          desc = { enumerable: true, get: function() {
            return m[k];
          } };
        }
        Object.defineProperty(o, k2, desc);
      }) : (function(o, m, k, k2) {
        if (k2 === void 0) k2 = k;
        o[k2] = m[k];
      }));
      var __exportStar = exports && exports.__exportStar || function(m, exports2) {
        for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports2, p)) __createBinding(exports2, m, p);
      };
      Object.defineProperty(exports, "__esModule", { value: true });
      __exportStar(require_uuid(), exports);
      __exportStar(require_types(), exports);
    }
  });

  // node_modules/@eleven-am/pondsocket-common/schema.js
  var require_schema = __commonJS({
    "node_modules/@eleven-am/pondsocket-common/schema.js"(exports) {
      "use strict";
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.channelEventSchema = exports.serverMessageSchema = exports.presenceMessageSchema = exports.clientMessageSchema = exports.ValidationError = void 0;
      var enums_1 = require_enums();
      var ValidationError = class extends Error {
        constructor(message, path) {
          super(path ? `${path}: ${message}` : message);
          this.path = path;
          this.name = "ValidationError";
        }
      };
      exports.ValidationError = ValidationError;
      function isObject(value) {
        return typeof value === "object" && value !== null && !Array.isArray(value);
      }
      function isString(value) {
        return typeof value === "string";
      }
      function isArray(value) {
        return Array.isArray(value);
      }
      function isRecord(value) {
        if (!isObject(value)) {
          return false;
        }
        return Object.keys(value).every((key) => typeof key === "string");
      }
      function validateString(value, fieldName) {
        if (!isString(value)) {
          throw new ValidationError(`Expected string, got ${typeof value}`, fieldName);
        }
      }
      function validateObject(value, fieldName) {
        if (!isObject(value)) {
          throw new ValidationError(`Expected object, got ${typeof value}`, fieldName);
        }
      }
      function validateRecord(value, fieldName) {
        if (!isRecord(value)) {
          throw new ValidationError(`Expected record with string keys, got ${typeof value}`, fieldName);
        }
      }
      function validateArray(value, fieldName) {
        if (!isArray(value)) {
          throw new ValidationError(`Expected array, got ${typeof value}`, fieldName);
        }
      }
      function validateEnum(value, enumObj, fieldName) {
        const validValues = Object.values(enumObj);
        if (!validValues.includes(value)) {
          throw new ValidationError(`Expected one of [${validValues.join(", ")}], got ${JSON.stringify(value)}`, fieldName);
        }
      }
      exports.clientMessageSchema = {
        parse(data) {
          validateObject(data, "clientMessage");
          const obj = data;
          if (!("event" in obj)) {
            throw new ValidationError("Missing required field", "event");
          }
          if (!("requestId" in obj)) {
            throw new ValidationError("Missing required field", "requestId");
          }
          if (!("channelName" in obj)) {
            throw new ValidationError("Missing required field", "channelName");
          }
          if (!("payload" in obj)) {
            throw new ValidationError("Missing required field", "payload");
          }
          if (!("action" in obj)) {
            throw new ValidationError("Missing required field", "action");
          }
          validateString(obj.event, "event");
          validateString(obj.requestId, "requestId");
          validateString(obj.channelName, "channelName");
          validateRecord(obj.payload, "payload");
          validateEnum(obj.action, enums_1.ClientActions, "action");
          return {
            event: obj.event,
            requestId: obj.requestId,
            channelName: obj.channelName,
            payload: obj.payload,
            action: obj.action
          };
        }
      };
      exports.presenceMessageSchema = {
        parse(data) {
          validateObject(data, "presenceMessage");
          const obj = data;
          if (!("requestId" in obj)) {
            throw new ValidationError("Missing required field", "requestId");
          }
          if (!("channelName" in obj)) {
            throw new ValidationError("Missing required field", "channelName");
          }
          if (!("event" in obj)) {
            throw new ValidationError("Missing required field", "event");
          }
          if (!("action" in obj)) {
            throw new ValidationError("Missing required field", "action");
          }
          if (!("payload" in obj)) {
            throw new ValidationError("Missing required field", "payload");
          }
          validateString(obj.requestId, "requestId");
          validateString(obj.channelName, "channelName");
          validateEnum(obj.event, enums_1.PresenceEventTypes, "event");
          if (obj.action !== enums_1.ServerActions.PRESENCE) {
            throw new ValidationError(`Expected ${enums_1.ServerActions.PRESENCE}, got ${JSON.stringify(obj.action)}`, "action");
          }
          validateObject(obj.payload, "payload");
          const payload = obj.payload;
          if (!("presence" in payload)) {
            throw new ValidationError("Missing required field", "payload.presence");
          }
          if (!("changed" in payload)) {
            throw new ValidationError("Missing required field", "payload.changed");
          }
          validateArray(payload.presence, "payload.presence");
          payload.presence.forEach((item, index) => {
            validateRecord(item, `payload.presence[${index}]`);
          });
          validateRecord(payload.changed, "payload.changed");
          return {
            requestId: obj.requestId,
            channelName: obj.channelName,
            event: obj.event,
            action: enums_1.ServerActions.PRESENCE,
            payload: {
              presence: payload.presence,
              changed: payload.changed
            }
          };
        }
      };
      exports.serverMessageSchema = {
        parse(data) {
          validateObject(data, "serverMessage");
          const obj = data;
          if (!("event" in obj)) {
            throw new ValidationError("Missing required field", "event");
          }
          if (!("requestId" in obj)) {
            throw new ValidationError("Missing required field", "requestId");
          }
          if (!("channelName" in obj)) {
            throw new ValidationError("Missing required field", "channelName");
          }
          if (!("payload" in obj)) {
            throw new ValidationError("Missing required field", "payload");
          }
          if (!("action" in obj)) {
            throw new ValidationError("Missing required field", "action");
          }
          validateString(obj.event, "event");
          validateString(obj.requestId, "requestId");
          validateString(obj.channelName, "channelName");
          validateRecord(obj.payload, "payload");
          const validActions = [
            enums_1.ServerActions.BROADCAST,
            enums_1.ServerActions.CONNECT,
            enums_1.ServerActions.ERROR,
            enums_1.ServerActions.SYSTEM
          ];
          if (!validActions.includes(obj.action)) {
            throw new ValidationError(`Expected one of [${validActions.join(", ")}], got ${JSON.stringify(obj.action)}`, "action");
          }
          return {
            event: obj.event,
            requestId: obj.requestId,
            channelName: obj.channelName,
            payload: obj.payload,
            action: obj.action
          };
        }
      };
      exports.channelEventSchema = {
        parse(data) {
          validateObject(data, "channelEvent");
          const obj = data;
          if (!("action" in obj)) {
            throw new ValidationError("Missing required field", "action");
          }
          if (obj.action === enums_1.ServerActions.PRESENCE) {
            return exports.presenceMessageSchema.parse(data);
          }
          return exports.serverMessageSchema.parse(data);
        }
      };
    }
  });

  // node_modules/@eleven-am/pondsocket-common/index.js
  var require_pondsocket_common = __commonJS({
    "node_modules/@eleven-am/pondsocket-common/index.js"(exports) {
      "use strict";
      var __createBinding = exports && exports.__createBinding || (Object.create ? (function(o, m, k, k2) {
        if (k2 === void 0) k2 = k;
        var desc = Object.getOwnPropertyDescriptor(m, k);
        if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
          desc = { enumerable: true, get: function() {
            return m[k];
          } };
        }
        Object.defineProperty(o, k2, desc);
      }) : (function(o, m, k, k2) {
        if (k2 === void 0) k2 = k;
        o[k2] = m[k];
      }));
      var __exportStar = exports && exports.__exportStar || function(m, exports2) {
        for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports2, p)) __createBinding(exports2, m, p);
      };
      Object.defineProperty(exports, "__esModule", { value: true });
      __exportStar(require_subjects(), exports);
      __exportStar(require_enums(), exports);
      __exportStar(require_misc(), exports);
      __exportStar(require_schema(), exports);
    }
  });

  // node_modules/@eleven-am/pondsocket-client/core/channel.js
  var require_channel = __commonJS({
    "node_modules/@eleven-am/pondsocket-client/core/channel.js"(exports) {
      "use strict";
      var __classPrivateFieldSet = exports && exports.__classPrivateFieldSet || function(receiver, state, value, kind, f) {
        if (kind === "m") throw new TypeError("Private method is not writable");
        if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
        if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
        return kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value), value;
      };
      var __classPrivateFieldGet = exports && exports.__classPrivateFieldGet || function(receiver, state, kind, f) {
        if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
        if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
        return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
      };
      var _Channel_instances;
      var _Channel_name;
      var _Channel_queue;
      var _Channel_presence;
      var _Channel_presenceSub;
      var _Channel_publisher;
      var _Channel_joinParams;
      var _Channel_receiver;
      var _Channel_clientState;
      var _Channel_joinState;
      var _Channel_emptyQueue;
      var _Channel_init;
      var _Channel_onMessage;
      var _Channel_publish;
      var _Channel_subscribeToPresence;
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.Channel = void 0;
      var pondsocket_common_1 = require_pondsocket_common();
      var Channel = class {
        constructor(publisher, clientState, name, params) {
          _Channel_instances.add(this);
          _Channel_name.set(this, void 0);
          _Channel_queue.set(this, void 0);
          _Channel_presence.set(this, void 0);
          _Channel_presenceSub.set(this, void 0);
          _Channel_publisher.set(this, void 0);
          _Channel_joinParams.set(this, void 0);
          _Channel_receiver.set(this, void 0);
          _Channel_clientState.set(this, void 0);
          _Channel_joinState.set(this, void 0);
          __classPrivateFieldSet(this, _Channel_name, name, "f");
          __classPrivateFieldSet(this, _Channel_queue, [], "f");
          __classPrivateFieldSet(this, _Channel_presence, [], "f");
          __classPrivateFieldSet(this, _Channel_joinParams, params, "f");
          __classPrivateFieldSet(this, _Channel_publisher, publisher, "f");
          __classPrivateFieldSet(this, _Channel_clientState, clientState, "f");
          __classPrivateFieldSet(this, _Channel_receiver, new pondsocket_common_1.Subject(), "f");
          __classPrivateFieldSet(this, _Channel_joinState, new pondsocket_common_1.BehaviorSubject(pondsocket_common_1.ChannelState.IDLE), "f");
          __classPrivateFieldSet(this, _Channel_presenceSub, () => {
          }, "f");
        }
        /**
         * @desc Gets the current connection state of the channel.
         */
        get channelState() {
          return __classPrivateFieldGet(this, _Channel_joinState, "f").value;
        }
        /**
         * @desc Gets the current presence of the channel.
         */
        get presence() {
          return __classPrivateFieldGet(this, _Channel_presence, "f");
        }
        /**
         * @desc Acknowledges the channel has been joined on the server.
         * @param receiver - The receiver to subscribe to.
         */
        acknowledge(receiver) {
          __classPrivateFieldGet(this, _Channel_joinState, "f").publish(pondsocket_common_1.ChannelState.JOINED);
          __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_init).call(this, receiver);
          __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_emptyQueue).call(this);
        }
        /**
         * @desc Connects to the channel.
         */
        join() {
          const message = {
            action: pondsocket_common_1.ClientActions.JOIN_CHANNEL,
            event: pondsocket_common_1.ClientActions.JOIN_CHANNEL,
            payload: __classPrivateFieldGet(this, _Channel_joinParams, "f"),
            channelName: __classPrivateFieldGet(this, _Channel_name, "f"),
            requestId: (0, pondsocket_common_1.uuid)()
          };
          if (__classPrivateFieldGet(this, _Channel_joinState, "f").value === pondsocket_common_1.ChannelState.JOINED) {
            return;
          }
          __classPrivateFieldGet(this, _Channel_joinState, "f").publish(pondsocket_common_1.ChannelState.JOINING);
          if (__classPrivateFieldGet(this, _Channel_clientState, "f").value) {
            __classPrivateFieldGet(this, _Channel_publisher, "f").call(this, message);
          } else {
            const unsubscribe = __classPrivateFieldGet(this, _Channel_clientState, "f").subscribe((state) => {
              if (state) {
                unsubscribe();
                if (__classPrivateFieldGet(this, _Channel_joinState, "f").value === pondsocket_common_1.ChannelState.JOINING) {
                  __classPrivateFieldGet(this, _Channel_publisher, "f").call(this, message);
                }
              }
            });
          }
        }
        /**
         * @desc Disconnects from the channel.
         */
        leave() {
          const message = {
            action: pondsocket_common_1.ClientActions.LEAVE_CHANNEL,
            event: pondsocket_common_1.ClientActions.LEAVE_CHANNEL,
            channelName: __classPrivateFieldGet(this, _Channel_name, "f"),
            requestId: (0, pondsocket_common_1.uuid)(),
            payload: {}
          };
          __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_publish).call(this, message);
          __classPrivateFieldGet(this, _Channel_joinState, "f").publish(pondsocket_common_1.ChannelState.CLOSED);
          __classPrivateFieldGet(this, _Channel_presenceSub, "f").call(this);
        }
        /**
         * @desc Monitors the channel state of the channel.
         * @param callback - The callback to call when the connection state changes.
         */
        onChannelStateChange(callback) {
          return __classPrivateFieldGet(this, _Channel_joinState, "f").subscribe((data) => {
            callback(data);
          });
        }
        /**
         * @desc Detects when clients join the channel.
         * @param callback - The callback to call when a client joins the channel.
         */
        onJoin(callback) {
          return __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_subscribeToPresence).call(this, (event, payload) => {
            if (event === pondsocket_common_1.PresenceEventTypes.JOIN) {
              return callback(payload.changed);
            }
          });
        }
        /**
         * @desc Detects when clients leave the channel.
         * @param callback - The callback to call when a client leaves the channel.
         */
        onLeave(callback) {
          return __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_subscribeToPresence).call(this, (event, payload) => {
            if (event === pondsocket_common_1.PresenceEventTypes.LEAVE) {
              return callback(payload.changed);
            }
          });
        }
        /**
         * @desc Monitors the channel for messages.
         * @param callback - The callback to call when a message is received.
         */
        onMessage(callback) {
          return __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_onMessage).call(this, (event, message) => {
            callback(event, message);
          });
        }
        /**
         * @desc Monitors the channel for messages.
         * @param event - The event to monitor.
         * @param callback - The callback to call when a message is received.
         */
        onMessageEvent(event, callback) {
          return this.onMessage((eventReceived, message) => {
            if (eventReceived === event) {
              return callback(message);
            }
          });
        }
        /**
         * @desc Detects when clients change their presence in the channel.
         * @param callback - The callback to call when a client changes their presence in the channel.
         */
        onPresenceChange(callback) {
          return __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_subscribeToPresence).call(this, (event, payload) => {
            if (event === pondsocket_common_1.PresenceEventTypes.UPDATE) {
              return callback(payload);
            }
          });
        }
        /**
         * @desc Monitors the presence of the channel.
         * @param callback - The callback to call when the presence changes.
         */
        onUsersChange(callback) {
          return __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_subscribeToPresence).call(this, (_event, payload) => callback(payload.presence));
        }
        /**
         * @desc Sends a message to specific clients in the channel.
         * @param event - The event to send.
         * @param payload - The message to send.
         */
        sendMessage(event, payload) {
          const requestId = (0, pondsocket_common_1.uuid)();
          const message = {
            action: pondsocket_common_1.ClientActions.BROADCAST,
            channelName: __classPrivateFieldGet(this, _Channel_name, "f"),
            requestId,
            event,
            payload
          };
          __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_publish).call(this, message);
        }
        /**
         * @desc Sends a message to the server and waits for a response.
         * @param sentEvent - The event to send.
         * @param payload - The message to send.
         */
        sendForResponse(sentEvent, payload) {
          const requestId = (0, pondsocket_common_1.uuid)();
          return new Promise((resolve) => {
            const unsub = __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_onMessage).call(this, (_, message2, responseId) => {
              if (requestId === responseId) {
                resolve(message2);
                unsub();
              }
            });
            const message = {
              action: pondsocket_common_1.ClientActions.BROADCAST,
              channelName: __classPrivateFieldGet(this, _Channel_name, "f"),
              requestId,
              event: sentEvent,
              payload
            };
            __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_publish).call(this, message);
          });
        }
      };
      exports.Channel = Channel;
      _Channel_name = /* @__PURE__ */ new WeakMap(), _Channel_queue = /* @__PURE__ */ new WeakMap(), _Channel_presence = /* @__PURE__ */ new WeakMap(), _Channel_presenceSub = /* @__PURE__ */ new WeakMap(), _Channel_publisher = /* @__PURE__ */ new WeakMap(), _Channel_joinParams = /* @__PURE__ */ new WeakMap(), _Channel_receiver = /* @__PURE__ */ new WeakMap(), _Channel_clientState = /* @__PURE__ */ new WeakMap(), _Channel_joinState = /* @__PURE__ */ new WeakMap(), _Channel_instances = /* @__PURE__ */ new WeakSet(), _Channel_emptyQueue = function _Channel_emptyQueue2() {
        __classPrivateFieldGet(this, _Channel_queue, "f").forEach((message) => __classPrivateFieldGet(this, _Channel_publisher, "f").call(this, message));
        __classPrivateFieldSet(this, _Channel_queue, [], "f");
      }, _Channel_init = function _Channel_init2(receiver) {
        __classPrivateFieldGet(this, _Channel_presenceSub, "f").call(this);
        const unsubMessages = receiver.subscribe((data) => {
          if (data.channelName === __classPrivateFieldGet(this, _Channel_name, "f") && this.channelState === pondsocket_common_1.ChannelState.JOINED) {
            __classPrivateFieldGet(this, _Channel_receiver, "f").publish(data);
          }
        });
        const unsubStateChange = __classPrivateFieldGet(this, _Channel_clientState, "f").subscribe((state) => {
          if (state && __classPrivateFieldGet(this, _Channel_joinState, "f").value === pondsocket_common_1.ChannelState.STALLED) {
            const message = {
              action: pondsocket_common_1.ClientActions.JOIN_CHANNEL,
              event: pondsocket_common_1.ClientActions.JOIN_CHANNEL,
              payload: __classPrivateFieldGet(this, _Channel_joinParams, "f"),
              channelName: __classPrivateFieldGet(this, _Channel_name, "f"),
              requestId: (0, pondsocket_common_1.uuid)()
            };
            __classPrivateFieldGet(this, _Channel_publisher, "f").call(this, message);
          } else if (!state && __classPrivateFieldGet(this, _Channel_joinState, "f").value === pondsocket_common_1.ChannelState.JOINED) {
            __classPrivateFieldGet(this, _Channel_joinState, "f").publish(pondsocket_common_1.ChannelState.STALLED);
          }
        });
        const unsubPresence = __classPrivateFieldGet(this, _Channel_instances, "m", _Channel_subscribeToPresence).call(this, (_, payload) => {
          __classPrivateFieldSet(this, _Channel_presence, payload.presence, "f");
        });
        __classPrivateFieldSet(this, _Channel_presenceSub, () => {
          unsubMessages();
          unsubStateChange();
          unsubPresence();
        }, "f");
      }, _Channel_onMessage = function _Channel_onMessage2(callback) {
        return __classPrivateFieldGet(this, _Channel_receiver, "f").subscribe((data) => {
          if (data.action !== pondsocket_common_1.ServerActions.PRESENCE) {
            return callback(data.event, data.payload, data.requestId);
          }
        });
      }, _Channel_publish = function _Channel_publish2(data) {
        if (__classPrivateFieldGet(this, _Channel_joinState, "f").value === pondsocket_common_1.ChannelState.JOINED) {
          return __classPrivateFieldGet(this, _Channel_publisher, "f").call(this, data);
        }
        __classPrivateFieldGet(this, _Channel_queue, "f").push(data);
      }, _Channel_subscribeToPresence = function _Channel_subscribeToPresence2(callback) {
        return __classPrivateFieldGet(this, _Channel_receiver, "f").subscribe((data) => {
          if (data.action === pondsocket_common_1.ServerActions.PRESENCE) {
            return callback(data.event, data.payload);
          }
        });
      };
    }
  });

  // node_modules/@eleven-am/pondsocket-client/browser/client.js
  var require_client = __commonJS({
    "node_modules/@eleven-am/pondsocket-client/browser/client.js"(exports) {
      "use strict";
      var __classPrivateFieldSet = exports && exports.__classPrivateFieldSet || function(receiver, state, value, kind, f) {
        if (kind === "m") throw new TypeError("Private method is not writable");
        if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
        if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
        return kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value), value;
      };
      var __classPrivateFieldGet = exports && exports.__classPrivateFieldGet || function(receiver, state, kind, f) {
        if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
        if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
        return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
      };
      var _PondClient_instances;
      var _PondClient_channels;
      var _PondClient_createPublisher;
      var _PondClient_handleAcknowledge;
      var _PondClient_init;
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.PondClient = void 0;
      var pondsocket_common_1 = require_pondsocket_common();
      var channel_1 = require_channel();
      var PondClient2 = class {
        constructor(endpoint, params = {}) {
          _PondClient_instances.add(this);
          _PondClient_channels.set(this, void 0);
          let address;
          try {
            address = new URL(endpoint);
          } catch (e) {
            address = new URL(window.location.toString());
            address.pathname = endpoint;
          }
          this._disconnecting = false;
          const query = new URLSearchParams(params);
          address.search = query.toString();
          const protocol = address.protocol === "https:" ? "wss:" : "ws:";
          if (address.protocol !== "wss:" && address.protocol !== "ws:") {
            address.protocol = protocol;
          }
          this._address = address;
          __classPrivateFieldSet(this, _PondClient_channels, /* @__PURE__ */ new Map(), "f");
          this._broadcaster = new pondsocket_common_1.Subject();
          this._connectionState = new pondsocket_common_1.BehaviorSubject(false);
          __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_init).call(this);
        }
        /**
         * @desc Connects to the server and returns the socket.
         */
        connect() {
          this._disconnecting = false;
          const socket = new WebSocket(this._address.toString());
          socket.onmessage = (message) => {
            const lines = message.data.trim().split("\n");
            for (const line of lines) {
              if (line.trim()) {
                const data = JSON.parse(line);
                const event = pondsocket_common_1.channelEventSchema.parse(data);
                this._broadcaster.publish(event);
              }
            }
          };
          socket.onerror = () => socket.close();
          socket.onclose = () => {
            this._connectionState.publish(false);
            if (this._disconnecting) {
              return;
            }
            setTimeout(() => {
              this.connect();
            }, 1e3);
          };
          this._socket = socket;
        }
        /**
         * @desc Returns the current state of the socket.
         */
        getState() {
          return this._connectionState.value;
        }
        /**
         * @desc Disconnects the socket.
         */
        disconnect() {
          var _a;
          this._connectionState.publish(false);
          this._disconnecting = true;
          (_a = this._socket) === null || _a === void 0 ? void 0 : _a.close();
          __classPrivateFieldGet(this, _PondClient_channels, "f").clear();
        }
        /**
         * @desc Creates a channel with the given name and params.
         * @param name - The name of the channel.
         * @param params - The params to send to the server.
         */
        createChannel(name, params) {
          const channel = __classPrivateFieldGet(this, _PondClient_channels, "f").get(name);
          if (channel && channel.channelState !== pondsocket_common_1.ChannelState.CLOSED) {
            return channel;
          }
          const publisher = __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_createPublisher).call(this);
          const newChannel = new channel_1.Channel(publisher, this._connectionState, name, params || {});
          __classPrivateFieldGet(this, _PondClient_channels, "f").set(name, newChannel);
          return newChannel;
        }
        /**
         * @desc Subscribes to the connection state.
         * @param callback - The callback to call when the state changes.
         */
        onConnectionChange(callback) {
          return this._connectionState.subscribe(callback);
        }
      };
      exports.PondClient = PondClient2;
      _PondClient_channels = /* @__PURE__ */ new WeakMap(), _PondClient_instances = /* @__PURE__ */ new WeakSet(), _PondClient_createPublisher = function _PondClient_createPublisher2() {
        return (message) => {
          if (this._connectionState.value) {
            this._socket.send(JSON.stringify(message));
          }
        };
      }, _PondClient_handleAcknowledge = function _PondClient_handleAcknowledge2(message) {
        var _a;
        const channel = (_a = __classPrivateFieldGet(this, _PondClient_channels, "f").get(message.channelName)) !== null && _a !== void 0 ? _a : new channel_1.Channel(__classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_createPublisher).call(this), this._connectionState, message.channelName, {});
        __classPrivateFieldGet(this, _PondClient_channels, "f").set(message.channelName, channel);
        channel.acknowledge(this._broadcaster);
      }, _PondClient_init = function _PondClient_init2() {
        this._broadcaster.subscribe((message) => {
          if (message.event === pondsocket_common_1.Events.ACKNOWLEDGE) {
            __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_handleAcknowledge).call(this, message);
          } else if (message.event === pondsocket_common_1.Events.CONNECTION && message.action === pondsocket_common_1.ServerActions.CONNECT) {
            this._connectionState.publish(true);
          }
        });
      };
    }
  });

  // node_modules/es5-ext/global.js
  var require_global = __commonJS({
    "node_modules/es5-ext/global.js"(exports, module) {
      var naiveFallback = function() {
        if (typeof self === "object" && self) return self;
        if (typeof window === "object" && window) return window;
        throw new Error("Unable to resolve global `this`");
      };
      module.exports = (function() {
        if (this) return this;
        if (typeof globalThis === "object" && globalThis) return globalThis;
        try {
          Object.defineProperty(Object.prototype, "__global__", {
            get: function() {
              return this;
            },
            configurable: true
          });
        } catch (error) {
          return naiveFallback();
        }
        try {
          if (!__global__) return naiveFallback();
          return __global__;
        } finally {
          delete Object.prototype.__global__;
        }
      })();
    }
  });

  // node_modules/websocket/package.json
  var require_package = __commonJS({
    "node_modules/websocket/package.json"(exports, module) {
      module.exports = {
        name: "websocket",
        description: "Websocket Client & Server Library implementing the WebSocket protocol as specified in RFC 6455.",
        keywords: [
          "websocket",
          "websockets",
          "socket",
          "networking",
          "comet",
          "push",
          "RFC-6455",
          "realtime",
          "server",
          "client"
        ],
        author: "Brian McKelvey <theturtle32@gmail.com> (https://github.com/theturtle32)",
        contributors: [
          "I\xF1aki Baz Castillo <ibc@aliax.net> (http://dev.sipdoc.net)"
        ],
        version: "1.0.35",
        repository: {
          type: "git",
          url: "https://github.com/theturtle32/WebSocket-Node.git"
        },
        homepage: "https://github.com/theturtle32/WebSocket-Node",
        engines: {
          node: ">=4.0.0"
        },
        dependencies: {
          bufferutil: "^4.0.1",
          debug: "^2.2.0",
          "es5-ext": "^0.10.63",
          "typedarray-to-buffer": "^3.1.5",
          "utf-8-validate": "^5.0.2",
          yaeti: "^0.0.6"
        },
        devDependencies: {
          "buffer-equal": "^1.0.0",
          gulp: "^4.0.2",
          "gulp-jshint": "^2.0.4",
          "jshint-stylish": "^2.2.1",
          jshint: "^2.0.0",
          tape: "^4.9.1"
        },
        config: {
          verbose: false
        },
        scripts: {
          test: "tape test/unit/*.js",
          gulp: "gulp"
        },
        main: "index",
        directories: {
          lib: "./lib"
        },
        browser: "lib/browser.js",
        license: "Apache-2.0"
      };
    }
  });

  // node_modules/websocket/lib/version.js
  var require_version = __commonJS({
    "node_modules/websocket/lib/version.js"(exports, module) {
      module.exports = require_package().version;
    }
  });

  // node_modules/websocket/lib/browser.js
  var require_browser = __commonJS({
    "node_modules/websocket/lib/browser.js"(exports, module) {
      var _globalThis;
      if (typeof globalThis === "object") {
        _globalThis = globalThis;
      } else {
        try {
          _globalThis = require_global();
        } catch (error) {
        } finally {
          if (!_globalThis && typeof window !== "undefined") {
            _globalThis = window;
          }
          if (!_globalThis) {
            throw new Error("Could not determine global this");
          }
        }
      }
      var NativeWebSocket = _globalThis.WebSocket || _globalThis.MozWebSocket;
      var websocket_version = require_version();
      function W3CWebSocket(uri, protocols) {
        var native_instance;
        if (protocols) {
          native_instance = new NativeWebSocket(uri, protocols);
        } else {
          native_instance = new NativeWebSocket(uri);
        }
        return native_instance;
      }
      if (NativeWebSocket) {
        ["CONNECTING", "OPEN", "CLOSING", "CLOSED"].forEach(function(prop) {
          Object.defineProperty(W3CWebSocket, prop, {
            get: function() {
              return NativeWebSocket[prop];
            }
          });
        });
      }
      module.exports = {
        "w3cwebsocket": NativeWebSocket ? W3CWebSocket : null,
        "version": websocket_version
      };
    }
  });

  // node_modules/@eleven-am/pondsocket-client/node/node.js
  var require_node = __commonJS({
    "node_modules/@eleven-am/pondsocket-client/node/node.js"(exports) {
      "use strict";
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.PondClient = void 0;
      var pondsocket_common_1 = require_pondsocket_common();
      var client_1 = require_client();
      var WebSocket2 = require_browser().w3cwebsocket;
      var PondClient2 = class extends client_1.PondClient {
        /**
         * @desc Connects to the server and returns the socket.
         */
        connect(backoff = 1) {
          this._disconnecting = false;
          const socket = new WebSocket2(this._address.toString());
          socket.onopen = () => this._connectionState.publish(true);
          socket.onmessage = (message) => {
            const lines = message.data.trim().split("\n");
            for (const line of lines) {
              if (line.trim()) {
                const data = JSON.parse(line);
                const event = pondsocket_common_1.channelEventSchema.parse(data);
                this._broadcaster.publish(event);
              }
            }
          };
          socket.onerror = () => socket.close();
          socket.onclose = () => {
            this._connectionState.publish(false);
            if (this._disconnecting) {
              return;
            }
            setTimeout(() => {
              this.connect();
            }, 1e3);
          };
        }
      };
      exports.PondClient = PondClient2;
    }
  });

  // node_modules/@eleven-am/pondsocket-client/index.js
  var require_pondsocket_client = __commonJS({
    "node_modules/@eleven-am/pondsocket-client/index.js"(exports) {
      "use strict";
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.PondClient = exports.ChannelState = void 0;
      var pondsocket_common_1 = require_pondsocket_common();
      Object.defineProperty(exports, "ChannelState", { enumerable: true, get: function() {
        return pondsocket_common_1.ChannelState;
      } });
      var client_1 = require_client();
      var node_1 = require_node();
      var PondClient2 = typeof window === "undefined" ? node_1.PondClient : client_1.PondClient;
      exports.PondClient = PondClient2;
    }
  });

  // src/runtime.ts
  var import_pondsocket_client = __toESM(require_pondsocket_client(), 1);

  // src/logger.ts
  var Logger = class {
    static configure(options) {
      this.debugMode = options.debug ?? false;
    }
    static debug(tag, message, data) {
      if (!this.debugMode) return;
      if (data) {
        console.debug(`[${tag}] ${message}`, data);
      } else {
        console.debug(`[${tag}] ${message}`);
      }
    }
    static warn(tag, message, error) {
      if (error) {
        console.warn(`[${tag}] ${message}`, error);
      } else {
        console.warn(`[${tag}] ${message}`);
      }
    }
    static error(tag, message, error) {
      if (error) {
        console.error(`[${tag}] ${message}`, error);
      } else {
        console.error(`[${tag}] ${message}`);
      }
    }
  };
  Logger.debugMode = false;

  // src/vdom.ts
  function hydrate(json, dom, refs) {
    const isWrapper = !json.tag && !json.text && !json.comment && json.children || json.fragment;
    const clientNode = { ...json, el: isWrapper ? null : dom, children: void 0 };
    if (!isWrapper) {
      dom.__pondNode = clientNode;
    }
    if (json.refId && refs) {
      refs.set(json.refId, clientNode);
    }
    if (json.tag) {
      if (dom.nodeType !== Node.ELEMENT_NODE) {
        throw new Error(`Hydration error: expected element <${json.tag}> but found nodeType ${dom.nodeType}`);
      } else {
        const el = dom;
        if (el.tagName.toLowerCase() !== json.tag.toLowerCase()) {
          throw new Error(`Hydration error: expected tag <${json.tag}> but found <${el.tagName}>`);
        }
      }
    } else if (json.text !== void 0 && json.text !== "") {
      if (dom.nodeType !== Node.TEXT_NODE) {
        throw new Error(`Hydration error: expected text node but found nodeType ${dom.nodeType}`);
      }
    }
    if (json.children && json.children.length > 0) {
      clientNode.children = [];
      const domChildren = Array.from(dom.childNodes).filter(shouldHydrate);
      const consumed = hydrateChildren(clientNode.children, json.children, domChildren, dom, refs);
      const expected = countRenderableNodes(json.children);
      if (consumed !== expected) {
        throw new Error(`Hydration error: expected ${expected} renderable children, hydrated ${consumed}`);
      }
    }
    return clientNode;
  }
  function hydrateChildren(target, jsonChildren, domChildren, parentDom, refs) {
    let domIdx = 0;
    for (let i = 0; i < jsonChildren.length; i++) {
      const childJson = jsonChildren[i];
      const isWrapper = !childJson.tag && !childJson.text && !childJson.comment && childJson.children || childJson.fragment;
      if (isWrapper) {
        const wrapperNode = { ...childJson, el: null, children: [] };
        if (childJson.refId && refs) refs.set(childJson.refId, wrapperNode);
        if (childJson.children) {
          const consumed = hydrateChildrenWithConsumption(
            wrapperNode.children,
            childJson.children,
            domChildren,
            domIdx,
            parentDom,
            refs
          );
          domIdx += consumed;
        }
        target.push(wrapperNode);
        continue;
      }
      if (childJson.text === "") {
        const childDom2 = domChildren[domIdx];
        if (!childDom2 || childDom2.nodeType !== Node.TEXT_NODE) {
          throw new Error(`Hydration error: expected empty text node at index ${i}`);
        }
        const childNode2 = hydrate(childJson, childDom2, refs);
        target.push(childNode2);
        domIdx++;
        continue;
      }
      const childDom = domChildren[domIdx];
      if (!childDom) {
        throw new Error(`Hydration error: missing DOM node for child index ${i}`);
      }
      const childNode = hydrate(childJson, childDom, refs);
      target.push(childNode);
      domIdx++;
    }
    return domIdx;
  }
  function hydrateChildrenWithConsumption(target, jsonChildren, domChildren, startIdx, parentDom, refs) {
    let domIdx = startIdx;
    for (let i = 0; i < jsonChildren.length; i++) {
      const childJson = jsonChildren[i];
      const isWrapper = !childJson.tag && !childJson.text && !childJson.comment && childJson.children || childJson.fragment;
      if (isWrapper) {
        const wrapperNode = { ...childJson, el: null, children: [] };
        if (childJson.refId && refs) refs.set(childJson.refId, wrapperNode);
        if (childJson.children) {
          const consumed = hydrateChildrenWithConsumption(
            wrapperNode.children,
            childJson.children,
            domChildren,
            domIdx,
            parentDom,
            refs
          );
          domIdx += consumed;
        }
        target.push(wrapperNode);
        continue;
      }
      if (childJson.text === "") {
        const childDom2 = domChildren[domIdx];
        if (!childDom2 || childDom2.nodeType !== Node.TEXT_NODE) {
          throw new Error(`Hydration error: expected empty text node at index ${i}`);
        }
        const childNode2 = hydrate(childJson, childDom2, refs);
        target.push(childNode2);
        domIdx++;
        continue;
      }
      const childDom = domChildren[domIdx];
      if (!childDom) {
        throw new Error(`Hydration error: missing DOM node for child index ${i}`);
      }
      const childNode = hydrate(childJson, childDom, refs);
      target.push(childNode);
      domIdx++;
    }
    return domIdx - startIdx;
  }
  function shouldHydrate(_node) {
    return true;
  }
  function countRenderableNodes(nodes) {
    if (!nodes || nodes.length === 0) return 0;
    let count = 0;
    for (const n of nodes) {
      const isWrapper = !n.tag && !n.text && !n.comment && n.children || n.fragment;
      if (isWrapper) {
        count += countRenderableNodes(n.children);
        continue;
      }
      if (n.tag || n.text !== void 0 || n.comment) {
        count++;
      }
    }
    return count;
  }

  // src/patcher.ts
  var Patcher = class {
    constructor(root, events, router, uploads, refs) {
      this.root = root;
      this.events = events;
      this.router = router;
      this.uploads = uploads;
      this.refs = refs;
    }
    apply(patch) {
      const target = this.traverse(patch.path);
      if (!target) {
        Logger.warn("Patcher", "Target not found for path", patch.path);
        return;
      }
      switch (patch.op) {
        case "setText":
          this.setText(target, patch.value);
          break;
        case "setAttr":
          this.setAttr(target, patch.value);
          break;
        case "delAttr":
          this.delAttr(target, patch.name);
          break;
        case "setStyleDecl":
          this.setStyleDecl(target, patch.selector, patch.name, patch.value);
          break;
        case "delStyleDecl":
          this.delStyleDecl(target, patch.selector, patch.name);
          break;
        case "replaceNode":
          this.replaceNode(target, patch.value, patch.path);
          break;
        case "addChild":
          this.addChild(target, patch.value, patch.index);
          break;
        case "delChild":
          this.delChild(target, patch.index, patch.value?.key);
          break;
        case "moveChild":
          this.moveChild(target, patch.value);
          break;
        case "setRef":
          this.setRef(target, patch.value);
          break;
        case "delRef":
          this.delRef(target);
          break;
        case "setComment":
          this.setComment(target, patch.value);
          break;
        case "setStyle":
          this.setStyle(target, patch.value);
          break;
        case "delStyle":
          this.delStyle(target, patch.name);
          break;
        case "setHandlers":
          this.setHandlers(target, patch.value);
          break;
        case "setRouter":
          this.setRouter(target, patch.value);
          break;
        case "delRouter":
          this.delRouter(target);
          break;
        case "setUpload":
          this.setUpload(target, patch.value);
          break;
        case "delUpload":
          this.delUpload(target);
          break;
        case "setComponent":
          target.componentId = patch.value;
          break;
        default:
          Logger.warn("Patcher", "Unsupported op", patch.op);
      }
    }
    traverse(path) {
      let current = this.root;
      for (const idx of path) {
        if (!current.children || !current.children[idx]) {
          return null;
        }
        current = current.children[idx];
      }
      return current;
    }
    setText(node, text) {
      if (node.el) {
        node.el.textContent = text;
      }
      node.text = text;
    }
    setAttr(node, attrs) {
      if (node.el && node.el instanceof Element) {
        for (const [name, tokens] of Object.entries(attrs)) {
          node.el.setAttribute(name, tokens.join(" "));
        }
      }
      if (!node.attrs) node.attrs = {};
      Object.assign(node.attrs, attrs);
    }
    delAttr(node, name) {
      if (node.el && node.el instanceof Element) {
        node.el.removeAttribute(name);
      }
      if (node.attrs) delete node.attrs[name];
    }
    replaceNode(oldNode, newJson, path) {
      if (!oldNode.el || !oldNode.el.parentNode) {
        Logger.warn("Patcher", "Cannot replace node without parent", oldNode);
        return;
      }
      const newDom = this.render(newJson);
      oldNode.el.parentNode.replaceChild(newDom, oldNode.el);
      this.events.detach(oldNode);
      this.router.detach(oldNode);
      this.uploads.unbind(oldNode);
      this.detachRefsRecursively(oldNode);
      const parentPath = path.slice(0, -1);
      const childIdx = path[path.length - 1];
      const parent = this.traverse(parentPath);
      if (parent && parent.children) {
        const newNode = hydrate(newJson, newDom, this.refs);
        parent.children[childIdx] = newNode;
        this.events.attach(newNode);
        this.router.attach(newNode);
      }
    }
    render(json) {
      if (json.text !== void 0) {
        return document.createTextNode(json.text);
      }
      if (json.tag) {
        const el = document.createElement(json.tag);
        if (json.attrs) {
          for (const [k, v] of Object.entries(json.attrs)) {
            el.setAttribute(k, v.join(" "));
          }
        }
        if (json.style && el instanceof HTMLElement) {
          for (const [name, value] of Object.entries(json.style)) {
            el.style.setProperty(name, value);
          }
        }
        if (json.styles && el instanceof HTMLStyleElement) {
          el.textContent = this.buildStyleContent(json.styles);
        }
        if (json.children) {
          for (const child of json.children) {
            el.appendChild(this.render(child));
          }
        }
        return el;
      }
      if (json.children && json.children.length > 0) {
        const fragment = document.createDocumentFragment();
        for (const child of json.children) {
          fragment.appendChild(this.render(child));
        }
        return fragment;
      }
      return document.createComment(json.comment || "");
    }
    addChild(parent, childJson, index) {
      if (!parent.el || !parent.el.childNodes) {
        Logger.warn("Patcher", "Cannot add child to non-element", parent);
        return;
      }
      const newDom = this.render(childJson);
      const newClientNode = hydrate(childJson, newDom, this.refs);
      if (!parent.children) parent.children = [];
      const safeIndex = Math.max(0, Math.min(index, parent.children.length));
      if (safeIndex >= parent.children.length) {
        parent.el.appendChild(newDom);
        parent.children.push(newClientNode);
      } else {
        let referenceNode = null;
        for (let i = safeIndex; i < parent.children.length; i++) {
          if (parent.children[i].el) {
            referenceNode = parent.children[i].el;
            break;
          }
        }
        if (referenceNode) {
          parent.el.insertBefore(newDom, referenceNode);
        } else {
          parent.el.appendChild(newDom);
        }
        parent.children.splice(safeIndex, 0, newClientNode);
      }
      this.events.attach(newClientNode);
      this.router.attach(newClientNode);
    }
    delChild(parent, index, key) {
      if (!parent.children || !parent.children[index]) {
        if (key && parent.children) {
          const idxByKey = parent.children.findIndex((c) => c && c.key === key);
          if (idxByKey >= 0) {
            index = idxByKey;
          }
        }
        if (!parent.children || !parent.children[index]) {
          Logger.warn("Patcher", "Cannot delete missing child", {
            index,
            childrenLength: parent.children?.length,
            parentTag: parent.tag,
            parentComponentId: parent.componentId
          });
          return;
        }
      }
      const child = parent.children[index];
      if (child.el && parent.el && child.el.parentNode === parent.el) {
        parent.el.removeChild(child.el);
      }
      parent.children.splice(index, 1);
      this.events.detach(child);
      this.router.detach(child);
      this.uploads.unbind(child);
      this.detachRefsRecursively(child);
    }
    moveChild(parent, value) {
      if (!parent.children || parent.children.length === 0) {
        Logger.warn("Patcher", "Cannot move child in empty parent", { value });
        return;
      }
      let fromIdx = -1;
      const toIdx = Math.max(0, Math.min(value.newIdx, parent.children.length - 1));
      if (value.key) {
        const key = value.key;
        const found = parent.children.findIndex((c) => c && c.key === key);
        if (found >= 0) {
          fromIdx = found;
        }
      }
      if (fromIdx < 0 || fromIdx >= parent.children.length) {
        Logger.warn("Patcher", "Cannot move missing child", { fromIdx, value });
        return;
      }
      const child = parent.children[fromIdx];
      parent.children.splice(fromIdx, 1);
      const insertIdx = Math.max(0, Math.min(toIdx, parent.children.length));
      parent.children.splice(insertIdx, 0, child);
      if (!parent.el || !child.el) return;
      let referenceNode = null;
      for (let i = insertIdx + 1; i < parent.children.length; i++) {
        if (parent.children[i].el) {
          referenceNode = parent.children[i].el;
          break;
        }
      }
      if (referenceNode) {
        parent.el.insertBefore(child.el, referenceNode);
      } else {
        parent.el.appendChild(child.el);
      }
    }
    setRef(node, refId) {
      node.refId = refId;
      this.refs.set(refId, node);
    }
    delRef(node) {
      if (node.refId) {
        this.refs.delete(node.refId);
        delete node.refId;
      }
    }
    setComment(node, comment) {
      if (node.el) {
        node.el.textContent = comment;
      }
      node.comment = comment;
    }
    setStyle(node, styles) {
      if (node.el && node.el instanceof HTMLElement) {
        for (const [name, value] of Object.entries(styles)) {
          node.el.style.setProperty(name, value);
        }
      }
      if (!node.style) node.style = {};
      Object.assign(node.style, styles);
    }
    setStyleDecl(node, selector, name, value) {
      if (node.el && node.el instanceof HTMLStyleElement && node.el.sheet) {
        const sheet = node.el.sheet;
        for (let i = 0; i < sheet.cssRules.length; i++) {
          const rule = sheet.cssRules[i];
          if (rule.selectorText === selector) {
            rule.style.setProperty(name, value);
            return;
          }
        }
        const idx = sheet.cssRules.length;
        try {
          sheet.insertRule(`${selector} { ${name}: ${value}; }`, idx);
        } catch (e) {
          Logger.warn("Patcher", "Failed to insert rule", { selector, error: e });
        }
      }
    }
    delStyleDecl(node, selector, name) {
      if (node.el && node.el instanceof HTMLStyleElement && node.el.sheet) {
        const sheet = node.el.sheet;
        for (let i = 0; i < sheet.cssRules.length; i++) {
          const rule = sheet.cssRules[i];
          if (rule.selectorText === selector) {
            rule.style.removeProperty(name);
            return;
          }
        }
      }
    }
    delStyle(node, name) {
      if (node.el && node.el instanceof HTMLElement) {
        node.el.style.removeProperty(name);
      }
      if (node.style) delete node.style[name];
    }
    setHandlers(node, handlers) {
      this.events.detach(node);
      node.handlers = handlers;
      this.events.attach(node);
    }
    setRouter(node, router) {
      this.router.detach(node);
      node.router = router;
      this.router.attach(node);
    }
    delRouter(node) {
      this.router.detach(node);
      delete node.router;
    }
    setUpload(node, meta) {
      this.uploads.bind(node, meta);
    }
    delUpload(node) {
      this.uploads.unbind(node);
    }
    detachRefsRecursively(node) {
      this.uploads.unbind(node);
      if (node.refId) {
        this.refs.delete(node.refId);
      }
      if (node.children) {
        for (const child of node.children) {
          this.detachRefsRecursively(child);
        }
      }
    }
    buildStyleContent(styles) {
      const blocks = [];
      for (const [selector, props] of Object.entries(styles)) {
        const entries = [];
        for (const [name, value] of Object.entries(props)) {
          entries.push(`${name}: ${value};`);
        }
        if (entries.length > 0) {
          blocks.push(`${selector} { ${entries.join(" ")} }`);
        }
      }
      return blocks.join("\n");
    }
  };

  // src/event-detail.ts
  function extractEventDetail(event, props, options) {
    if (!Array.isArray(props) || props.length === 0) {
      return void 0;
    }
    const detail = {};
    props.forEach((path) => {
      if (typeof path !== "string" || path.length === 0) {
        return;
      }
      const value = resolvePath(path, event, options);
      if (value !== void 0) {
        detail[path] = value;
      }
    });
    return Object.keys(detail).length > 0 ? detail : void 0;
  }
  function resolvePath(path, event, options) {
    const segments = path.split(".").map((segment) => segment.trim()).filter(Boolean);
    if (segments.length === 0) {
      return void 0;
    }
    const root = segments.shift();
    let current;
    switch (root) {
      case "event":
        current = event;
        break;
      case "target":
        current = event.target ?? null;
        break;
      case "currentTarget":
        current = event.currentTarget ?? null;
        break;
      case "element":
      case "ref":
        current = options?.refElement ?? (event.currentTarget instanceof Element ? event.currentTarget : null);
        break;
      default:
        return void 0;
    }
    for (const segment of segments) {
      if (current == null) {
        return void 0;
      }
      try {
        current = current[segment];
      } catch {
        return void 0;
      }
    }
    return serializeValue(current);
  }
  function serializeValue(value) {
    if (value === null || value === void 0) {
      return null;
    }
    const type = typeof value;
    if (type === "string" || type === "number" || type === "boolean") {
      return value;
    }
    if (Array.isArray(value)) {
      const mapped = value.map(serializeValue).filter((entry) => entry !== void 0);
      return mapped.length > 0 ? mapped : null;
    }
    if (value instanceof Date) {
      return value.toISOString();
    }
    if (value instanceof DOMTokenList) {
      return Array.from(value);
    }
    if (value instanceof Node) {
      return void 0;
    }
    try {
      return JSON.parse(JSON.stringify(value));
    } catch {
      return void 0;
    }
  }

  // src/events.ts
  var EventManager = class {
    constructor(channel, sid) {
      this.channel = channel;
      this.sid = sid;
      this.listeners = /* @__PURE__ */ new WeakMap();
    }
    attach(node) {
      if (!node) return;
      if (node.el && node.handlers && node.handlers.length > 0) {
        this.bindEvents(node);
      }
      if (node.children) {
        for (const child of node.children) {
          this.attach(child);
        }
      }
    }
    detach(node) {
      if (!node) return;
      if (node.el) {
        this.unbindEvents(node.el);
      }
      if (node.children) {
        for (const child of node.children) {
          this.detach(child);
        }
      }
    }
    bindEvents(node) {
      if (!node.el || !node.handlers) return;
      const el = node.el;
      let nodeListeners = this.listeners.get(el);
      if (!nodeListeners) {
        nodeListeners = /* @__PURE__ */ new Map();
        this.listeners.set(el, nodeListeners);
      }
      for (const h of node.handlers) {
        if (!h || !h.event || !h.handler) continue;
        const existing = nodeListeners.get(h.event) || [];
        const duplicate = existing.some((rec) => rec.handlerId === h.handler);
        if (duplicate) {
          continue;
        }
        const listener = (e) => {
          const preventDefault = !(h.listen && h.listen.includes("allowDefault"));
          if (preventDefault && e.cancelable) {
            e.preventDefault();
          }
          this.triggerHandler(h, e, node);
          if (!h.listen || !h.listen.includes("bubble")) {
            e.stopPropagation();
          }
        };
        el.addEventListener(h.event, listener);
        nodeListeners.set(h.event, [...existing, { handlerId: h.handler, listener }]);
        Logger.debug("Events", "Attached listener", { event: h.event, handler: h.handler });
      }
    }
    unbindEvents(el) {
      const nodeListeners = this.listeners.get(el);
      if (!nodeListeners) return;
      for (const [event, records] of nodeListeners.entries()) {
        for (const rec of records) {
          el.removeEventListener(event, rec.listener);
        }
      }
      this.listeners.delete(el);
    }
    triggerHandler(handler, e, node) {
      Logger.debug("Events", "Triggering handler", { handlerId: handler.handler, type: e.type });
      const refElement = node.el instanceof Element ? node.el : void 0;
      const detail = extractEventDetail(e, handler.props, { refElement });
      const payload = {
        name: e.type
      };
      if (detail !== void 0) {
        payload.detail = detail;
      }
      this.channel.sendMessage("evt", {
        t: "evt",
        sid: this.sid,
        hid: handler.handler,
        payload
      });
    }
  };

  // src/router.ts
  var Router = class {
    constructor(channel, sessionId) {
      this.channel = channel;
      this.sessionId = sessionId;
      this.listeners = /* @__PURE__ */ new WeakMap();
      window.addEventListener("popstate", (e) => this.onPopState(e));
    }
    attach(node) {
      if (!node || !node.router || !node.el) return;
      const el = node.el;
      if (this.listeners.has(el)) return;
      const listener = (e) => {
        e.preventDefault();
        this.navigate(node.router);
      };
      el.addEventListener("click", listener);
      this.listeners.set(el, listener);
    }
    detach(node) {
      if (!node || !node.el) return;
      const el = node.el;
      const listener = this.listeners.get(el);
      if (listener) {
        el.removeEventListener("click", listener);
        this.listeners.delete(el);
      }
    }
    navigate(meta) {
      const path = meta.path ?? window.location.pathname;
      const query = meta.query !== void 0 ? meta.query : window.location.search;
      const hash = meta.hash !== void 0 ? meta.hash : window.location.hash;
      const cleanQuery = query.startsWith("?") ? query.substring(1) : query;
      const url = path + (cleanQuery ? "?" + cleanQuery : "") + (hash ? "#" + hash : "");
      if (meta.replace) {
        window.history.replaceState({}, "", url);
      } else {
        window.history.pushState({}, "", url);
      }
      this.sendNav("nav", path, cleanQuery, hash);
    }
    onPopState(_e) {
      const path = window.location.pathname;
      const query = window.location.search;
      const hash = window.location.hash;
      this.sendNav("pop", path, query, hash);
    }
    sendNav(type, path, query, hash) {
      Logger.debug("Router", `Sending ${type}`, { path, query, hash });
      const q = query.startsWith("?") ? query.substring(1) : query;
      this.channel.sendMessage(type, {
        sid: this.sessionId,
        path,
        q,
        hash
      });
    }
  };

  // src/dom_actions.ts
  var DOMActionExecutor = class {
    constructor(refs) {
      this.refs = refs;
    }
    execute(effects) {
      if (!effects || effects.length === 0) return;
      for (const effect of effects) {
        this.executeOne(effect);
      }
    }
    executeOne(effect) {
      const node = this.refs.get(effect.ref);
      if (!node || !node.el) {
        Logger.warn("DOMAction", "Ref not found", { ref: effect.ref });
        return;
      }
      const el = node.el;
      try {
        switch (effect.kind) {
          case "dom.call":
            if (effect.method && typeof el[effect.method] === "function") {
              el[effect.method](...effect.args || []);
            } else {
              Logger.warn("DOMAction", "Method not found", { method: effect.method });
            }
            break;
          case "dom.set":
            if (effect.prop) {
              el[effect.prop] = effect.value;
            }
            break;
          case "dom.toggle":
            if (effect.prop) {
              el[effect.prop] = !el[effect.prop];
            }
            break;
          case "dom.class":
            if (effect.class) {
              if (effect.on === true) {
                el.classList.add(effect.class);
              } else if (effect.on === false) {
                el.classList.remove(effect.class);
              } else {
                el.classList.toggle(effect.class);
              }
            }
            break;
          case "dom.scroll":
            if (el.scrollIntoView) {
              const opts = {};
              if (effect.behavior) opts.behavior = effect.behavior;
              if (effect.block) opts.block = effect.block;
              if (effect.inline) opts.inline = effect.inline;
              el.scrollIntoView(opts);
            }
            break;
          default:
            Logger.warn("DOMAction", "Unknown action kind", { kind: effect.kind });
        }
      } catch (e) {
        Logger.error("DOMAction", "Execution failed", e);
      }
    }
  };

  // src/uploads.ts
  var UploadManager = class {
    constructor(runtime) {
      this.runtime = runtime;
      this.bindings = /* @__PURE__ */ new Map();
      this.active = /* @__PURE__ */ new Map();
    }
    bind(node, meta) {
      if (!node.el || !(node.el instanceof HTMLInputElement)) {
        Logger.warn("Uploads", "Upload binding requires an input element", node);
        return;
      }
      const element = node.el;
      const uploadId = meta.uploadId;
      this.unbind(node);
      const handler = () => this.handleInputChange(uploadId, element, meta);
      element.addEventListener("change", handler);
      if (meta.accept && meta.accept.length > 0) {
        element.setAttribute("accept", meta.accept.join(","));
      } else {
        element.removeAttribute("accept");
      }
      if (meta.multiple) {
        Logger.warn("Uploads", "Multiple file selection not supported; forcing single file");
        element.removeAttribute("multiple");
      } else {
        element.removeAttribute("multiple");
      }
      this.bindings.set(uploadId, { node, element, meta, changeHandler: handler });
      Logger.debug("Uploads", "Bound upload", uploadId);
    }
    unbind(node) {
      for (const [id, binding] of this.bindings.entries()) {
        if (binding.node === node) {
          this.detachBinding(id);
          return;
        }
      }
    }
    detachBinding(uploadId) {
      const binding = this.bindings.get(uploadId);
      if (!binding) return;
      binding.element.removeEventListener("change", binding.changeHandler);
      this.bindings.delete(uploadId);
      this.abortUpload(uploadId, false);
      Logger.debug("Uploads", "Unbound upload", uploadId);
    }
    handleControl(message) {
      if (!message || !message.id) return;
      Logger.debug("Uploads", "Control message", message);
      if (message.op === "cancel" || message.op === "error") {
        this.abortUpload(message.id, true);
      }
    }
    handleInputChange(uploadId, element, meta) {
      const files = element.files;
      if (!files || files.length === 0) {
        this.sendMessage({ op: "cancelled", id: uploadId });
        this.abortUpload(uploadId, true);
        return;
      }
      const file = files[0];
      if (meta.multiple && files.length > 1) {
        this.sendMessage({
          op: "error",
          id: uploadId,
          error: "Multiple file uploads are not supported yet"
        });
        element.value = "";
        return;
      }
      if (!file) {
        this.sendMessage({ op: "cancelled", id: uploadId });
        return;
      }
      if (meta.maxSize && meta.maxSize > 0 && file.size > meta.maxSize) {
        this.sendMessage({
          op: "error",
          id: uploadId,
          error: `File exceeds maximum size (${meta.maxSize} bytes)`
        });
        element.value = "";
        return;
      }
      const fileMeta = { name: file.name, size: file.size, type: file.type };
      this.sendMessage({ op: "change", id: uploadId, meta: fileMeta });
      this.startUpload(uploadId, file, element);
    }
    startUpload(uploadId, file, element) {
      const sid = this.runtime.getSessionId();
      if (!sid) return;
      const base = this.runtime.getUploadEndpoint();
      const target = `${base.replace(/\/+$/, "")}/${encodeURIComponent(sid)}/${encodeURIComponent(uploadId)}`;
      this.abortUpload(uploadId, false);
      const xhr = new XMLHttpRequest();
      xhr.upload.onprogress = (event) => {
        const loaded = event.loaded;
        const total = event.lengthComputable ? event.total : file.size;
        this.sendMessage({ op: "progress", id: uploadId, loaded, total });
      };
      xhr.onerror = () => {
        this.active.delete(uploadId);
        this.sendMessage({ op: "error", id: uploadId, error: "Upload failed" });
      };
      xhr.onabort = () => {
        this.active.delete(uploadId);
        this.sendMessage({ op: "cancelled", id: uploadId });
      };
      xhr.onload = () => {
        this.active.delete(uploadId);
        if (xhr.status < 200 || xhr.status >= 300) {
          this.sendMessage({ op: "error", id: uploadId, error: `Upload failed (${xhr.status})` });
        } else {
          this.sendMessage({ op: "progress", id: uploadId, loaded: file.size, total: file.size });
          element.value = "";
        }
      };
      const form = new FormData();
      form.append("file", file);
      xhr.open("POST", target, true);
      xhr.send(form);
      this.active.set(uploadId, { xhr, element });
      Logger.debug("Uploads", "Started upload", { uploadId, target });
    }
    abortUpload(uploadId, clearInput) {
      const active = this.active.get(uploadId);
      if (!active) return;
      active.xhr.abort();
      if (clearInput) {
        active.element.value = "";
      }
      this.active.delete(uploadId);
    }
    sendMessage(payload) {
      this.runtime.sendUploadMessage(payload);
    }
  };

  // src/runtime.ts
  var LiveRuntime = class {
    constructor() {
      this.config = {};
      this.root = null;
      this.refs = /* @__PURE__ */ new Map();
      this.sessionId = "";
      const boot = this.getBootPayload();
      if (!boot) {
        Logger.error("Runtime", "No boot payload found");
        return;
      }
      this.sessionId = boot.sid;
      this.config = boot.client || {};
      Logger.configure({ debug: boot.client?.debug });
      Logger.debug("Runtime", "Booting...", boot);
      this.connect(boot);
      this.hydrate(boot);
    }
    getBootPayload() {
      if (typeof window === "undefined") return null;
      const script = document.getElementById("live-boot");
      if (script && script.textContent) {
        try {
          return JSON.parse(script.textContent);
        } catch (e) {
          Logger.error("Runtime", "Failed to parse boot payload", e);
        }
      }
      return window.__LIVEUI_BOOT__ || null;
    }
    hydrate(boot) {
      try {
        let findHtmlElement2 = function(node) {
          if (node.tag === "html") {
            return document.documentElement;
          }
          if (node.children) {
            for (const child of node.children) {
              const result = findHtmlElement2(child);
              if (result) return result;
            }
          }
          return null;
        };
        var findHtmlElement = findHtmlElement2;
        const jsonTree = JSON.parse(boot.json);
        const htmlElement = findHtmlElement2(jsonTree);
        if (!htmlElement) {
          Logger.error("Runtime", "Could not find <html> element in JSON tree");
          return;
        }
        this.root = this.hydrateWithComponentWrappers(jsonTree, htmlElement);
        if (this.eventManager && this.root) {
          this.eventManager.attach(this.root);
        }
        if (this.router && this.root) {
          this.attachRouterRecursively(this.root);
        }
        if (this.eventManager && this.router && this.uploadManager) {
          this.patcher = new Patcher(this.root, this.eventManager, this.router, this.uploadManager, this.refs);
        }
        Logger.debug("Runtime", "Hydration complete");
      } catch (e) {
        Logger.error("Runtime", "Hydration failed", e);
      }
    }
    hydrateWithComponentWrappers(jsonNode, htmlElement) {
      if (jsonNode.tag === "html") {
        return hydrate(jsonNode, htmlElement, this.refs);
      }
      const clientNode = {
        ...jsonNode,
        el: null,
        children: void 0
      };
      if (jsonNode.componentId) {
        clientNode.componentId = jsonNode.componentId;
      }
      if (jsonNode.children && jsonNode.children.length > 0) {
        clientNode.children = [];
        for (const child of jsonNode.children) {
          const childNode = this.hydrateWithComponentWrappers(child, htmlElement);
          clientNode.children.push(childNode);
        }
      }
      return clientNode;
    }
    connect(boot) {
      const endpoint = boot.client?.endpoint || "/live";
      this.client = new import_pondsocket_client.PondClient(endpoint);
      const joinPayload = {
        sid: boot.sid,
        ver: boot.ver,
        ack: boot.seq,
        loc: boot.location
      };
      this.channel = this.client.createChannel(`live/${boot.sid}`, joinPayload);
      this.eventManager = new EventManager(this.channel, boot.sid);
      this.router = new Router(this.channel, boot.sid);
      this.uploadManager = new UploadManager(this);
      this.domActions = new DOMActionExecutor(this.refs);
      this.channel.onChannelStateChange((state) => {
        Logger.debug("Runtime", "Channel state:", state);
      });
      this.channel.onMessage((_event, payload) => {
        this.handleMessage(payload);
      });
      this.client.connect();
      this.channel.join();
    }
    getSessionId() {
      return this.sessionId;
    }
    getUploadEndpoint() {
      return this.config.upload || "/pondlive/upload";
    }
    sendUploadMessage(payload) {
      this.channel.sendMessage({ t: "upload", ...payload });
    }
    handleMessage(msg) {
      switch (msg.t) {
        case "frame":
          this.handleFrame(msg);
          break;
        case "init":
          this.handleInit(msg);
          break;
        case "domreq":
          this.handleDOMRequest(msg);
          break;
        case "upload":
          this.uploadManager.handleControl(msg);
          break;
        default:
          Logger.debug("Runtime", "Unknown message type", msg.t);
      }
    }
    handleFrame(frame) {
      Logger.debug("Runtime", "Received frame", { seq: frame.seq, ops: frame.patch.length });
      if (this.patcher && frame.patch) {
        for (const op of frame.patch) {
          this.patcher.apply(op);
        }
      }
      if (frame.effects) {
        this.domActions.execute(frame.effects);
      }
    }
    handleInit(init) {
      Logger.debug("Runtime", "Re-initialized", init);
    }
    handleDOMRequest(req) {
      const { id, ref, props, method, args } = req;
      const node = this.refs.get(ref);
      if (!node || !node.el) {
        this.sendDOMResponse({ t: "domres", id, error: "ref not found" });
        return;
      }
      const el = node.el;
      try {
        let result;
        let values;
        if (props && Array.isArray(props)) {
          values = {};
          for (const prop of props) {
            values[prop] = el[prop];
          }
        }
        if (method && typeof el[method] === "function") {
          result = el[method](...args || []);
        }
        this.sendDOMResponse({ t: "domres", id, result, values });
      } catch (e) {
        this.sendDOMResponse({ t: "domres", id, error: e.message || "unknown error" });
      }
    }
    sendDOMResponse(response) {
      this.channel.sendMessage("domres", {
        ...response,
        sid: this.sessionId
      });
    }
    attachRouterRecursively(node) {
      if (this.router) {
        this.router.attach(node);
      }
      if (node.children) {
        for (const child of node.children) {
          this.attachRouterRecursively(child);
        }
      }
    }
  };

  // src/entry.ts
  if (typeof window !== "undefined") {
    window.addEventListener("DOMContentLoaded", () => {
      const instance = new LiveRuntime();
      window.__LIVEUI__ = instance;
    });
  }
})();
//# sourceMappingURL=pondlive-dev.js.map

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
  var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
    // If the importer is in node compatibility mode or this is not an ESM
    // file that has been converted to a CommonJS file using a Babel-
    // compatible transform (i.e. "__esModule" has not been set), then set
    // "default" to the CommonJS "module.exports" for node compatibility.
    isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
    mod
  ));
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

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
      var ChannelState;
      (function(ChannelState2) {
        ChannelState2["IDLE"] = "IDLE";
        ChannelState2["JOINING"] = "JOINING";
        ChannelState2["JOINED"] = "JOINED";
        ChannelState2["STALLED"] = "STALLED";
        ChannelState2["CLOSED"] = "CLOSED";
      })(ChannelState || (exports.ChannelState = ChannelState = {}));
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

  // src/entry.ts
  var entry_exports = {};
  __export(entry_exports, {
    ComputedSignal: () => ComputedSignal,
    Signal: () => Signal,
    applyOps: () => applyOps,
    bootClient: () => bootClient,
    clearPatcherCaches: () => clearPatcherCaches,
    configurePatcher: () => configurePatcher,
    default: () => index_default,
    dom: () => dom_index_exports,
    getPatcherConfig: () => getPatcherConfig,
    getPatcherStats: () => getPatcherStats,
    getRefElement: () => getRefElement,
    getRefMeta: () => getRefMeta,
    morphElement: () => morphElement
  });

  // src/index.ts
  var import_pondsocket_client = __toESM(require_pondsocket_client(), 1);

  // src/dom-index.ts
  var dom_index_exports = {};
  __export(dom_index_exports, {
    deleteRow: () => deleteRow,
    ensureList: () => ensureList,
    getRow: () => getRow,
    getSlot: () => getSlot,
    initLists: () => initLists,
    registerList: () => registerList,
    registerSlot: () => registerSlot,
    reset: () => reset,
    setRow: () => setRow,
    unregisterList: () => unregisterList,
    unregisterSlot: () => unregisterSlot
  });

  // src/refs.ts
  var registry = /* @__PURE__ */ new Map();
  function escapeAttributeValue(value) {
    if (typeof value !== "string") {
      return "";
    }
    if (typeof CSS !== "undefined" && typeof CSS.escape === "function") {
      return CSS.escape(value);
    }
    return value.replace(/["]|\\/g, "\\$&");
  }
  function queryRefElement(id) {
    if (typeof document === "undefined" || !id) {
      return null;
    }
    const selector = `[data-live-ref="${escapeAttributeValue(id)}"]`;
    try {
      return document.querySelector(selector);
    } catch (_error) {
      return document.querySelector(`[data-live-ref="${id}"]`);
    }
  }
  function ensureRecord(id, meta) {
    let record = registry.get(id);
    if (!record) {
      record = {
        id,
        meta: meta ?? { tag: "" },
        element: queryRefElement(id)
      };
      registry.set(id, record);
    } else if (meta) {
      record.meta = meta;
      if (!record.element) {
        record.element = queryRefElement(id);
      }
    }
    return record;
  }
  function collectRefEventProps(meta, eventType) {
    if (!meta || !meta.events) {
      return [];
    }
    const selectors = [];
    const seen = /* @__PURE__ */ new Set();
    for (const [primary, eventMeta] of Object.entries(meta.events)) {
      if (!eventMeta) {
        continue;
      }
      const listens = Array.isArray(eventMeta.listen) ? eventMeta.listen : [];
      if (primary !== eventType && !listens.includes(eventType)) {
        continue;
      }
      if (!Array.isArray(eventMeta.props)) {
        continue;
      }
      for (const prop of eventMeta.props) {
        if (typeof prop !== "string" || prop.length === 0) {
          continue;
        }
        if (seen.has(prop)) {
          continue;
        }
        seen.add(prop);
        selectors.push(prop);
      }
    }
    return selectors;
  }
  function resolveRefEventContext(element, eventType) {
    if (!element || !eventType) {
      return null;
    }
    const refElement = findClosestRefElement(element);
    if (!refElement) {
      return null;
    }
    const id = refElement.getAttribute("data-live-ref");
    if (!id) {
      return null;
    }
    const record = ensureRecord(id);
    record.element = refElement;
    if (!refSupportsEvent(record.meta, eventType)) {
      return null;
    }
    const props = collectRefEventProps(record.meta, eventType);
    return {
      id,
      element: refElement,
      props,
      notify(payload) {
        if (!record.lastPayloads) {
          record.lastPayloads = /* @__PURE__ */ Object.create(null);
        }
        record.lastPayloads[eventType] = { ...payload };
      }
    };
  }
  function attachRef(id, element) {
    if (!id) {
      return;
    }
    const record = ensureRecord(id);
    record.element = element;
    if (!record.meta.tag) {
      record.meta = { ...record.meta, tag: element.tagName.toLowerCase() };
    }
  }
  function detachRef(id, element) {
    const record = registry.get(id);
    if (!record) {
      return;
    }
    if (!element || record.element === element) {
      record.element = null;
    }
  }
  function forEachRefElement(root, visit) {
    if (root instanceof Element) {
      visit(root);
    }
    const selectorAll = root.querySelectorAll?.bind(root);
    if (typeof selectorAll !== "function") {
      return;
    }
    const matches = selectorAll("[data-live-ref]");
    matches.forEach((node) => {
      if (node instanceof Element) {
        visit(node);
      }
    });
  }
  function clearRefs() {
    registry.clear();
  }
  function registerRefs(refs) {
    if (!refs) {
      return;
    }
    for (const [id, meta] of Object.entries(refs)) {
      if (!id) {
        continue;
      }
      ensureRecord(id, meta);
    }
  }
  function unregisterRefs(ids) {
    if (!Array.isArray(ids)) {
      return;
    }
    for (const id of ids) {
      if (!id) {
        continue;
      }
      registry.delete(id);
    }
  }
  function bindRefsInTree(root) {
    if (!root || typeof document === "undefined") {
      return;
    }
    forEachRefElement(root, (el) => {
      const id = el.getAttribute("data-live-ref");
      if (id) {
        attachRef(id, el);
      }
    });
  }
  function unbindRefsInTree(root) {
    if (!root || typeof document === "undefined") {
      return;
    }
    forEachRefElement(root, (el) => {
      const id = el.getAttribute("data-live-ref");
      if (id) {
        detachRef(id, el);
      }
    });
  }
  function updateRefBinding(element, previousId, nextId) {
    if (previousId && (!nextId || previousId !== nextId)) {
      detachRef(previousId, element);
    }
    if (nextId) {
      attachRef(nextId, element);
    } else if (previousId && !nextId) {
      detachRef(previousId, element);
    }
  }
  function getRefElement(id) {
    const record = registry.get(id);
    return record?.element ?? null;
  }
  function getRefMeta(id) {
    const record = registry.get(id);
    return record?.meta ?? null;
  }
  function findClosestRefElement(element) {
    let current = element;
    while (current) {
      if (typeof current.getAttribute === "function") {
        const id = current.getAttribute("data-live-ref");
        if (id) {
          return current;
        }
      }
      current = current.parentElement;
    }
    return null;
  }
  function refSupportsEvent(meta, eventType) {
    if (!meta || !meta.events) {
      return false;
    }
    for (const [primary, eventMeta] of Object.entries(meta.events)) {
      if (primary === eventType) {
        return true;
      }
      if (eventMeta?.listen && eventMeta.listen.includes(eventType)) {
        return true;
      }
    }
    return false;
  }

  // src/dom.ts
  function pickSource(scope, ctx) {
    switch (scope) {
      case "event":
        return ctx.event ?? void 0;
      case "target":
        return ctx.target ?? void 0;
      case "currentTarget":
        if (ctx.handlerElement) {
          return ctx.handlerElement;
        }
        return ctx.event && "currentTarget" in ctx.event ? ctx.event.currentTarget ?? void 0 : void 0;
      case "element":
      case "ref":
        return ctx.refElement ?? ctx.handlerElement ?? ctx.target ?? void 0;
      default:
        return void 0;
    }
  }
  function resolvePath(source, parts) {
    let value = source;
    for (const part of parts) {
      if (!part) {
        continue;
      }
      if (value == null) {
        return void 0;
      }
      try {
        value = value[part];
      } catch (_error) {
        return void 0;
      }
    }
    return value;
  }
  function resolvePropertySelector(selector, ctx) {
    if (typeof selector !== "string") {
      return void 0;
    }
    const trimmed = selector.trim();
    if (!trimmed) {
      return void 0;
    }
    const parts = trimmed.split(".");
    if (parts.length === 0) {
      return void 0;
    }
    const scope = parts.shift();
    if (!scope) {
      return void 0;
    }
    const source = pickSource(scope, ctx);
    if (source === void 0) {
      return void 0;
    }
    return resolvePath(source, parts);
  }
  function domGetSync(selectors, ctx) {
    if (!Array.isArray(selectors) || selectors.length === 0) {
      return null;
    }
    const result = {};
    for (const selector of selectors) {
      if (typeof selector !== "string" || selector.length === 0) {
        continue;
      }
      const raw = resolvePropertySelector(selector, ctx);
      if (raw === void 0) {
        continue;
      }
      const normalized = normalizePropertyValue(raw);
      if (normalized !== void 0) {
        result[selector] = normalized;
      }
    }
    return Object.keys(result).length > 0 ? result : null;
  }
  function normalizePropertyValue(value) {
    if (value === null) {
      return null;
    }
    const type = typeof value;
    if (type === "string" || type === "number" || type === "boolean") {
      return value;
    }
    if (value instanceof Date) {
      return value.toISOString();
    }
    if (typeof FileList !== "undefined" && value instanceof FileList) {
      return serializeFileList(value);
    }
    if (typeof File !== "undefined" && value instanceof File) {
      return { name: value.name, size: value.size, type: value.type };
    }
    if (typeof DOMTokenList !== "undefined" && value instanceof DOMTokenList) {
      return Array.from(value);
    }
    if (typeof TimeRanges !== "undefined" && value instanceof TimeRanges) {
      const ranges = [];
      for (let i = 0; i < value.length; i++) {
        try {
          ranges.push({ start: value.start(i), end: value.end(i) });
        } catch (_error) {
        }
      }
      return ranges;
    }
    if (Array.isArray(value)) {
      return value.map((item) => normalizePropertyValue(item)).filter((item) => item !== void 0);
    }
    if (value && typeof value === "object") {
      try {
        return JSON.parse(JSON.stringify(value));
      } catch (_error) {
        return void 0;
      }
    }
    return void 0;
  }
  function serializeFileList(list) {
    const files = [];
    for (let i = 0; i < list.length; i++) {
      const file = list.item(i);
      if (file) {
        files.push({ name: file.name, size: file.size, type: file.type });
      }
    }
    return files;
  }
  function callElementMethod(refId, method, args = [], options) {
    const element = refId ? getRefElement(refId) : null;
    if (!element) {
      if (!options?.allowMissing) {
        console.warn("[LiveUI] DOMCall target missing", { refId, method });
      }
      return void 0;
    }
    const fn = element[method];
    if (typeof fn !== "function") {
      console.warn("[LiveUI] DOMCall method missing", { refId, method });
      return void 0;
    }
    try {
      const result = fn.apply(element, Array.isArray(args) ? args : []);
      if (result && typeof result.then === "function") {
        result.catch((error) => {
          console.warn("[LiveUI] DOMCall promise rejected", {
            refId,
            method,
            error
          });
        });
      }
      return result;
    } catch (error) {
      console.error("[LiveUI] DOMCall failed", { refId, method, error });
      return void 0;
    }
  }

  // src/events.ts
  var handlers = /* @__PURE__ */ new Map();
  var eventUsageCounts = /* @__PURE__ */ new Map();
  var installedListeners = /* @__PURE__ */ new Map();
  var handlerBindings = /* @__PURE__ */ new WeakMap();
  var slotBindingSpecs = /* @__PURE__ */ new Map();
  var slotElements = /* @__PURE__ */ new Map();
  var routerBindings = /* @__PURE__ */ new WeakMap();
  var DATA_EVENT_ATTR_PREFIX = "data-on";
  var DATA_ROUTER_ATTR_PREFIX2 = "data-router-";
  var ALWAYS_ACTIVE_EVENTS = ["click", "input", "change", "submit"];
  function setRouterMetaValue(element, field, value) {
    let meta = routerBindings.get(element);
    if (value === null || value === void 0 || value === "") {
      if (!meta) {
        return;
      }
      delete meta[field];
      if (!meta.path && !meta.query && !meta.hash && !meta.replace) {
        routerBindings.delete(element);
      }
      return;
    }
    if (!meta) {
      meta = {};
      routerBindings.set(element, meta);
    }
    meta[field] = value;
  }
  function applyRouterAttribute(element, key, value) {
    if (!element || typeof key !== "string" || key.length === 0) {
      return;
    }
    switch (key) {
      case "path":
      case "query":
      case "hash":
      case "replace":
        setRouterMetaValue(element, key, value);
        break;
      default:
        break;
    }
  }
  function clearRouterAttributes(element) {
    if (!element) {
      return;
    }
    routerBindings.delete(element);
  }
  function cloneSlotBindingList(specs) {
    if (!Array.isArray(specs)) {
      return [];
    }
    return specs.map((spec) => {
      const clone = {
        event: spec?.event || "",
        handler: spec?.handler || ""
      };
      if (Array.isArray(spec?.listen) && spec.listen.length > 0) {
        clone.listen = [...spec.listen];
      }
      if (Array.isArray(spec?.props) && spec.props.length > 0) {
        clone.props = [...spec.props];
      }
      return clone;
    });
  }
  function applySlotBindings(slotId) {
    const element = slotElements.get(slotId);
    if (!element) {
      return;
    }
    const specs = slotBindingSpecs.get(slotId) ?? [];
    if (!Array.isArray(specs) || specs.length === 0) {
      handlerBindings.delete(element);
      return;
    }
    const map = /* @__PURE__ */ new Map();
    for (const spec of specs) {
      if (!spec || typeof spec.event !== "string" || typeof spec.handler !== "string") {
        continue;
      }
      const eventName = spec.event.trim();
      const handlerId = spec.handler.trim();
      if (eventName.length === 0 || handlerId.length === 0) {
        continue;
      }
      map.set(eventName, handlerId);
    }
    if (map.size === 0) {
      handlerBindings.delete(element);
      return;
    }
    handlerBindings.set(element, map);
  }
  function primeSlotBindings(table) {
    slotBindingSpecs.clear();
    if (table && typeof table === "object") {
      for (const [key, value] of Object.entries(table)) {
        const slotId = Number(key);
        if (Number.isNaN(slotId)) {
          continue;
        }
        slotBindingSpecs.set(slotId, cloneSlotBindingList(value));
      }
    }
    slotElements.forEach((_element, slotId) => {
      applySlotBindings(slotId);
    });
  }
  function registerBindingsForSlot(slotId, specs) {
    if (!Number.isFinite(slotId)) {
      return;
    }
    if (!Array.isArray(specs)) {
      slotBindingSpecs.set(slotId, []);
    } else {
      slotBindingSpecs.set(slotId, cloneSlotBindingList(specs));
    }
    applySlotBindings(slotId);
  }
  function getRegisteredSlotBindings(slotId) {
    const specs = slotBindingSpecs.get(slotId);
    if (!specs) {
      return void 0;
    }
    return cloneSlotBindingList(specs);
  }
  function onSlotRegistered(slotId, node) {
    if (!Number.isFinite(slotId) || !node) {
      return;
    }
    if (node instanceof Element) {
      slotElements.set(slotId, node);
      applySlotBindings(slotId);
      return;
    }
    slotElements.delete(slotId);
  }
  function onSlotUnregistered(slotId) {
    const element = slotElements.get(slotId);
    if (element) {
      handlerBindings.delete(element);
      clearRouterAttributes(element);
    }
    slotElements.delete(slotId);
  }
  function mergeSelectorLists(primary, secondary) {
    const merged = [];
    const seen = /* @__PURE__ */ new Set();
    const add = (list) => {
      if (!Array.isArray(list)) {
        return;
      }
      for (const value of list) {
        if (typeof value !== "string" || value.length === 0) {
          continue;
        }
        if (seen.has(value)) {
          continue;
        }
        seen.add(value);
        merged.push(value);
      }
    };
    add(primary);
    add(secondary);
    return merged.length > 0 ? merged : void 0;
  }
  function isLiveAnchor(element) {
    const hasHref = element.hasAttribute("href") || element.hasAttribute("xlink:href");
    if (!hasHref) {
      return false;
    }
    const isHtmlAnchor = element instanceof HTMLAnchorElement;
    const isSvgAnchor = element.namespaceURI === "http://www.w3.org/2000/svg" && element.tagName.toLowerCase() === "a";
    return isHtmlAnchor || isSvgAnchor;
  }
  function registerHandlers(handlerMap) {
    if (!handlerMap) return;
    for (const [id, meta] of Object.entries(handlerMap)) {
      const previous = handlers.get(id);
      if (previous) {
        for (const eventName of collectEventTypes(previous)) {
          decrementEventUsage(eventName);
        }
      }
      handlers.set(id, meta);
      for (const eventName of collectEventTypes(meta)) {
        incrementEventUsage(eventName);
      }
    }
  }
  function unregisterHandlers(handlerIds) {
    if (!Array.isArray(handlerIds)) return;
    for (const id of handlerIds) {
      const handler = handlers.get(id);
      handlers.delete(id);
      if (handler) {
        for (const eventName of collectEventTypes(handler)) {
          decrementEventUsage(eventName);
        }
      }
    }
  }
  function clearHandlers() {
    handlers.clear();
    eventUsageCounts.clear();
    syncEventListeners();
  }
  function extractPrimaryEventName(attrName) {
    if (!attrName || !attrName.startsWith(DATA_EVENT_ATTR_PREFIX)) {
      return null;
    }
    const suffix = attrName.slice(DATA_EVENT_ATTR_PREFIX.length);
    if (suffix.length === 0) {
      return null;
    }
    const normalized = suffix.startsWith("-") ? suffix.slice(1) : suffix;
    if (normalized.length === 0) {
      return null;
    }
    const separatorIndex = normalized.indexOf("-");
    const eventName = separatorIndex === -1 ? normalized : normalized.slice(0, separatorIndex);
    return eventName.length > 0 ? eventName : null;
  }
  function collectAttributeNames(element) {
    if (!element) return [];
    if (typeof element.getAttributeNames === "function") {
      return element.getAttributeNames();
    }
    const out = [];
    if (element.attributes) {
      for (let i = 0; i < element.attributes.length; i++) {
        const attr = element.attributes.item(i);
        if (!attr) continue;
        out.push(attr.name);
      }
    }
    return out;
  }
  function refreshHandlerBindings(element) {
    if (!element) {
      return;
    }
    const attributeNames = collectAttributeNames(element);
    if (attributeNames.length === 0) {
      if (handlerBindings.has(element)) {
        handlerBindings.delete(element);
      }
      if (routerBindings.has(element)) {
        routerBindings.delete(element);
      }
      return;
    }
    let bindings = null;
    let sawEventAttr = false;
    for (const name of attributeNames) {
      if (!name.startsWith(DATA_EVENT_ATTR_PREFIX)) {
        if (name.startsWith(DATA_ROUTER_ATTR_PREFIX2)) {
          const key = name.slice(DATA_ROUTER_ATTR_PREFIX2.length);
          const value2 = element.getAttribute(name);
          element.removeAttribute(name);
          applyRouterAttribute(element, key, value2 ?? "");
        }
        continue;
      }
      const remainder = name.slice(DATA_EVENT_ATTR_PREFIX.length);
      const value = element.getAttribute(name);
      element.removeAttribute(name);
      const isMeta = remainder.includes("-");
      if (isMeta) {
        continue;
      }
      sawEventAttr = true;
      const eventName = extractPrimaryEventName(name);
      if (!eventName || !value) {
        continue;
      }
      if (!bindings) {
        bindings = /* @__PURE__ */ new Map();
      }
      bindings.set(eventName, value);
    }
    if (bindings && bindings.size > 0) {
      handlerBindings.set(element, bindings);
    } else if (sawEventAttr || handlerBindings.has(element)) {
      handlerBindings.delete(element);
    }
  }
  function primeHandlerBindings(root) {
    if (!root) return;
    const doc = root instanceof Document ? root : root.ownerDocument ?? (typeof document !== "undefined" ? document : null);
    if (!doc) return;
    const startNode = root instanceof Document ? doc.documentElement ?? null : root;
    if (!startNode) {
      return;
    }
    const walker = doc.createTreeWalker(startNode, NodeFilter.SHOW_ELEMENT);
    if (startNode instanceof Element) {
      refreshHandlerBindings(startNode);
    }
    let current = walker.nextNode();
    while (current) {
      if (current instanceof Element) {
        refreshHandlerBindings(current);
      }
      current = walker.nextNode();
    }
  }
  var sendEventCallback = null;
  var navigationHandler = null;
  var uploadDelegate = null;
  function setupEventDelegation(sendEvent) {
    sendEventCallback = sendEvent;
    ALWAYS_ACTIVE_EVENTS.forEach((eventType) => {
      installListener(eventType);
    });
  }
  function teardownEventDelegation() {
    installedListeners.forEach((listener, eventType) => {
      const useCapture = eventType === "blur" || eventType === "focus";
      document.removeEventListener(eventType, listener, useCapture);
    });
    installedListeners.clear();
    sendEventCallback = null;
    uploadDelegate = null;
  }
  function registerUploadDelegate(delegate) {
    uploadDelegate = delegate;
  }
  function registerNavigationHandler(handler) {
    navigationHandler = handler;
    if (!installedListeners.has("click")) {
      installListener("click");
    }
  }
  function unregisterNavigationHandler() {
    navigationHandler = null;
  }
  function installListener(eventType) {
    if (installedListeners.has(eventType)) return;
    const useCapture = eventType === "blur" || eventType === "focus";
    const listener = (e) => {
      if (sendEventCallback) {
        handleEvent(e, eventType, sendEventCallback);
      }
    };
    document.addEventListener(eventType, listener, useCapture);
    installedListeners.set(eventType, listener);
  }
  function removeListener(eventType) {
    const listener = installedListeners.get(eventType);
    if (!listener) return;
    const useCapture = eventType === "blur" || eventType === "focus";
    document.removeEventListener(eventType, listener, useCapture);
    installedListeners.delete(eventType);
  }
  function syncEventListeners() {
    eventUsageCounts.forEach((_, eventType) => {
      installListener(eventType);
    });
    installedListeners.forEach((_, eventType) => {
      if (!eventUsageCounts.has(eventType) && !ALWAYS_ACTIVE_EVENTS.includes(eventType)) {
        removeListener(eventType);
      }
    });
  }
  function handleEvent(e, eventType, sendEvent) {
    const target = e.target;
    if (!target || !(target instanceof Element)) return;
    if (uploadDelegate) {
      try {
        uploadDelegate(e, eventType);
      } catch (error) {
        console.error("[LiveUI] upload delegate error", error);
      }
    }
    const handlerInfo = findHandlerInfo(target, eventType);
    if (handlerInfo) {
      const handler = handlers.get(handlerInfo.id);
      if (handler && handlerSupportsEvent(handler, eventType)) {
        const refContext = resolveRefEventContext(handlerInfo.element, eventType);
        const combinedProps = mergeSelectorLists(handler.props, refContext?.props);
        const payload = extractEventPayload(
          e,
          target,
          combinedProps,
          handlerInfo.element,
          refContext?.element ?? null
        );
        const routerMeta = routerBindings.get(handlerInfo.element);
        if (routerMeta) {
          if (routerMeta.path !== void 0 && payload["currentTarget.dataset.routerPath"] === void 0) {
            payload["currentTarget.dataset.routerPath"] = routerMeta.path;
          }
          if (routerMeta.query !== void 0 && payload["currentTarget.dataset.routerQuery"] === void 0) {
            payload["currentTarget.dataset.routerQuery"] = routerMeta.query;
          }
          if (routerMeta.hash !== void 0 && payload["currentTarget.dataset.routerHash"] === void 0) {
            payload["currentTarget.dataset.routerHash"] = routerMeta.hash;
          }
          if (routerMeta.replace !== void 0 && payload["currentTarget.dataset.routerReplace"] === void 0) {
            payload["currentTarget.dataset.routerReplace"] = routerMeta.replace;
          }
        }
        if (refContext) {
          refContext.notify(payload);
        }
        if (eventType === "submit") {
          e.preventDefault();
        }
        if (eventType === "click") {
          const anchor = findAnchorElement(target);
          if (anchor && shouldInterceptNavigation(e, anchor)) {
            e.preventDefault();
          }
        }
        sendEvent({
          hid: handlerInfo.id,
          payload
        });
        return;
      }
    }
    if (eventType === "click" && navigationHandler && e instanceof MouseEvent) {
      const anchor = findAnchorElement(target);
      if (anchor && shouldInterceptNavigation(e, anchor)) {
        const href = anchor.getAttribute("href") ?? anchor.getAttribute("xlink:href");
        if (href) {
          try {
            const url = new URL(href, window.location.href);
            const path = url.pathname;
            const query = url.search.substring(1);
            const hash = url.hash.startsWith("#") ? url.hash.substring(1) : url.hash;
            if (navigationHandler(path, query, hash)) {
              e.preventDefault();
              e.stopPropagation();
              return;
            }
          } catch (err) {
          }
        }
      }
    }
  }
  function findHandlerInfo(element, eventType) {
    let current = element;
    while (current && current !== document.documentElement) {
      let binding = handlerBindings.get(current);
      if (!binding) {
        const attributeNames = collectAttributeNames(current);
        if (attributeNames.some((name) => name.startsWith(DATA_EVENT_ATTR_PREFIX))) {
          refreshHandlerBindings(current);
          binding = handlerBindings.get(current);
        }
      }
      if (binding && binding.size > 0) {
        const directId = binding.get(eventType);
        if (directId) {
          const meta = handlers.get(directId);
          if (!meta || handlerSupportsEvent(meta, eventType)) {
            return { id: directId, element: current };
          }
        }
        for (const [registeredEvent, handlerId] of binding.entries()) {
          if (registeredEvent === eventType) {
            continue;
          }
          const meta = handlers.get(handlerId);
          if (meta && handlerSupportsEvent(meta, eventType)) {
            return { id: handlerId, element: current };
          }
        }
      }
      current = current.parentElement;
    }
    return null;
  }
  function extractEventPayload(e, target, props, handlerElement, refElement) {
    const payload = {
      type: e.type
    };
    if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement || target instanceof HTMLSelectElement) {
      payload.value = target.value;
      if (target instanceof HTMLInputElement) {
        if (target.type === "checkbox" || target.type === "radio") {
          payload.checked = target.checked;
        }
      }
    }
    if (e instanceof KeyboardEvent) {
      payload.key = e.key;
      payload.keyCode = e.keyCode;
      payload.altKey = e.altKey;
      payload.ctrlKey = e.ctrlKey;
      payload.metaKey = e.metaKey;
      payload.shiftKey = e.shiftKey;
    }
    if (e instanceof MouseEvent) {
      payload.clientX = e.clientX;
      payload.clientY = e.clientY;
    }
    if (Array.isArray(props)) {
      const domValues = domGetSync(props, {
        event: e,
        target,
        handlerElement: handlerElement ?? null,
        refElement: refElement ?? null
      });
      if (domValues) {
        for (const [key, value] of Object.entries(domValues)) {
          payload[key] = value;
        }
      }
    }
    return payload;
  }
  function findAnchorElement(element) {
    let current = element;
    while (current && current !== document.documentElement) {
      if (isLiveAnchor(current)) {
        return current;
      }
      current = current.parentElement;
    }
    return null;
  }
  function shouldInterceptNavigation(e, anchor) {
    if (e.button !== 0) {
      return false;
    }
    if (e.ctrlKey || e.metaKey || e.altKey || e.shiftKey) {
      return false;
    }
    if (!isLiveAnchor(anchor)) {
      return false;
    }
    const href = anchor.getAttribute("href") ?? anchor.getAttribute("xlink:href") ?? void 0;
    if (!href) {
      return false;
    }
    const target = anchor.getAttribute("target");
    if (target && target !== "_self") {
      return false;
    }
    try {
      const url = new URL(href, window.location.href);
      if (url.origin !== window.location.origin) {
        return false;
      }
      if (url.pathname === window.location.pathname && url.search === window.location.search && url.hash) {
        return false;
      }
      return true;
    } catch (err) {
      return false;
    }
  }
  function incrementEventUsage(eventType) {
    const current = eventUsageCounts.get(eventType) ?? 0;
    eventUsageCounts.set(eventType, current + 1);
  }
  function decrementEventUsage(eventType) {
    const current = eventUsageCounts.get(eventType);
    if (!current) {
      return;
    }
    if (current <= 1) {
      eventUsageCounts.delete(eventType);
    } else {
      eventUsageCounts.set(eventType, current - 1);
    }
  }
  function collectEventTypes(meta) {
    if (!meta) {
      return [];
    }
    const seen = /* @__PURE__ */ new Set();
    const order = [];
    if (meta.event) {
      seen.add(meta.event);
      order.push(meta.event);
    }
    if (Array.isArray(meta.listen)) {
      for (const evt of meta.listen) {
        if (typeof evt !== "string" || evt.length === 0) {
          continue;
        }
        if (!seen.has(evt)) {
          seen.add(evt);
          order.push(evt);
        }
      }
    }
    return order;
  }
  function handlerSupportsEvent(meta, eventType) {
    if (!meta) {
      return false;
    }
    if (meta.event === eventType) {
      return true;
    }
    if (Array.isArray(meta.listen)) {
      return meta.listen.includes(eventType);
    }
    return false;
  }

  // src/dom-index.ts
  var slotMap = /* @__PURE__ */ new Map();
  var listMap = /* @__PURE__ */ new Map();
  function collectRows(container) {
    const rows = /* @__PURE__ */ new Map();
    if (!container) return rows;
    const elements = container.querySelectorAll("[data-row-key]");
    elements.forEach((el) => {
      const key = el.getAttribute("data-row-key");
      if (!key) return;
      rows.set(key, el);
    });
    return rows;
  }
  function registerSlot(index, node) {
    if (!node) return;
    slotMap.set(index, node);
    onSlotRegistered(index, node);
  }
  function getSlot(index) {
    return slotMap.get(index) ?? null;
  }
  function unregisterSlot(index) {
    onSlotUnregistered(index);
    slotMap.delete(index);
  }
  function unregisterList(slotIndex) {
    listMap.delete(slotIndex);
  }
  function reset() {
    slotMap.forEach((_node, index) => {
      onSlotUnregistered(index);
    });
    slotMap.clear();
    listMap.clear();
  }
  function initLists(slotIndexes) {
    if (!Array.isArray(slotIndexes)) return;
    if (typeof document === "undefined") return;
    for (const slotIndex of slotIndexes) {
      if (!listMap.has(slotIndex)) {
        const container = document.querySelector(
          `[data-list-slot="${slotIndex}"]`
        );
        if (container) {
          listMap.set(slotIndex, { container, rows: collectRows(container) });
        }
      }
    }
  }
  function ensureList(slotIndex) {
    if (listMap.has(slotIndex)) {
      return listMap.get(slotIndex);
    }
    const container = document.querySelector(`[data-list-slot="${slotIndex}"]`);
    if (!container) {
      throw new Error(`liveui: list slot ${slotIndex} not registered`);
    }
    const record = { container, rows: collectRows(container) };
    listMap.set(slotIndex, record);
    return record;
  }
  function registerList(slotIndex, container, rows) {
    if (!container) return;
    listMap.set(slotIndex, { container, rows: rows ?? collectRows(container) });
  }
  function setRow(slotIndex, key, root) {
    const list = ensureList(slotIndex);
    list.rows.set(key, root);
  }
  function getRow(slotIndex, key) {
    const list = ensureList(slotIndex);
    return list.rows.get(key) ?? null;
  }
  function deleteRow(slotIndex, key) {
    const list = ensureList(slotIndex);
    list.rows.delete(key);
  }

  // src/componentMarkers.ts
  var componentMarkerIndex = /* @__PURE__ */ new Map();
  function ensureDocument(root) {
    if (!root) {
      return typeof document !== "undefined" ? document : null;
    }
    if (root instanceof Document) {
      return root;
    }
    const node = root;
    if (node && node.ownerDocument) {
      return node.ownerDocument;
    }
    return typeof document !== "undefined" ? document : null;
  }
  function cloneDescriptors(descriptors) {
    const out = {};
    for (const [id, descriptor] of Object.entries(descriptors)) {
      if (!descriptor) continue;
      const start = Number(descriptor.start);
      const end = Number(descriptor.end);
      if (!Number.isFinite(start) || !Number.isFinite(end)) continue;
      out[id] = { start, end };
    }
    return out;
  }
  function assignMarkers(descriptors, root, resetExisting) {
    const target = root ?? (typeof document !== "undefined" ? document : null);
    if (!target) {
      return;
    }
    const doc = ensureDocument(target);
    if (!doc) {
      return;
    }
    if (resetExisting) {
      componentMarkerIndex.clear();
    }
    const startLookup = /* @__PURE__ */ new Map();
    const endLookup = /* @__PURE__ */ new Map();
    for (const [id, descriptor] of Object.entries(descriptors)) {
      if (!descriptor) continue;
      const start = Number(descriptor.start);
      const end = Number(descriptor.end);
      if (!Number.isFinite(start) || !Number.isFinite(end)) continue;
      startLookup.set(start, id);
      endLookup.set(end, id);
    }
    if (startLookup.size === 0 && endLookup.size === 0) {
      return;
    }
    const walker = doc.createTreeWalker(
      target instanceof Document ? target : target,
      NodeFilter.SHOW_COMMENT
    );
    let index = 0;
    let current = walker.nextNode();
    while (current) {
      const startId = startLookup.get(index);
      if (startId) {
        registerComponentMarker(startId, current, componentMarkerIndex.get(startId)?.end ?? null);
      }
      const endId = endLookup.get(index);
      if (endId) {
        registerComponentMarker(endId, componentMarkerIndex.get(endId)?.start ?? null, current);
      }
      current.data = "";
      index++;
      current = walker.nextNode();
    }
  }
  function resetComponentMarkers() {
    componentMarkerIndex.clear();
  }
  function initializeComponentMarkers(descriptors, root) {
    if (!descriptors) {
      resetComponentMarkers();
      return;
    }
    assignMarkers(cloneDescriptors(descriptors), root ?? null, true);
  }
  function registerComponentMarkers(descriptors, root) {
    if (!descriptors) {
      return;
    }
    assignMarkers(cloneDescriptors(descriptors), root, false);
  }
  function registerComponentMarker(id, start, end) {
    if (!id) return;
    const entry = componentMarkerIndex.get(id) ?? { start: null, end: null };
    if (start) {
      entry.start = start;
      start.data = "";
    }
    if (end) {
      entry.end = end;
      end.data = "";
    }
    componentMarkerIndex.set(id, entry);
  }
  function getComponentBounds(id) {
    if (!id) return null;
    const entry = componentMarkerIndex.get(id);
    if (entry && entry.start?.isConnected && entry.end?.isConnected) {
      return { start: entry.start, end: entry.end };
    }
    if (entry?.start && entry?.end) {
      return { start: entry.start, end: entry.end };
    }
    return null;
  }

  // src/patcher.ts
  var config = {
    enableDirtyChecking: true,
    enableBatching: true,
    enableFragmentPooling: true,
    enableMemoization: true,
    enableVirtualScrolling: true,
    virtualScrollThreshold: 100,
    // Start virtual scrolling after 100 items
    memoizationCacheSize: 1e3
  };
  var DATA_EVENT_ATTR_PREFIX2 = "data-on";
  var templateCache = document.createElement("template");
  var htmlCache = /* @__PURE__ */ new Map();
  function createFragment(html) {
    if (config.enableFragmentPooling && htmlCache.has(html)) {
      return htmlCache.get(html).cloneNode(true);
    }
    templateCache.innerHTML = html;
    const fragment = templateCache.content.cloneNode(true);
    if (config.enableFragmentPooling && htmlCache.size < 50) {
      htmlCache.set(html, fragment.cloneNode(true));
    }
    return fragment;
  }
  var PatchMemoizer = class {
    constructor(maxSize) {
      this.cache = /* @__PURE__ */ new Map();
      this.maxSize = maxSize;
    }
    key(op, slotIndex, value) {
      return `${op}:${slotIndex}:${JSON.stringify(value)}`;
    }
    has(key) {
      return this.cache.has(key);
    }
    get(key) {
      return this.cache.get(key);
    }
    set(key, value) {
      if (this.cache.size >= this.maxSize) {
        const firstKey = this.cache.keys().next().value;
        if (firstKey !== void 0) {
          this.cache.delete(firstKey);
        }
      }
      this.cache.set(key, value);
    }
    clear() {
      this.cache.clear();
    }
  };
  var memoizer = new PatchMemoizer(config.memoizationCacheSize);
  var DOMBatcher = class {
    constructor() {
      this.batch = { reads: [], writes: [] };
      this.scheduled = false;
    }
    scheduleRead(fn) {
      this.batch.reads.push(fn);
      this.schedule();
    }
    scheduleWrite(fn) {
      this.batch.writes.push(fn);
      this.schedule();
    }
    schedule() {
      if (this.scheduled) return;
      this.scheduled = true;
      requestAnimationFrame(() => {
        this.flush();
      });
    }
    flush() {
      this.batch.reads.forEach((fn) => fn());
      this.batch.writes.forEach((fn) => fn());
      this.batch = { reads: [], writes: [] };
      this.scheduled = false;
    }
    immediate() {
      if (this.scheduled) {
        this.flush();
      }
    }
  };
  var batcher = new DOMBatcher();
  var virtualScrollStates = /* @__PURE__ */ new Map();
  function initVirtualScroll(slotIndex, container) {
    if (!config.enableVirtualScrolling) return;
    const state = {
      container,
      totalItems: 0,
      visibleRange: { start: 0, end: 50 },
      itemHeight: 0
    };
    state.observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            const target = entry.target;
            if (target instanceof HTMLElement || target instanceof SVGElement) {
              target.style.display = "";
            }
          }
        });
      },
      { root: container, threshold: 0.1 }
    );
    virtualScrollStates.set(slotIndex, state);
  }
  function shouldVirtualize(_slotIndex, itemCount) {
    return config.enableVirtualScrolling && itemCount > config.virtualScrollThreshold;
  }
  function ensureTextNode(node, slotIndex) {
    if (!node) {
      throw new Error(`liveui: slot ${slotIndex} not registered`);
    }
    return node;
  }
  function applySetText(slotIndex, text) {
    const node = ensureTextNode(getSlot(slotIndex), slotIndex);
    if (config.enableDirtyChecking && node.textContent === text) {
      return;
    }
    if (config.enableBatching) {
      batcher.scheduleWrite(() => {
        node.textContent = text;
      });
    } else {
      node.textContent = text;
    }
  }
  function applySetAttrs(slotIndex, upsert, remove) {
    const node = getSlot(slotIndex);
    if (!(node instanceof Element)) {
      throw new Error(`liveui: slot ${slotIndex} is not an element`);
    }
    const existingBindings = getRegisteredSlotBindings(slotIndex) ?? [];
    const bindingMap = /* @__PURE__ */ new Map();
    for (const spec of existingBindings) {
      if (!spec || typeof spec.event !== "string") continue;
      bindingMap.set(spec.event, {
        handler: spec.handler,
        listen: spec.listen ? [...spec.listen] : void 0,
        props: spec.props ? [...spec.props] : void 0
      });
    }
    const parseTokens = (value) => {
      if (typeof value !== "string") {
        return void 0;
      }
      const tokens = value.split(/\s+/).map((token) => token.trim()).filter((token) => token.length > 0);
      return tokens.length > 0 ? tokens : void 0;
    };
    let bindingChanged = false;
    if (upsert) {
      for (const key of Object.keys(upsert)) {
        if (typeof key === "string" && key.startsWith(DATA_ROUTER_ATTR_PREFIX)) {
          const routerKey = key.slice(DATA_ROUTER_ATTR_PREFIX.length);
          applyRouterAttribute(node, routerKey, upsert[key]);
          delete upsert[key];
          continue;
        }
        if (typeof key !== "string" || !key.startsWith(DATA_EVENT_ATTR_PREFIX2)) {
          continue;
        }
        bindingChanged = true;
        const value = upsert[key];
        delete upsert[key];
        let remainder = key.slice(DATA_EVENT_ATTR_PREFIX2.length);
        if (remainder.startsWith("-")) {
          remainder = remainder.slice(1);
        }
        if (remainder.length === 0) {
          continue;
        }
        let metaType = "";
        const dashIndex = remainder.indexOf("-");
        if (dashIndex !== -1) {
          metaType = remainder.slice(dashIndex + 1);
          remainder = remainder.slice(0, dashIndex);
        }
        const eventName = remainder.trim();
        if (eventName.length === 0) {
          continue;
        }
        const entry = bindingMap.get(eventName) ?? {};
        if (metaType === "listen") {
          entry.listen = parseTokens(value);
        } else if (metaType === "props") {
          entry.props = parseTokens(value);
        } else {
          entry.handler = typeof value === "string" ? value : value != null ? String(value) : "";
        }
        bindingMap.set(eventName, entry);
      }
    }
    if (Array.isArray(remove) && remove.length > 0) {
      for (let i = remove.length - 1; i >= 0; i--) {
        const attrName = remove[i];
        if (typeof attrName === "string" && attrName.startsWith(DATA_ROUTER_ATTR_PREFIX)) {
          const routerKey = attrName.slice(DATA_ROUTER_ATTR_PREFIX.length);
          applyRouterAttribute(node, routerKey, void 0);
          remove.splice(i, 1);
          continue;
        }
        if (typeof attrName !== "string" || !attrName.startsWith(DATA_EVENT_ATTR_PREFIX2)) {
          continue;
        }
        bindingChanged = true;
        remove.splice(i, 1);
        let remainder = attrName.slice(DATA_EVENT_ATTR_PREFIX2.length);
        if (remainder.startsWith("-")) {
          remainder = remainder.slice(1);
        }
        if (remainder.length === 0) {
          continue;
        }
        const dashIndex = remainder.indexOf("-");
        if (dashIndex !== -1) {
          remainder = remainder.slice(0, dashIndex);
        }
        const eventName = remainder.trim();
        if (eventName.length === 0) {
          continue;
        }
        bindingMap.delete(eventName);
      }
    }
    if (bindingChanged) {
      const specs = [];
      bindingMap.forEach((info, eventName) => {
        const handlerId = info.handler?.trim();
        if (!handlerId) {
          return;
        }
        const spec = { event: eventName, handler: handlerId };
        if (info.listen && info.listen.length > 0) {
          spec.listen = [...info.listen];
        }
        if (info.props && info.props.length > 0) {
          spec.props = [...info.props];
        }
        specs.push(spec);
      });
      registerBindingsForSlot(slotIndex, specs);
    }
    const previousRef = node.getAttribute("data-live-ref");
    const applyAttrs = () => {
      if (upsert) {
        for (const [k, v] of Object.entries(upsert)) {
          if (v === void 0 || v === null) continue;
          if (config.enableDirtyChecking && node.getAttribute(k) === String(v)) {
            continue;
          }
          node.setAttribute(k, String(v));
        }
      }
      if (remove) {
        for (const key of remove) {
          if (node.hasAttribute(key)) {
            node.removeAttribute(key);
          }
        }
      }
      const nextRef = node.getAttribute("data-live-ref");
      updateRefBinding(node, previousRef, nextRef);
      refreshHandlerBindings(node);
    };
    if (config.enableBatching) {
      batcher.scheduleWrite(applyAttrs);
    } else {
      applyAttrs();
    }
  }
  function morphElement(fromEl, toEl) {
    const fromAttrs = fromEl.attributes;
    const toAttrs = toEl.attributes;
    for (let i = fromAttrs.length - 1; i >= 0; i--) {
      const attr = fromAttrs[i];
      if (!toEl.hasAttribute(attr.name)) {
        fromEl.removeAttribute(attr.name);
      }
    }
    for (let i = 0; i < toAttrs.length; i++) {
      const attr = toAttrs[i];
      if (fromEl.getAttribute(attr.name) !== attr.value) {
        fromEl.setAttribute(attr.name, attr.value);
      }
    }
    if (fromEl.childNodes.length === 1 && fromEl.firstChild?.nodeType === Node.TEXT_NODE) {
      if (toEl.childNodes.length === 1 && toEl.firstChild?.nodeType === Node.TEXT_NODE) {
        if (fromEl.firstChild.textContent !== toEl.firstChild.textContent) {
          fromEl.firstChild.textContent = toEl.firstChild.textContent;
        }
      }
    }
  }
  function registerRowSlots(slotIndexes, fragment) {
    if (!Array.isArray(slotIndexes) || slotIndexes.length === 0) {
      return;
    }
    const pending = new Set(slotIndexes);
    const walker = document.createTreeWalker(fragment, NodeFilter.SHOW_ELEMENT);
    let current = walker.nextNode();
    const foundAnyElements = current !== null;
    while (current && pending.size > 0) {
      const element = current;
      const attr = element.getAttribute("data-slot-index");
      if (attr !== null) {
        attr.split(/\s+/).map((token) => token.trim()).filter((token) => token.length > 0).forEach((token) => {
          const [slotPart, childPart] = token.split("@");
          const slotId = Number(slotPart);
          if (Number.isNaN(slotId) || !pending.has(slotId)) {
            return;
          }
          let target = element;
          if (childPart !== void 0) {
            const childIndex = Number(childPart);
            if (!Number.isNaN(childIndex)) {
              const child = element.childNodes.item(childIndex);
              if (child) {
                target = child;
              }
            }
          }
          registerSlot(slotId, target);
          pending.delete(slotId);
        });
      }
      current = walker.nextNode();
    }
    if (pending.size > 0 && foundAnyElements) {
      pending.forEach((idx) => {
        console.warn(`liveui: slot ${idx} not resolved in inserted row`);
      });
    }
  }
  function applyList(slotIndex, childOps) {
    if (!Array.isArray(childOps) || childOps.length === 0) return;
    const record = ensureList(slotIndex);
    const container = record.container;
    const children = config.enableBatching ? Array.from(container.children) : null;
    let itemCount = record.rows.size;
    for (const op of childOps) {
      if (!op || !op.length) continue;
      const kind = op[0];
      switch (kind) {
        case "del": {
          const key = op[1];
          const row = getRow(slotIndex, key);
          if (row && row.parentNode === container) {
            const removeNode = () => {
              unbindRefsInTree(row);
              container.removeChild(row);
            };
            if (config.enableBatching) {
              batcher.scheduleWrite(removeNode);
            } else {
              removeNode();
            }
          }
          deleteRow(slotIndex, key);
          itemCount--;
          break;
        }
        case "ins": {
          const pos = op[1];
          const payload = op[2] || { key: "", html: "" };
          if (payload && typeof payload === "object" && payload.bindings) {
            for (const [slotKey, specs] of Object.entries(payload.bindings)) {
              const slotId = Number(slotKey);
              if (Number.isNaN(slotId)) {
                continue;
              }
              registerBindingsForSlot(slotId, specs ?? []);
            }
          }
          const memoKey = memoizer.key("ins", slotIndex, payload.html);
          let fragment;
          if (config.enableMemoization && memoizer.has(memoKey)) {
            fragment = memoizer.get(memoKey).cloneNode(true);
          } else {
            fragment = createFragment(payload.html || "");
            if (config.enableMemoization) {
              memoizer.set(memoKey, fragment.cloneNode(true));
            }
          }
          primeHandlerBindings(fragment);
          if (payload.markers) {
            registerComponentMarkers(payload.markers, fragment);
          }
          const nodes = Array.from(fragment.childNodes);
          if (nodes.length === 0) {
            console.warn("live: insertion payload missing nodes for key", payload.key);
            break;
          }
          const insertNode = () => {
            const refNode = config.enableBatching ? children[pos] || null : container.children[pos] || null;
            container.insertBefore(fragment, refNode);
            for (const inserted of nodes) {
              if (inserted instanceof Element) {
                bindRefsInTree(inserted);
              }
            }
            const root = nodes[0];
            if (root instanceof Element) {
              setRow(slotIndex, payload.key, root);
              registerRowSlots(payload.slots || [], root);
              if (shouldVirtualize(slotIndex, itemCount)) {
                const state = virtualScrollStates.get(slotIndex);
                if (state && (pos < state.visibleRange.start || pos > state.visibleRange.end)) {
                  if (root instanceof HTMLElement || root instanceof SVGElement) {
                    root.style.display = "none";
                  }
                }
              }
            } else {
              console.warn("live: row root is not an element for key", payload.key);
            }
          };
          if (config.enableBatching) {
            batcher.scheduleWrite(insertNode);
          } else {
            insertNode();
          }
          itemCount++;
          break;
        }
        case "mov": {
          const from = op[1];
          const to = op[2];
          if (from === to) break;
          const moveNode = () => {
            const childArray = config.enableBatching ? children : Array.from(container.children);
            const child = childArray[from];
            if (child) {
              const refNode = to < childArray.length ? childArray[to] : null;
              container.insertBefore(child, refNode);
            }
          };
          if (config.enableBatching) {
            batcher.scheduleWrite(moveNode);
          } else {
            moveNode();
          }
          break;
        }
        default:
          console.warn("live: unknown list child op", op);
      }
    }
    if (shouldVirtualize(slotIndex, itemCount)) {
      const state = virtualScrollStates.get(slotIndex);
      if (state) {
        state.totalItems = itemCount;
      } else {
        initVirtualScroll(slotIndex, container);
      }
    }
  }
  function applyOps(ops) {
    if (!Array.isArray(ops)) return;
    for (const op of ops) {
      if (!op || op.length === 0) continue;
      const kind = op[0];
      switch (kind) {
        case "setText":
          applySetText(op[1], op[2]);
          break;
        case "setAttrs":
          applySetAttrs(op[1], op[2] || {}, op[3] || []);
          break;
        case "list":
          applyList(op[1], op.slice(2));
          break;
        default:
          console.warn("live: unknown op", op);
      }
    }
    if (config.enableBatching) {
      batcher.immediate();
    }
  }
  function configurePatcher(options) {
    Object.assign(config, options);
  }
  function getPatcherConfig() {
    return { ...config };
  }
  function clearPatcherCaches() {
    htmlCache.clear();
    memoizer.clear();
    virtualScrollStates.clear();
  }
  function getPatcherStats() {
    return {
      htmlCacheSize: htmlCache.size,
      memoizerCacheSize: memoizer["cache"].size,
      virtualScrollCount: virtualScrollStates.size
    };
  }
  function computeInverseOps(patches) {
    const inverseOps = [];
    for (const patch of patches) {
      const [opType, slotId, ...args] = patch;
      if (opType === "setText") {
        const element = getSlot(slotId);
        if (element) {
          const currentText = element.textContent || "";
          inverseOps.push(["setText", slotId, currentText]);
        }
      } else if (opType === "setAttrs") {
        const element = getSlot(slotId);
        if (element && element instanceof Element) {
          const [newAttrs, removeKeys] = args;
          const oldAttrs = {};
          const keysToRemove = [];
          for (const key of Object.keys(newAttrs)) {
            const currentValue = element.getAttribute(key);
            if (currentValue !== null) {
              oldAttrs[key] = currentValue;
            } else {
              keysToRemove.push(key);
            }
          }
          for (const key of removeKeys) {
            const currentValue = element.getAttribute(key);
            if (currentValue !== null) {
              oldAttrs[key] = currentValue;
            }
          }
          inverseOps.push(["setAttrs", slotId, oldAttrs, keysToRemove]);
        }
      } else if (opType === "list") {
        const childOps = args;
        const inverseChildOps = [];
        for (const childOp of childOps) {
          const [childOpType, ...childArgs] = childOp;
          if (childOpType === "ins") {
            const [, { key }] = childArgs;
            inverseChildOps.push(["del", key]);
          } else if (childOpType === "del") {
          } else if (childOpType === "mov") {
            const [fromIdx, toIdx] = childArgs;
            inverseChildOps.push(["mov", toIdx, fromIdx]);
          }
        }
        if (inverseChildOps.length > 0) {
          inverseOps.push(["list", slotId, ...inverseChildOps.reverse()]);
        }
      }
    }
    return inverseOps.reverse();
  }

  // src/uploads.ts
  var UploadManager = class {
    constructor(options) {
      this.uploads = /* @__PURE__ */ new Map();
      this.options = options;
      registerUploadDelegate((event, type) => this.handleDomEvent(event, type));
    }
    dispose() {
      registerUploadDelegate(null);
      this.abortAll();
    }
    onDisconnect() {
      this.abortAll();
    }
    handleControl(message) {
      if (!message || message.op !== "cancel") {
        return;
      }
      const active = this.uploads.get(message.id);
      if (active?.xhr) {
        active.xhr.abort();
      }
    }
    handleDomEvent(event, eventType) {
      if (eventType !== "change") {
        return;
      }
      const target = event.target;
      if (!target) {
        return;
      }
      const input = this.resolveInput(target);
      if (!input) {
        return;
      }
      const uploadId = input.dataset?.pondUpload;
      if (!uploadId) {
        return;
      }
      const files = input.files;
      if (!files || files.length === 0) {
        return;
      }
      const file = files[0];
      const sessionId = this.options.getSessionId();
      if (!sessionId || !this.options.isConnected()) {
        console.warn(
          "[LiveUI] upload ignored because the session is not connected"
        );
        if (sessionId) {
          this.sendMessage({
            t: "upload",
            sid: sessionId,
            id: uploadId,
            op: "error" /* Error */,
            error: "not connected"
          });
        }
        return;
      }
      this.sendMessage({
        t: "upload",
        sid: sessionId,
        id: uploadId,
        op: "change" /* Change */,
        meta: this.buildMeta(file)
      });
      this.startUpload(sessionId, uploadId, file, input);
    }
    startUpload(sessionId, uploadId, file, input) {
      const current = this.uploads.get(uploadId);
      if (current?.xhr) {
        current.xhr.abort();
      }
      const endpoint = this.options.getEndpoint();
      if (!endpoint) {
        console.warn("[LiveUI] upload endpoint missing; aborting file upload");
        this.sendMessage({
          t: "upload",
          sid: sessionId,
          id: uploadId,
          op: "error" /* Error */,
          error: "upload endpoint missing"
        });
        return;
      }
      const url = this.buildUploadURL(endpoint, sessionId, uploadId);
      const xhr = new XMLHttpRequest();
      xhr.open("POST", url, true);
      xhr.upload.onprogress = (evt) => {
        if (!evt.lengthComputable) {
          return;
        }
        const sid = this.options.getSessionId();
        if (!sid) {
          return;
        }
        this.sendMessage({
          t: "upload",
          sid,
          id: uploadId,
          op: "progress" /* Progress */,
          loaded: evt.loaded,
          total: evt.total
        });
      };
      const finalize = () => {
        const active = this.uploads.get(uploadId);
        if (active?.xhr === xhr) {
          this.uploads.delete(uploadId);
        }
      };
      xhr.onload = () => {
        finalize();
        if (xhr.status >= 200 && xhr.status < 300) {
          return;
        }
        const sid = this.options.getSessionId();
        if (!sid) {
          return;
        }
        this.sendMessage({
          t: "upload",
          sid,
          id: uploadId,
          op: "error" /* Error */,
          error: `HTTP ${xhr.status}`
        });
      };
      xhr.onerror = () => {
        finalize();
        const sid = this.options.getSessionId();
        if (!sid) {
          return;
        }
        this.sendMessage({
          t: "upload",
          sid,
          id: uploadId,
          op: "error" /* Error */,
          error: "network error"
        });
      };
      xhr.onabort = () => {
        finalize();
        const sid = this.options.getSessionId();
        if (!sid) {
          return;
        }
        this.sendMessage({
          t: "upload",
          sid,
          id: uploadId,
          op: "cancelled" /* Cancelled */
        });
      };
      const data = new FormData();
      data.append("file", file, file.name);
      xhr.send(data);
      this.uploads.set(uploadId, { xhr, input });
      input.value = "";
    }
    buildUploadURL(base, sid, uploadId) {
      const normalized = base.endsWith("/") ? base : `${base}/`;
      return `${normalized}${encodeURIComponent(sid)}/${encodeURIComponent(uploadId)}`;
    }
    buildMeta(file) {
      return {
        name: file.name,
        size: file.size,
        type: file.type || void 0
      };
    }
    resolveInput(target) {
      if (target instanceof HTMLInputElement) {
        return target;
      }
      if (target.closest) {
        const match = target.closest('input[type="file"][data-pond-upload]');
        return match instanceof HTMLInputElement ? match : null;
      }
      return null;
    }
    sendMessage(payload) {
      try {
        this.options.send(payload);
      } catch (error) {
        console.error("[LiveUI] failed to send upload message", error);
      }
    }
    abortAll() {
      for (const [, active] of this.uploads) {
        try {
          active.xhr.abort();
        } catch (error) {
          console.error("[LiveUI] failed to abort upload", error);
        }
      }
      this.uploads.clear();
    }
  };

  // src/reactive.ts
  var Signal = class {
    constructor(initial) {
      this.subscribers = /* @__PURE__ */ new Set();
      this.value = initial;
    }
    get() {
      return this.value;
    }
    set(newValue) {
      if (this.value !== newValue) {
        this.value = newValue;
        this.notify();
      }
    }
    update(updater) {
      this.set(updater(this.value));
    }
    subscribe(fn) {
      this.subscribers.add(fn);
      fn(this.value);
      return () => this.subscribers.delete(fn);
    }
    notify() {
      this.subscribers.forEach((fn) => fn(this.value));
    }
  };
  var ComputedSignal = class {
    constructor(compute, dependencies) {
      this.subscribers = /* @__PURE__ */ new Set();
      this.unsubscribers = [];
      this.value = compute();
      dependencies.forEach((dep) => {
        const unsub = dep.subscribe(() => {
          const newValue = compute();
          if (this.value !== newValue) {
            this.value = newValue;
            this.notify();
          }
        });
        this.unsubscribers.push(unsub);
      });
    }
    get() {
      return this.value;
    }
    subscribe(fn) {
      this.subscribers.add(fn);
      fn(this.value);
      return () => this.subscribers.delete(fn);
    }
    destroy() {
      this.unsubscribers.forEach((unsub) => unsub());
      this.unsubscribers = [];
      this.subscribers.clear();
    }
    notify() {
      this.subscribers.forEach((fn) => fn(this.value));
    }
  };

  // src/emitter.ts
  var EventEmitter = class {
    constructor() {
      this.listeners = /* @__PURE__ */ new Map();
    }
    on(event, handler) {
      if (!this.listeners.has(event)) {
        this.listeners.set(event, /* @__PURE__ */ new Set());
      }
      this.listeners.get(event).add(handler);
      return () => this.off(event, handler);
    }
    off(event, handler) {
      const handlers2 = this.listeners.get(event);
      if (handlers2) {
        handlers2.delete(handler);
        if (handlers2.size === 0) {
          this.listeners.delete(event);
        }
      }
    }
    emit(event, data) {
      const handlers2 = this.listeners.get(event);
      if (handlers2) {
        handlers2.forEach((handler) => {
          try {
            handler(data);
          } catch (error) {
            console.error(`Error in event handler for ${String(event)}:`, error);
          }
        });
      }
    }
    once(event, handler) {
      const wrappedHandler = (data) => {
        handler(data);
        this.off(event, wrappedHandler);
      };
      return this.on(event, wrappedHandler);
    }
    removeAllListeners(event) {
      if (event) {
        this.listeners.delete(event);
      } else {
        this.listeners.clear();
      }
    }
    listenerCount(event) {
      return this.listeners.get(event)?.size ?? 0;
    }
  };

  // src/optimistic.ts
  var OptimisticUpdateManager = class {
    constructor(options) {
      this.updates = /* @__PURE__ */ new Map();
      this.nextId = 0;
      this.debug = false;
      this.onRollback = options?.onRollback;
      this.onError = options?.onError;
      this.debug = options?.debug || false;
    }
    /**
     * Apply optimistic update
     */
    apply(patches) {
      const id = `opt_${this.nextId++}`;
      const inverseOps = computeInverseOps(patches);
      const update = {
        id,
        patches,
        inverseOps,
        timestamp: Date.now()
      };
      this.updates.set(id, update);
      applyOps(patches);
      if (this.debug) {
        console.log("[optimistic] Applied update:", id, "patches:", patches.length);
      }
      return id;
    }
    /**
     * Commit optimistic update (server confirmed)
     */
    commit(id) {
      if (this.updates.delete(id) && this.debug) {
        console.log("[optimistic] Committed update:", id);
      }
    }
    /**
     * Rollback optimistic update (server rejected)
     */
    rollback(id) {
      const update = this.updates.get(id);
      if (!update) return;
      if (this.debug) {
        console.log("[optimistic] Rolling back update:", id, "inverse ops:", update.inverseOps.length);
      }
      try {
        applyOps(update.inverseOps);
        this.onRollback?.(id, update.patches);
      } catch (error) {
        console.error("[optimistic] Rollback error:", error);
        this.onError?.(error, "rollback");
      }
      this.updates.delete(id);
    }
    /**
     * Get pending update count
     */
    getPendingCount() {
      return this.updates.size;
    }
    /**
     * Clear all pending updates
     */
    clear() {
      this.updates.clear();
    }
  };

  // src/boot.ts
  var BootHandler = class {
    constructor(options) {
      this.boot = null;
      this.debug = false;
      this.debug = options?.debug || false;
    }
    /**
     * Load boot payload from explicit source or auto-detect
     */
    load(explicit) {
      const candidate = explicit ?? this.detect();
      if (!candidate || typeof candidate.sid !== "string" || candidate.sid.length === 0) {
        this.log("No boot payload detected or payload invalid");
        return null;
      }
      this.apply(candidate);
      return candidate;
    }
    /**
     * Detect boot payload from window or DOM
     */
    detect() {
      if (typeof window !== "undefined") {
        const globalBoot = window.__LIVEUI_BOOT__;
        if (globalBoot && typeof globalBoot === "object" && typeof globalBoot.sid === "string") {
          return globalBoot;
        }
      }
      if (typeof document !== "undefined") {
        const script = document.getElementById("live-boot");
        const payload = script?.textContent;
        if (payload) {
          try {
            return JSON.parse(payload);
          } catch (error) {
            this.log("Failed to parse boot payload from DOM", error);
          }
        }
      }
      return null;
    }
    /**
     * Apply boot payload: register handlers, slots, and sync location
     */
    apply(boot) {
      this.boot = boot;
      if (boot.handlers) {
        clearHandlers();
        registerHandlers(boot.handlers);
        syncEventListeners();
      }
      primeSlotBindings(boot.bindings);
      clearRefs();
      registerRefs(boot.refs);
      if (typeof document !== "undefined") {
        bindRefsInTree(document);
        initializeComponentMarkers(boot.markers ?? null, document);
      }
      this.registerInitialDom(boot);
      if (typeof window !== "undefined" && boot.location) {
        const queryPart = boot.location.q ? `?${boot.location.q}` : "";
        const hashPart = boot.location.hash ? `#${boot.location.hash}` : "";
        const target = `${boot.location.path}${queryPart}${hashPart}`;
        const current = `${window.location.pathname}${window.location.search}${window.location.hash}`;
        if (target && current !== target) {
          window.history.replaceState({}, "", target);
        }
      }
    }
    /**
     * Register initial DOM slots from boot payload
     */
    registerInitialDom(boot) {
      if (typeof document === "undefined") {
        return;
      }
      reset();
      const anchors = this.collectSlotAnchors();
      if (Array.isArray(boot.slots)) {
        for (const slot of boot.slots) {
          if (!slot || typeof slot.anchorId !== "number") continue;
          const node = anchors.get(slot.anchorId);
          if (node) {
            registerSlot(slot.anchorId, node);
          } else if (this.debug) {
            console.warn(`liveui: slot ${slot.anchorId} not registered during boot`);
          }
        }
      }
      const listSlots = this.collectListSlotIndexes(boot.d);
      if (listSlots.length > 0) {
        initLists(listSlots);
      }
    }
    /**
     * Collect list slot indexes from dynamic slots
     */
    collectListSlotIndexes(dynamics) {
      if (!Array.isArray(dynamics)) {
        return [];
      }
      const result = [];
      dynamics.forEach((dyn, index) => {
        if (dyn && dyn.kind === "list") {
          result.push(index);
        }
      });
      return result;
    }
    collectSlotAnchors() {
      const anchors = /* @__PURE__ */ new Map();
      if (typeof document === "undefined") {
        return anchors;
      }
      const elements = document.querySelectorAll("[data-slot-index]");
      elements.forEach((element) => {
        const raw = element.getAttribute("data-slot-index");
        if (!raw) return;
        raw.split(/\s+/).map((token) => token.trim()).filter((token) => token.length > 0).forEach((token) => {
          const [slotPart, childPart] = token.split("@");
          const slotId = Number(slotPart);
          if (Number.isNaN(slotId) || anchors.has(slotId)) {
            return;
          }
          let node = element;
          if (childPart !== void 0) {
            const childIndex = Number(childPart);
            if (!Number.isNaN(childIndex)) {
              const child = element.childNodes.item(childIndex);
              if (child) {
                node = child;
              }
            }
          }
          anchors.set(slotId, node);
        });
      });
      return anchors;
    }
    /**
     * Get current boot payload
     */
    get() {
      return this.boot;
    }
    /**
     * Ensure boot payload exists, throw if missing
     */
    ensure() {
      if (!this.boot || !this.boot.sid) {
        throw new Error("LiveUI: boot payload is required before connecting");
      }
      return this.boot;
    }
    /**
     * Get join location from browser or boot fallback
     */
    getJoinLocation() {
      const fallback = this.boot?.location ?? { path: "/", q: "", hash: "" };
      if (typeof window === "undefined") {
        return fallback;
      }
      const path = window.location.pathname || fallback.path || "/";
      const rawQuery = window.location.search ?? "";
      const query = rawQuery.startsWith("?") ? rawQuery.substring(1) : rawQuery;
      const rawHash = window.location.hash ?? "";
      const hash = rawHash.startsWith("#") ? rawHash.substring(1) : rawHash;
      return {
        path,
        q: query,
        hash
      };
    }
    log(...args) {
      if (this.debug) {
        console.log("[boot]", ...args);
      }
    }
  };

  // src/index.ts
  var isServerMessage = (value) => {
    if (value === null || typeof value !== "object") {
      return false;
    }
    const candidate = value;
    return typeof candidate.t === "string";
  };
  var DEFAULT_COOKIE_ENDPOINT = "/pondlive/cookie";
  var LiveUI = class extends EventEmitter {
    constructor(options = {}) {
      super();
      this.client = null;
      this.channel = null;
      this.pubsubChannels = /* @__PURE__ */ new Map();
      // Reactive state
      this.connectionState = new Signal({
        status: "disconnected"
      });
      this.sessionId = new Signal(null);
      this.version = new Signal(0);
      this.lastAck = 0;
      this.hasBootPayload = false;
      this.autoConnectCleanup = null;
      // Frame sequence validation
      this.expectedSeq = null;
      this.frameBuffer = /* @__PURE__ */ new Map();
      this.MAX_FRAME_BUFFER_SIZE = 50;
      this.eventQueue = [];
      // Frame batching for patch operations
      this.patchQueue = [];
      this.batchScheduled = false;
      this.rafHandle = null;
      this.rafUsesTimeoutFallback = false;
      this.cookieRequests = /* @__PURE__ */ new Set();
      // Reconnection
      this.reconnectAttempts = 0;
      this.reconnectTimer = null;
      this.isReconnecting = false;
      // Runtime diagnostics
      this.diagnostics = [];
      this.errorOverlay = null;
      this.errorListEl = null;
      this.errorOverlayVisible = false;
      this.handleDebugKeydown = (event) => this.onDebugKeydown(event);
      // Performance metrics
      this.metrics = {
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
        sequenceGaps: 0
      };
      this.startTime = 0;
      this.patchTimes = [];
      this.patchTimeTotal = 0;
      this.uploads = null;
      // Navigation tracking (to prevent double pushState)
      this.lastOptimisticNavTime = 0;
      this.OPTIMISTIC_NAV_WINDOW = 100;
      // ms
      // Event debouncing
      this.eventDebouncer = /* @__PURE__ */ new Map();
      /**
       * Handle browser back/forward navigation
       */
      this.handlePopState = () => {
        const state = this.connectionState.get();
        if (state.status !== "connected" || !this.channel) {
          this.log("Not connected, cannot handle popstate");
          return;
        }
        const path = window.location.pathname;
        const query = window.location.search.substring(1);
        const hash = window.location.hash.startsWith("#") ? window.location.hash.substring(1) : window.location.hash;
        const msg = {
          t: "pop",
          sid: this.sessionId.get(),
          path,
          q: query,
          hash
        };
        this.log("Sending popstate navigation:", msg);
        this.channel.sendMessage("pop", msg);
      };
      this.bootHandler = new BootHandler({ debug: options.debug || false });
      this.optimistic = new OptimisticUpdateManager({
        onRollback: (id, patches) => this.emit("rollback", { id, patches }),
        onError: (error, context) => this.emit("error", { error, context }),
        debug: options.debug || false
      });
      this.uploads = new UploadManager({
        getSessionId: () => this.sessionId.get(),
        getEndpoint: () => this.options.uploadEndpoint ?? null,
        send: (payload) => this.sendUploadMessage(payload),
        isConnected: () => this.connectionState.get().status === "connected" && this.channel !== null
      });
      const baseOptions = { ...options ?? {} };
      const providedBoot = baseOptions.boot ?? null;
      delete baseOptions.boot;
      this.options = {
        endpoint: baseOptions.endpoint ?? "/live",
        uploadEndpoint: baseOptions.uploadEndpoint ?? "/pondlive/upload/",
        autoConnect: baseOptions.autoConnect !== false,
        debug: baseOptions.debug ?? false,
        reconnect: baseOptions.reconnect !== false,
        maxReconnectAttempts: baseOptions.maxReconnectAttempts ?? 5,
        reconnectDelay: baseOptions.reconnectDelay ?? 1e3,
        ...baseOptions
      };
      if (providedBoot) {
        this.options.boot = providedBoot;
      }
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
    async connect() {
      this.clearAutoConnect();
      const currentState = this.connectionState.get();
      if (currentState.status === "connected" || currentState.status === "connecting") {
        this.log("Already connected or connecting");
        return;
      }
      let boot;
      try {
        boot = this.bootHandler.ensure();
      } catch (error) {
        this.log("Boot payload error:", error);
        this.setState({ status: "error", error });
        this.emit("error", { error, context: "boot" });
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
            hash: joinLocation.hash
          }
        };
        this.client = new import_pondsocket_client.PondClient(this.options.endpoint);
        if (typeof document !== "undefined") {
          setupEventDelegation((event) => this.sendEvent(event));
        }
        if (typeof document !== "undefined") {
          registerNavigationHandler(
            (path, query, hash) => this.sendNavigation(path, query, hash)
          );
        }
        if (typeof window !== "undefined") {
          window.addEventListener("popstate", this.handlePopState);
        }
        const topic = `live/${sid}`;
        this.channel = this.client.createChannel(
          topic,
          joinPayload
        );
        this.channel.onChannelStateChange((state) => {
          this.log("Channel state changed:", state);
          if (state === "JOINED") {
            this.onConnected();
          }
        });
        this.channel.onMessage((event, message) => {
          if (isServerMessage(message)) {
            this.handleMessage(message);
          } else {
            this.log("Ignoring non-server payload", event, message);
          }
        });
        this.channel.onLeave(() => {
          this.log("Channel left");
          this.onDisconnected();
        });
        this.client.connect();
        this.channel.join();
      } catch (error) {
        this.log("Connection error:", error);
        this.setState({ status: "error", error });
        this.emit("error", { error, context: "connect" });
        if (this.options.reconnect && !this.isReconnecting) {
          this.scheduleReconnect();
        }
      }
    }
    /**
     * Disconnect from the server
     */
    disconnect() {
      this.clearReconnectTimer();
      this.isReconnecting = false;
      this.cancelScheduledBatch();
      this.patchQueue = [];
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
      if (typeof window !== "undefined") {
        window.removeEventListener("popstate", this.handlePopState);
      }
      this.setState({ status: "disconnected" });
      this.sessionId.set(null);
      this.version.set(0);
      clearHandlers();
      clearRefs();
      teardownEventDelegation();
      unregisterNavigationHandler();
      this.uploads?.onDisconnect();
      this.emit("disconnected", void 0);
    }
    setupAutoConnect() {
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
    clearAutoConnect() {
      if (this.autoConnectCleanup) {
        this.autoConnectCleanup();
        this.autoConnectCleanup = null;
      }
    }
    /**
     * Get current connection state
     */
    getConnectionState() {
      return this.connectionState.get();
    }
    /**
     * Get current metrics
     */
    getMetrics() {
      return {
        ...this.metrics,
        uptime: this.startTime ? Date.now() - this.startTime : 0
      };
    }
    /**
     * Subscribe to connection state changes
     */
    onStateChange(callback) {
      return this.connectionState.subscribe(callback);
    }
    /**
     * Subscribe to session ID changes
     */
    onSessionIdChange(callback) {
      return this.sessionId.subscribe(callback);
    }
    /**
     * Subscribe to version changes
     */
    onVersionChange(callback) {
      return this.version.subscribe(callback);
    }
    /**
     * Send event with optional debouncing
     */
    sendEventDebounced(event, delay = 300) {
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
    applyOptimistic(patches) {
      return this.optimistic.apply(patches);
    }
    /**
     * Commit optimistic update (server confirmed)
     */
    commitOptimistic(id) {
      this.optimistic.commit(id);
    }
    /**
     * Rollback optimistic update (server rejected)
     */
    rollbackOptimistic(id) {
      this.optimistic.rollback(id);
    }
    // ========================================================================
    // Private methods
    // ========================================================================
    setState(newState) {
      const oldState = this.connectionState.get();
      if (!this.hasStateChanged(oldState, newState)) {
        return;
      }
      this.connectionState.set(newState);
      this.emit("stateChanged", { from: oldState, to: newState });
    }
    onConnected() {
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
    onDisconnected() {
      this.setState({ status: "disconnected" });
      if (this.options.reconnect && !this.isReconnecting) {
        this.scheduleReconnect();
      }
      this.uploads?.onDisconnect();
    }
    scheduleReconnect() {
      if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
        this.log("Max reconnect attempts reached");
        this.emit("error", {
          error: new Error("Max reconnect attempts reached"),
          context: "reconnect"
        });
        return;
      }
      this.isReconnecting = true;
      this.reconnectAttempts++;
      const delay = this.options.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
      this.log(
        `Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.options.maxReconnectAttempts})`
      );
      this.setState({ status: "reconnecting", attempt: this.reconnectAttempts });
      this.emit("reconnecting", { attempt: this.reconnectAttempts });
      this.reconnectTimer = setTimeout(() => {
        this.metrics.reconnections++;
        this.connect();
      }, delay);
    }
    clearReconnectTimer() {
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }
    }
    /**
     * Handle incoming messages from the server
     */
    handleMessage(msg) {
      if (!msg || !msg.t) {
        this.log("Invalid message", msg);
        return;
      }
      this.log("Received message:", msg.t, msg);
      switch (msg.t) {
        case "init":
          this.handleInit(msg);
          break;
        case "frame":
          this.handleFrame(msg);
          break;
        case "join":
          this.handleJoin(msg);
          break;
        case "resume":
          this.handleResume(msg);
          break;
        case "error":
          this.handleError(msg);
          break;
        case "pubsub":
          this.handlePubsub(msg);
          break;
        case "upload":
          this.uploads?.handleControl(msg);
          break;
        case "domreq":
          this.handleDOMRequest(msg);
          break;
        default:
          this.log("Unknown message type:", msg.t);
      }
    }
    /**
     * Handle init message
     */
    handleInit(msg) {
      this.sessionId.set(msg.sid);
      this.version.set(msg.ver);
      this.expectedSeq = msg.seq !== void 0 ? msg.seq + 1 : null;
      this.frameBuffer.clear();
      if (Array.isArray(msg.errors) && msg.errors.length > 0) {
        for (const err of msg.errors) {
          this.recordDiagnostic(err);
        }
      }
      if (msg.handlers) {
        registerHandlers(msg.handlers);
        syncEventListeners();
      }
      primeSlotBindings(msg.bindings ?? null);
      clearRefs();
      registerRefs(msg.refs);
      if (typeof document !== "undefined") {
        bindRefsInTree(document);
        primeHandlerBindings(document);
        initializeComponentMarkers(msg.markers ?? null, document);
      }
      if (msg.seq !== void 0) {
        this.lastAck = msg.seq;
        this.sendAck(msg.seq);
      }
      this.log("Session initialized:", msg.sid, "version:", msg.ver);
      this.onConnected();
    }
    /**
     * Handle frame message with sequence validation
     */
    handleFrame(msg) {
      const sid = this.sessionId.get();
      if (msg.sid !== sid) {
        this.log("Session mismatch, ignoring frame");
        return;
      }
      this.metrics.framesReceived++;
      if (msg.seq !== void 0) {
        if (this.expectedSeq === null) {
          this.expectedSeq = msg.seq + 1;
          this.applyFrame(msg);
          this.drainFrameBuffer();
          return;
        }
        if (msg.seq === this.expectedSeq) {
          this.expectedSeq = msg.seq + 1;
          this.applyFrame(msg);
          this.drainFrameBuffer();
          return;
        }
        if (msg.seq > this.expectedSeq) {
          this.metrics.sequenceGaps++;
          this.log("Frame arrived out of order", {
            expected: this.expectedSeq,
            received: msg.seq,
            gap: msg.seq - this.expectedSeq
          });
          if (this.frameBuffer.size >= this.MAX_FRAME_BUFFER_SIZE) {
            this.metrics.framesDropped++;
            this.log("Frame buffer full, dropping oldest frame");
            const oldestSeq = Math.min(...this.frameBuffer.keys());
            this.frameBuffer.delete(oldestSeq);
          }
          this.frameBuffer.set(msg.seq, msg);
          this.metrics.framesBuffered++;
          return;
        }
        if (msg.seq < this.expectedSeq) {
          this.metrics.framesDropped++;
          this.log("Dropping duplicate/late frame", {
            expected: this.expectedSeq,
            received: msg.seq
          });
          return;
        }
      } else {
        this.applyFrame(msg);
      }
    }
    /**
     * Apply a frame's contents (extracted for reuse)
     */
    applyFrame(msg) {
      this.version.set(msg.ver);
      if (msg.patch && msg.patch.length > 0) {
        this.log("Queueing", msg.patch.length, "operations");
        this.patchQueue.push(...msg.patch);
        this.scheduleBatch();
      }
      if (msg.handlers) {
        if (msg.handlers.add) {
          registerHandlers(msg.handlers.add);
        }
        if (msg.handlers.del) {
          unregisterHandlers(msg.handlers.del);
        }
        syncEventListeners();
      }
      if (msg.refs) {
        if (msg.refs.del) {
          unregisterRefs(msg.refs.del);
        }
        if (msg.refs.add) {
          registerRefs(msg.refs.add);
        }
      }
      if (msg.nav) {
        const now = Date.now();
        const wasOptimistic = now - this.lastOptimisticNavTime < this.OPTIMISTIC_NAV_WINDOW;
        if (msg.nav.push) {
          if (wasOptimistic) {
            window.history.replaceState({}, "", msg.nav.push);
          } else {
            window.history.pushState({}, "", msg.nav.push);
          }
        } else if (msg.nav.replace) {
          window.history.replaceState({}, "", msg.nav.replace);
        }
        this.lastOptimisticNavTime = 0;
      }
      if (msg.effects && msg.effects.length > 0) {
        this.applyEffects(msg.effects);
      }
      if (msg.seq !== void 0) {
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
    /**
     * Drain buffered frames that are now in sequence
     */
    drainFrameBuffer() {
      while (this.expectedSeq !== null && this.frameBuffer.has(this.expectedSeq)) {
        const bufferedFrame = this.frameBuffer.get(this.expectedSeq);
        this.frameBuffer.delete(this.expectedSeq);
        this.expectedSeq = bufferedFrame.seq + 1;
        this.log("Draining buffered frame", { seq: bufferedFrame.seq });
        this.applyFrame(bufferedFrame);
      }
    }
    /**
     * Apply effects from server
     */
    applyEffects(effects) {
      for (const effect of effects) {
        try {
          this.log("Applying effect:", effect);
          this.emit("effect", { effect });
          const effectType = typeof effect.type === "string" ? effect.type.toLowerCase() : "";
          switch (effectType) {
            case "boot":
              this.applyBootEffect(effect);
              break;
            case "componentboot":
              this.applyComponentBootEffect(effect);
              break;
            case "domcall":
              this.applyDOMCallEffect(effect);
              break;
            case "scroll":
            case "scrolltop":
              this.applyScrollEffect(effect);
              break;
            case "focus":
              this.applyFocusEffect(effect);
              break;
            case "alert":
              window.alert(effect.message);
              break;
            case "toast":
              this.applyToastEffect(effect);
              break;
            case "push":
              this.applyPushEffect(effect);
              break;
            case "replace":
              this.applyReplaceEffect(effect);
              break;
            case "dispatch":
              this.dispatchCustomEvent(
                effect.eventName,
                effect.detail
              );
              break;
            case "metadata":
              this.applyMetadataEffect(effect);
              break;
            case "cookies":
              this.handleCookieEffect(effect);
              break;
            case "custom":
              break;
            default:
              this.log("Unknown effect type:", effect.type);
              break;
          }
        } catch (error) {
          this.log("Error applying effect:", effect, error);
          this.emit("error", { error, context: "effect" });
        }
      }
    }
    handleDOMRequest(msg) {
      if (!msg || !msg.id) {
        return;
      }
      const refId = msg.ref ?? "";
      const selectors = Array.isArray(msg.props) ? msg.props : [];
      const response = { t: "domres", id: msg.id };
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
      if (selectors.length > 0) {
        try {
          const values = domGetSync(selectors, {
            event: null,
            target: element,
            handlerElement: element,
            refElement: element
          });
          if (values && Object.keys(values).length > 0) {
            response.values = values;
          } else {
            response.values = {};
          }
        } catch (error) {
          response.error = error instanceof Error ? error.message : String(error);
        }
      } else {
        response.values = {};
      }
      this.sendDOMResponse(response);
    }
    applyDOMCallEffect(effect) {
      const refId = effect.ref ?? effect.Ref ?? "";
      const method = effect.method ?? effect.Method ?? "";
      if (!refId || !method) {
        this.log("DOMCall effect missing ref or method", effect);
        return;
      }
      let args = [];
      if (Array.isArray(effect.args)) {
        args = effect.args;
      } else if (Array.isArray(effect.Args)) {
        args = effect.Args;
      } else if (effect.args !== void 0) {
        args = [effect.args];
      } else if (effect.Args !== void 0) {
        args = [effect.Args];
      }
      callElementMethod(refId, method, args);
    }
    sendDOMResponse(payload) {
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
    applyScrollEffect(effect) {
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
          block: effect.block || "start"
        });
      }
    }
    applyMetadataEffect(effect) {
      if (typeof document === "undefined") {
        return;
      }
      if (Object.prototype.hasOwnProperty.call(effect, "title") && effect.title !== void 0) {
        document.title = effect.title;
      }
      const head = document.head;
      if (!head) {
        return;
      }
      const keyAttr = "data-live-key";
      const typeAttr = "data-live-head";
      const removeByKeys = (keys) => {
        if (!Array.isArray(keys) || keys.length === 0) {
          return;
        }
        keys.forEach((key) => {
          head.querySelectorAll(`[${keyAttr}="${key}"]`).forEach((node) => {
            node.parentNode?.removeChild(node);
          });
        });
      };
      const resetAttributes = (el, preserve) => {
        const keep = new Set(preserve);
        Array.from(el.attributes).forEach((attr) => {
          if (!keep.has(attr.name)) {
            el.removeAttribute(attr.name);
          }
        });
      };
      const upsertMeta = (payload, typeValue) => {
        if (!payload || !payload.key) {
          return;
        }
        let element = head.querySelector(
          `meta[${keyAttr}="${payload.key}"]`
        );
        if (!element) {
          element = document.createElement("meta");
          head.appendChild(element);
        }
        resetAttributes(element, [keyAttr]);
        element.setAttribute(keyAttr, payload.key);
        element.setAttribute(typeAttr, typeValue);
        if (payload.name !== void 0)
          element.setAttribute("name", payload.name);
        if (payload.content !== void 0)
          element.setAttribute("content", payload.content);
        if (payload.property !== void 0)
          element.setAttribute("property", payload.property);
        if (payload.charset !== void 0)
          element.setAttribute("charset", payload.charset);
        if (payload.httpEquiv !== void 0)
          element.setAttribute("http-equiv", payload.httpEquiv);
        if (payload.itemProp !== void 0)
          element.setAttribute("itemprop", payload.itemProp);
        if (payload.attrs) {
          for (const [k, v] of Object.entries(payload.attrs)) {
            if (v != null) {
              element.setAttribute(k, String(v));
            }
          }
        }
      };
      const upsertLink = (payload) => {
        if (!payload || !payload.key) {
          return;
        }
        let element = head.querySelector(
          `link[${keyAttr}="${payload.key}"]`
        );
        if (!element) {
          element = document.createElement("link");
          head.appendChild(element);
        }
        resetAttributes(element, [keyAttr]);
        element.setAttribute(keyAttr, payload.key);
        element.setAttribute(typeAttr, "link");
        if (payload.rel !== void 0) element.setAttribute("rel", payload.rel);
        if (payload.href !== void 0)
          element.setAttribute("href", payload.href);
        if (payload.type !== void 0)
          element.setAttribute("type", payload.type);
        if (payload.as !== void 0) element.setAttribute("as", payload.as);
        if (payload.media !== void 0)
          element.setAttribute("media", payload.media);
        if (payload.hreflang !== void 0)
          element.setAttribute("hreflang", payload.hreflang);
        if (payload.title !== void 0)
          element.setAttribute("title", payload.title);
        if (payload.crossorigin !== void 0)
          element.setAttribute("crossorigin", payload.crossorigin);
        if (payload.integrity !== void 0)
          element.setAttribute("integrity", payload.integrity);
        if (payload.referrerpolicy !== void 0)
          element.setAttribute("referrerpolicy", payload.referrerpolicy);
        if (payload.sizes !== void 0)
          element.setAttribute("sizes", payload.sizes);
        if (payload.attrs) {
          for (const [k, v] of Object.entries(payload.attrs)) {
            if (v != null) {
              element.setAttribute(k, String(v));
            }
          }
        }
      };
      const upsertScript = (payload) => {
        if (!payload || !payload.key) {
          return;
        }
        let element = head.querySelector(
          `script[${keyAttr}="${payload.key}"]`
        );
        if (!element) {
          element = document.createElement("script");
          head.appendChild(element);
        }
        resetAttributes(element, [keyAttr]);
        element.setAttribute(keyAttr, payload.key);
        element.setAttribute(typeAttr, "script");
        if (payload.module) {
          element.setAttribute("type", "module");
        } else if (payload.type !== void 0) {
          element.setAttribute("type", payload.type);
        }
        if (payload.src !== void 0) element.setAttribute("src", payload.src);
        if (payload.async) element.setAttribute("async", "async");
        if (payload.defer) element.setAttribute("defer", "defer");
        if (payload.noModule) element.setAttribute("nomodule", "nomodule");
        if (payload.crossorigin !== void 0)
          element.setAttribute("crossorigin", payload.crossorigin);
        if (payload.integrity !== void 0)
          element.setAttribute("integrity", payload.integrity);
        if (payload.referrerpolicy !== void 0)
          element.setAttribute("referrerpolicy", payload.referrerpolicy);
        if (payload.nonce !== void 0)
          element.setAttribute("nonce", payload.nonce);
        if (payload.attrs) {
          for (const [k, v] of Object.entries(payload.attrs)) {
            if (v != null) {
              element.setAttribute(k, String(v));
            }
          }
        }
        if (payload.inner !== void 0) {
          element.textContent = payload.inner;
        }
      };
      if (effect.clearDescription) {
        removeByKeys(["description"]);
      }
      if (Object.prototype.hasOwnProperty.call(effect, "description") && effect.description !== void 0) {
        const content = effect.description;
        if (content.trim().length > 0) {
          upsertMeta(
            { key: "description", name: "description", content },
            "description"
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
    handleCookieEffect(effect) {
      void this.performCookieSync(effect);
    }
    async performCookieSync(effect) {
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
            Accept: "application/json"
          },
          body: JSON.stringify({ sid, token })
        });
        if (!response.ok && response.status !== 204) {
          this.log("Cookie negotiation failed", {
            endpoint,
            status: response.status,
            statusText: response.statusText
          });
        }
      } catch (error) {
        this.log("Cookie negotiation error", error);
        this.emit("error", { error, context: "cookies" });
      } finally {
        this.cookieRequests.delete(token);
      }
    }
    applyFocusEffect(effect) {
      const selector = effect.selector || effect.Selector;
      if (!selector) return;
      const element = document.querySelector(selector);
      if (element && (element instanceof HTMLElement || element instanceof SVGElement) && typeof element.focus === "function") {
        element.focus();
      }
    }
    applyToastEffect(effect) {
      const message = effect.message || effect.Message;
      const duration = effect.duration || 3e3;
      const variant = effect.variant || "info";
      this.emit("effect", {
        effect: {
          type: "toast",
          message,
          duration,
          variant
        }
      });
      if (!this.listenerCount("effect")) {
        console.info(`[Toast ${variant}] ${message}`);
      }
    }
    applyPushEffect(effect) {
      const url = effect.url || effect.URL;
      if (url) {
        window.history.pushState({}, "", url);
      }
    }
    applyReplaceEffect(effect) {
      const url = effect.url || effect.URL;
      if (url) {
        window.history.replaceState({}, "", url);
      }
    }
    applyBootEffect(effect) {
      const boot = effect?.boot;
      if (!boot || typeof boot.html !== "string") {
        return;
      }
      clearPatcherCaches();
      reset();
      if (typeof document !== "undefined") {
        document.body.innerHTML = boot.html;
        primeHandlerBindings(document);
        initializeComponentMarkers(boot.markers ?? null, document);
      }
      this.bootHandler.load(boot);
      syncEventListeners();
    }
    applyComponentBootEffect(effect) {
      if (typeof document === "undefined") return;
      if (!effect || !effect.componentId) return;
      const { componentId, html, slots, listSlots, bindings } = effect;
      const bounds = this.findComponentBounds(componentId);
      if (!bounds) {
        if (this.options.debug) {
          console.warn(`liveui: component ${componentId} bounds not found for componentBoot effect`);
        }
        return;
      }
      if (Array.isArray(slots)) {
        for (const slot of slots) {
          unregisterSlot(slot);
        }
      }
      if (Array.isArray(listSlots)) {
        for (const slot of listSlots) {
          unregisterList(slot);
        }
      }
      clearPatcherCaches();
      const template = document.createElement("template");
      template.innerHTML = html || "";
      const fragment = template.content.cloneNode(true);
      primeHandlerBindings(fragment);
      if (effect.markers) {
        registerComponentMarkers(effect.markers, fragment);
      }
      const range = document.createRange();
      range.setStartBefore(bounds.start);
      range.setEndAfter(bounds.end);
      range.deleteContents();
      range.insertNode(fragment);
      range.detach();
      const refreshed = this.findComponentBounds(componentId);
      if (!refreshed) {
        if (this.options.debug) {
          console.warn(`liveui: component ${componentId} bounds missing after componentBoot replacement`);
        }
        return;
      }
      if (bindings && typeof bindings === "object") {
        for (const [slotKey, specs] of Object.entries(bindings)) {
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
        Array.isArray(listSlots) ? listSlots : []
      );
    }
    findComponentBounds(id) {
      if (typeof document === "undefined" || !id) return null;
      return getComponentBounds(id);
    }
    registerComponentAnchors(bounds, slots, listSlots) {
      const slotSet = new Set(slots ?? []);
      const listSet = new Set(listSlots ?? []);
      const queue = [];
      for (let node = bounds.start.nextSibling; node && node !== bounds.end; node = node.nextSibling) {
        queue.push(node);
      }
      while (queue.length > 0) {
        const current = queue.shift();
        if (!current) continue;
        if (current instanceof Element) {
          this.registerElementSlots(current, slotSet);
          this.registerListContainer(current, listSet);
          for (let child = current.firstChild; child; child = child.nextSibling) {
            queue.push(child);
          }
        }
      }
    }
    registerElementSlots(element, slots) {
      const raw = element.getAttribute("data-slot-index");
      if (!raw) return;
      const tokens = raw.split(/\s+/);
      for (const token of tokens) {
        const trimmed = token.trim();
        if (trimmed.length === 0) continue;
        const [slotPart, childPart] = trimmed.split("@", 2);
        const slotId = Number(slotPart);
        if (Number.isNaN(slotId)) continue;
        if (slots.size > 0 && !slots.has(slotId)) continue;
        let target = element;
        if (childPart !== void 0) {
          const childIndex = Number(childPart);
          if (!Number.isNaN(childIndex)) {
            const childNode = element.childNodes.item(childIndex);
            if (childNode) {
              target = childNode;
            }
          }
        }
        registerSlot(slotId, target);
      }
    }
    registerListContainer(element, listSlots) {
      const attr = element.getAttribute("data-list-slot");
      if (!attr) return;
      const slotId = Number(attr);
      if (Number.isNaN(slotId)) return;
      if (listSlots.size > 0 && !listSlots.has(slotId)) return;
      registerList(slotId, element);
    }
    dispatchCustomEvent(eventName, detail) {
      const event = new CustomEvent(eventName, {
        detail,
        bubbles: true,
        cancelable: true
      });
      document.dispatchEvent(event);
    }
    /**
     * Schedule a batch to be applied on the next animation frame
     */
    scheduleBatch() {
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
    flushBatch() {
      if (this.patchQueue.length === 0) {
        this.batchScheduled = false;
        this.rafHandle = null;
        this.rafUsesTimeoutFallback = false;
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
      const duration = performance.now() - startTime;
      this.metrics.patchesApplied += patches.length;
      this.patchTimes.push(duration);
      this.patchTimeTotal += duration;
      if (this.patchTimes.length > 100) {
        const removed = this.patchTimes.shift();
        if (removed !== void 0) {
          this.patchTimeTotal -= removed;
        }
      }
      const windowLength = this.patchTimes.length;
      this.metrics.averagePatchTime = windowLength ? this.patchTimeTotal / windowLength : 0;
      this.emit("frameApplied", { operations: patches.length, duration });
      this.emit("metricsUpdated", this.getMetrics());
    }
    cancelScheduledBatch() {
      if (this.rafHandle === null) {
        return;
      }
      if (this.rafUsesTimeoutFallback) {
        clearTimeout(this.rafHandle);
      } else if (typeof cancelAnimationFrame === "function") {
        cancelAnimationFrame(this.rafHandle);
      }
      this.rafHandle = null;
      this.batchScheduled = false;
      this.rafUsesTimeoutFallback = false;
    }
    hasStateChanged(oldState, newState) {
      if (oldState.status !== newState.status) {
        return true;
      }
      switch (newState.status) {
        case "connected":
          return oldState.status !== "connected" || oldState.sessionId !== newState.sessionId || oldState.version !== newState.version;
        case "reconnecting":
          return oldState.status !== "reconnecting" || oldState.attempt !== newState.attempt;
        case "error":
          return oldState.status !== "error" || oldState.error !== newState.error;
        default:
          return false;
      }
    }
    /**
     * Handle join message
     */
    handleJoin(msg) {
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
    handleResume(msg) {
      this.log("Resume from", msg.from, "to", msg.to);
      const ack = msg.from > 0 ? msg.from - 1 : 0;
      if (ack > this.lastAck) {
        this.lastAck = ack;
      }
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
    handleError(msg) {
      const error = new Error(msg.message);
      console.error("LiveUI error:", msg.code, msg.message);
      this.emit("error", { error, context: msg.code });
      this.recordDiagnostic(msg);
    }
    handlePubsub(msg) {
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
    recordDiagnostic(msg) {
      if (!this.options.debug) {
        return;
      }
      const key = this.buildDiagnosticKey(msg);
      const entry = {
        key,
        code: msg.code ?? "runtime_panic",
        message: msg.message ?? "Runtime panic recovered",
        details: msg.details,
        timestamp: Date.now()
      };
      const existingIndex = this.diagnostics.findIndex(
        (item) => item.key === key
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
    buildDiagnosticKey(msg) {
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
        panic
      ].join("|");
    }
    joinPubsubTopic(topic) {
      if (!topic || !this.client) {
        return;
      }
      if (this.pubsubChannels.has(topic)) {
        return;
      }
      const channel = this.client.createChannel(
        `pubsub/${topic}`,
        {
          sid: this.sessionId.get() ?? void 0
        }
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
    leavePubsubTopic(topic) {
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
    ensureErrorOverlayElements() {
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
        "backdrop-filter:blur(6px)"
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
        'font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif'
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
        "z-index:1"
      ].join(";");
      const title = document.createElement("h2");
      title.textContent = "LiveUI Runtime Errors";
      title.style.cssText = "margin:0;font-size:18px;font-weight:700;color:#f8fafc;";
      const headerMeta = document.createElement("div");
      headerMeta.style.cssText = "display:flex;flex-direction:column;gap:4px;";
      headerMeta.appendChild(title);
      const hint = document.createElement("span");
      hint.textContent = "\u21E7\u2318E / \u21E7Ctrl+E to toggle \u2022 Esc to dismiss";
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
        "cursor:pointer"
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
        "cursor:pointer"
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
        "cursor:pointer"
      ].join(";");
      closeButton.addEventListener("click", () => this.hideErrorOverlay());
      controls.appendChild(retryButton);
      controls.appendChild(clearButton);
      controls.appendChild(closeButton);
      header.appendChild(headerMeta);
      header.appendChild(controls);
      const list = document.createElement("div");
      list.style.cssText = "padding:16px 24px 24px;display:flex;flex-direction:column;gap:16px;";
      panel.appendChild(header);
      panel.appendChild(list);
      overlay.appendChild(panel);
      document.body.appendChild(overlay);
      this.errorOverlay = overlay;
      this.errorListEl = list;
    }
    renderErrorOverlay() {
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
          (a, b) => b.timestamp - a.timestamp
        );
        for (const entry of ordered) {
          list.appendChild(this.createDiagnosticElement(entry));
        }
      }
      this.errorOverlay.style.display = "flex";
      this.errorOverlayVisible = true;
    }
    createDiagnosticElement(entry) {
      const container = document.createElement("article");
      container.style.cssText = [
        "background:rgba(15,23,42,0.65)",
        "border:1px solid rgba(148,163,184,0.25)",
        "border-radius:12px",
        "padding:18px 20px",
        "display:flex",
        "flex-direction:column",
        "gap:12px"
      ].join(";");
      const header = document.createElement("div");
      header.style.cssText = "display:flex;flex-wrap:wrap;align-items:center;justify-content:space-between;gap:12px;";
      const title = document.createElement("h3");
      title.textContent = entry.message || "Runtime panic recovered";
      title.style.cssText = "margin:0;font-size:16px;font-weight:700;color:#f8fafc;";
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
        "padding:4px 10px"
      ].join(";");
      const timeEl = document.createElement("time");
      const captured = entry.details?.capturedAt ? new Date(entry.details.capturedAt) : new Date(entry.timestamp);
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
      metaContainer.style.cssText = "display:flex;flex-direction:column;gap:4px;font-size:12px;color:#cbd5f5;";
      const componentLabel = details?.componentName && details.componentId ? `${details.componentName} (${details.componentId})` : details?.componentName ?? details?.componentId ?? "";
      this.appendMeta(metaContainer, "Component", componentLabel);
      this.appendMeta(metaContainer, "Phase", details?.phase);
      if (details?.hook) {
        const hookInfo = details.hookIndex != null ? `${details.hook} (#${details.hookIndex})` : details.hook;
        this.appendMeta(metaContainer, "Hook", hookInfo);
      }
      this.appendMeta(metaContainer, "Panic", details?.panic);
      if (metaContainer.childNodes.length > 0) {
        container.appendChild(metaContainer);
      }
      if (details?.suggestion) {
        const suggestion = document.createElement("p");
        suggestion.textContent = details.suggestion;
        suggestion.style.cssText = "margin:0;padding:12px 14px;border-radius:8px;background:rgba(34,197,94,0.12);color:#bbf7d0;font-size:13px;border-left:3px solid rgba(34,197,94,0.45);";
        container.appendChild(suggestion);
      }
      const metadata = details?.metadata;
      if (metadata && Object.keys(metadata).length > 0) {
        const dl = document.createElement("dl");
        dl.style.cssText = "display:grid;grid-template-columns:max-content 1fr;gap:4px 12px;padding:12px;border-radius:8px;background:rgba(15,23,42,0.5);";
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
      if (details?.stack) {
        const stack = document.createElement("pre");
        stack.textContent = details.stack.trim();
        stack.style.cssText = "margin:0;padding:12px;border-radius:8px;background:#020617;color:#f1f5f9;max-height:220px;overflow:auto;font-size:12px;line-height:1.45;";
        container.appendChild(stack);
      }
      return container;
    }
    appendMeta(container, label, value) {
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
    formatMetadataValue(value) {
      if (value === null || value === void 0) {
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
    hideErrorOverlay() {
      if (!this.errorOverlay) {
        return;
      }
      this.errorOverlay.style.display = "none";
      this.errorOverlayVisible = false;
    }
    clearDiagnostics() {
      this.diagnostics = [];
      if (this.errorListEl) {
        this.errorListEl.innerHTML = "";
      }
      this.hideErrorOverlay();
    }
    requestRecovery() {
      const sid = this.sessionId.get();
      if (!sid || !this.channel) {
        this.log("Cannot request recovery without active session/channel");
        return;
      }
      this.log("Requesting runtime recovery");
      this.channel.sendMessage("recover", { t: "recover", sid });
    }
    onDebugKeydown(event) {
      if (!this.options.debug) {
        return;
      }
      if (event.key === "Escape" && this.errorOverlayVisible) {
        this.hideErrorOverlay();
        return;
      }
      if ((event.key === "e" || event.key === "E") && event.shiftKey && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        this.renderErrorOverlay();
      }
    }
    /**
     * Send event to the server
     */
    sendEvent(event) {
      const state = this.connectionState.get();
      if (state.status !== "connected" || !this.channel) {
        this.log("Not connected, queueing event");
        this.eventQueue.push(event);
        return;
      }
      const msg = {
        t: "evt",
        sid: this.sessionId.get(),
        hid: event.hid,
        payload: event.payload
      };
      this.log("Sending event:", msg);
      this.channel.sendMessage("evt", msg);
      this.metrics.eventsProcessed++;
    }
    sendUploadMessage(payload) {
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
    sendNavigation(path, query, hash) {
      const state = this.connectionState.get();
      if (state.status !== "connected" || !this.channel) {
        this.log("Not connected, cannot navigate");
        return false;
      }
      const msg = {
        t: "nav",
        sid: this.sessionId.get(),
        path,
        q: query,
        hash
      };
      this.log("Sending navigation:", msg);
      this.channel.sendMessage("nav", msg);
      const queryPart = query ? `?${query}` : "";
      const hashPart = hash ? `#${hash}` : "";
      window.history.pushState({}, "", `${path}${queryPart}${hashPart}`);
      this.lastOptimisticNavTime = Date.now();
      return true;
    }
    /**
     * Send acknowledgement to the server
     */
    sendAck(seq) {
      if (!this.channel) return;
      const msg = {
        t: "ack",
        sid: this.sessionId.get(),
        seq
      };
      this.channel.sendMessage("ack", msg);
      if (typeof seq === "number" && seq > this.lastAck) {
        this.lastAck = seq;
      }
    }
    /**
     * Flush queued events
     */
    flushEventQueue() {
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
    log(...args) {
      if (this.options.debug) {
        console.log("[LiveUI]", ...args);
      }
    }
  };
  var index_default = LiveUI;

  // src/entry.ts
  var bootPromise = null;
  function getWindow() {
    if (typeof window === "undefined") {
      return null;
    }
    return window;
  }
  function detectBootPayload(target) {
    const existing = target.__LIVEUI_BOOT__;
    if (existing && typeof existing === "object" && typeof existing.sid === "string") {
      return existing;
    }
    if (typeof document === "undefined") {
      return null;
    }
    const script = document.getElementById("live-boot");
    const content = script?.textContent;
    if (!content) {
      return null;
    }
    try {
      const payload = JSON.parse(content);
      target.__LIVEUI_BOOT__ = payload;
      return payload;
    } catch (error) {
      console.error("[LiveUI] Failed to parse boot payload", error);
      return null;
    }
  }
  function attachGlobals(target, instance) {
    const augmented = index_default;
    Object.assign(augmented, {
      instance,
      dom: dom_index_exports,
      applyOps,
      patcher: {
        configure: configurePatcher,
        getConfig: getPatcherConfig,
        clearCaches: clearPatcherCaches,
        getStats: getPatcherStats,
        morphElement
      },
      boot: bootClient
    });
    target.LiveUI = augmented;
    target.LiveUIInstance = instance;
    if (target.__LIVEUI_DEVTOOLS__) {
      target.__LIVEUI_DEVTOOLS__.installed = true;
      target.__LIVEUI_DEVTOOLS__.instance = instance;
    }
  }
  function createClient(target) {
    const inlineOptions = { ...target.__LIVEUI_OPTIONS__ ?? {} };
    const bootPayload = detectBootPayload(target);
    const inlineBoot = inlineOptions.boot;
    const resolvedBootPayload = bootPayload ?? inlineBoot ?? null;
    const shouldAutoConnect = inlineOptions.autoConnect !== false;
    const options = { ...inlineOptions, autoConnect: false };
    const bootDebug = typeof resolvedBootPayload?.client?.debug === "boolean" ? resolvedBootPayload.client.debug : void 0;
    if (typeof bootDebug !== "undefined") {
      options.debug = bootDebug;
    } else if (typeof options.debug === "undefined") {
      options.debug = true;
    }
    if (resolvedBootPayload) {
      options.boot = resolvedBootPayload;
      if (!bootPayload) {
        target.__LIVEUI_BOOT__ = resolvedBootPayload;
      }
    }
    const client = new index_default(options);
    attachGlobals(target, client);
    if (shouldAutoConnect) {
      if (resolvedBootPayload) {
        void client.connect().catch((error) => {
          console.error("[LiveUI] Failed to connect during boot", error);
        });
      } else {
        console.warn("[LiveUI] Boot payload missing; auto-connect skipped.");
      }
    }
    return client;
  }
  function scheduleBoot(target) {
    return new Promise((resolve, reject) => {
      const start = () => {
        try {
          const instance = createClient(target);
          resolve(instance);
        } catch (error) {
          console.error("[LiveUI] Boot failed", error);
          reject(error);
        }
      };
      if (typeof document !== "undefined" && document.readyState === "loading") {
        const handler = () => {
          document.removeEventListener("DOMContentLoaded", handler);
          start();
        };
        document.addEventListener("DOMContentLoaded", handler);
      } else {
        start();
      }
    });
  }
  function bootClient({ force = false } = {}) {
    const globalWindow = getWindow();
    if (!globalWindow) {
      return Promise.reject(new Error("LiveUI: window is not available for bootstrapping."));
    }
    if (force) {
      bootPromise = null;
    }
    if (!bootPromise) {
      bootPromise = scheduleBoot(globalWindow);
    }
    return bootPromise;
  }
  if (typeof window !== "undefined") {
    void bootClient().catch(() => {
    });
  }
  return __toCommonJS(entry_exports);
})();

// Export to global window object
if (typeof window !== 'undefined') {
  window.LiveUI = LiveUIModule.default;
  window.LiveUI.dom = LiveUIModule.dom;
  window.LiveUI.applyOps = LiveUIModule.applyOps;
}

//# sourceMappingURL=pondlive-dev.js.map

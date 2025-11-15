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

  // src/entry.ts
  var entry_exports = {};
  __export(entry_exports, {
    LiveRuntime: () => LiveRuntime,
    LiveUI: () => LiveUI,
    bootClient: () => bootClient
  });

  // src/runtime.ts
  var import_pondsocket_client = __toESM(require_pondsocket_client(), 1);

  // src/logger.ts
  var _Logger = class _Logger {
    static configure(options) {
      _Logger.debugEnabled = Boolean(options?.debug);
    }
    static debug(...args) {
      if (!_Logger.debugEnabled) {
        return;
      }
      _Logger.emit("log", "debug", args);
    }
    static info(...args) {
      _Logger.emit("log", "info", args);
    }
    static warn(...args) {
      _Logger.emit("warn", "warn", args);
    }
    static error(...args) {
      _Logger.emit("error", "error", args);
    }
    static emit(method, level, args) {
      if (typeof console === "undefined") {
        return;
      }
      const emitter = console[method] ?? console.log;
      emitter(`[LiveUI][${level}]`, ...args);
    }
  };
  _Logger.debugEnabled = false;
  var Logger = _Logger;

  // src/boot.ts
  var BootLoader = class {
    constructor(options) {
      this.payload = null;
      this.options = {
        debug: options?.debug ?? false,
        scriptId: options?.scriptId ?? "live-boot"
      };
    }
    load(explicit) {
      const candidate = explicit ?? this.readWindowPayload() ?? this.readScriptPayload();
      if (candidate && typeof candidate.sid === "string") {
        this.payload = candidate;
        this.cacheToWindow(candidate);
        if (this.options.debug) {
          Logger.debug("[boot]", "payload loaded", {
            sid: candidate.sid,
            version: candidate.ver,
            hasHtml: Boolean(candidate.html)
          });
        }
        return this.payload;
      }
      if (this.options.debug) {
        this.log("boot payload unavailable");
      }
      return this.payload;
    }
    get() {
      return this.payload;
    }
    ensure() {
      const boot = this.payload ?? this.load();
      if (!boot || typeof boot.sid !== "string") {
        throw new Error("[LiveUI] boot payload is required before connecting");
      }
      return boot;
    }
    readWindowPayload() {
      if (typeof window === "undefined") {
        return null;
      }
      const globalAny = window;
      const payload = globalAny.__LIVEUI_BOOT__;
      if (payload && typeof payload.sid === "string") {
        return payload;
      }
      return null;
    }
    readScriptPayload() {
      if (typeof document === "undefined") {
        return null;
      }
      const script = document.getElementById(this.options.scriptId);
      const content = script?.textContent;
      if (!content) {
        return null;
      }
      try {
        return JSON.parse(content);
      } catch (error) {
        this.log("failed to parse boot payload", error);
        return null;
      }
    }
    cacheToWindow(payload) {
      if (typeof window === "undefined") {
        return;
      }
      const globalAny = window;
      globalAny.__LIVEUI_BOOT__ = payload;
    }
    log(message, error) {
      if (!this.options.debug) {
        return;
      }
      if (error) {
        Logger.warn("[boot]", message, error);
      } else {
        Logger.warn("[boot]", message);
      }
    }
  };

  // src/emitter.ts
  var TypedEventEmitter = class {
    constructor() {
      this.listeners = /* @__PURE__ */ new Map();
    }
    on(event, listener) {
      const bucket = this.listeners.get(event) ?? /* @__PURE__ */ new Set();
      bucket.add(listener);
      this.listeners.set(event, bucket);
      return () => this.off(event, listener);
    }
    once(event, listener) {
      const unsubscribe = this.on(event, (payload) => {
        unsubscribe();
        listener(payload);
      });
      return unsubscribe;
    }
    off(event, listener) {
      const bucket = this.listeners.get(event);
      if (!bucket) {
        return;
      }
      bucket.delete(listener);
      if (bucket.size === 0) {
        this.listeners.delete(event);
      }
    }
    emit(event, payload) {
      const bucket = this.listeners.get(event);
      if (!bucket) {
        return;
      }
      for (const listener of Array.from(bucket)) {
        try {
          listener(payload);
        } catch (error) {
          Logger.error("event listener error", event, error);
        }
      }
    }
    clear() {
      this.listeners.clear();
    }
  };

  // src/runtime.ts
  var DEFAULT_OPTIONS = {
    endpoint: "/live",
    uploadEndpoint: "/pondlive/upload/",
    autoConnect: true,
    debug: false,
    reconnect: true,
    reconnectDelay: 1e3,
    maxReconnectAttempts: 5
  };
  var LiveRuntime = class {
    constructor(options) {
      this.events = new TypedEventEmitter();
      this.client = null;
      this.channel = null;
      this.bootPayload = null;
      this.connectPromise = null;
      this.reconnectTimer = null;
      this.reconnectAttempts = 0;
      this.disposed = false;
      this.state = { status: "disconnected" };
      this.lastAck = 0;
      this.sessionId = null;
      this.version = 0;
      this.options = {
        ...DEFAULT_OPTIONS,
        ...options ?? {}
      };
      Logger.configure({ debug: this.options.debug });
      this.bootLoader = new BootLoader({ debug: this.options.debug });
      this.bootPayload = this.options.boot ? this.bootLoader.load(this.options.boot) : this.bootLoader.load();
      this.sessionId = this.bootPayload?.sid ?? null;
      this.version = this.bootPayload?.ver ?? 0;
      Logger.debug("[Runtime]", "initialized", {
        sessionId: this.sessionId,
        version: this.version,
        hasBoot: Boolean(this.bootPayload)
      });
      if (this.options.autoConnect && this.bootPayload) {
        void this.connect();
      }
    }
    on(event, listener) {
      return this.events.on(event, listener);
    }
    once(event, listener) {
      return this.events.once(event, listener);
    }
    off(event, listener) {
      this.events.off(event, listener);
    }
    getState() {
      return this.state;
    }
    getBootPayload() {
      return this.bootPayload;
    }
    getSessionId() {
      return this.sessionId ?? this.bootPayload?.sid ?? null;
    }
    getUploadEndpoint() {
      return this.bootPayload?.client?.upload ?? this.options.uploadEndpoint;
    }
    sendUploadMessage(payload) {
      const sid = this.getSessionId();
      if (!this.channel || !sid || !payload.id) {
        return;
      }
      const message = {
        t: "upload",
        sid,
        id: payload.id,
        op: payload.op
      };
      if (payload.meta) {
        message.meta = payload.meta;
      }
      if (typeof payload.loaded === "number") {
        message.loaded = payload.loaded;
      }
      if (typeof payload.total === "number") {
        message.total = payload.total;
      }
      if (payload.error) {
        message.error = payload.error;
      }
      Logger.debug("[Runtime]", "\u2192 sending upload message", message);
      this.channel.sendMessage("upload", message);
    }
    sendDOMResponse(payload) {
      const sid = this.getSessionId();
      if (!this.channel || !sid || !payload.id) {
        return;
      }
      const message = {
        t: "domres",
        sid,
        id: payload.id
      };
      if (payload.values) {
        message.values = payload.values;
      }
      if (payload.result !== void 0) {
        message.result = payload.result;
      }
      if (payload.error) {
        message.error = payload.error;
      }
      Logger.debug("[Runtime]", "\u2192 sending domres", message);
      this.channel.sendMessage("domres", message);
    }
    async connect() {
      if (this.disposed) {
        throw new Error("[LiveUI] runtime disposed");
      }
      Logger.debug("[Runtime]", "connect requested", {
        hasBoot: Boolean(this.bootPayload),
        disposed: this.disposed,
        state: this.state.status
      });
      if (this.channel && this.state.status === "connected") {
        return;
      }
      if (this.connectPromise) {
        return this.connectPromise;
      }
      this.connectPromise = new Promise((resolve, reject) => {
        try {
          const boot = this.bootPayload ?? this.bootLoader.load();
          if (!boot || typeof boot.sid !== "string") {
            throw new Error("[LiveUI] missing boot payload; call load() before connecting");
          }
          this.bootPayload = boot;
          this.updateState({ status: "connecting" });
          Logger.debug("[Runtime]", "opening socket", { endpoint: this.options.endpoint });
          const client = new import_pondsocket_client.PondClient(this.options.endpoint);
          this.client = client;
          const joinPayload = this.buildJoinPayload(boot);
          Logger.debug("[Runtime]", "joining channel", {
            sid: boot.sid,
            ack: joinPayload.ack,
            version: joinPayload.ver
          });
          const channel = client.createChannel(`live/${boot.sid}`, joinPayload);
          this.channel = channel;
          channel.onChannelStateChange((state) => {
            Logger.debug("[Runtime]", "channel state changed", state);
            if (state === import_pondsocket_client.ChannelState.JOINED) {
              this.reconnectAttempts = 0;
              this.sessionId = boot.sid;
              this.version = boot.ver ?? 0;
              this.updateState({ status: "connected", sessionId: boot.sid, version: this.version });
              this.events.emit("connected", { sid: boot.sid, version: this.version });
              Logger.debug("[Runtime]", "session joined", { sid: boot.sid, version: this.version });
              resolve();
              this.connectPromise = null;
            }
          });
          channel.onMessage((_event, payload) => {
            Logger.debug("[Runtime]", "\u2190 received message", payload);
            this.routeMessage(payload);
          });
          channel.onLeave(() => {
            this.handleChannelLeave();
          });
          client.connect();
          channel.join();
        } catch (error) {
          Logger.debug("[Runtime]", "connect failed", error);
          this.connectPromise = null;
          this.handleErrorEvent(error);
          reject(error);
        }
      });
      return this.connectPromise;
    }
    disconnect() {
      this.clearReconnectTimer();
      this.reconnectAttempts = 0;
      this.connectPromise = null;
      if (this.channel) {
        try {
          this.channel.leave();
        } catch (error) {
          Logger.debug("[Runtime]", "channel leave error", error);
        }
        this.channel = null;
      }
      if (this.client) {
        try {
          this.client.disconnect();
        } catch (error) {
          Logger.debug("[Runtime]", "client disconnect error", error);
        }
        this.client = null;
      }
      this.updateState({ status: "disconnected" });
      this.events.emit("disconnected", void 0);
    }
    destroy() {
      this.disposed = true;
      this.disconnect();
      this.events.clear();
    }
    sendEvent(hid, payload, cseq) {
      const sid = this.sessionId ?? this.bootPayload?.sid;
      if (!this.channel || !sid) {
        return;
      }
      Logger.debug("[Runtime]", "send event", { handler: hid, cseq });
      const message = {
        t: "evt",
        sid,
        hid,
        payload,
        cseq
      };
      Logger.debug("[Runtime]", "\u2192 sending event", message);
      this.channel.sendMessage("evt", message);
    }
    sendNavigation(path, q, hash = "") {
      const sid = this.sessionId ?? this.bootPayload?.sid;
      if (!this.channel || !sid) {
        return;
      }
      Logger.debug("[Runtime]", "send navigation", { path, q, hash });
      const message = {
        t: "nav",
        sid,
        path,
        q,
        hash
      };
      Logger.debug("[Runtime]", "\u2192 sending navigation", message);
      this.channel.sendMessage("nav", message);
    }
    sendPopNavigation(path, q, hash = "") {
      const sid = this.sessionId ?? this.bootPayload?.sid;
      if (!this.channel || !sid) {
        return;
      }
      Logger.debug("[Runtime]", "send pop navigation", { path, q, hash });
      const message = {
        t: "pop",
        sid,
        path,
        q,
        hash
      };
      Logger.debug("[Runtime]", "\u2192 sending pop navigation", message);
      this.channel.sendMessage("pop", message);
    }
    requestRecover() {
      const sid = this.sessionId ?? this.bootPayload?.sid;
      if (!this.channel || !sid) {
        return;
      }
      Logger.debug("[Runtime]", "request recover", { sid });
      const payload = {
        t: "recover",
        sid
      };
      Logger.debug("[Runtime]", "\u2192 sending recover", payload);
      this.channel.sendMessage("recover", payload);
    }
    buildJoinPayload(boot) {
      const ack = this.lastAck || boot.seq || 0;
      return {
        sid: boot.sid,
        ver: boot.ver ?? 0,
        ack,
        loc: {
          path: boot.location?.path ?? "/",
          q: boot.location?.q ?? "",
          hash: boot.location?.hash ?? ""
        }
      };
    }
    routeMessage(msg) {
      if (!msg || typeof msg.t !== "string") {
        return;
      }
      Logger.debug("[Runtime]", "received message", { type: msg?.t });
      this.events.emit("message", msg);
      switch (msg.t) {
        case "init":
          this.handleInit(msg);
          break;
        case "frame":
          this.handleFrame(msg);
          break;
        case "template":
          this.events.emit("template", msg);
          break;
        case "resume":
          this.events.emit("resume", msg);
          break;
        case "join":
          this.events.emit("join", msg);
          break;
        case "diagnostic":
          this.events.emit("diagnostic", msg);
          break;
        case "error":
          this.handleErrorMessage(msg);
          break;
        case "upload":
          this.events.emit("upload", msg);
          break;
        case "domreq":
          this.events.emit("domreq", msg);
          break;
        default:
          break;
      }
    }
    handleInit(msg) {
      this.sessionId = msg.sid;
      this.version = msg.ver;
      Logger.debug("[Runtime]", "init received", { sid: msg.sid, version: msg.ver, seq: msg.seq });
      if (typeof msg.seq === "number") {
        this.lastAck = msg.seq;
        this.sendAck(msg.seq);
      }
      this.events.emit("init", msg);
    }
    handleFrame(msg) {
      this.version = msg.ver;
      Logger.debug("[Runtime]", "frame received", {
        version: msg.ver,
        seq: msg.seq,
        ops: Array.isArray(msg.patch) ? msg.patch.length : 0
      });
      if (typeof msg.seq === "number") {
        this.lastAck = msg.seq;
        this.sendAck(msg.seq);
      }
      this.events.emit("frame", msg);
    }
    handleErrorMessage(msg) {
      const error = new Error(msg.message ?? "server error");
      error.name = msg.code ?? "ServerError";
      this.handleErrorEvent(error);
    }
    handleChannelLeave() {
      Logger.debug("[Runtime]", "channel left, cleaning up");
      this.channel = null;
      this.sessionId = null;
      this.version = 0;
      if (this.client) {
        try {
          this.client.disconnect();
        } catch (error) {
          Logger.debug("[Runtime]", "client disconnect error", error);
        }
        this.client = null;
      }
      this.updateState({ status: "disconnected" });
      this.events.emit("disconnected", void 0);
      if (!this.disposed && this.options.reconnect) {
        this.scheduleReconnect();
      }
    }
    scheduleReconnect() {
      if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
        return;
      }
      if (this.reconnectTimer) {
        return;
      }
      this.reconnectAttempts += 1;
      const delay = this.options.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
      Logger.debug("[Runtime]", "scheduling reconnect", {
        attempt: this.reconnectAttempts,
        delay
      });
      this.updateState({ status: "reconnecting", attempt: this.reconnectAttempts });
      this.reconnectTimer = setTimeout(() => {
        this.reconnectTimer = null;
        void this.connect();
      }, delay);
    }
    clearReconnectTimer() {
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }
    }
    sendAck(seq) {
      const sid = this.sessionId ?? this.bootPayload?.sid;
      if (!this.channel || !sid) {
        return;
      }
      Logger.debug("[Runtime]", "acknowledging frame", { seq });
      const payload = {
        t: "ack",
        sid,
        seq
      };
      Logger.debug("[Runtime]", "\u2192 sending ack", payload);
      this.channel.sendMessage("ack", payload);
    }
    handleErrorEvent(error) {
      Logger.warn("[Runtime]", "runtime error", error);
      this.events.emit("error", { error });
      this.updateState({ status: "error", error });
    }
    updateState(next) {
      this.state = next;
      this.events.emit("state", next);
    }
  };

  // src/manifest.ts
  var componentRanges = /* @__PURE__ */ new Map();
  function registerComponentRanges(ranges) {
    ranges.forEach((range, id) => {
      if (id) {
        componentRanges.set(id, range);
      }
    });
  }
  function getComponentRange(id) {
    return componentRanges.get(id);
  }
  function resolveSlotAnchors(descriptors, overrides) {
    const anchors = /* @__PURE__ */ new Map();
    if (!Array.isArray(descriptors)) {
      return anchors;
    }
    for (const descriptor of descriptors) {
      if (!descriptor) continue;
      const slotId = Number(descriptor.slot);
      if (!Number.isInteger(slotId) || slotId < 0 || anchors.has(slotId)) {
        continue;
      }
      const range = overrides?.get(descriptor.componentId) ?? getComponentRange(descriptor.componentId);
      if (!range) {
        continue;
      }
      const anchor = resolveNodeBySegments(range, descriptor.path);
      if (!anchor) {
        continue;
      }
      let target = anchor;
      const textIndex = Number(descriptor.textChildIndex);
      if (Number.isInteger(textIndex) && textIndex >= 0) {
        if (anchor instanceof Text) {
          target = anchor;
        } else if (anchor instanceof Element || anchor instanceof DocumentFragment) {
          const existing = anchor.childNodes.item(textIndex);
          if (existing) {
            target = existing;
          } else if (typeof document !== "undefined") {
            const textNode = document.createTextNode("");
            const before = anchor.childNodes.item(textIndex) ?? null;
            anchor.insertBefore(textNode, before);
            target = textNode;
          }
        } else {
          target = null;
        }
      }
      if (!target) {
        continue;
      }
      anchors.set(slotId, target);
    }
    return anchors;
  }
  function resolveListContainers(descriptors, overrides) {
    const containers = /* @__PURE__ */ new Map();
    if (!Array.isArray(descriptors)) {
      return containers;
    }
    for (const descriptor of descriptors) {
      if (!descriptor) continue;
      const slotId = Number(descriptor.slot);
      if (!Number.isInteger(slotId) || slotId < 0 || containers.has(slotId)) {
        continue;
      }
      const range = overrides?.get(descriptor.componentId) ?? getComponentRange(descriptor.componentId);
      if (!range) {
        continue;
      }
      if (descriptor.atRoot) {
        const element = resolveRangeContainerElement(range);
        if (element) {
          containers.set(slotId, element);
        }
        continue;
      }
      const node = resolveNodeBySegments(range, descriptor.path);
      if (!(node instanceof Element)) {
        continue;
      }
      containers.set(slotId, node);
    }
    return containers;
  }
  function applyComponentRanges(descriptors, options) {
    const ranges = computeComponentRanges(descriptors, options);
    registerComponentRanges(ranges);
    return ranges;
  }
  function computeComponentRanges(descriptors, options) {
    const ranges = /* @__PURE__ */ new Map();
    if (!Array.isArray(descriptors)) {
      return ranges;
    }
    const root = options?.root ?? document.body ?? document;
    const baseRange = root ? { container: root, startIndex: 0, endIndex: root.childNodes.length - 1 } : null;
    const pending = /* @__PURE__ */ new Map();
    descriptors.forEach((descriptor) => {
      if (descriptor && typeof descriptor.componentId === "string") {
        pending.set(descriptor.componentId, descriptor);
      }
    });
    let progressed = true;
    while (pending.size > 0 && progressed) {
      progressed = false;
      for (const [id, descriptor] of Array.from(pending.entries())) {
        const parentRange = descriptor.parentId ? ranges.get(descriptor.parentId) : baseRange;
        if (!parentRange) {
          continue;
        }
        Logger.debug("[Manifest]", "computing component range", {
          id,
          parentId: descriptor.parentId,
          parentPath: descriptor.parentPath,
          firstChild: descriptor.firstChild,
          lastChild: descriptor.lastChild,
          parentRangeContainer: parentRange.container.nodeName,
          parentRangeStart: parentRange.startIndex,
          parentRangeEnd: parentRange.endIndex
        });
        const firstNode = resolveNodeForComponent(parentRange, descriptor.parentPath, descriptor.firstChild);
        const lastNode = resolveNodeForComponent(parentRange, descriptor.parentPath, descriptor.lastChild);
        Logger.debug("[Manifest]", "resolved boundary nodes", {
          id,
          firstNode: firstNode?.nodeName,
          lastNode: lastNode?.nodeName
        });
        const container = chooseComponentContainer(firstNode, lastNode, parentRange.container);
        const topLevelFirst = ascendToContainer(firstNode, container) ?? firstNode;
        const topLevelLast = ascendToContainer(lastNode, container) ?? lastNode ?? topLevelFirst;
        let startIndex = topLevelFirst ? getNodeIndex(container, topLevelFirst) : parentRange.startIndex;
        if (startIndex < 0) {
          startIndex = parentRange.startIndex;
        }
        let endIndex = topLevelLast ? getNodeIndex(container, topLevelLast) : startIndex;
        if (endIndex < startIndex) {
          endIndex = startIndex;
        }
        Logger.debug("[Manifest]", "computed component range", {
          id,
          container: container.nodeName,
          startIndex,
          endIndex
        });
        ranges.set(id, { container, startIndex, endIndex });
        pending.delete(id);
        progressed = true;
      }
    }
    return ranges;
  }
  function resolveNodeInComponent(componentId, path, overrides) {
    const range = overrides?.get(componentId) ?? getComponentRange(componentId);
    if (!range) {
      return null;
    }
    return resolveNodeBySegments(range, path);
  }
  function resolveNodeBySegments(range, segments) {
    if (!range || range.endIndex < range.startIndex) {
      Logger.debug("[Manifest]", "resolveNodeBySegments: invalid range", { range });
      return null;
    }
    const container = range.container;
    if (!container) {
      Logger.debug("[Manifest]", "resolveNodeBySegments: no container", { range });
      return null;
    }
    Logger.debug("[Manifest]", "resolveNodeBySegments START", {
      segments,
      rangeInfo: {
        containerNodeName: container.nodeName,
        startIndex: range.startIndex,
        endIndex: range.endIndex,
        childCount: container.childNodes.length
      }
    });
    if (!Array.isArray(segments) || segments.length === 0) {
      const root = resolveRangeRoot(range);
      Logger.debug("[Manifest]", "resolveNodeBySegments: using range root", { result: root?.nodeName });
      return root;
    }
    let current = null;
    for (let i = 0; i < segments.length; i++) {
      const segment = segments[i];
      if (!segment) {
        continue;
      }
      if (segment.kind === "range") {
        current = resolveRangeChild(range, segment.index);
        Logger.debug("[Manifest]", "resolveNodeBySegments: range segment", {
          step: i,
          offset: segment.index,
          node: current?.nodeName
        });
        if (!current) {
          return null;
        }
        continue;
      }
      if (!current) {
        current = resolveRangeChild(range, segment.index);
        Logger.debug("[Manifest]", "resolveNodeBySegments: selecting top-level child", {
          step: i,
          offset: segment.index,
          node: current?.nodeName
        });
        if (!current) {
          return null;
        }
        continue;
      }
      if (!(current instanceof Element || current instanceof DocumentFragment)) {
        Logger.debug("[Manifest]", "resolveNodeBySegments: current not Element/Fragment", {
          step: i,
          current: current?.nodeName
        });
        return null;
      }
      const clamped = clampIndex(current, segment.index);
      const next = current.childNodes.item(clamped) ?? null;
      Logger.debug("[Manifest]", "resolveNodeBySegments: navigating dom segment", {
        step: i,
        index: segment.index,
        clamped,
        currentNodeName: current.nodeName,
        nextNodeName: next?.nodeName
      });
      current = next;
      if (!current) {
        return null;
      }
    }
    Logger.debug("[Manifest]", "resolveNodeBySegments END", { result: current?.nodeName });
    return current ?? resolveRangeRoot(range);
  }
  function resolveNodeForComponent(range, parentPath, childPath) {
    const combined = combineSegments(parentPath, childPath);
    if (combined) {
      return resolveNodeBySegments(range, combined);
    }
    return resolveNodeBySegments(range, parentPath) ?? resolveNodeBySegments(range, childPath);
  }
  function combineSegments(base, extra) {
    const result = [];
    if (Array.isArray(base) && base.length > 0) {
      result.push(...base);
    }
    if (Array.isArray(extra) && extra.length > 0) {
      result.push(...extra);
    }
    return result.length > 0 ? result : void 0;
  }
  function chooseComponentContainer(firstNode, lastNode, fallback) {
    const firstAncestors = collectAncestorParents(firstNode);
    if (!lastNode) {
      return firstAncestors[0] ?? fallback;
    }
    const lastAncestors = collectAncestorParents(lastNode);
    for (const ancestor of firstAncestors) {
      if (lastAncestors.includes(ancestor)) {
        return ancestor;
      }
    }
    return firstAncestors[0] ?? lastAncestors[0] ?? fallback;
  }
  function collectAncestorParents(node) {
    const parents = [];
    let current = node;
    while (current && current.parentNode) {
      const parent = current.parentNode;
      parents.push(parent);
      current = parent;
    }
    return parents;
  }
  function ascendToContainer(node, container) {
    if (!node || !container) {
      return node;
    }
    let current = node;
    while (current && current.parentNode && current.parentNode !== container) {
      current = current.parentNode;
    }
    return current;
  }
  function clampIndex(container, index) {
    const max = container.childNodes.length - 1;
    if (max < 0) {
      return 0;
    }
    if (index < 0) {
      return 0;
    }
    if (index > max) {
      return max;
    }
    return index;
  }
  function getNodeIndex(container, node) {
    if (!container || !node) {
      return -1;
    }
    const nodes = container.childNodes;
    for (let i = 0; i < nodes.length; i++) {
      if (nodes.item(i) === node) {
        return i;
      }
    }
    return -1;
  }
  function resolveRangeRoot(range) {
    return resolveRangeChild(range, 0);
  }
  function resolveRangeChild(range, offset) {
    const container = range.container;
    if (!container || container.childNodes.length === 0) {
      return null;
    }
    const start = clampIndex(container, range.startIndex);
    const end = Math.max(start, clampIndex(container, range.endIndex));
    const span = end - start;
    const normalizedOffset = Number.isFinite(offset) ? offset : 0;
    const clampedOffset = normalizedOffset <= 0 ? 0 : normalizedOffset >= span ? span : normalizedOffset;
    const index = start + clampedOffset;
    return container.childNodes.item(index) ?? null;
  }
  function resolveRangeContainerElement(range) {
    const container = range.container;
    if (container instanceof Element) {
      return container;
    }
    if (container instanceof Document) {
      return container.body ?? container.documentElement ?? null;
    }
    if (container instanceof DocumentFragment) {
      const first = container.firstElementChild;
      return first ?? null;
    }
    return null;
  }

  // src/dom-registry.ts
  var DomRegistry = class {
    constructor() {
      this.slots = /* @__PURE__ */ new Map();
      this.lists = /* @__PURE__ */ new Map();
    }
    reset() {
      this.slots.clear();
      this.lists.clear();
      Logger.debug("[DomRegistry]", "reset");
    }
    prime(componentPaths, options) {
      return applyComponentRanges(componentPaths, options);
    }
    registerSlotAnchors(descriptors, overrides) {
      const anchors = resolveSlotAnchors(descriptors, overrides);
      anchors.forEach((node, id) => this.slots.set(id, node));
      Logger.debug("[DomRegistry]", "registered slot anchors", { count: anchors.size });
    }
    registerListContainers(descriptors, overrides, rowMeta) {
      const containers = resolveListContainers(descriptors, overrides);
      containers.forEach((element, id) => {
        const info = rowMeta?.get(id);
        const { rows, order } = collectRows(element, info);
        this.lists.set(id, { container: element, rows, order });
      });
      Logger.debug("[DomRegistry]", "registered list containers", { count: containers.size });
    }
    registerSlots(slotDescriptors) {
      if (!Array.isArray(slotDescriptors)) {
        return;
      }
      slotDescriptors.forEach((slot) => {
        if (slot && typeof slot.anchorId === "number" && !this.slots.has(slot.anchorId)) {
          const placeholder = typeof document !== "undefined" ? document.createTextNode("") : null;
          if (placeholder) {
            this.slots.set(slot.anchorId, placeholder);
          }
        }
      });
      Logger.debug("[DomRegistry]", "registered additional slots", { total: this.slots.size });
    }
    getSlot(id) {
      return this.slots.get(id);
    }
    registerLists(listDescriptors, overrides, rowMeta) {
      this.registerListContainers(listDescriptors, overrides, rowMeta);
    }
    getList(id) {
      return this.lists.get(id)?.container;
    }
    getRow(slotId, key) {
      return this.lists.get(slotId)?.rows.get(key);
    }
    insertRow(slotId, key, nodes, index) {
      if (!key || nodes.length === 0) {
        return;
      }
      const list = this.lists.get(slotId);
      if (!list) {
        return;
      }
      list.rows.set(key, { nodes: [...nodes] });
      const clamped = clampIndex2(list.order.length, index);
      list.order.splice(clamped, 0, key);
      Logger.debug("[DomRegistry]", "inserted row", { slotId, key, index: clamped });
    }
    deleteRow(slotId, key) {
      const list = this.lists.get(slotId);
      if (!list) {
        return;
      }
      list.rows.delete(key);
      const idx = list.order.indexOf(key);
      if (idx >= 0) {
        list.order.splice(idx, 1);
      }
      Logger.debug("[DomRegistry]", "deleted row", { slotId, key });
    }
    moveRow(slotId, key, toIndex) {
      const list = this.lists.get(slotId);
      if (!list) {
        return;
      }
      const current = list.order.indexOf(key);
      if (current === -1) {
        return;
      }
      list.order.splice(current, 1);
      const clamped = clampIndex2(list.order.length, toIndex);
      list.order.splice(clamped, 0, key);
      Logger.debug("[DomRegistry]", "moved row", { slotId, key, to: clamped });
    }
    getRowKeyAt(slotId, index) {
      const list = this.lists.get(slotId);
      if (!list) {
        return void 0;
      }
      const order = list.order ?? [];
      if (index < 0 || index >= order.length) {
        return void 0;
      }
      return order[index];
    }
    getRowFirstNode(slotId, key) {
      const record = this.getRow(slotId, key);
      return record?.nodes[0];
    }
  };
  function collectRows(container, meta) {
    const rows = /* @__PURE__ */ new Map();
    const order = [];
    if (!container || !Array.isArray(meta) || meta.length === 0) {
      return { rows, order };
    }
    const nodes = Array.from(container.childNodes);
    let cursor = 0;
    for (const info of meta) {
      if (!info || !info.key) {
        continue;
      }
      const count = Math.max(1, Number(info.count) || 1);
      const span = [];
      for (let i = 0; i < count && cursor < nodes.length; i += 1, cursor += 1) {
        const node = nodes[cursor];
        if (node) {
          span.push(node);
        }
      }
      if (span.length === 0) {
        continue;
      }
      rows.set(info.key, { nodes: span });
      order.push(info.key);
    }
    return { rows, order };
  }
  function clampIndex2(length, index) {
    if (!Number.isFinite(index) || index < 0) {
      return 0;
    }
    if (index > length) {
      return length;
    }
    return index;
  }

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

  // src/refs.ts
  var refEventObservers = /* @__PURE__ */ new Set();
  var registeredRefEvents = /* @__PURE__ */ new Set();
  function observeRefEvents(observer) {
    refEventObservers.add(observer);
    return () => refEventObservers.delete(observer);
  }
  function getRegisteredRefEvents() {
    return Array.from(registeredRefEvents);
  }
  function registerRefEvents(events) {
    if (!Array.isArray(events)) {
      return;
    }
    const newlyAdded = [];
    events.forEach((event) => {
      if (!event) {
        return;
      }
      if (!registeredRefEvents.has(event)) {
        registeredRefEvents.add(event);
        newlyAdded.push(event);
      }
    });
    if (newlyAdded.length === 0) {
      return;
    }
    newlyAdded.forEach((event) => {
      refEventObservers.forEach((observer) => {
        try {
          observer(event);
        } catch {
        }
      });
    });
  }
  var RefRegistry = class {
    constructor(runtime) {
      this.runtime = runtime;
      this.meta = /* @__PURE__ */ new Map();
      this.bindings = /* @__PURE__ */ new Map();
    }
    clear() {
      Array.from(this.bindings.keys()).forEach((id) => this.detach(id));
      this.meta.clear();
      Logger.debug("[Refs]", "cleared all bindings");
    }
    apply(delta) {
      if (!delta) {
        return;
      }
      if (Array.isArray(delta.del)) {
        for (const id of delta.del) {
          this.detach(id);
          this.meta.delete(id);
        }
      }
      if (delta.add) {
        for (const [id, meta] of Object.entries(delta.add)) {
          if (id) {
            this.meta.set(id, meta);
            registerRefEvents(Object.keys(meta.events ?? {}));
          }
        }
      }
      Logger.debug("[Refs]", "applied ref delta", {
        added: delta.add ? Object.keys(delta.add).length : 0,
        removed: delta.del?.length ?? 0
      });
    }
    registerBindings(descriptors, overrides) {
      if (!Array.isArray(descriptors)) {
        return;
      }
      Logger.debug("[Refs]", "registering ref bindings", { count: descriptors.length });
      descriptors.forEach((descriptor, index) => {
        if (!descriptor || typeof descriptor.refId !== "string" || descriptor.refId.length === 0) {
          return;
        }
        Logger.debug("[Refs]", "processing ref binding", {
          index,
          descriptor: {
            componentId: descriptor.componentId,
            path: descriptor.path,
            refId: descriptor.refId
          },
          hasRefId: !!descriptor.refId,
          refIdLength: descriptor.refId?.length ?? 0
        });
        Logger.debug("[Refs]", "resolving node for ref", {
          refId: descriptor.refId,
          componentId: descriptor.componentId,
          path: descriptor.path
        });
        const node = resolveNodeInComponent(descriptor.componentId, descriptor.path, overrides);
        Logger.debug("[Refs]", "node resolved", {
          refId: descriptor.refId,
          node,
          isElement: node instanceof Element,
          nodeType: node?.nodeType,
          nodeName: node?.nodeName
        });
        if (node instanceof Element) {
          Logger.debug("[Refs]", "attaching ref", { refId: descriptor.refId, node });
          this.attach(descriptor.refId, node);
        } else {
          Logger.debug("[Refs]", "node is not Element, detaching", { refId: descriptor.refId, node });
          this.detach(descriptor.refId);
        }
      });
    }
    get(id) {
      return this.bindings.get(id)?.element;
    }
    attach(refId, element) {
      const meta = this.meta.get(refId);
      if (!meta) {
        return;
      }
      const existing = this.bindings.get(refId);
      if (existing && existing.element === element) {
        return;
      }
      this.detach(refId);
      const listeners = /* @__PURE__ */ new Map();
      const events = meta.events ?? {};
      registerRefEvents(Object.keys(events));
      Object.entries(events).forEach(([eventName, spec]) => {
        if (!eventName) {
          return;
        }
        const listener = (event) => {
          const detail = extractEventDetail(event, spec?.props, { refElement: element });
          const payload = detail ? { name: eventName, detail } : { name: eventName };
          const handlerKey = spec?.handler || `${refId}/${eventName}`;
          this.runtime.sendEvent(handlerKey, payload);
        };
        element.addEventListener(eventName, listener);
        listeners.set(eventName, listener);
      });
      this.bindings.set(refId, { element, listeners });
      Logger.debug("[Refs]", "attached ref", {
        refId,
        tag: element.tagName,
        events: Object.keys(events).length
      });
    }
    detach(refId) {
      const binding = this.bindings.get(refId);
      if (!binding) {
        return;
      }
      binding.listeners.forEach((listener, event) => {
        binding.element.removeEventListener(event, listener);
      });
      this.bindings.delete(refId);
      Logger.debug("[Refs]", "detached ref", { refId });
    }
  };

  // src/events.ts
  var slotTable = /* @__PURE__ */ new Map();
  var eventSlots = /* @__PURE__ */ new Map();
  var eventObservers = /* @__PURE__ */ new Set();
  function registerSlotTable(table) {
    slotTable.clear();
    eventSlots.clear();
    if (!table) {
      Logger.debug("[Events]", "cleared slot table");
      return;
    }
    for (const [key, value] of Object.entries(table)) {
      const slotId = Number(key);
      if (Number.isNaN(slotId)) {
        continue;
      }
      const bindings = cloneBindings(value);
      slotTable.set(slotId, bindings);
      addEventIndex(slotId, bindings);
    }
    Logger.debug("[Events]", "registered slot table", { slots: slotTable.size });
  }
  function registerBindingsForSlot(slotId, specs) {
    if (!Number.isFinite(slotId)) {
      return;
    }
    slotTable.delete(slotId);
    eventSlots.forEach((set) => set.delete(slotId));
    if (!Array.isArray(specs)) {
      return;
    }
    const bindings = cloneBindings(specs);
    slotTable.set(slotId, bindings);
    addEventIndex(slotId, bindings);
    Logger.debug("[Events]", "registered slot bindings", { slotId, count: bindings.length });
  }
  function getSlotBindings(slotId) {
    const bindings = slotTable.get(slotId);
    return bindings ? cloneBindings(bindings) : void 0;
  }
  function getSlotsForEvent(event) {
    const set = eventSlots.get(event);
    return set ? Array.from(set) : [];
  }
  function getRegisteredSlotEvents() {
    return Array.from(eventSlots.keys());
  }
  function observeSlotEvents(observer) {
    eventObservers.add(observer);
    return () => eventObservers.delete(observer);
  }
  function addEventIndex(slotId, bindings) {
    bindings.forEach((binding) => {
      const event = binding.event;
      if (!event) {
        return;
      }
      const set = eventSlots.get(event) ?? /* @__PURE__ */ new Set();
      const hadEvent = set.size > 0;
      set.add(slotId);
      eventSlots.set(event, set);
      if (!hadEvent && set.size === 1) {
        notifyObservers([event]);
      }
      Logger.debug("[Events]", "indexed event", { event, slotId, totalSlots: set.size });
    });
  }
  function notifyObservers(events) {
    if (!Array.isArray(events)) {
      return;
    }
    events.forEach((event) => {
      if (!event) {
        return;
      }
      eventObservers.forEach((observer) => {
        try {
          observer(event);
        } catch {
        }
      });
    });
  }
  function cloneBindings(specs) {
    if (!Array.isArray(specs)) {
      return [];
    }
    return specs.map((spec) => ({
      event: spec?.event ?? "",
      handler: spec?.handler ?? "",
      listen: Array.isArray(spec?.listen) ? [...spec.listen] : void 0,
      props: Array.isArray(spec?.props) ? [...spec.props] : void 0
    }));
  }

  // src/router-bindings.ts
  var routerMeta = /* @__PURE__ */ new WeakMap();
  function applyRouterBindings(descriptors, overrides) {
    if (!Array.isArray(descriptors)) {
      return;
    }
    let applied = 0;
    descriptors.forEach((descriptor) => {
      if (!descriptor || typeof descriptor.componentId !== "string") {
        return;
      }
      const node = resolveNodeInComponent(descriptor.componentId, descriptor.path, overrides);
      if (!(node instanceof Element)) {
        return;
      }
      routerMeta.set(node, {
        path: descriptor.pathValue ?? void 0,
        query: descriptor.query ?? void 0,
        hash: descriptor.hash ?? void 0,
        replace: descriptor.replace ?? void 0
      });
      applied += 1;
    });
    Logger.debug("[Router]", "applied router bindings", { count: applied });
  }
  function getRouterMeta(element) {
    if (!element) {
      return void 0;
    }
    let current = element;
    while (current) {
      const meta = routerMeta.get(current);
      if (meta?.path) {
        return meta;
      }
      current = current.parentElement;
    }
    return void 0;
  }

  // src/patcher.ts
  var Patcher = class {
    constructor(dom, refs, uploads) {
      this.dom = dom;
      this.refs = refs;
      this.uploads = uploads;
    }
    applyFrame(frame) {
      if (!Array.isArray(frame.patch)) {
        return;
      }
      Logger.debug("[Patcher]", "applying frame patch", { opCount: frame.patch.length });
      this.applyOps(frame.patch);
    }
    applyOps(ops) {
      Logger.debug("[Patcher]", "applying ops", { count: ops.length });
      for (const op of ops) {
        if (!Array.isArray(op) || op.length === 0) {
          continue;
        }
        const kind = op[0];
        switch (kind) {
          case "setText":
            this.applySetText(op[1], op[2]);
            break;
          case "setAttrs":
            this.applySetAttrs(op[1], op[2] || {});
            break;
          case "list":
            this.applyList(op[1], op.slice(2));
            break;
        }
      }
    }
    applySetText(slotId, value) {
      const node = this.dom.getSlot(slotId);
      if (!node) {
        Logger.debug("[Patcher]", "setText skipped (missing slot)", { slotId });
        return;
      }
      if (node instanceof Text) {
        node.textContent = value ?? "";
        Logger.debug("[Patcher]", "setText applied", { slotId, nodeType: "Text" });
        return;
      }
      if (node instanceof Element) {
        node.textContent = value ?? "";
        Logger.debug("[Patcher]", "setText applied", { slotId, nodeType: node.tagName });
      }
    }
    applySetAttrs(slotId, attrs) {
      const node = this.dom.getSlot(slotId);
      if (!(node instanceof Element)) {
        Logger.debug("[Patcher]", "setAttrs skipped (non-element)", { slotId });
        return;
      }
      Object.entries(attrs ?? {}).forEach(([key, value]) => {
        if (value === null || value === void 0 || value === "") {
          node.removeAttribute(key);
        } else {
          node.setAttribute(key, value);
        }
      });
      Logger.debug("[Patcher]", "setAttrs applied", { slotId, keys: Object.keys(attrs ?? {}) });
    }
    applyList(slotId, childOps) {
      const container = this.dom.getList(slotId);
      if (!(container instanceof Element)) {
        Logger.debug("[Patcher]", "list op skipped (no container)", { slotId });
        return;
      }
      Logger.debug("[Patcher]", "list patch", { slotId, opCount: childOps.length });
      childOps.forEach((op) => {
        if (!Array.isArray(op)) {
          return;
        }
        switch (op[0]) {
          case "del":
            this.applyListDelete(slotId, container, op);
            break;
          case "ins":
            this.applyListInsert(slotId, container, op);
            break;
          case "mov":
            this.applyListMove(slotId, container, op);
            break;
        }
      });
    }
    applyListDelete(slotId, container, op) {
      const key = op[1];
      const record = this.dom.getRow(slotId, key);
      if (!record) {
        Logger.debug("[Patcher]", "list delete skipped (missing row)", { slotId, key });
        return;
      }
      record.nodes.forEach((node) => {
        if (node.parentNode === container) {
          container.removeChild(node);
        }
      });
      this.dom.deleteRow(slotId, key);
      Logger.debug("[Patcher]", "list delete applied", { slotId, key });
    }
    applyListInsert(slotId, container, op) {
      const [_, index, payload] = op;
      const html = payload?.html ?? "";
      if (!html) {
        Logger.debug("[Patcher]", "list insert skipped (no html)", { slotId, index });
        return;
      }
      const template = document.createElement("template");
      template.innerHTML = html;
      const fragment = template.content;
      const nodes = Array.from(fragment.childNodes);
      if (nodes.length === 0) {
        Logger.debug("[Patcher]", "list insert skipped (no nodes)", { slotId, index });
        return;
      }
      const beforeKey = this.dom.getRowKeyAt(slotId, index);
      const before = beforeKey ? this.dom.getRowFirstNode(slotId, beforeKey) : null;
      container.insertBefore(fragment, before ?? null);
      this.dom.insertRow(slotId, payload?.key ?? "", nodes, index);
      const root = nodes.find((node) => node instanceof Element) ?? null;
      if (root) {
        this.registerRowMetadata(
          root,
          payload?.componentPaths ?? [],
          payload?.slotPaths,
          payload?.listPaths,
          payload?.bindings
        );
      }
      Logger.debug("[Patcher]", "list insert applied", {
        slotId,
        index,
        key: payload?.key,
        nodeCount: nodes.length
      });
    }
    applyListMove(slotId, container, op) {
      const from = op[1];
      const to = op[2];
      if (from === to) {
        return;
      }
      const currentKey = this.dom.getRowKeyAt(slotId, from);
      if (!currentKey) {
        Logger.debug("[Patcher]", "list move skipped (missing source key)", { slotId, from, to });
        return;
      }
      const record = this.dom.getRow(slotId, currentKey);
      if (!record) {
        Logger.debug("[Patcher]", "list move skipped (missing record)", { slotId, from, to });
        return;
      }
      const targetKey = this.dom.getRowKeyAt(slotId, to);
      const beforeNode = targetKey ? this.dom.getRowFirstNode(slotId, targetKey) : null;
      record.nodes.forEach((node) => {
        if (node.parentNode !== container) {
          return;
        }
        container.insertBefore(node, beforeNode ?? null);
      });
      this.dom.moveRow(slotId, currentKey, to);
      Logger.debug("[Patcher]", "list move applied", { slotId, from, to, key: currentKey });
    }
    registerRowMetadata(root, componentPaths, slotPaths, listPaths, bindings) {
      const overrides = applyComponentRanges(componentPaths ?? [], { root });
      this.dom.registerSlotAnchors(slotPaths ?? void 0, overrides);
      this.dom.registerLists(listPaths ?? void 0, overrides);
      if (bindings?.slots) {
        Object.entries(bindings.slots).forEach(([slot, specs]) => {
          registerBindingsForSlot(Number(slot), specs);
        });
      }
      if (bindings?.router) {
        applyRouterBindings(bindings.router, overrides);
      }
      if (bindings?.refs) {
        this.refs?.registerBindings(bindings.refs, overrides);
      }
      if (bindings?.uploads) {
        this.uploads?.registerBindings(bindings.uploads, overrides, { replace: false });
      }
      Logger.debug("[Patcher]", "row metadata registered", {
        slots: bindings?.slots ? Object.keys(bindings.slots).length : 0,
        routers: bindings?.router?.length ?? 0,
        refs: bindings?.refs?.length ?? 0,
        uploads: bindings?.uploads?.length ?? 0
      });
    }
  };

  // src/uploads.ts
  var UploadManager = class {
    constructor(runtime) {
      this.runtime = runtime;
      this.bindings = /* @__PURE__ */ new Map();
      this.elementToUpload = /* @__PURE__ */ new WeakMap();
      this.active = /* @__PURE__ */ new Map();
      this.componentBindings = /* @__PURE__ */ new Map();
    }
    clear() {
      this.bindings.forEach((binding) => {
        binding.element.removeEventListener("change", binding.changeHandler);
      });
      this.bindings.clear();
      this.componentBindings.clear();
      this.active.forEach((upload) => upload.xhr.abort());
      this.active.clear();
      Logger.debug("[Uploads]", "cleared bindings and active uploads");
    }
    prime(descriptors, overrides) {
      this.clear();
      this.registerBindings(descriptors, overrides);
      Logger.debug("[Uploads]", "primed upload bindings", { count: descriptors?.length ?? 0 });
    }
    registerBindings(descriptors, overrides, options) {
      if (!Array.isArray(descriptors)) {
        return;
      }
      Logger.debug("[Uploads]", "register bindings", {
        count: descriptors.length,
        replace: options?.replace !== false
      });
      if (options?.replace === false) {
        descriptors.forEach((descriptor) => {
          if (!descriptor || typeof descriptor.uploadId !== "string" || descriptor.uploadId.length === 0) {
            return;
          }
          this.attachDescriptor(descriptor, overrides);
        });
        return;
      }
      const grouped = /* @__PURE__ */ new Map();
      descriptors.forEach((descriptor) => {
        if (!descriptor || typeof descriptor.uploadId !== "string" || descriptor.uploadId.length === 0) {
          return;
        }
        const componentId = descriptor.componentId || "__root__";
        const list = grouped.get(componentId) ?? [];
        list.push(descriptor);
        grouped.set(componentId, list);
      });
      grouped.forEach((list, componentId) => {
        this.replaceBindingsForComponent(componentId, list, overrides);
      });
    }
    replaceBindingsForComponent(componentId, descriptors, overrides) {
      const id = componentId || "__root__";
      const existing = this.componentBindings.get(id);
      if (existing) {
        existing.forEach((uploadId) => this.detachBinding(uploadId));
        this.componentBindings.delete(id);
      }
      if (!descriptors || descriptors.length === 0) {
        return;
      }
      const next = /* @__PURE__ */ new Set();
      descriptors.forEach((descriptor) => {
        if (!descriptor || typeof descriptor.uploadId !== "string" || descriptor.uploadId.length === 0) {
          return;
        }
        this.attachDescriptor(descriptor, overrides);
        next.add(descriptor.uploadId);
      });
      if (next.size > 0) {
        this.componentBindings.set(id, next);
      }
      Logger.debug("[Uploads]", "component bindings replaced", {
        componentId: id,
        count: descriptors?.length ?? 0
      });
    }
    handleControl(message) {
      if (!message || !message.id) {
        return;
      }
      Logger.debug("[Uploads]", "control message", { op: message.op, id: message.id });
      if (message.op === "cancel") {
        this.abortUpload(message.id, true);
      } else if (message.op === "error") {
        this.abortUpload(message.id, true);
      }
    }
    attachDescriptor(descriptor, overrides) {
      const node = resolveNodeInComponent(descriptor.componentId, descriptor.path, overrides);
      const element = this.resolveInput(node);
      if (!element) {
        return;
      }
      const uploadId = descriptor.uploadId;
      this.detachBinding(uploadId);
      const handler = () => this.handleInputChange(uploadId, element, descriptor);
      element.addEventListener("change", handler);
      this.syncAttributes(element, descriptor);
      this.bindings.set(uploadId, { id: uploadId, element, descriptor, changeHandler: handler });
      this.elementToUpload.set(element, uploadId);
      const componentId = descriptor.componentId || "__root__";
      const set = this.componentBindings.get(componentId) ?? /* @__PURE__ */ new Set();
      set.add(uploadId);
      this.componentBindings.set(componentId, set);
      Logger.debug("[Uploads]", "attached upload descriptor", {
        uploadId,
        componentId,
        multiple: Boolean(descriptor.multiple)
      });
    }
    detachBinding(uploadId) {
      const binding = this.bindings.get(uploadId);
      if (!binding) {
        return;
      }
      binding.element.removeEventListener("change", binding.changeHandler);
      this.bindings.delete(uploadId);
      this.elementToUpload.delete(binding.element);
      this.abortUpload(uploadId, false);
      const componentId = binding.descriptor.componentId || "__root__";
      const set = this.componentBindings.get(componentId);
      if (set) {
        set.delete(uploadId);
        if (set.size === 0) {
          this.componentBindings.delete(componentId);
        }
      }
    }
    resolveInput(node) {
      if (!node) {
        return null;
      }
      if (node instanceof HTMLInputElement) {
        return node;
      }
      if (node instanceof Element) {
        if (node.tagName.toLowerCase() === "input" && node.getAttribute("type") === "file") {
          return node;
        }
        const descendant = node.querySelector('input[type="file"]');
        if (descendant instanceof HTMLInputElement) {
          return descendant;
        }
      }
      return null;
    }
    syncAttributes(element, descriptor) {
      if (Array.isArray(descriptor.accept) && descriptor.accept.length > 0) {
        element.setAttribute("accept", descriptor.accept.join(","));
      } else {
        element.removeAttribute("accept");
      }
      if (descriptor.multiple) {
        element.multiple = true;
      } else {
        element.multiple = false;
        element.removeAttribute("multiple");
      }
    }
    handleInputChange(uploadId, element, descriptor) {
      const files = element.files;
      if (!files || files.length === 0) {
        this.sendUploadMessage({ op: "cancelled", id: uploadId });
        this.abortUpload(uploadId, true);
        return;
      }
      const file = typeof files.item === "function" ? files.item(0) : files[0] ?? null;
      if (!file) {
        this.sendUploadMessage({ op: "cancelled", id: uploadId });
        return;
      }
      if (typeof descriptor.maxSize === "number" && descriptor.maxSize > 0 && file.size > descriptor.maxSize) {
        this.sendUploadMessage({
          op: "error",
          id: uploadId,
          error: `File exceeds maximum size (${descriptor.maxSize} bytes)`
        });
        element.value = "";
        return;
      }
      const meta = { name: file.name, size: file.size, type: file.type };
      this.sendUploadMessage({ op: "change", id: uploadId, meta });
      this.startUpload(uploadId, file, element);
      Logger.debug("[Uploads]", "input change processed", {
        uploadId,
        file: file.name,
        size: file.size
      });
    }
    startUpload(uploadId, file, element) {
      const sid = this.runtime.getSessionId();
      if (!sid) {
        return;
      }
      const base = this.runtime.getUploadEndpoint();
      const target = this.buildUploadURL(base, sid, uploadId);
      this.abortUpload(uploadId, false);
      const xhr = new XMLHttpRequest();
      xhr.upload.onprogress = (event) => {
        const loaded = event.loaded ?? 0;
        const total = event.lengthComputable ? event.total : file.size;
        this.sendUploadMessage({ op: "progress", id: uploadId, loaded, total });
      };
      xhr.onerror = () => {
        this.active.delete(uploadId);
        this.sendUploadMessage({ op: "error", id: uploadId, error: "Upload failed" });
      };
      xhr.onabort = () => {
        this.active.delete(uploadId);
        this.sendUploadMessage({ op: "cancelled", id: uploadId });
      };
      xhr.onload = () => {
        this.active.delete(uploadId);
        if (xhr.status < 200 || xhr.status >= 300) {
          this.sendUploadMessage({
            op: "error",
            id: uploadId,
            error: `Upload failed (${xhr.status})`
          });
        } else {
          this.sendUploadMessage({ op: "progress", id: uploadId, loaded: file.size, total: file.size });
          element.value = "";
        }
      };
      const form = new FormData();
      form.append("file", file);
      xhr.open("POST", target, true);
      xhr.send(form);
      this.active.set(uploadId, { xhr, element });
      Logger.debug("[Uploads]", "upload started", { uploadId, target });
    }
    abortUpload(uploadId, clearInput) {
      const active = this.active.get(uploadId);
      if (!active) {
        return;
      }
      active.xhr.abort();
      if (clearInput) {
        active.element.value = "";
      }
      this.active.delete(uploadId);
      Logger.debug("[Uploads]", "upload aborted", { uploadId, cleared: clearInput });
    }
    sendUploadMessage(payload) {
      if (!payload.id) {
        return;
      }
      this.runtime.sendUploadMessage(payload);
      Logger.debug("[Uploads]", "sent upload message", payload);
    }
    buildUploadURL(base, sid, uploadId) {
      const normalized = (base && base.length > 0 ? base : "/pondlive/upload/").replace(/\/+$/, "");
      return `${normalized}/${encodeURIComponent(sid)}/${encodeURIComponent(uploadId)}`;
    }
  };

  // src/metadata.ts
  var MetadataManager = class {
    constructor() {
      this.metaTags = /* @__PURE__ */ new Map();
      this.linkTags = /* @__PURE__ */ new Map();
      this.scriptTags = /* @__PURE__ */ new Map();
      this.descriptionMeta = null;
      this.indexExistingTags();
    }
    indexExistingTags() {
      if (typeof document === "undefined") {
        return;
      }
      document.querySelectorAll("meta[data-live-key]").forEach((el) => {
        if (el instanceof HTMLMetaElement) {
          const key = el.getAttribute("data-live-key");
          if (key) {
            this.metaTags.set(key, el);
            if (el.name === "description") {
              this.descriptionMeta = el;
            }
          }
        }
      });
      document.querySelectorAll("link[data-live-key]").forEach((el) => {
        if (el instanceof HTMLLinkElement) {
          const key = el.getAttribute("data-live-key");
          if (key) {
            this.linkTags.set(key, el);
          }
        }
      });
      document.querySelectorAll("script[data-live-key]").forEach((el) => {
        if (el instanceof HTMLScriptElement) {
          const key = el.getAttribute("data-live-key");
          if (key) {
            this.scriptTags.set(key, el);
          }
        }
      });
    }
    applyEffect(effect) {
      if (typeof document === "undefined") {
        return;
      }
      Logger.debug("[Metadata]", "applying effect", effect);
      if (effect.title !== void 0) {
        document.title = effect.title;
      }
      if (effect.description !== void 0) {
        this.updateDescription(effect.description);
      } else if (effect.clearDescription) {
        this.clearDescription();
      }
      if (effect.metaRemove) {
        for (const key of effect.metaRemove) {
          this.removeMeta(key);
        }
      }
      if (effect.metaAdd) {
        for (const payload of effect.metaAdd) {
          this.addOrUpdateMeta(payload);
        }
      }
      if (effect.linkRemove) {
        for (const key of effect.linkRemove) {
          this.removeLink(key);
        }
      }
      if (effect.linkAdd) {
        for (const payload of effect.linkAdd) {
          this.addOrUpdateLink(payload);
        }
      }
      if (effect.scriptRemove) {
        for (const key of effect.scriptRemove) {
          this.removeScript(key);
        }
      }
      if (effect.scriptAdd) {
        for (const payload of effect.scriptAdd) {
          this.addOrUpdateScript(payload);
        }
      }
    }
    updateDescription(content) {
      if (!this.descriptionMeta) {
        this.descriptionMeta = document.createElement("meta");
        this.descriptionMeta.name = "description";
        this.descriptionMeta.setAttribute("data-live-managed", "true");
        document.head.appendChild(this.descriptionMeta);
      }
      this.descriptionMeta.content = content;
    }
    clearDescription() {
      if (this.descriptionMeta && this.descriptionMeta.hasAttribute("data-live-managed")) {
        this.descriptionMeta.remove();
        this.descriptionMeta = null;
      }
    }
    addOrUpdateMeta(payload) {
      let el = this.metaTags.get(payload.key);
      if (!el) {
        el = document.createElement("meta");
        el.setAttribute("data-live-key", payload.key);
        document.head.appendChild(el);
        this.metaTags.set(payload.key, el);
      }
      if (payload.name) el.name = payload.name;
      if (payload.content !== void 0) el.content = payload.content;
      if (payload.property) el.setAttribute("property", payload.property);
      if (payload.charset) el.setAttribute("charset", payload.charset);
      if (payload.httpEquiv) el.setAttribute("http-equiv", payload.httpEquiv);
      if (payload.itemProp) el.setAttribute("itemprop", payload.itemProp);
      if (payload.attrs) {
        for (const [key, value] of Object.entries(payload.attrs)) {
          el.setAttribute(key, value);
        }
      }
      if (payload.name === "description") {
        this.descriptionMeta = el;
      }
    }
    removeMeta(key) {
      const el = this.metaTags.get(key);
      if (el) {
        if (el === this.descriptionMeta) {
          this.descriptionMeta = null;
        }
        el.remove();
        this.metaTags.delete(key);
      }
    }
    addOrUpdateLink(payload) {
      let el = this.linkTags.get(payload.key);
      if (!el) {
        el = document.createElement("link");
        el.setAttribute("data-live-key", payload.key);
        document.head.appendChild(el);
        this.linkTags.set(payload.key, el);
      }
      if (payload.rel) el.rel = payload.rel;
      if (payload.href) el.href = payload.href;
      if (payload.type) el.type = payload.type;
      if (payload.as) el.setAttribute("as", payload.as);
      if (payload.media) el.media = payload.media;
      if (payload.hreflang) el.hreflang = payload.hreflang;
      if (payload.title) el.title = payload.title;
      if (payload.crossorigin) el.setAttribute("crossorigin", payload.crossorigin);
      if (payload.integrity) el.integrity = payload.integrity;
      if (payload.referrerpolicy) el.setAttribute("referrerpolicy", payload.referrerpolicy);
      if (payload.sizes) el.setAttribute("sizes", payload.sizes);
      if (payload.attrs) {
        for (const [key, value] of Object.entries(payload.attrs)) {
          el.setAttribute(key, value);
        }
      }
    }
    removeLink(key) {
      const el = this.linkTags.get(key);
      if (el) {
        el.remove();
        this.linkTags.delete(key);
      }
    }
    addOrUpdateScript(payload) {
      const existing = this.scriptTags.get(payload.key);
      if (existing) {
        existing.remove();
        this.scriptTags.delete(payload.key);
      }
      const el = document.createElement("script");
      el.setAttribute("data-live-key", payload.key);
      if (payload.src) el.src = payload.src;
      if (payload.type) el.type = payload.type;
      if (payload.async) el.async = true;
      if (payload.defer) el.defer = true;
      if (payload.module) el.type = "module";
      if (payload.noModule) el.setAttribute("nomodule", "");
      if (payload.crossorigin) el.setAttribute("crossorigin", payload.crossorigin);
      if (payload.integrity) el.integrity = payload.integrity;
      if (payload.referrerpolicy) el.setAttribute("referrerpolicy", payload.referrerpolicy);
      if (payload.nonce) el.nonce = payload.nonce;
      if (payload.inner) el.textContent = payload.inner;
      if (payload.attrs) {
        for (const [key, value] of Object.entries(payload.attrs)) {
          el.setAttribute(key, value);
        }
      }
      document.head.appendChild(el);
      this.scriptTags.set(payload.key, el);
    }
    removeScript(key) {
      const el = this.scriptTags.get(key);
      if (el) {
        el.remove();
        this.scriptTags.delete(key);
      }
    }
    dispose() {
      this.metaTags.forEach((el) => {
        if (el.hasAttribute("data-live-key")) {
          el.remove();
        }
      });
      this.linkTags.forEach((el) => {
        if (el.hasAttribute("data-live-key")) {
          el.remove();
        }
      });
      this.scriptTags.forEach((el) => {
        if (el.hasAttribute("data-live-key")) {
          el.remove();
        }
      });
      this.metaTags.clear();
      this.linkTags.clear();
      this.scriptTags.clear();
      this.descriptionMeta = null;
    }
  };

  // src/hydration.ts
  var HydrationManager = class {
    constructor(runtime, options) {
      this.dom = new DomRegistry();
      this.runtime = runtime;
      this.options = options ?? {};
      this.uploads = new UploadManager(runtime);
      this.refs = new RefRegistry(runtime);
      this.metadata = new MetadataManager();
      this.patcher = new Patcher(this.dom, this.refs, this.uploads);
      this.runtime.on("init", (msg) => this.applyTemplate(msg));
      this.runtime.on("template", (msg) => this.applyTemplate(msg));
      this.runtime.on("frame", (msg) => this.applyFrame(msg));
      this.runtime.on("upload", (msg) => this.uploads.handleControl(msg));
      this.runtime.on("domreq", (msg) => this.handleDOMRequest(msg));
      const boot = this.runtime.getBootPayload();
      if (boot) {
        this.applyTemplate(boot);
      }
    }
    applyTemplate(payload) {
      if (typeof document === "undefined") {
        return;
      }
      if (typeof payload.html !== "string") {
        return;
      }
      const root = this.resolveRoot();
      if (!root) {
        return;
      }
      const componentPaths = payload.componentPaths;
      Logger.debug("[Hydration]", "applying template", {
        slotPaths: payload.slotPaths?.length ?? 0,
        listPaths: payload.listPaths?.length ?? 0,
        componentPaths: componentPaths?.length ?? 0,
        htmlLength: payload.html?.length ?? 0
      });
      if (root instanceof Document) {
        const container = root.body ?? root.documentElement ?? root;
        if (container) {
          container.innerHTML = payload.html;
          pruneWhitespace(container);
        }
        const overrides2 = this.dom.prime(componentPaths, { root });
        this.componentRanges = overrides2;
        this.primeRegistries(payload, container ?? root, overrides2);
        return;
      }
      if (root instanceof ShadowRoot) {
        root.innerHTML = payload.html;
        pruneWhitespace(root);
        const overrides2 = this.dom.prime(componentPaths, { root });
        this.componentRanges = overrides2;
        this.primeRegistries(payload, root, overrides2);
        return;
      }
      root.innerHTML = payload.html;
      pruneWhitespace(root);
      const overrides = this.dom.prime(componentPaths, { root });
      this.componentRanges = overrides;
      this.primeRegistries(payload, root, overrides);
    }
    applyFrame(frame) {
      Logger.debug("[Hydration]", "applying frame", {
        seq: frame.seq,
        ver: frame.ver,
        patchOps: Array.isArray(frame.patch) ? frame.patch.length : 0,
        effects: frame.effects?.length ?? 0
      });
      this.patcher.applyFrame(frame);
      if (frame.bindings?.slots) {
        registerSlotTable(frame.bindings.slots);
      }
      if (frame.bindings?.router) {
        applyRouterBindings(frame.bindings.router, this.componentRanges);
      }
      if (frame.bindings?.refs) {
        this.refs.registerBindings(frame.bindings.refs, this.componentRanges);
      }
      if (frame.bindings?.uploads) {
        this.uploads.registerBindings(frame.bindings.uploads, this.componentRanges);
      }
      if (frame.bindings?.slots === void 0 && frame.bindings?.router) {
      }
      this.refs.apply(frame.refs);
      if (frame.effects) {
        this.applyEffects(frame.effects);
      }
      if (frame.nav) {
        this.applyNavigation(frame.nav);
      }
      Logger.debug("[Hydration]", "frame applied", {
        hasSlots: Boolean(frame.bindings?.slots),
        hasRouter: Boolean(frame.bindings?.router?.length),
        hasRefs: Boolean(frame.bindings?.refs?.length),
        hasUploads: Boolean(frame.bindings?.uploads?.length),
        refDeltaAdd: frame.refs?.add ? Object.keys(frame.refs.add).length : 0,
        refDeltaDel: frame.refs?.del?.length ?? 0,
        effectsApplied: frame.effects?.length ?? 0,
        navApplied: Boolean(frame.nav?.push || frame.nav?.replace)
      });
    }
    applyEffects(effects) {
      for (const effect of effects) {
        if (effect.type === "metadata") {
          this.metadata.applyEffect(effect);
        } else if (effect.type === "dom") {
          this.applyDOMAction(effect);
        } else if (effect.type === "cookies") {
          this.applyCookieEffect(effect);
        }
      }
    }
    applyDOMAction(effect) {
      if (!effect.ref || !effect.kind) {
        return;
      }
      const element = this.refs.get(effect.ref);
      if (!element) {
        Logger.debug("[Hydration]", "DOM action skipped (element not found)", { ref: effect.ref, kind: effect.kind });
        return;
      }
      try {
        const kind = effect.kind;
        if (kind === "dom.call" && effect.method) {
          if (typeof element[effect.method] === "function") {
            const args = Array.isArray(effect.args) ? effect.args : [];
            element[effect.method](...args);
            Logger.debug("[Hydration]", "DOM action call", { ref: effect.ref, method: effect.method });
          }
        } else if (kind === "dom.set" && effect.prop) {
          element[effect.prop] = effect.value;
          Logger.debug("[Hydration]", "DOM action set", { ref: effect.ref, prop: effect.prop });
        } else if (kind === "dom.toggle" && effect.prop) {
          element[effect.prop] = effect.value;
          Logger.debug("[Hydration]", "DOM action toggle", { ref: effect.ref, prop: effect.prop, value: effect.value });
        } else if (kind === "dom.class" && effect.class && element instanceof Element) {
          if (effect.on === true) {
            element.classList.add(effect.class);
          } else {
            element.classList.remove(effect.class);
          }
          Logger.debug("[Hydration]", "DOM action class", { ref: effect.ref, class: effect.class, on: effect.on });
        } else if (kind === "dom.scroll" && element instanceof Element) {
          const options = {};
          if (effect.behavior) options.behavior = effect.behavior;
          if (effect.block) options.block = effect.block;
          if (effect.inline) options.inline = effect.inline;
          element.scrollIntoView(options);
          Logger.debug("[Hydration]", "DOM action scroll", { ref: effect.ref, options });
        }
      } catch (error) {
        Logger.warn("[Hydration]", "DOM action failed", { ref: effect.ref, kind: effect.kind, error });
      }
    }
    applyCookieEffect(effect) {
      if (typeof window === "undefined" || typeof fetch !== "function") {
        return;
      }
      if (!effect.endpoint || !effect.sid || !effect.token) {
        Logger.debug("[Hydration]", "Cookie effect skipped (missing required fields)", effect);
        return;
      }
      const method = effect.method || "POST";
      Logger.debug("[Hydration]", "Applying cookie effect", { endpoint: effect.endpoint, method });
      fetch(effect.endpoint, {
        method,
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          sid: effect.sid,
          token: effect.token
        }),
        credentials: "include"
      }).then((response) => {
        if (response.ok) {
          Logger.debug("[Hydration]", "Cookie effect succeeded", { endpoint: effect.endpoint });
        } else {
          Logger.warn("[Hydration]", "Cookie effect failed", { endpoint: effect.endpoint, status: response.status });
        }
      }).catch((error) => {
        Logger.warn("[Hydration]", "Cookie effect error", { endpoint: effect.endpoint, error });
      });
    }
    applyNavigation(nav) {
      if (typeof window === "undefined" || typeof history === "undefined") {
        return;
      }
      if (!nav.push && !nav.replace && !nav.back) {
        return;
      }
      if (nav.back) {
        try {
          history.back();
          Logger.debug("[Hydration]", "Navigation back");
        } catch (error) {
          Logger.warn("[Hydration]", "Navigation back failed", { error });
        }
        return;
      }
      const url = nav.replace || nav.push;
      if (!url || typeof url !== "string") {
        Logger.debug("[Hydration]", "Navigation skipped (invalid URL)", nav);
        return;
      }
      try {
        if (nav.replace) {
          history.replaceState(null, "", nav.replace);
          Logger.debug("[Hydration]", "Navigation replace", { url: nav.replace });
        } else {
          history.pushState(null, "", nav.push);
          Logger.debug("[Hydration]", "Navigation push", { url: nav.push });
        }
      } catch (error) {
        Logger.warn("[Hydration]", "Navigation failed", { url, error });
      }
    }
    resolveRoot() {
      const root = this.options.root ?? document.body ?? document;
      if (!root) {
        return null;
      }
      if (root instanceof Element || root instanceof Document || root instanceof ShadowRoot) {
        return root;
      }
      return document.body ?? document;
    }
    primeRegistries(payload, root, overrides) {
      if (!root) {
        return;
      }
      this.dom.reset();
      this.refs.clear();
      this.refs.apply(payload.refs);
      if (payload.bindings?.slots) {
        registerSlotTable(payload.bindings.slots);
      } else {
        registerSlotTable(void 0);
      }
      const listRowIndex = buildListRowIndex(payload);
      this.dom.registerSlotAnchors(payload.slotPaths, overrides);
      if (Array.isArray(payload.slots)) {
        this.dom.registerSlots(payload.slots);
      }
      this.dom.registerListContainers(payload.listPaths, overrides, listRowIndex);
      if (payload.bindings?.router) {
        applyRouterBindings(payload.bindings.router, overrides);
      }
      if (payload.bindings?.refs) {
        this.refs.registerBindings(payload.bindings.refs, overrides);
      }
      this.uploads.prime(payload.bindings?.uploads ?? null, overrides);
      Logger.debug("[Hydration]", "registries primed", {
        slots: payload.slotPaths?.length ?? 0,
        lists: payload.listPaths?.length ?? 0,
        routers: payload.bindings?.router?.length ?? 0,
        refs: payload.bindings?.refs?.length ?? 0,
        uploads: payload.bindings?.uploads?.length ?? 0
      });
    }
    handleDOMRequest(msg) {
      if (!msg || !msg.id || !msg.ref) {
        this.runtime.sendDOMResponse({ id: msg?.id ?? "", error: "invalid request" });
        return;
      }
      const element = this.refs.get(msg.ref);
      if (!element) {
        this.runtime.sendDOMResponse({ id: msg.id, error: "element not found" });
        return;
      }
      try {
        let result;
        const values = {};
        if (msg.method && typeof element[msg.method] === "function") {
          const args = Array.isArray(msg.args) ? msg.args : [];
          result = element[msg.method](...args);
        }
        if (Array.isArray(msg.props)) {
          msg.props.forEach((prop) => {
            if (prop) {
              values[prop] = element[prop];
            }
          });
        }
        this.runtime.sendDOMResponse({
          id: msg.id,
          result,
          values: Object.keys(values).length > 0 ? values : void 0
        });
      } catch (error) {
        this.runtime.sendDOMResponse({
          id: msg.id,
          error: error instanceof Error ? error.message : "unknown error"
        });
      }
    }
    getRegistry() {
      return this.dom;
    }
  };
  function pruneWhitespace(root) {
    if (!root || typeof Node === "undefined") {
      return;
    }
    const doc = (root instanceof Document ? root : root.ownerDocument) ?? document;
    if (!doc || typeof doc.createTreeWalker !== "function") {
      return;
    }
    const walker = doc.createTreeWalker(root, NodeFilter.SHOW_TEXT, null);
    const removals = [];
    let current = walker.nextNode();
    while (current) {
      if (current.nodeType === Node.TEXT_NODE) {
        const text = current.textContent ?? "";
        if (!text.trim()) {
          removals.push(current);
        }
      }
      current = walker.nextNode();
    }
    removals.forEach((node) => {
      if (node.parentNode) {
        node.parentNode.removeChild(node);
      }
    });
  }
  function buildListRowIndex(payload) {
    const map = /* @__PURE__ */ new Map();
    if (!payload || !Array.isArray(payload.d)) {
      return map;
    }
    const slots = Array.isArray(payload.slots) ? payload.slots : [];
    payload.d.forEach((slot, index) => {
      if (!slot || slot.kind !== "list" || !Array.isArray(slot.list) || slot.list.length === 0) {
        return;
      }
      const entries = slot.list.map((row) => {
        if (!row || typeof row.key !== "string" || row.key.length === 0) {
          return void 0;
        }
        const count = Math.max(1, Number(row.rootCount) || 1);
        return { key: row.key, count };
      }).filter((entry) => Boolean(entry));
      if (entries.length === 0) {
        return;
      }
      const slotId = slots[index]?.anchorId ?? index;
      map.set(slotId, entries);
    });
    return map;
  }

  // src/event-delegation.ts
  var EventDelegation = class {
    constructor(dom, runtime) {
      this.dom = dom;
      this.runtime = runtime;
      this.handlers = /* @__PURE__ */ new Map();
    }
    setup() {
      if (typeof document === "undefined") {
        return;
      }
      this.registerEvents(["click"]);
      this.registerEvents(getRegisteredSlotEvents());
      this.registerEvents(getRegisteredRefEvents());
      this.stopSlotObserver = observeSlotEvents((event) => this.bind(event));
      this.stopRefObserver = observeRefEvents((event) => this.bind(event));
      Logger.debug("[Delegation]", "event delegation setup complete", {
        handlers: this.handlers.size
      });
    }
    teardown() {
      if (typeof document === "undefined") {
        return;
      }
      if (this.stopSlotObserver) {
        this.stopSlotObserver();
        this.stopSlotObserver = void 0;
      }
      if (this.stopRefObserver) {
        this.stopRefObserver();
        this.stopRefObserver = void 0;
      }
      this.handlers.forEach((listener, event) => {
        document.removeEventListener(event, listener, true);
      });
      this.handlers.clear();
      Logger.debug("[Delegation]", "event delegation torn down");
    }
    registerEvents(events) {
      events.forEach((event) => this.bind(event));
    }
    bind(event) {
      if (this.handlers.has(event) || typeof document === "undefined") {
        return;
      }
      const listener = (e) => this.handleEvent(event, e);
      document.addEventListener(event, listener, true);
      this.handlers.set(event, listener);
      Logger.debug("[Delegation]", "bound event listener", { event });
    }
    handleEvent(event, e) {
      const target = e.target;
      if (!(target instanceof Element)) {
        return;
      }
      const router = getRouterMeta(target);
      if (router && router.path && event === "click") {
        Logger.debug("[Delegation]", "router navigation triggered", {
          path: router.path,
          hash: router.hash
        });
        this.runtime.sendNavigation(router.path, router.query ?? "", router.hash ?? "");
        e.preventDefault();
        return;
      }
      const slotIds = getSlotsForEvent(event);
      for (const slotId of slotIds) {
        const specs = getSlotBindings(slotId) ?? [];
        const node = this.dom.getSlot(slotId);
        if (node instanceof Element && (node === target || node.contains(target))) {
          const binding = specs.find((spec) => spec.event === event);
          if (binding) {
            const detail = extractEventDetail(e, binding.props);
            this.runtime.sendEvent(binding.handler, detail ? { name: event, detail } : { name: event });
            Logger.debug("[Delegation]", "slot event dispatched", {
              slotId,
              handler: binding.handler,
              event
            });
            break;
          }
        }
      }
    }
  };

  // src/index.ts
  var LiveUI = class {
    constructor(options) {
      this.runtime = new LiveRuntime(options);
      this._hydration = new HydrationManager(this.runtime);
      this.events = new EventDelegation(this._hydration.getRegistry(), this.runtime);
      this.events.setup();
    }
    connect() {
      return this.runtime.connect();
    }
    disconnect() {
      this.runtime.disconnect();
      this.events.teardown();
    }
    destroy() {
      this.runtime.destroy();
      this.events.teardown();
    }
    getState() {
      return this.runtime.getState();
    }
    getBootPayload() {
      return this.runtime.getBootPayload();
    }
    on(event, listener) {
      return this.runtime.on(event, listener);
    }
    once(event, listener) {
      return this.runtime.once(event, listener);
    }
    off(event, listener) {
      this.runtime.off(event, listener);
    }
    sendEvent(handlerId, payload, cseq) {
      this.runtime.sendEvent(handlerId, payload, cseq);
    }
    sendNavigation(path, q, hash) {
      this.runtime.sendNavigation(path, q, hash);
    }
    getHydrationManager() {
      return this._hydration;
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
    if (target.__LIVEUI_BOOT__ && typeof target.__LIVEUI_BOOT__ === "object") {
      return target.__LIVEUI_BOOT__;
    }
    if (typeof document === "undefined") {
      return null;
    }
    const script = document.getElementById("live-boot");
    const content = script?.textContent?.trim();
    if (!content) {
      return null;
    }
    try {
      const payload = JSON.parse(content);
      target.__LIVEUI_BOOT__ = payload;
      return payload;
    } catch (error) {
      Logger.error("Failed to parse boot payload", error);
      return null;
    }
  }
  function attachGlobals(target, instance) {
    const LiveUIExport = index_default;
    LiveUIExport.boot = bootClient;
    LiveUIExport.instance = instance;
    target.LiveUI = LiveUIExport;
    target.LiveUIInstance = instance;
    if (target.__LIVEUI_DEVTOOLS__) {
      target.__LIVEUI_DEVTOOLS__.installed = true;
      target.__LIVEUI_DEVTOOLS__.instance = instance;
    }
  }
  function createClient(target) {
    const inlineOptions = { ...target.__LIVEUI_OPTIONS__ ?? {} };
    const bootPayload = detectBootPayload(target);
    const resolvedBoot = inlineOptions.boot ?? bootPayload ?? null;
    if (resolvedBoot) {
      inlineOptions.boot = resolvedBoot;
      target.__LIVEUI_BOOT__ = resolvedBoot;
      if (typeof resolvedBoot.client?.debug === "boolean") {
        inlineOptions.debug = resolvedBoot.client.debug;
      }
    }
    if (typeof inlineOptions.debug === "undefined") {
      inlineOptions.debug = false;
    }
    target.__LIVEUI_OPTIONS__ = inlineOptions;
    const autoConnect = inlineOptions.autoConnect !== false;
    inlineOptions.autoConnect = false;
    const client = new index_default(inlineOptions);
    attachGlobals(target, client);
    Logger.debug("[Entry]", "LiveUI client created", {
      autoConnect,
      debug: inlineOptions.debug,
      hasBoot: Boolean(resolvedBoot)
    });
    if (autoConnect && resolvedBoot) {
      void client.connect().catch((error) => {
        Logger.error("Failed to connect after boot", error);
      });
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
  function bootClient(options = {}) {
    const target = getWindow();
    if (!target) {
      return Promise.reject(new Error("[LiveUI] window is not available in this environment"));
    }
    if (options.force) {
      bootPromise = null;
    }
    if (!bootPromise) {
      bootPromise = scheduleBoot(target);
    }
    return bootPromise;
  }
  if (typeof window !== "undefined") {
    void bootClient().catch((error) => {
      Logger.error("Boot failed", error);
    });
  }
  return __toCommonJS(entry_exports);
})();
//# sourceMappingURL=pondlive-dev.js.map

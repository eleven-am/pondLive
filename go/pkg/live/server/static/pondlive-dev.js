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

  // src/index.ts
  var index_exports = {};
  __export(index_exports, {
    EffectExecutor: () => EffectExecutor,
    Logger: () => Logger,
    Patcher: () => Patcher,
    Router: () => Router,
    Runtime: () => Runtime,
    Transport: () => Transport,
    Uploader: () => Uploader,
    boot: () => boot
  });

  // src/transport.ts
  var import_pondsocket_client = __toESM(require_pondsocket_client(), 1);
  var Transport = class {
    constructor(config) {
      this.handler = null;
      this._sessionId = config.sessionId;
      this.client = new import_pondsocket_client.PondClient(config.endpoint);
      const joinPayload = {
        sid: config.sessionId,
        ver: config.version,
        ack: config.ack,
        loc: config.location
      };
      this.channel = this.client.createChannel(`live/${config.sessionId}`, joinPayload);
      this.channel.join();
      this.channel.onMessage((_event, payload) => {
        this.handler?.(payload);
      });
    }
    get sessionId() {
      return this._sessionId;
    }
    connect() {
      this.client.connect();
    }
    disconnect() {
      this.channel.leave();
      this.client.disconnect();
    }
    send(msg) {
      this.channel.sendMessage(msg.t, msg);
    }
    onMessage(handler) {
      this.handler = handler;
    }
    onStateChange(handler) {
      this.channel.onChannelStateChange(handler);
    }
  };

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
      if (!this.enabled) return;
      if (levels[level] < levels[this.level]) return;
      const prefix = `[LiveUI:${tag}]`;
      const fn = console[level] || console.log;
      if (args.length > 0) {
        fn(prefix, message, ...args);
      } else {
        fn(prefix, message);
      }
    }
  };
  var Logger = new LoggerImpl();

  // src/patcher.ts
  var Patcher = class {
    constructor(root, callbacks) {
      this.handlerStore = /* @__PURE__ */ new WeakMap();
      this.routerStore = /* @__PURE__ */ new WeakMap();
      this.uploadStore = /* @__PURE__ */ new WeakMap();
      this.root = root;
      this.callbacks = callbacks;
    }
    apply(patches) {
      const sorted = [...patches].sort((a, b) => a.seq - b.seq);
      for (const patch of sorted) {
        this.applyPatch(patch);
      }
    }
    applyPatch(patch) {
      const node = this.resolvePath(patch.path);
      if (!node) {
        Logger.warn("Patcher", "Could not resolve path", patch.path);
        return;
      }
      switch (patch.op) {
        case "setText":
          this.setText(node, patch.value);
          break;
        case "setComment":
          this.setComment(node, patch.value);
          break;
        case "setAttr":
          this.setAttr(node, patch.value);
          break;
        case "delAttr":
          this.delAttr(node, patch.name);
          break;
        case "setStyle":
          this.setStyle(node, patch.value);
          break;
        case "delStyle":
          this.delStyle(node, patch.name);
          break;
        case "setStyleDecl":
          this.setStyleDecl(node, patch.selector, patch.name, patch.value);
          break;
        case "delStyleDecl":
          this.delStyleDecl(node, patch.selector, patch.name);
          break;
        case "setHandlers":
          this.setHandlers(node, patch.value);
          break;
        case "setRouter":
          this.setRouter(node, patch.value);
          break;
        case "delRouter":
          this.delRouter(node);
          break;
        case "setUpload":
          this.setUpload(node, patch.value);
          break;
        case "delUpload":
          this.delUpload(node);
          break;
        case "setRef":
          this.callbacks.onRef(patch.value, node);
          break;
        case "delRef":
          this.callbacks.onRefDelete(patch.value);
          break;
        case "replaceNode":
          this.replaceNode(node, patch.value);
          break;
        case "addChild":
          this.addChild(node, patch.index, patch.value);
          break;
        case "delChild":
          this.delChild(node, patch.index);
          break;
        case "moveChild":
          this.moveChild(node, patch.value);
          break;
      }
    }
    resolvePath(path) {
      let node = this.root;
      for (const index of path) {
        if (!node) return null;
        node = node.childNodes[index] ?? null;
      }
      return node;
    }
    setText(node, text) {
      node.textContent = text;
    }
    setComment(node, text) {
      node.textContent = text;
    }
    setAttr(el, attrs) {
      Logger.info("Patcher", "setAttr", el, attrs);
      for (const [name, values] of Object.entries(attrs)) {
        if (name === "class") {
          el.className = values.join(" ");
        } else if (name === "value" && el instanceof HTMLInputElement) {
          el.value = values[0] ?? "";
        } else if (name === "checked" && el instanceof HTMLInputElement) {
          el.checked = values.length > 0 && values[0] !== "false";
        } else if (name === "selected" && el instanceof HTMLOptionElement) {
          el.selected = values.length > 0 && values[0] !== "false";
        } else if (values.length === 0) {
          el.setAttribute(name, "");
        } else {
          el.setAttribute(name, values.join(" "));
        }
      }
    }
    delAttr(el, name) {
      Logger.info("Patcher", "delAttr", el, name);
      if (name === "value" && el instanceof HTMLInputElement) {
        el.value = "";
      } else if (name === "checked" && el instanceof HTMLInputElement) {
        el.checked = false;
      } else if (name === "selected" && el instanceof HTMLOptionElement) {
        el.selected = false;
      } else {
        el.removeAttribute(name);
      }
    }
    setStyle(el, styles) {
      Logger.info("Patcher", "setStyle", el, styles);
      for (const [prop, value] of Object.entries(styles)) {
        el.style.setProperty(prop, value);
      }
    }
    delStyle(el, prop) {
      Logger.info("Patcher", "delStyle", el, prop);
      el.style.removeProperty(prop);
    }
    setStyleDecl(styleEl, selector, prop, value) {
      Logger.info("Patcher", "setStyleDecl", styleEl, selector, prop, value);
      const sheet = styleEl.sheet;
      if (!sheet) return;
      const rule = this.findOrCreateRule(sheet, selector);
      if (rule) {
        rule.style.setProperty(prop, value);
      }
    }
    delStyleDecl(styleEl, selector, prop) {
      Logger.info("Patcher", "delStyleDecl", styleEl, selector, prop);
      const sheet = styleEl.sheet;
      if (!sheet) return;
      const rule = this.findRule(sheet, selector);
      if (rule) {
        rule.style.removeProperty(prop);
      }
    }
    findRule(sheet, selector) {
      for (let i = 0; i < sheet.cssRules.length; i++) {
        const rule = sheet.cssRules[i];
        if (rule instanceof CSSStyleRule && rule.selectorText === selector) {
          return rule;
        }
      }
      return null;
    }
    findOrCreateRule(sheet, selector) {
      let rule = this.findRule(sheet, selector);
      if (!rule) {
        const index = sheet.insertRule(`${selector} {}`, sheet.cssRules.length);
        rule = sheet.cssRules[index];
      }
      return rule;
    }
    setHandlers(el, handlers) {
      Logger.info("Patcher", "setHandlers", el, handlers);
      const oldHandlers = this.handlerStore.get(el);
      if (oldHandlers) {
        for (const [event, listener] of oldHandlers) {
          el.removeEventListener(event, listener);
        }
      }
      const newHandlers = /* @__PURE__ */ new Map();
      for (const meta of handlers) {
        const listen = meta.listen ?? [];
        const listener = (e) => {
          if (!listen.includes("allowDefault") && e.cancelable) {
            e.preventDefault();
          }
          const data = this.extractEventData(e, meta.props ?? []);
          this.callbacks.onEvent(meta.event, meta.handler, data);
          if (!listen.includes("bubble")) {
            e.stopPropagation();
          }
        };
        el.addEventListener(meta.event, listener);
        newHandlers.set(meta.event, listener);
      }
      this.handlerStore.set(el, newHandlers);
    }
    extractEventData(e, props, el) {
      Logger.info("Patcher", "extractEventData", e, props, el);
      return Object.fromEntries(props.map((prop) => [prop, this.resolveProp(e, prop, el)]).filter(([_, value]) => value !== void 0));
    }
    resolveProp(e, path, el) {
      const segments = path.split(".").map((s) => s.trim()).filter(Boolean);
      if (segments.length === 0) return void 0;
      const root = segments.shift();
      let current;
      switch (root) {
        case "event":
          current = e;
          break;
        case "target":
          current = e.target;
          break;
        case "currentTarget":
          current = e.currentTarget;
          break;
        case "element":
        case "ref":
          current = el ?? (e.currentTarget instanceof Element ? e.currentTarget : null);
          break;
        default:
          current = e[root];
      }
      for (const segment of segments) {
        if (current == null) return void 0;
        try {
          current = current[segment];
        } catch {
          return void 0;
        }
      }
      return this.serializeValue(current);
    }
    serializeValue(value) {
      if (value === null || value === void 0) return null;
      const type = typeof value;
      if (type === "string" || type === "number" || type === "boolean") return value;
      if (Array.isArray(value)) {
        const mapped = value.map((v) => this.serializeValue(v)).filter((v) => v !== void 0);
        return mapped.length > 0 ? mapped : null;
      }
      if (value instanceof Date) return value.toISOString();
      if (value instanceof DOMTokenList) return Array.from(value);
      if (value instanceof Node) return void 0;
      try {
        return JSON.parse(JSON.stringify(value));
      } catch {
        return void 0;
      }
    }
    setRouter(el, meta) {
      Logger.info("Patcher", "setRouter", el, meta);
      this.delRouter(el);
      const listener = (e) => {
        e.preventDefault();
        this.callbacks.onRouter(meta);
      };
      el.addEventListener("click", listener);
      this.routerStore.set(el, listener);
    }
    delRouter(el) {
      Logger.info("Patcher", "delRouter", el);
      const listener = this.routerStore.get(el);
      if (listener) {
        el.removeEventListener("click", listener);
        this.routerStore.delete(el);
      }
    }
    setUpload(el, meta) {
      Logger.info("Patcher", "setUpload", el, meta);
      this.delUpload(el);
      if (meta.multiple) {
        el.multiple = true;
      }
      if (meta.accept && meta.accept.length > 0) {
        el.accept = meta.accept.join(",");
      }
      const listener = () => {
        if (el.files && el.files.length > 0) {
          this.callbacks.onUpload(meta, el.files);
        }
      };
      el.addEventListener("change", listener);
      this.uploadStore.set(el, listener);
    }
    delUpload(el) {
      Logger.info("Patcher", "delUpload", el);
      const listener = this.uploadStore.get(el);
      if (listener) {
        el.removeEventListener("change", listener);
        this.uploadStore.delete(el);
      }
      el.multiple = false;
      el.accept = "";
    }
    replaceNode(oldNode, newNodeData) {
      Logger.info("Patcher", "replaceNode", oldNode, newNodeData);
      const newNode = this.createNode(newNodeData);
      if (newNode && oldNode.parentNode) {
        oldNode.parentNode.replaceChild(newNode, oldNode);
      }
    }
    addChild(parent, index, nodeData) {
      Logger.info("Patcher", "addChild", parent, index, nodeData);
      const newNode = this.createNode(nodeData);
      if (!newNode) return;
      const refChild = parent.childNodes[index] ?? null;
      parent.insertBefore(newNode, refChild);
    }
    delChild(parent, index) {
      Logger.info("Patcher", "delChild", parent, index);
      const child = parent.childNodes[index];
      if (child) {
        parent.removeChild(child);
      }
    }
    moveChild(parent, move) {
      Logger.info("Patcher", "moveChild", parent, move);
      const child = parent.childNodes[move.fromIndex];
      if (!child) return;
      parent.removeChild(child);
      const refChild = parent.childNodes[move.newIdx] ?? null;
      parent.insertBefore(child, refChild);
    }
    createNode(data) {
      Logger.info("Patcher", "createNode", data);
      if (data.text !== void 0) {
        return document.createTextNode(data.text);
      }
      if (data.comment !== void 0) {
        return document.createComment(data.comment);
      }
      if (!data.tag) return null;
      const el = document.createElement(data.tag);
      if (data.attrs) {
        this.setAttr(el, data.attrs);
      }
      if (data.style) {
        this.setStyle(el, data.style);
      }
      if (data.handlers && data.handlers.length > 0) {
        this.setHandlers(el, data.handlers);
      }
      if (data.router) {
        this.setRouter(el, data.router);
      }
      if (data.upload && el instanceof HTMLInputElement) {
        this.setUpload(el, data.upload);
      }
      if (data.refId) {
        this.callbacks.onRef(data.refId, el);
      }
      if (data.unsafeHTML) {
        el.innerHTML = data.unsafeHTML;
      } else if (data.children) {
        for (const child of data.children) {
          const childNode = this.createNode(child);
          if (childNode) {
            el.appendChild(childNode);
          }
        }
      }
      return el;
    }
  };

  // src/router.ts
  var Router = class {
    constructor(onNav) {
      this.onNav = onNav;
      window.addEventListener("popstate", () => this.handlePopState());
    }
    navigate(meta) {
      const path = meta.pathValue ?? window.location.pathname;
      const query = meta.query !== void 0 ? meta.query : window.location.search;
      const hash = meta.hash !== void 0 ? meta.hash : window.location.hash;
      const cleanQuery = query.startsWith("?") ? query.substring(1) : query;
      const cleanHash = hash.startsWith("#") ? hash.substring(1) : hash;
      const url = path + (cleanQuery ? "?" + cleanQuery : "") + (cleanHash ? "#" + cleanHash : "");
      if (meta.replace) {
        window.history.replaceState({}, "", url);
      } else {
        window.history.pushState({}, "", url);
      }
      this.onNav("nav", path, cleanQuery, cleanHash);
    }
    handlePopState() {
      const path = window.location.pathname;
      const query = window.location.search;
      const hash = window.location.hash;
      const cleanQuery = query.startsWith("?") ? query.substring(1) : query;
      const cleanHash = hash.startsWith("#") ? hash.substring(1) : hash;
      this.onNav("pop", path, cleanQuery, cleanHash);
    }
    destroy() {
      window.removeEventListener("popstate", () => this.handlePopState());
    }
  };

  // src/uploader.ts
  var Uploader = class {
    constructor(config) {
      this.active = /* @__PURE__ */ new Map();
      this.endpoint = config.endpoint.replace(/\/+$/, "");
      this.sessionId = config.sessionId;
      this.onMessage = config.onMessage;
    }
    upload(meta, files, input) {
      const uploadId = meta.uploadId;
      if (files.length === 0) {
        this.send({ t: "upload", op: "cancelled", id: uploadId });
        return;
      }
      const file = files[0];
      if (meta.maxSize && meta.maxSize > 0 && file.size > meta.maxSize) {
        this.send({
          t: "upload",
          op: "error",
          id: uploadId,
          error: `File exceeds maximum size (${meta.maxSize} bytes)`
        });
        if (input) input.value = "";
        return;
      }
      this.send({
        t: "upload",
        op: "change",
        id: uploadId,
        meta: { name: file.name, size: file.size, type: file.type }
      });
      this.startUpload(uploadId, file, input ?? null);
    }
    cancel(uploadId) {
      const active = this.active.get(uploadId);
      if (active) {
        active.xhr.abort();
        if (active.input) active.input.value = "";
        this.active.delete(uploadId);
      }
    }
    startUpload(uploadId, file, input) {
      this.cancel(uploadId);
      const target = `${this.endpoint}/${encodeURIComponent(this.sessionId)}/${encodeURIComponent(uploadId)}`;
      const xhr = new XMLHttpRequest();
      xhr.upload.onprogress = (event) => {
        const loaded = event.loaded;
        const total = event.lengthComputable ? event.total : file.size;
        this.send({ t: "upload", op: "progress", id: uploadId, loaded, total });
      };
      xhr.onerror = () => {
        this.active.delete(uploadId);
        this.send({ t: "upload", op: "error", id: uploadId, error: "Upload failed" });
      };
      xhr.onabort = () => {
        this.active.delete(uploadId);
        this.send({ t: "upload", op: "cancelled", id: uploadId });
      };
      xhr.onload = () => {
        this.active.delete(uploadId);
        if (xhr.status < 200 || xhr.status >= 300) {
          this.send({ t: "upload", op: "error", id: uploadId, error: `Upload failed (${xhr.status})` });
        } else {
          this.send({ t: "upload", op: "progress", id: uploadId, loaded: file.size, total: file.size });
          if (input) input.value = "";
        }
      };
      const form = new FormData();
      form.append("file", file);
      xhr.open("POST", target, true);
      xhr.send(form);
      this.active.set(uploadId, { xhr, input });
    }
    send(msg) {
      this.onMessage(msg);
    }
  };

  // src/effects.ts
  var EffectExecutor = class {
    constructor(config) {
      this.sessionId = config.sessionId;
      this.resolveRef = config.resolveRef;
      this.onDOMResponse = config.onDOMResponse;
    }
    execute(effects) {
      if (!effects || effects.length === 0) return;
      for (const effect of effects) {
        this.executeOne(effect);
      }
    }
    handleDOMRequest(req) {
      const el = this.resolveRef(req.ref);
      if (!el) {
        this.sendDOMResponse(req.id, void 0, void 0, `ref not found: ${req.ref}`);
        return;
      }
      try {
        if (req.props && req.props.length > 0) {
          const values = this.readProps(el, req.props);
          this.sendDOMResponse(req.id, values, void 0, void 0);
        } else if (req.method) {
          const result = this.callMethod(el, req.method, req.args ?? []);
          this.sendDOMResponse(req.id, void 0, result, void 0);
        } else {
          this.sendDOMResponse(req.id, void 0, void 0, "no props or method specified");
        }
      } catch (e) {
        const error = e instanceof Error ? e.message : String(e);
        this.sendDOMResponse(req.id, void 0, void 0, error);
      }
    }
    executeOne(effect) {
      switch (effect.type) {
        case "dom":
          this.executeDOMAction(effect);
          break;
        case "cookies":
          this.executeCookieSync(effect);
          break;
      }
    }
    executeDOMAction(effect) {
      const el = this.resolveRef(effect.ref);
      if (!el) {
        Logger.warn("Effects", "Ref not found", effect.ref);
        return;
      }
      try {
        switch (effect.kind) {
          case "dom.call":
            this.executeCall(el, effect);
            break;
          case "dom.set":
            this.executeSet(el, effect);
            break;
          case "dom.toggle":
            this.executeToggle(el, effect);
            break;
          case "dom.class":
            this.executeClass(el, effect);
            break;
          case "dom.scroll":
            this.executeScroll(el, effect);
            break;
          default:
            Logger.warn("Effects", "Unknown kind", effect.kind);
        }
      } catch (e) {
        Logger.error("Effects", "Execution failed", e);
      }
    }
    executeCall(el, effect) {
      if (!effect.method) return;
      const method = el[effect.method];
      if (typeof method === "function") {
        method.apply(el, effect.args ?? []);
      } else {
        Logger.warn("Effects", "Method not found", effect.method);
      }
    }
    executeSet(el, effect) {
      if (!effect.prop) return;
      el[effect.prop] = effect.value;
    }
    executeToggle(el, effect) {
      if (!effect.prop) return;
      const current = el[effect.prop];
      el[effect.prop] = !current;
    }
    executeClass(el, effect) {
      if (!effect.class) return;
      if (effect.on === true) {
        el.classList.add(effect.class);
      } else if (effect.on === false) {
        el.classList.remove(effect.class);
      } else {
        el.classList.toggle(effect.class);
      }
    }
    executeScroll(el, effect) {
      if (!el.scrollIntoView) return;
      const opts = {};
      if (effect.behavior) opts.behavior = effect.behavior;
      if (effect.block) opts.block = effect.block;
      if (effect.inline) opts.inline = effect.inline;
      el.scrollIntoView(opts);
    }
    executeCookieSync(effect) {
      const url = `${effect.endpoint}?sid=${encodeURIComponent(effect.sid)}&token=${encodeURIComponent(effect.token)}`;
      fetch(url, {
        method: effect.method ?? "GET",
        credentials: "include"
      }).catch((e) => {
        Logger.error("Effects", "Cookie sync failed", e);
      });
    }
    readProps(el, props) {
      const values = {};
      for (const prop of props) {
        values[prop] = this.resolveProp(el, prop);
      }
      return values;
    }
    resolveProp(el, path) {
      const segments = path.split(".");
      let current = el;
      for (const segment of segments) {
        if (current == null) return void 0;
        current = current[segment];
      }
      return this.serializeValue(current);
    }
    callMethod(el, method, args) {
      const fn = el[method];
      if (typeof fn !== "function") {
        throw new Error(`method not found: ${method}`);
      }
      const result = fn.apply(el, args);
      return this.serializeValue(result);
    }
    serializeValue(value) {
      if (value === null || value === void 0) return null;
      const type = typeof value;
      if (type === "string" || type === "number" || type === "boolean") return value;
      if (Array.isArray(value)) {
        return value.map((v) => this.serializeValue(v));
      }
      if (value instanceof Date) return value.toISOString();
      if (value instanceof DOMTokenList) return Array.from(value);
      if (value instanceof Node) return void 0;
      try {
        return JSON.parse(JSON.stringify(value));
      } catch {
        return void 0;
      }
    }
    sendDOMResponse(id, values, result, error) {
      const response = {
        t: "dom_res",
        sid: this.sessionId,
        id
      };
      if (values !== void 0) response.values = values;
      if (result !== void 0) response.result = result;
      if (error !== void 0) response.error = error;
      this.onDOMResponse(response);
    }
  };

  // src/runtime.ts
  var import_pondsocket_client2 = __toESM(require_pondsocket_client(), 1);
  var Runtime = class {
    constructor(config) {
      this.connectedState = true;
      this.refs = /* @__PURE__ */ new Map();
      this.cseq = 0;
      this.sessionId = config.sessionId;
      Logger.configure({ enabled: config.debug ?? false, level: "debug" });
      const resolveRef = (refId) => this.refs.get(refId);
      this.transport = new Transport({
        endpoint: config.endpoint,
        sessionId: config.sessionId,
        version: config.version,
        ack: config.seq,
        location: config.location
      });
      this.patcher = new Patcher(config.root, {
        onEvent: (_event, handler, data) => this.handleEvent(handler, data),
        onRef: (refId, el) => this.refs.set(refId, el),
        onRefDelete: (refId) => this.refs.delete(refId),
        onRouter: (meta) => this.handleRouterClick(meta),
        onUpload: (meta, files) => this.handleUpload(meta, files)
      });
      this.router = new Router((type, path, query, hash) => {
        this.sendNav(type, path, query, hash);
      });
      this.uploader = new Uploader({
        endpoint: config.uploadEndpoint,
        sessionId: config.sessionId,
        onMessage: (msg) => this.transport.send(msg)
      });
      this.effects = new EffectExecutor({
        sessionId: config.sessionId,
        resolveRef,
        onDOMResponse: (res) => this.transport.send(res)
      });
      this.transport.onMessage((msg) => this.handleMessage(msg));
      this.transport.onStateChange((state) => this.handleStateChange(state));
    }
    connect() {
      this.transport.connect();
      Logger.info("Runtime", "Connected");
    }
    disconnect() {
      this.transport.disconnect();
      Logger.info("Runtime", "Disconnected");
    }
    connected() {
      return this.connectedState;
    }
    handleMessage(msg) {
      Logger.debug("Runtime", "Received", msg.t);
      switch (msg.t) {
        case "boot":
          this.handleBoot(msg);
          break;
        case "init":
          this.handleInit(msg);
          break;
        case "frame":
          this.handleFrame(msg);
          break;
        case "resume_ok":
          this.handleResumeOK(msg);
          break;
        case "dom_req":
          this.handleDOMRequest(msg);
          break;
        case "evt_ack":
          break;
        case "error":
          Logger.error("Runtime", "Server error", msg.code, msg.message);
          break;
        case "diagnostic":
          Logger.warn("Runtime", "Diagnostic", msg.code, msg.message);
          break;
      }
    }
    handleBoot(boot2) {
      Logger.info("Runtime", "Boot received", { ver: boot2.ver, seq: boot2.seq, patches: boot2.patch?.length ?? 0 });
      if (boot2.patch && boot2.patch.length > 0) {
        this.patcher.apply(boot2.patch);
      }
      this.sendAck(boot2.seq);
    }
    handleInit(init) {
      Logger.info("Runtime", "Init received", { ver: init.ver, seq: init.seq });
      this.sendAck(init.seq);
    }
    handleFrame(frame) {
      Logger.debug("Runtime", "Frame", { seq: frame.seq, ops: frame.patch?.length ?? 0 });
      if (frame.patch && frame.patch.length > 0) {
        this.patcher.apply(frame.patch);
      }
      if (frame.effects && frame.effects.length > 0) {
        this.effects.execute(frame.effects);
      }
      if (frame.nav) {
        this.handleServerNav(frame.nav);
      }
      this.sendAck(frame.seq);
    }
    handleResumeOK(resume) {
      Logger.info("Runtime", "Resume OK", { from: resume.from, to: resume.to });
    }
    handleDOMRequest(req) {
      this.effects.handleDOMRequest(req);
    }
    handleServerNav(nav) {
      if (nav.push) {
        window.history.pushState({}, "", nav.push);
      } else if (nav.replace) {
        window.history.replaceState({}, "", nav.replace);
      } else if (nav.back) {
        window.history.back();
      }
    }
    handleEvent(handler, data) {
      const event = {
        t: "evt",
        sid: this.sessionId,
        hid: handler,
        cseq: ++this.cseq,
        payload: data ?? {}
      };
      this.transport.send(event);
      Logger.debug("Runtime", "Event sent", handler);
    }
    handleRouterClick(meta) {
      this.router.navigate(meta);
    }
    handleUpload(meta, files) {
      this.uploader.upload(meta, files);
    }
    sendNav(type, path, query, hash) {
      const msg = { t: type, sid: this.sessionId, path, q: query, hash };
      this.transport.send(msg);
      Logger.debug("Runtime", "Nav sent", type, path);
    }
    sendAck(seq) {
      this.transport.send({ t: "ack", sid: this.sessionId, seq });
    }
    handleStateChange(state) {
      Logger.debug("Runtime", "Channel state", state);
      if (state === import_pondsocket_client2.ChannelState.STALLED || state === import_pondsocket_client2.ChannelState.CLOSED) {
        this.connectedState = false;
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
      } catch (e) {
        Logger.error("Runtime", "Failed to parse boot payload", e);
      }
    }
    if (!bootData) {
      bootData = window.__LIVEUI_BOOT__ ?? null;
    }
    if (!bootData) {
      Logger.error("Runtime", "No boot payload found");
      return null;
    }
    const config = {
      root: document.documentElement,
      sessionId: bootData.sid,
      version: bootData.ver,
      seq: bootData.seq,
      endpoint: bootData.client?.endpoint ?? "/live",
      uploadEndpoint: bootData.client?.upload ?? "/pondlive/upload",
      location: bootData.location,
      debug: bootData.client?.debug
    };
    const runtime = new Runtime(config);
    runtime.handleBoot(bootData);
    runtime.connect();
    return runtime;
  }

  // src/index.ts
  if (typeof window !== "undefined") {
    window.addEventListener("DOMContentLoaded", () => {
      const runtime = boot();
      if (runtime) {
        window.__LIVEUI__ = runtime;
      }
    });
  }
  return __toCommonJS(index_exports);
})();
//# sourceMappingURL=pondlive-dev.js.map

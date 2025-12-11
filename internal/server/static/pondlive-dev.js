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
        ChannelState3["DECLINED"] = "DECLINED";
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
        Events2["UNAUTHORIZED"] = "UNAUTHORIZED";
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

  // node_modules/@eleven-am/pondsocket-client/types.js
  var require_types2 = __commonJS({
    "node_modules/@eleven-am/pondsocket-client/types.js"(exports) {
      "use strict";
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.ConnectionState = void 0;
      var ConnectionState3;
      (function(ConnectionState4) {
        ConnectionState4["DISCONNECTED"] = "disconnected";
        ConnectionState4["CONNECTING"] = "connecting";
        ConnectionState4["CONNECTED"] = "connected";
      })(ConnectionState3 || (exports.ConnectionState = ConnectionState3 = {}));
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
      var types_1 = require_types2();
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
         * @desc Marks the channel join as declined by the server.
         * @param payload - The decline payload containing status code and message.
         */
        decline(payload) {
          __classPrivateFieldGet(this, _Channel_joinState, "f").publish(pondsocket_common_1.ChannelState.DECLINED);
          __classPrivateFieldSet(this, _Channel_queue, [], "f");
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
          if (__classPrivateFieldGet(this, _Channel_clientState, "f").value === types_1.ConnectionState.CONNECTED) {
            __classPrivateFieldGet(this, _Channel_publisher, "f").call(this, message);
          } else {
            const unsubscribe = __classPrivateFieldGet(this, _Channel_clientState, "f").subscribe((state) => {
              if (state === types_1.ConnectionState.CONNECTED) {
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
          if (state === types_1.ConnectionState.CONNECTED && __classPrivateFieldGet(this, _Channel_joinState, "f").value === pondsocket_common_1.ChannelState.STALLED) {
            const message = {
              action: pondsocket_common_1.ClientActions.JOIN_CHANNEL,
              event: pondsocket_common_1.ClientActions.JOIN_CHANNEL,
              payload: __classPrivateFieldGet(this, _Channel_joinParams, "f"),
              channelName: __classPrivateFieldGet(this, _Channel_name, "f"),
              requestId: (0, pondsocket_common_1.uuid)()
            };
            __classPrivateFieldGet(this, _Channel_publisher, "f").call(this, message);
          } else if (state !== types_1.ConnectionState.CONNECTED && __classPrivateFieldGet(this, _Channel_joinState, "f").value === pondsocket_common_1.ChannelState.JOINED) {
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
      var _PondClient_clearTimeouts;
      var _PondClient_createPublisher;
      var _PondClient_handleAcknowledge;
      var _PondClient_handleUnauthorized;
      var _PondClient_init;
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.PondClient = void 0;
      var pondsocket_common_1 = require_pondsocket_common();
      var channel_1 = require_channel();
      var types_1 = require_types2();
      var DEFAULT_CONNECTION_TIMEOUT = 1e4;
      var DEFAULT_MAX_RECONNECT_DELAY = 3e4;
      var PondClient2 = class {
        constructor(endpoint, params = {}, options = {}) {
          var _a, _b;
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
          this._reconnectAttempts = 0;
          const query = new URLSearchParams(params);
          address.search = query.toString();
          const protocol = address.protocol === "https:" ? "wss:" : "ws:";
          if (address.protocol !== "wss:" && address.protocol !== "ws:") {
            address.protocol = protocol;
          }
          this._address = address;
          this._options = {
            connectionTimeout: (_a = options.connectionTimeout) !== null && _a !== void 0 ? _a : DEFAULT_CONNECTION_TIMEOUT,
            maxReconnectDelay: (_b = options.maxReconnectDelay) !== null && _b !== void 0 ? _b : DEFAULT_MAX_RECONNECT_DELAY,
            pingInterval: options.pingInterval
          };
          __classPrivateFieldSet(this, _PondClient_channels, /* @__PURE__ */ new Map(), "f");
          this._broadcaster = new pondsocket_common_1.Subject();
          this._connectionState = new pondsocket_common_1.BehaviorSubject(types_1.ConnectionState.DISCONNECTED);
          this._errorSubject = new pondsocket_common_1.Subject();
          __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_init).call(this);
        }
        /**
         * @desc Connects to the server and returns the socket.
         */
        connect() {
          this._disconnecting = false;
          this._connectionState.publish(types_1.ConnectionState.CONNECTING);
          const socket = new WebSocket(this._address.toString());
          this._connectionTimeoutId = setTimeout(() => {
            if (socket.readyState === WebSocket.CONNECTING) {
              const error = new Error("Connection timeout");
              this._errorSubject.publish(error);
              socket.close();
            }
          }, this._options.connectionTimeout);
          socket.onopen = () => {
            __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_clearTimeouts).call(this);
            this._reconnectAttempts = 0;
            if (this._options.pingInterval) {
              this._pingIntervalId = setInterval(() => {
                if (socket.readyState === WebSocket.OPEN) {
                  socket.send(JSON.stringify({ action: "ping" }));
                }
              }, this._options.pingInterval);
            }
          };
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
          socket.onerror = (event) => {
            const error = new Error("WebSocket error");
            this._errorSubject.publish(error);
            socket.close();
          };
          socket.onclose = () => {
            __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_clearTimeouts).call(this);
            this._connectionState.publish(types_1.ConnectionState.DISCONNECTED);
            if (this._disconnecting) {
              return;
            }
            const delay = Math.min(1e3 * Math.pow(2, this._reconnectAttempts), this._options.maxReconnectDelay);
            this._reconnectAttempts++;
            setTimeout(() => {
              this.connect();
            }, delay);
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
          __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_clearTimeouts).call(this);
          this._disconnecting = true;
          this._connectionState.publish(types_1.ConnectionState.DISCONNECTED);
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
        /**
         * @desc Subscribes to connection errors.
         * @param callback - The callback to call when an error occurs.
         */
        onError(callback) {
          return this._errorSubject.subscribe(callback);
        }
      };
      exports.PondClient = PondClient2;
      _PondClient_channels = /* @__PURE__ */ new WeakMap(), _PondClient_instances = /* @__PURE__ */ new WeakSet(), _PondClient_clearTimeouts = function _PondClient_clearTimeouts2() {
        if (this._connectionTimeoutId) {
          clearTimeout(this._connectionTimeoutId);
          this._connectionTimeoutId = void 0;
        }
        if (this._pingIntervalId) {
          clearInterval(this._pingIntervalId);
          this._pingIntervalId = void 0;
        }
      }, _PondClient_createPublisher = function _PondClient_createPublisher2() {
        return (message) => {
          if (this._connectionState.value === types_1.ConnectionState.CONNECTED) {
            this._socket.send(JSON.stringify(message));
          }
        };
      }, _PondClient_handleAcknowledge = function _PondClient_handleAcknowledge2(message) {
        var _a;
        const channel = (_a = __classPrivateFieldGet(this, _PondClient_channels, "f").get(message.channelName)) !== null && _a !== void 0 ? _a : new channel_1.Channel(__classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_createPublisher).call(this), this._connectionState, message.channelName, {});
        __classPrivateFieldGet(this, _PondClient_channels, "f").set(message.channelName, channel);
        channel.acknowledge(this._broadcaster);
      }, _PondClient_handleUnauthorized = function _PondClient_handleUnauthorized2(message) {
        const channel = __classPrivateFieldGet(this, _PondClient_channels, "f").get(message.channelName);
        if (channel) {
          const payload = message.payload;
          channel.decline(payload);
        }
      }, _PondClient_init = function _PondClient_init2() {
        this._broadcaster.subscribe((message) => {
          if (message.event === pondsocket_common_1.Events.ACKNOWLEDGE) {
            __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_handleAcknowledge).call(this, message);
          } else if (message.event === pondsocket_common_1.Events.UNAUTHORIZED) {
            __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_handleUnauthorized).call(this, message);
          } else if (message.event === pondsocket_common_1.Events.CONNECTION && message.action === pondsocket_common_1.ServerActions.CONNECT) {
            this._connectionState.publish(types_1.ConnectionState.CONNECTED);
          }
        });
      };
    }
  });

  // node_modules/@eleven-am/pondsocket-client/browser/sseClient.js
  var require_sseClient = __commonJS({
    "node_modules/@eleven-am/pondsocket-client/browser/sseClient.js"(exports) {
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
      var _SSEClient_instances;
      var _SSEClient_channels;
      var _SSEClient_handleMessage;
      var _SSEClient_clearTimeout;
      var _SSEClient_createPublisher;
      var _SSEClient_handleAcknowledge;
      var _SSEClient_handleUnauthorized;
      var _SSEClient_init;
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.SSEClient = void 0;
      var pondsocket_common_1 = require_pondsocket_common();
      var channel_1 = require_channel();
      var types_1 = require_types2();
      var DEFAULT_CONNECTION_TIMEOUT = 1e4;
      var DEFAULT_MAX_RECONNECT_DELAY = 3e4;
      var SSEClient = class {
        constructor(endpoint, params = {}, options = {}) {
          var _a, _b;
          _SSEClient_instances.add(this);
          _SSEClient_channels.set(this, void 0);
          let address;
          try {
            address = new URL(endpoint);
          } catch (e) {
            address = new URL(window.location.toString());
            address.pathname = endpoint;
          }
          this._disconnecting = false;
          this._reconnectAttempts = 0;
          const query = new URLSearchParams(params);
          address.search = query.toString();
          if (address.protocol !== "https:" && address.protocol !== "http:") {
            address.protocol = window.location.protocol;
          }
          this._address = address;
          this._postAddress = new URL(address.toString());
          this._options = {
            connectionTimeout: (_a = options.connectionTimeout) !== null && _a !== void 0 ? _a : DEFAULT_CONNECTION_TIMEOUT,
            maxReconnectDelay: (_b = options.maxReconnectDelay) !== null && _b !== void 0 ? _b : DEFAULT_MAX_RECONNECT_DELAY,
            pingInterval: options.pingInterval,
            withCredentials: options.withCredentials
          };
          __classPrivateFieldSet(this, _SSEClient_channels, /* @__PURE__ */ new Map(), "f");
          this._broadcaster = new pondsocket_common_1.Subject();
          this._connectionState = new pondsocket_common_1.BehaviorSubject(types_1.ConnectionState.DISCONNECTED);
          this._errorSubject = new pondsocket_common_1.Subject();
          __classPrivateFieldGet(this, _SSEClient_instances, "m", _SSEClient_init).call(this);
        }
        connect() {
          var _a;
          this._disconnecting = false;
          this._connectionState.publish(types_1.ConnectionState.CONNECTING);
          const eventSource = new EventSource(this._address.toString(), {
            withCredentials: (_a = this._options.withCredentials) !== null && _a !== void 0 ? _a : false
          });
          this._connectionTimeoutId = setTimeout(() => {
            if (eventSource.readyState === EventSource.CONNECTING) {
              const error = new Error("Connection timeout");
              this._errorSubject.publish(error);
              eventSource.close();
            }
          }, this._options.connectionTimeout);
          eventSource.onopen = () => {
            __classPrivateFieldGet(this, _SSEClient_instances, "m", _SSEClient_clearTimeout).call(this);
            this._reconnectAttempts = 0;
          };
          eventSource.onmessage = (event) => {
            __classPrivateFieldGet(this, _SSEClient_instances, "m", _SSEClient_handleMessage).call(this, event.data);
          };
          eventSource.onerror = () => {
            const error = new Error("SSE connection error");
            this._errorSubject.publish(error);
            eventSource.close();
            __classPrivateFieldGet(this, _SSEClient_instances, "m", _SSEClient_clearTimeout).call(this);
            this._connectionState.publish(types_1.ConnectionState.DISCONNECTED);
            if (this._disconnecting) {
              return;
            }
            const delay = Math.min(1e3 * Math.pow(2, this._reconnectAttempts), this._options.maxReconnectDelay);
            this._reconnectAttempts++;
            setTimeout(() => {
              this.connect();
            }, delay);
          };
          this._eventSource = eventSource;
        }
        getState() {
          return this._connectionState.value;
        }
        getConnectionId() {
          return this._connectionId;
        }
        disconnect() {
          var _a;
          __classPrivateFieldGet(this, _SSEClient_instances, "m", _SSEClient_clearTimeout).call(this);
          this._disconnecting = true;
          this._connectionState.publish(types_1.ConnectionState.DISCONNECTED);
          if (this._connectionId) {
            fetch(this._postAddress.toString(), {
              method: "DELETE",
              headers: {
                "X-Connection-ID": this._connectionId
              },
              credentials: this._options.withCredentials ? "include" : "same-origin"
            }).catch(() => {
            });
          }
          (_a = this._eventSource) === null || _a === void 0 ? void 0 : _a.close();
          this._connectionId = void 0;
          __classPrivateFieldGet(this, _SSEClient_channels, "f").clear();
        }
        createChannel(name, params) {
          const channel = __classPrivateFieldGet(this, _SSEClient_channels, "f").get(name);
          if (channel && channel.channelState !== pondsocket_common_1.ChannelState.CLOSED) {
            return channel;
          }
          const publisher = __classPrivateFieldGet(this, _SSEClient_instances, "m", _SSEClient_createPublisher).call(this);
          const newChannel = new channel_1.Channel(publisher, this._connectionState, name, params || {});
          __classPrivateFieldGet(this, _SSEClient_channels, "f").set(name, newChannel);
          return newChannel;
        }
        onConnectionChange(callback) {
          return this._connectionState.subscribe(callback);
        }
        onError(callback) {
          return this._errorSubject.subscribe(callback);
        }
      };
      exports.SSEClient = SSEClient;
      _SSEClient_channels = /* @__PURE__ */ new WeakMap(), _SSEClient_instances = /* @__PURE__ */ new WeakSet(), _SSEClient_handleMessage = function _SSEClient_handleMessage2(data) {
        try {
          const lines = data.trim().split("\n");
          for (const line of lines) {
            if (line.trim()) {
              const parsed = JSON.parse(line);
              const event = pondsocket_common_1.channelEventSchema.parse(parsed);
              if (event.event === pondsocket_common_1.Events.CONNECTION && event.action === pondsocket_common_1.ServerActions.CONNECT) {
                if (event.payload && typeof event.payload === "object" && "connectionId" in event.payload) {
                  this._connectionId = event.payload.connectionId;
                }
              }
              this._broadcaster.publish(event);
            }
          }
        } catch (e) {
          this._errorSubject.publish(e instanceof Error ? e : new Error("Failed to parse SSE message"));
        }
      }, _SSEClient_clearTimeout = function _SSEClient_clearTimeout2() {
        if (this._connectionTimeoutId) {
          clearTimeout(this._connectionTimeoutId);
          this._connectionTimeoutId = void 0;
        }
      }, _SSEClient_createPublisher = function _SSEClient_createPublisher2() {
        return (message) => {
          if (this._connectionState.value === types_1.ConnectionState.CONNECTED && this._connectionId) {
            fetch(this._postAddress.toString(), {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
                "X-Connection-ID": this._connectionId
              },
              body: JSON.stringify(message),
              credentials: this._options.withCredentials ? "include" : "same-origin"
            }).catch((err) => {
              this._errorSubject.publish(err instanceof Error ? err : new Error("Failed to send message"));
            });
          }
        };
      }, _SSEClient_handleAcknowledge = function _SSEClient_handleAcknowledge2(message) {
        var _a;
        const channel = (_a = __classPrivateFieldGet(this, _SSEClient_channels, "f").get(message.channelName)) !== null && _a !== void 0 ? _a : new channel_1.Channel(__classPrivateFieldGet(this, _SSEClient_instances, "m", _SSEClient_createPublisher).call(this), this._connectionState, message.channelName, {});
        __classPrivateFieldGet(this, _SSEClient_channels, "f").set(message.channelName, channel);
        channel.acknowledge(this._broadcaster);
      }, _SSEClient_handleUnauthorized = function _SSEClient_handleUnauthorized2(message) {
        const channel = __classPrivateFieldGet(this, _SSEClient_channels, "f").get(message.channelName);
        if (channel) {
          const payload = message.payload;
          channel.decline(payload);
        }
      }, _SSEClient_init = function _SSEClient_init2() {
        this._broadcaster.subscribe((message) => {
          if (message.event === pondsocket_common_1.Events.ACKNOWLEDGE) {
            __classPrivateFieldGet(this, _SSEClient_instances, "m", _SSEClient_handleAcknowledge).call(this, message);
          } else if (message.event === pondsocket_common_1.Events.UNAUTHORIZED) {
            __classPrivateFieldGet(this, _SSEClient_instances, "m", _SSEClient_handleUnauthorized).call(this, message);
          } else if (message.event === pondsocket_common_1.Events.CONNECTION && message.action === pondsocket_common_1.ServerActions.CONNECT) {
            this._connectionState.publish(types_1.ConnectionState.CONNECTED);
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
      var __classPrivateFieldGet = exports && exports.__classPrivateFieldGet || function(receiver, state, kind, f) {
        if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
        if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
        return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
      };
      var _PondClient_instances;
      var _PondClient_clearTimeouts;
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.PondClient = void 0;
      var pondsocket_common_1 = require_pondsocket_common();
      var client_1 = require_client();
      var types_1 = require_types2();
      var WebSocket2 = require_browser().w3cwebsocket;
      var PondClient2 = class extends client_1.PondClient {
        constructor() {
          super(...arguments);
          _PondClient_instances.add(this);
        }
        /**
         * @desc Connects to the server and returns the socket.
         */
        connect() {
          this._disconnecting = false;
          this._connectionState.publish(types_1.ConnectionState.CONNECTING);
          const socket = new WebSocket2(this._address.toString());
          this._connectionTimeoutId = setTimeout(() => {
            if (socket.readyState === WebSocket2.CONNECTING) {
              const error = new Error("Connection timeout");
              this._errorSubject.publish(error);
              socket.close();
            }
          }, this._options.connectionTimeout);
          socket.onopen = () => {
            __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_clearTimeouts).call(this);
            this._reconnectAttempts = 0;
            if (this._options.pingInterval) {
              this._pingIntervalId = setInterval(() => {
                if (socket.readyState === WebSocket2.OPEN) {
                  socket.send(JSON.stringify({ action: "ping" }));
                }
              }, this._options.pingInterval);
            }
          };
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
          socket.onerror = () => {
            const error = new Error("WebSocket error");
            this._errorSubject.publish(error);
            socket.close();
          };
          socket.onclose = () => {
            __classPrivateFieldGet(this, _PondClient_instances, "m", _PondClient_clearTimeouts).call(this);
            this._connectionState.publish(types_1.ConnectionState.DISCONNECTED);
            if (this._disconnecting) {
              return;
            }
            const delay = Math.min(1e3 * Math.pow(2, this._reconnectAttempts), this._options.maxReconnectDelay);
            this._reconnectAttempts++;
            setTimeout(() => {
              this.connect();
            }, delay);
          };
          this._socket = socket;
        }
      };
      exports.PondClient = PondClient2;
      _PondClient_instances = /* @__PURE__ */ new WeakSet(), _PondClient_clearTimeouts = function _PondClient_clearTimeouts2() {
        if (this._connectionTimeoutId) {
          clearTimeout(this._connectionTimeoutId);
          this._connectionTimeoutId = void 0;
        }
        if (this._pingIntervalId) {
          clearInterval(this._pingIntervalId);
          this._pingIntervalId = void 0;
        }
      };
    }
  });

  // node_modules/@eleven-am/pondsocket-client/index.js
  var require_pondsocket_client = __commonJS({
    "node_modules/@eleven-am/pondsocket-client/index.js"(exports) {
      "use strict";
      Object.defineProperty(exports, "__esModule", { value: true });
      exports.SSEClient = exports.PondClient = exports.ConnectionState = exports.ChannelState = void 0;
      var pondsocket_common_1 = require_pondsocket_common();
      Object.defineProperty(exports, "ChannelState", { enumerable: true, get: function() {
        return pondsocket_common_1.ChannelState;
      } });
      var client_1 = require_client();
      var sseClient_1 = require_sseClient();
      Object.defineProperty(exports, "SSEClient", { enumerable: true, get: function() {
        return sseClient_1.SSEClient;
      } });
      var node_1 = require_node();
      var types_1 = require_types2();
      Object.defineProperty(exports, "ConnectionState", { enumerable: true, get: function() {
        return types_1.ConnectionState;
      } });
      var PondClient2 = typeof window === "undefined" ? node_1.PondClient : client_1.PondClient;
      exports.PondClient = PondClient2;
    }
  });

  // src/index.ts
  var index_exports = {};
  __export(index_exports, {
    Bus: () => Bus,
    Executor: () => Executor,
    Logger: () => Logger,
    OpKinds: () => OpKinds,
    Patcher: () => Patcher,
    Runtime: () => Runtime,
    ScriptExecutor: () => ScriptExecutor,
    Topics: () => Topics,
    Transport: () => Transport,
    boot: () => boot,
    isBoot: () => isBoot,
    isMessage: () => isMessage,
    isServerAck: () => isServerAck,
    isServerError: () => isServerError,
    isServerEvt: () => isServerEvt
  });

  // src/protocol.ts
  var Topics = {
    Router: "router",
    DOM: "dom",
    Frame: "frame",
    Ack: "ack"
  };
  function scriptTopic(scriptId) {
    return `script:${scriptId}`;
  }
  function handlerTopic(handlerId) {
    return handlerId;
  }
  var OpKinds = {
    SetText: "setText",
    SetComment: "setComment",
    SetAttr: "setAttr",
    DelAttr: "delAttr",
    SetStyle: "setStyle",
    DelStyle: "delStyle",
    SetStyleDecl: "setStyleDecl",
    DelStyleDecl: "delStyleDecl",
    SetHandlers: "setHandlers",
    SetScript: "setScript",
    DelScript: "delScript",
    SetRef: "setRef",
    DelRef: "delRef",
    ReplaceNode: "replaceNode",
    AddChild: "addChild",
    DelChild: "delChild",
    MoveChild: "moveChild"
  };
  function isBoot(msg) {
    return typeof msg === "object" && msg !== null && msg.t === "boot";
  }
  function isServerError(msg) {
    return typeof msg === "object" && msg !== null && msg.t === "error";
  }
  function isServerEvt(msg) {
    return typeof msg === "object" && msg !== null && typeof msg.t === "string" && typeof msg.a === "string";
  }
  function isServerAck(msg) {
    return typeof msg === "object" && msg !== null && msg.t === "ack" && typeof msg.seq === "number";
  }
  function isMessage(msg) {
    return typeof msg === "object" && msg !== null && typeof msg.seq === "number" && typeof msg.topic === "string" && typeof msg.event === "string";
  }

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
      const key = `${topic}:${String(action)}`;
      const subs = this.subscribers.get(key) ?? [];
      for (const sub of subs) {
        try {
          sub.callback(payload);
        } catch {
        }
      }
      for (const sub of this.wildcardSubscribers) {
        try {
          sub.callback(topic, String(action), payload);
        } catch {
        }
      }
    }
    subscriberCount(topic, action) {
      const key = `${topic}:${String(action)}`;
      return this.subscribers.get(key)?.length ?? 0;
    }
    subscribeScript(scriptId, action, callback) {
      return this.subscribe(scriptTopic(scriptId), action, callback);
    }
    publishScript(scriptId, action, payload) {
      const topic = scriptTopic(scriptId);
      const key = `${topic}:${String(action)}`;
      const subs = this.subscribers.get(key) ?? [];
      for (const sub of subs) {
        try {
          sub.callback(payload);
        } catch {
        }
      }
      for (const sub of this.wildcardSubscribers) {
        try {
          sub.callback(topic, String(action), payload);
        } catch {
        }
      }
    }
    publishHandler(handlerId, payload) {
      const topic = handlerTopic(handlerId);
      const key = `${topic}:invoke`;
      const subs = this.subscribers.get(key) ?? [];
      for (const sub of subs) {
        try {
          sub.callback(payload);
        } catch {
        }
      }
      for (const sub of this.wildcardSubscribers) {
        try {
          sub.callback(topic, "invoke", payload);
        } catch {
        }
      }
    }
    clear() {
      this.subscribers.clear();
      this.wildcardSubscribers = [];
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
  var import_pondsocket_client = __toESM(require_pondsocket_client(), 1);

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

  // src/transport.ts
  var Transport = class {
    constructor(config) {
      this.state = "disconnected";
      this.stateListeners = [];
      this.sessionId = config.sessionId;
      this.bus = config.bus;
      this.client = new import_pondsocket_client.PondClient(config.endpoint);
      const joinPayload = {
        sid: config.sessionId,
        ver: config.version,
        ack: config.lastAck,
        loc: config.location
      };
      this.channel = this.client.createChannel(`live/${config.sessionId}`, joinPayload);
      this.channel.onMessage((_event, payload) => {
        this.handleMessage(payload);
      });
      this.channel.onChannelStateChange((channelState) => {
        this.handleStateChange(channelState);
      });
    }
    get sid() {
      return this.sessionId;
    }
    get connectionState() {
      return this.state;
    }
    connect() {
      this.state = "connecting";
      this.notifyStateChange();
      this.channel.join();
      this.client.connect();
    }
    disconnect() {
      this.channel.leave();
      this.client.disconnect();
      this.state = "disconnected";
      this.notifyStateChange();
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
      const evt = {
        t: topic,
        sid: this.sessionId,
        a: String(action),
        p: payload
      };
      this.sendMessage("evt", evt);
    }
    sendAck(seq) {
      const ack = {
        t: Topics.Ack,
        sid: this.sessionId,
        seq
      };
      this.sendMessage("ack", ack);
    }
    sendHandler(handlerId, payload) {
      const evt = {
        t: handlerTopic(handlerId),
        sid: this.sessionId,
        a: "invoke",
        p: payload
      };
      this.sendMessage("evt", evt);
    }
    sendScript(scriptId, payload) {
      const evt = {
        t: `script:${scriptId}`,
        sid: this.sessionId,
        a: "message",
        p: payload
      };
      this.sendMessage("evt", evt);
    }
    handleMessage(payload) {
      Logger.info("TRANSPORT", "Transport received message:", payload);
      if (!isMessage(payload)) {
        return;
      }
      const { seq, topic, event, data } = payload;
      if (!this.isValidTopic(topic)) {
        return;
      }
      this.publishToBus(topic, event, data, seq);
    }
    isValidTopic(topic) {
      return topic === "router" || topic === "dom" || topic === "frame" || topic === "ack" || topic.startsWith("script:");
    }
    publishToBus(topic, action, data, seq) {
      switch (topic) {
        case "frame":
          if (action === "patch") {
            const payload = {
              seq,
              patches: data
            };
            this.bus.publish("frame", "patch", payload);
          }
          break;
        case "router":
          if (action === "push") {
            this.bus.publish("router", "push", data);
          } else if (action === "replace") {
            this.bus.publish("router", "replace", data);
          } else if (action === "back") {
            this.bus.publish("router", "back", void 0);
          } else if (action === "forward") {
            this.bus.publish("router", "forward", void 0);
          }
          break;
        case "dom":
          if (action === "call") {
            this.bus.publish("dom", "call", data);
          } else if (action === "set") {
            this.bus.publish("dom", "set", data);
          } else if (action === "query") {
            this.bus.publish("dom", "query", data);
          } else if (action === "async") {
            this.bus.publish("dom", "async", data);
          }
          break;
        case "ack":
          if (action === "ack") {
            this.bus.publish("ack", "ack", data);
          }
          break;
        default:
          if (topic.startsWith("script:") && action === "send") {
            const payload = data;
            this.bus.publishScript(payload.scriptId, "send", payload);
          }
          break;
      }
    }
    handleStateChange(channelState) {
      switch (channelState) {
        case import_pondsocket_client.ChannelState.JOINED:
          this.state = "connected";
          break;
        case import_pondsocket_client.ChannelState.STALLED:
          this.state = "stalled";
          break;
        case import_pondsocket_client.ChannelState.CLOSED:
          this.state = "disconnected";
          break;
        case import_pondsocket_client.ChannelState.DECLINED:
          this.state = "declined";
          break;
        case import_pondsocket_client.ChannelState.JOINING:
        case import_pondsocket_client.ChannelState.IDLE:
          this.state = "connecting";
          break;
      }
      this.notifyStateChange();
    }
    notifyStateChange() {
      for (const listener of this.stateListeners) {
        try {
          listener(this.state);
        } catch {
        }
      }
    }
    sendMessage(type, message) {
      Logger.info("TRANSPORT", "Transport sending message:", type, message);
      this.channel.sendMessage(type, message);
    }
  };

  // src/patcher.ts
  var _Patcher = class _Patcher {
    constructor(root, callbacks) {
      this.handlerStore = /* @__PURE__ */ new WeakMap();
      this.scriptStore = /* @__PURE__ */ new WeakMap();
      this.keyedElements = /* @__PURE__ */ new Map();
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
      if (!node) return;
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
        case "setScript":
          this.setScript(node, patch.value);
          break;
        case "delScript":
          this.delScript(node);
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
          this.addChild(node, patch.index, patch.value, patch.path ?? []);
          break;
        case "delChild":
          this.delChild(node, patch.index);
          break;
        case "moveChild":
          this.moveChild(node, patch.value, patch.path ?? []);
          break;
      }
    }
    resolvePath(path) {
      let node = this.root;
      if (path) {
        for (const index of path) {
          if (!node) return null;
          node = node.childNodes[index] ?? null;
        }
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
      for (const [name, values] of Object.entries(attrs)) {
        if (name === "class") {
          if (el instanceof SVGElement) {
            el.setAttribute("class", values.join(" "));
          } else {
            el.className = values.join(" ");
          }
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
      for (const [prop, value] of Object.entries(styles)) {
        el.style.setProperty(prop, value);
      }
    }
    delStyle(el, prop) {
      el.style.removeProperty(prop);
    }
    setStyleDecl(styleEl, selector, prop, value) {
      const sheet = styleEl.sheet;
      if (!sheet) return;
      const rule = this.findOrCreateRule(sheet, selector);
      if (rule) {
        rule.style.setProperty(prop, value);
      }
    }
    delStyleDecl(styleEl, selector, prop) {
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
      const oldHandlers = this.handlerStore.get(el);
      if (oldHandlers) {
        oldHandlers.forEach((state) => {
          if (state.cleanup) state.cleanup();
        });
      }
      const newHandlers = /* @__PURE__ */ new Map();
      for (const meta of handlers) {
        const state = this.createHandler(el, meta);
        newHandlers.set(meta.event, state);
      }
      this.handlerStore.set(el, newHandlers);
    }
    createHandler(el, meta) {
      let timeoutId = null;
      let lastCall = 0;
      const invoke = (e) => {
        if (meta.prevent && e.cancelable) {
          e.preventDefault();
        }
        if (meta.stop) {
          e.stopPropagation();
        }
        const data = this.extractEventData(e, meta.props ?? []);
        this.callbacks.onEvent(meta.handler, data);
      };
      let handler;
      if (meta.debounce && meta.debounce > 0) {
        handler = (e) => {
          if (meta.prevent && e.cancelable) e.preventDefault();
          if (meta.stop) e.stopPropagation();
          if (timeoutId) clearTimeout(timeoutId);
          timeoutId = setTimeout(() => {
            const data = this.extractEventData(e, meta.props ?? []);
            this.callbacks.onEvent(meta.handler, data);
          }, meta.debounce);
        };
      } else if (meta.throttle && meta.throttle > 0) {
        handler = (e) => {
          if (meta.prevent && e.cancelable) e.preventDefault();
          if (meta.stop) e.stopPropagation();
          const now = Date.now();
          if (now - lastCall >= meta.throttle) {
            lastCall = now;
            const data = this.extractEventData(e, meta.props ?? []);
            this.callbacks.onEvent(meta.handler, data);
          }
        };
      } else {
        handler = invoke;
      }
      const options = {};
      if (meta.passive) options.passive = true;
      if (meta.once) options.once = true;
      if (meta.capture) options.capture = true;
      el.addEventListener(meta.event, handler, options);
      return {
        listener: handler,
        cleanup: () => {
          el.removeEventListener(meta.event, handler, options);
          if (timeoutId) clearTimeout(timeoutId);
        }
      };
    }
    extractEventData(e, props) {
      const result = {};
      for (const prop of props) {
        const value = this.resolveProp(e, prop);
        if (value !== void 0) {
          result[prop] = value;
        }
      }
      return result;
    }
    resolveProp(e, path) {
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
    setScript(el, meta) {
      this.delScript(el);
      this.scriptStore.set(el, meta.scriptId);
      this.callbacks.onScript(meta, el);
    }
    delScript(el) {
      const scriptId = this.scriptStore.get(el);
      if (scriptId) {
        this.scriptStore.delete(el);
        this.callbacks.onScriptCleanup(scriptId);
      }
    }
    replaceNode(oldNode, newNodeData) {
      const newNode = this.createNode(newNodeData);
      if (newNode && oldNode.parentNode) {
        this.cleanupTree(oldNode);
        oldNode.parentNode.replaceChild(newNode, oldNode);
      }
    }
    addChild(parent, index, nodeData, parentPath) {
      const newNode = this.createNode(nodeData);
      if (!newNode) return;
      if (nodeData.key && newNode instanceof Element) {
        const keyId = `${parentPath.join(",")}-${nodeData.key}`;
        this.keyedElements.set(keyId, newNode);
      }
      const refChild = parent.childNodes[index] ?? null;
      parent.insertBefore(newNode, refChild);
    }
    delChild(parent, index) {
      const child = parent.childNodes[index];
      if (child) {
        this.cleanupTree(child);
        parent.removeChild(child);
      }
    }
    moveChild(parent, move, parentPath) {
      let child = null;
      if (move.key) {
        const keyId = `${parentPath.join(",")}-${move.key}`;
        child = this.keyedElements.get(keyId) ?? null;
      }
      if (!child) {
        child = parent.childNodes[move.fromIndex] ?? null;
      }
      if (!child) return;
      parent.removeChild(child);
      const refChild = parent.childNodes[move.newIdx] ?? null;
      parent.insertBefore(child, refChild);
    }
    cleanupTree(node) {
      if (node.nodeType === Node.ELEMENT_NODE) {
        const el = node;
        const handlers = this.handlerStore.get(el);
        if (handlers) {
          handlers.forEach((state) => {
            if (state.cleanup) state.cleanup();
          });
          this.handlerStore.delete(el);
        }
        const scriptId = this.scriptStore.get(el);
        if (scriptId) {
          this.scriptStore.delete(el);
          this.callbacks.onScriptCleanup(scriptId);
        }
        for (let i = 0; i < node.childNodes.length; i++) {
          this.cleanupTree(node.childNodes[i]);
        }
      }
    }
    createNode(data, isSvg = false) {
      if (data.text !== void 0) {
        return document.createTextNode(data.text);
      }
      if (data.comment !== void 0) {
        return document.createComment(data.comment);
      }
      if (!data.tag) return null;
      const isSvgElement = _Patcher.SVG_TAGS.has(data.tag);
      const useSvg = isSvg || isSvgElement;
      const el = useSvg ? document.createElementNS(_Patcher.SVG_NS, data.tag) : document.createElement(data.tag);
      if (data.attrs) {
        this.setAttr(el, data.attrs);
      }
      if (data.style) {
        this.setStyle(el, data.style);
      }
      if (data.handlers && data.handlers.length > 0) {
        this.setHandlers(el, data.handlers);
      }
      if (data.script) {
        this.setScript(el, data.script);
      }
      if (data.refId) {
        this.callbacks.onRef(data.refId, el);
      }
      if (data.unsafeHTML) {
        el.innerHTML = data.unsafeHTML;
      } else if (data.children) {
        const childSvg = useSvg && data.tag !== "foreignObject";
        for (const child of data.children) {
          const childNode = this.createNode(child, childSvg);
          if (childNode) {
            el.appendChild(childNode);
          }
        }
      }
      return el;
    }
  };
  _Patcher.SVG_NS = "http://www.w3.org/2000/svg";
  _Patcher.SVG_TAGS = /* @__PURE__ */ new Set([
    "svg",
    "animate",
    "animateMotion",
    "animateTransform",
    "circle",
    "clipPath",
    "defs",
    "desc",
    "ellipse",
    "feBlend",
    "feColorMatrix",
    "feComponentTransfer",
    "feComposite",
    "feConvolveMatrix",
    "feDiffuseLighting",
    "feDisplacementMap",
    "feDistantLight",
    "feDropShadow",
    "feFlood",
    "feFuncA",
    "feFuncB",
    "feFuncG",
    "feFuncR",
    "feGaussianBlur",
    "feImage",
    "feMerge",
    "feMergeNode",
    "feMorphology",
    "feOffset",
    "fePointLight",
    "feSpecularLighting",
    "feSpotLight",
    "feTile",
    "feTurbulence",
    "filter",
    "foreignObject",
    "g",
    "image",
    "line",
    "linearGradient",
    "marker",
    "mask",
    "metadata",
    "mpath",
    "path",
    "pattern",
    "polygon",
    "polyline",
    "radialGradient",
    "rect",
    "set",
    "stop",
    "switch",
    "symbol",
    "text",
    "textPath",
    "title",
    "tspan",
    "use",
    "view"
  ]);
  var Patcher = _Patcher;

  // src/executor.ts
  var Executor = class {
    constructor(config) {
      this.subscriptions = [];
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
      const el = this.resolveRef(payload.ref);
      if (!el) return;
      const method = el[payload.method];
      if (typeof method === "function") {
        method.apply(el, payload.args ?? []);
      }
    }
    handleSet(payload) {
      const el = this.resolveRef(payload.ref);
      if (!el) return;
      el[payload.prop] = payload.value;
    }
    handleQuery(payload) {
      const el = this.resolveRef(payload.ref);
      const response = { requestId: payload.requestId };
      if (!el) {
        response.error = `ref not found: ${payload.ref}`;
        this.transport.send("dom", "response", response);
        return;
      }
      const values = {};
      for (const selector of payload.selectors) {
        values[selector] = this.readProperty(el, selector);
      }
      response.values = values;
      this.transport.send("dom", "response", response);
    }
    handleAsync(payload) {
      const el = this.resolveRef(payload.ref);
      const response = { requestId: payload.requestId };
      if (!el) {
        response.error = `ref not found: ${payload.ref}`;
        this.transport.send("dom", "response", response);
        return;
      }
      const method = el[payload.method];
      if (typeof method !== "function") {
        response.error = `method not found: ${payload.method}`;
        this.transport.send("dom", "response", response);
        return;
      }
      Promise.resolve(method.apply(el, payload.args ?? [])).then((result) => {
        response.result = this.serializeValue(result);
        this.transport.send("dom", "response", response);
      }).catch((err) => {
        response.error = err instanceof Error ? err.message : String(err);
        this.transport.send("dom", "response", response);
      });
    }
    handlePush(payload) {
      const url = this.buildUrl(payload);
      window.history.pushState({}, "", url);
    }
    handleReplace(payload) {
      const url = this.buildUrl(payload);
      window.history.replaceState({}, "", url);
    }
    handleBack() {
      window.history.back();
    }
    handleForward() {
      window.history.forward();
    }
    buildUrl(payload) {
      let url = payload.path;
      if (payload.query) {
        url += "?" + payload.query;
      }
      if (payload.hash) {
        url += "#" + payload.hash;
      }
      return url;
    }
    readProperty(el, path) {
      const segments = path.split(".");
      let current = el;
      for (const segment of segments) {
        if (current == null) return void 0;
        current = current[segment];
      }
      return this.serializeValue(current);
    }
    serializeValue(value) {
      if (value === null || value === void 0) return null;
      const type = typeof value;
      if (type === "string" || type === "number" || type === "boolean") return value;
      if (Array.isArray(value)) {
        return value.map((v) => this.serializeValue(v)).filter((v) => v !== void 0);
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
  function isResumeOK(msg) {
    return typeof msg === "object" && msg !== null && msg.t === "resume_ok";
  }
  var Runtime = class {
    constructor(config) {
      this.refs = /* @__PURE__ */ new Map();
      this.cseq = 0;
      this.lastSeq = 0;
      this.connectedState = false;
      this.lastSeq = config.seq;
      Logger.configure({ enabled: config.debug ?? false, level: "debug" });
      Logger.info("Runtime", "Initializing", { sid: config.sessionId, ver: config.version });
      this.bus = new Bus();
      this.transport = new Transport({
        endpoint: config.endpoint,
        sessionId: config.sessionId,
        version: config.version,
        lastAck: config.seq,
        location: config.location,
        bus: this.bus
      });
      const resolveRef = (refId) => this.refs.get(refId);
      this.patcher = new Patcher(config.root, {
        onEvent: (handlerId, data) => this.handleEvent(handlerId, data),
        onRef: (refId, el) => {
          this.refs.set(refId, el);
          Logger.debug("Runtime", "Ref set", refId);
        },
        onRefDelete: (refId) => {
          this.refs.delete(refId);
          Logger.debug("Runtime", "Ref deleted", refId);
        },
        onScript: (meta, el) => this.handleScript(meta, el),
        onScriptCleanup: (scriptId) => this.handleScriptCleanup(scriptId)
      });
      this.executor = new Executor({
        bus: this.bus,
        transport: this.transport,
        resolveRef
      });
      this.scripts = new ScriptExecutor({ bus: this.bus, transport: this.transport });
      this.bus.subscribe("frame", "patch", (payload) => this.handlePatch(payload));
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
      this.executor.destroy();
      this.scripts.destroy();
      this.bus.clear();
      this.refs.clear();
    }
    connected() {
      return this.connectedState;
    }
    get seq() {
      return this.lastSeq;
    }
    handleBoot(boot2) {
      Logger.info("Runtime", "Boot received", { ver: boot2.ver, seq: boot2.seq, patches: boot2.patch?.length ?? 0 });
      if (boot2.patch && boot2.patch.length > 0) {
        this.applyPatches(boot2.patch);
      }
      this.lastSeq = boot2.seq;
      this.transport.sendAck(boot2.seq);
    }
    handleMessage(msg) {
      if (isServerError(msg)) {
        const err = msg;
        Logger.error("Runtime", "Server error", { code: err.code, message: err.message });
        return;
      }
      if (isResumeOK(msg)) {
        this.handleResumeOK(msg);
        return;
      }
    }
    handlePatch(payload) {
      Logger.debug("Runtime", "Patch received", { seq: payload.seq, count: payload.patches?.length ?? 0 });
      if (payload.patches && payload.patches.length > 0) {
        this.applyPatches(payload.patches);
      }
      this.lastSeq = payload.seq;
      this.transport.sendAck(payload.seq);
    }
    handleResumeOK(resume) {
      Logger.info("Runtime", "Resume OK", { from: resume.from, to: resume.to });
    }
    handleEvent(handlerId, data) {
      this.cseq++;
      Logger.debug("Runtime", "Event", { handler: handlerId, cseq: this.cseq });
      const payload = {
        ...data,
        cseq: this.cseq
      };
      this.transport.sendHandler(handlerId, payload);
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
      } else if (wasConnected && !this.connectedState) {
        Logger.warn("Runtime", "Disconnected");
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
    applyPatches(patches) {
      this.patcher.apply(patches);
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
      endpoint: "/live",
      location: bootData.location,
      debug: bootData.client?.debug
    };
    const runtime = new Runtime(config);
    runtime.handleBoot(bootData);
    runtime.connect();
    return runtime;
  }

  // src/index.ts
  if (typeof window !== "undefined" && typeof document !== "undefined") {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", () => boot());
    } else {
      boot();
    }
  }
  return __toCommonJS(index_exports);
})();
//# sourceMappingURL=pondlive-dev.js.map

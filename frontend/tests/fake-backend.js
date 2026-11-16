// Fake Wails backend for the UI tests, injected before the app loads.
//
// It replaces window.go.app.api (the bound Go methods) and window.runtime
// (Wails events). The schema comes from tests/fixtures/schema.json, which is
// generated from the real Go backend (see internal/app/fixtures_test.go), and
// the events are emitted in the same order as the Go code emits them.
//
// Tests configure it with window.__WOMBAT_FAKE_CONFIG__ (see tests/app.js)
// and inspect it through window.__fake.
(() => {
  const schema = window.__WOMBAT_SCHEMA__;
  const cfg = {
    addr: "localhost:5001",
    saved: {}, // method full name -> saved request JSON
    metadata: null, // saved request metadata [{key, val}]
    schemaDelay: 0, // ms until the services arrive (reflection time)
    responses: {}, // method full name -> response text (default: an echo)
    connectError: "", // if set, Connect fails with this error (after saving)
    streamCount: 1, // responses per Send (a server stream when > 1)
    workspaces: [], // more workspaces for the switcher: [{id, addr}]
    ...(window.__WOMBAT_FAKE_CONFIG__ || {}),
  };

  const calls = [];
  const listeners = new Map();

  // Wails delivers events asynchronously, in order.
  const emit = (name, ...data) =>
    setTimeout(() => {
      for (const l of [...(listeners.get(name) || [])]) {
        l.cb(...data);
        if (l.max > 0 && --l.max === 0) off(name, l);
      }
    }, 0);
  const off = (name, l) => {
    const ls = listeners.get(name) || [];
    const i = ls.indexOf(l);
    if (i >= 0) ls.splice(i, 1);
  };

  const options = () => ({
    id: "wksp_default",
    addr: cfg.addr,
    reflect: true,
    protos: { files: null, roots: null },
    insecure: false,
    plaintext: true,
    rootca: "",
    clientcert: "",
    clientkey: "",
  });

  const methodSelect = name => {
    for (const s of schema.services) for (const m of s.methods) if (m.full_name === name) return m;
    return undefined;
  };

  // The request as prototext, like api.formatPayload sends it (multiline,
  // two-space indent, trailing newline), using the method's field kinds.
  const QUOTED = new Set(["string", "bytes"]);
  const prototext = (message, value, indent = "") => {
    let out = "";
    const scalar = (field, v) => (QUOTED.has(field.kind) ? JSON.stringify(String(v)) : String(v));
    const entry = (name, field, v) => {
      if (v === null || v === undefined) return;
      if (field.kind === "message" || field.kind === "group") {
        out += `${indent}${name}: {\n${prototext(field.message, v, indent + "  ")}${indent}}\n`;
      } else {
        out += `${indent}${name}: ${scalar(field, v)}\n`;
      }
    };
    for (const field of message?.fields || []) {
      if (field.kind === "oneof") {
        for (const member of field.oneof) entry(member.name, member, value?.[member.name]);
        continue;
      }
      const v = value?.[field.name];
      if (field.kind === "map") {
        for (const [k, mv] of Object.entries(v || {})) {
          out += `${indent}${field.name}: {\n${indent}  key: ${scalar(field.map_key, k)}\n`;
          const inner = { fields: [{ ...field.map_value, name: "value" }] };
          out += prototext(inner, { value: mv }, indent + "  ") + `${indent}}\n`;
        }
      } else if (field.repeated) {
        for (const item of v || []) entry(field.name, { ...field, repeated: false }, item);
      } else {
        entry(field.name, field, v);
      }
    }
    return out;
  };

  // Mirrors api.Connect + loadProtoFiles.
  const connect = addr => {
    emit("wombat:client_connect_started", addr);
    emit("wombat:services_select_changed");
    emit("wombat:method_input_changed");
    emit("wombat:client_connected", addr);
    emit("wombat:client_state_changed", "READY");
    setTimeout(() => emit("wombat:services_select_changed", schema.services, "", "", null), cfg.schemaDelay);
  };

  const api = {
    FrontendReady() {
      if (cfg.addr) connect(cfg.addr);
    },
    GetWorkspaceOptions: () => options(),
    ListWorkspaces: () => [
      options(),
      ...(cfg.workspaces || []).filter(w => !(cfg.deleted || []).includes(w.id)).map(w => ({ ...options(), ...w })),
    ],
    SelectWorkspace(id) {
      connect(cfg.addr);
    },
    DeleteWorkspace(id) {
      cfg.deleted = [...(cfg.deleted || []), id];
    },
    Connect(opts, metadata, save) {
      cfg.addr = opts.addr; // saved first, like the real one
      if (cfg.connectError) {
        emit("wombat:client_connect_started", opts.addr);
        emit("wombat:error", { title: "Connection error", msg: cfg.connectError });
        throw new Error(cfg.connectError);
      }
      connect(opts.addr);
    },
    RetryConnection() {},
    GetMetadata: () => cfg.metadata,
    GetReflectMetadata: () => null,
    // rawStateDelay: method -> ms, to let a later method change overtake it.
    GetRawMessageState: method => {
      const delay = cfg.rawStateDelay?.[method];
      if (!delay) return cfg.saved[method] || "";
      return new Promise(resolve => setTimeout(() => resolve(cfg.saved[method] || ""), delay));
    },

    // Mirrors api.SelectMethod.
    SelectMethod(fullname, initState, metadata) {
      const m = schema.methods[fullname];
      if (!m) {
        emit("wombat:error", { title: "Failed to select method", msg: `unknown method ${fullname}` });
        emit("wombat:method_input_changed");
        return;
      }
      emit("wombat:method_input_changed", m, initState, metadata ?? null);
    },

    // Mirrors api.Send and the stats handler: rpc_started on its own, the
    // rest in batches (wombat:rpc_events) like the backend sends them.
    Send(method, json, metadata) {
      cfg.saved[method] = json;
      cfg.metadata = metadata;
      const m = methodSelect(method) || {};
      emit("wombat:rpc_started", { client_stream: !!m.client_stream, server_stream: !!m.server_stream });
      const ev = (name, ...data) => ({ name, data });
      const events = [
        ev("wombat:stat_begin", { BeginTime: "2026-11-16T10:00:00Z" }),
        ev("wombat:stat_out_payload", { Payload: json, Length: json.length }),
        ev("wombat:out_payload_received", prototext(schema.methods[method]?.message, JSON.parse(json))),
        ev("wombat:in_header_received", { "content-type": ["application/grpc"] }),
      ];
      for (let i = 0; i < cfg.streamCount; i++) {
        const text = cfg.streamCount > 1 ? `n: ${i + 1}\n` : cfg.responses[method] ?? `echo: ${json}\n`; // text ends with \n
        events.push(ev("wombat:stat_in_payload", { Payload: text, Length: text.length }));
        events.push(ev("wombat:in_payload_received", text));
      }
      events.push(ev("wombat:in_trailer_received", {}));
      events.push(ev("wombat:rpc_ended", { status: "OK", status_code: 0, duration: "1ms" }));
      for (let i = 0; i < events.length; i += 500) emit("wombat:rpc_events", events.slice(i, i + 500));
    },
    Cancel() {},
    CloseSend() {},

    ExportCommands(method, json, metadata) {
      return { grpcurl: `grpcurl -d '${json}' \\\n    -plaintext \\\n    ${cfg.addr} ${method.replace(/^\//, "")}` };
    },
    // Mirrors api.ImportCommand for the commands ExportCommands produces.
    ImportCommand(kind, command) {
      const data = (command.match(/-d '([^']*)'/) || [])[1] || "{}";
      const method = command.trim().split(/\s+/).pop();
      const headers = [...command.matchAll(/-(?:H|rpc-header) '([^:']*):\s*([^']*)'/g)].map(([, key, val]) => ({ key, val }));
      emit("wombat:services_select_changed", schema.services, "/" + method, data, headers);
    },

    FindProtoFiles: () => null,
    SelectDirectory: () => "",
  };

  // Every call is recorded and answered asynchronously, like Wails does.
  const go = {};
  for (const [name, fn] of Object.entries(api)) {
    go[name] = (...args) => {
      calls.push({ name, args: JSON.parse(JSON.stringify(args)) });
      return new Promise((resolve, reject) =>
        setTimeout(() => {
          try {
            resolve(fn(...args) ?? null);
          } catch (e) {
            reject(e);
          }
        }, 0));
    };
  }
  window.go = {
    app: {
      api: new Proxy(go, {
        get: (t, k) => t[k] || (() => Promise.reject(new Error(`fake backend: ${String(k)} is not implemented`))),
      }),
    },
  };

  const noop = () => {};
  window.runtime = new Proxy(
    {
      EventsOnMultiple(name, cb, max) {
        const l = { cb, max };
        if (!listeners.has(name)) listeners.set(name, []);
        listeners.get(name).push(l);
        return () => off(name, l);
      },
      EventsOff(...names) {
        for (const n of names) listeners.delete(n);
      },
      EventsOffAll() {
        listeners.clear();
      },
      EventsEmit(name, ...data) {
        emit(name, ...data);
      },
    },
    { get: (t, k) => t[k] || noop }, // Log*, Window*, BrowserOpenURL, ...
  );

  window.__fake = {
    calls,
    emit,
    config: cfg,
    callsTo: name => calls.filter(c => c.name === name),
  };
})();

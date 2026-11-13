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
    ListWorkspaces: () => [options()],
    SelectWorkspace(id) {
      connect(cfg.addr);
    },
    DeleteWorkspace() {},
    Connect(opts, metadata, save) {
      cfg.addr = opts.addr;
      connect(opts.addr);
    },
    RetryConnection() {},
    GetMetadata: () => cfg.metadata,
    GetReflectMetadata: () => null,
    GetRawMessageState: method => cfg.saved[method] || "",

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

    // Mirrors api.Send and the stats handler for a successful unary call.
    Send(method, json, metadata) {
      cfg.saved[method] = json;
      cfg.metadata = metadata;
      const m = methodSelect(method) || {};
      emit("wombat:rpc_started", { client_stream: !!m.client_stream, server_stream: !!m.server_stream });
      emit("wombat:out_payload_received", json);
      emit("wombat:in_header_received", { "content-type": ["application/grpc"] });
      emit("wombat:in_payload_received", `echo: ${json}\n`);
      emit("wombat:in_trailer_received", {});
      emit("wombat:rpc_ended", { status: "OK", status_code: 0, duration: "1ms" });
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
      emit("wombat:services_select_changed", schema.services, "/" + method, data, null);
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

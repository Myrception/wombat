<script>
  import { onDestroy } from "svelte";
  import Tab from "../controls/Tab.svelte";
  import Tabs from "../controls/Tabs.svelte";
  import TabList from "../controls/TabList.svelte";
  import TabPanel from "../controls/TabPanel.svelte";
  import OutputHeader from "./OutputHeader.svelte";
  import Response from "./Response.svelte";
  import HeadersTrailers from "./HeadersTrailers.svelte";
  import Statistics from "./Statistics.svelte";
  import { EventsOn } from '../../wailsjs/runtime/runtime';
  import { monaco } from '../monaco';
  import { maxMessages } from '../settings';

  let headers = {};
  let trailers = {};
  let rpc = {};
  let stats = [];
  let inflight = false;
  let client_stream = false;
  let server_stream = false;
  let outCount = 0;
  let inCount = 0;

  // Only the last $maxMessages messages (and their statistics) are kept;
  // `removed` counts the older ones that were dropped.
  let removed = 0;
  let statsRemoved = 0;
  let messages = []; // per message in the response: its decoration ids (the first marks its header line)
  let statsBuf = []; // the statistics, copied to `stats` once per batch
  let statsDirty = false;
  let payloadStats = 0; // payload entries in statsBuf

  const respModel = monaco.editor.createModel("", "javascript");

  const reset = data => {
    headers = {};
    trailers = {};
    rpc = {};
    statsBuf = [];
    stats = [];
    inflight = true;
    client_stream = data.client_stream;
    server_stream = data.server_stream;
    outCount = 0;
    inCount = 0;
    removed = 0;
    statsRemoved = 0;
    payloadStats = 0;
    messages = [];
    respModel.setValue("");
  };

  // Removes the oldest messages until at most `max` are left. A message runs
  // from its header line to the line before the next message's header.
  const trim = max => {
    if (messages.length <= max) return;
    const drop = messages.length - max;
    const next = respModel.getDecorationRange(messages[drop].ids[0]);
    if (!next) return;
    respModel.deltaDecorations(messages.slice(0, drop).flatMap(m => m.ids), []);
    respModel.applyEdits([{ range: new monaco.Range(1, 1, next.startLineNumber, 1), text: "" }]);
    messages = messages.slice(drop);
    removed += drop;
  };

  const append = (payload = "", type = "") => {
    let isEmpty = false;
    if(payload === "") {
      isEmpty = true;
      payload = "<empty>\n";
    }

    const eof = respModel.getLineCount();
    const range = new monaco.Range(eof, 1, eof, 1);
    // applyEdits: the response is read-only, so no undo history is kept.
    respModel.applyEdits([{ forceMoveMarkers: true, range, text: "\n"+payload }]);

    const newEof = respModel.getLineCount();
    const payloadStart = new monaco.Range(eof+1, 1, eof+1, 1)
    const payloadEnd = new monaco.Range(newEof-1, 1, newEof-1, 1)

    const decors = [
      { range, options: { isWholeLine: true, afterContentClassName: "decor-header "+type }},
      { range: payloadStart, options: { isWholeLine: true, linesDecorationsClassName: "decor-lines "+type }},
      { range: payloadEnd, options: { isWholeLine: true, linesDecorationsClassName: "decor-lines "+type }},
    ];

    if (isEmpty) {
      decors.push({ range: payloadStart, options: { isWholeLine: true, inlineClassName: "decor-empty" }});
    }

    if (newEof - eof > 3) {
      const payloadMiddle = new monaco.Range(eof+2, 1, newEof-2, 1)
      decors.push({ range: payloadMiddle, options: { isWholeLine: true, linesDecorationsClassName: "decor-lines-block "+type }});
    }

    messages.push({ ids: respModel.deltaDecorations([], decors) });
    trim($maxMessages);
  }

  const isPayloadStat = s => s.type === "inPayload" || s.type === "outPayload";

  const addStat = (type, data) => {
    data.type = type;
    statsBuf.push(data);
    statsDirty = true;
    if (!isPayloadStat(data)) return;
    // Keep as many payload statistics as messages.
    payloadStats++;
    while (payloadStats > $maxMessages) {
      statsBuf.splice(statsBuf.findIndex(isPayloadStat), 1);
      payloadStats--;
      statsRemoved++;
    }
  };

  // Once per batch, so a fast stream does not copy the list per message.
  const flushStats = () => {
    if (!statsDirty) return;
    statsDirty = false;
    stats = statsBuf.slice();
  };

  const handlers = {
    "wombat:rpc_started": reset,
    "wombat:in_header_received": data => headers = data,
    "wombat:in_trailer_received": data => trailers = data,
    "wombat:out_payload_received": data => append(data, "out-payload"),
    "wombat:in_payload_received": data => append(data, "in-payload"),
    "wombat:error_received": data => append(data, "error"),
    "wombat:rpc_ended": data => {
      rpc = data;
      inflight = false;
    },
    "wombat:stat_begin": data => addStat("begin", data),
    "wombat:stat_out_header": data => addStat("outHeader", data),
    "wombat:stat_out_payload": data => {
      addStat("outPayload", data);
      outCount = outCount + 1;
    },
    "wombat:stat_out_trailer": data => addStat("outTrailer", data),
    "wombat:stat_in_header": data => addStat("inHeader", data),
    "wombat:stat_in_payload": data => {
      addStat("inPayload", data);
      inCount = inCount + 1;
    },
    "wombat:stat_in_trailer": data => addStat("inTrailer", data),
    "wombat:stat_end": data => addStat("end", data),
  };

  // The backend sends the events of a call in batches (see batch.go); each
  // event can also arrive on its own.
  const unsubscribes = Object.entries(handlers).map(([name, fn]) =>
    EventsOn(name, (...data) => {
      fn(...data);
      flushStats();
    }));
  unsubscribes.push(EventsOn("wombat:rpc_events", events => {
    for (const e of events || []) handlers[e.name]?.(...(e.data || []));
    flushStats();
  }));

  // A lower limit applies at once.
  const unsubscribeMax = maxMessages.subscribe(max => trim(max));

  onDestroy(() => {
    for (const u of unsubscribes) u();
    unsubscribeMax();
  });
</script>

<style>
  .output-pane {
    width: 100%;
    height: 100%;
    overflow: hidden;
  }
</style>

<div class="output-pane">
  <OutputHeader {rpc} {inflight} {client_stream} {server_stream} {outCount} {inCount} {removed} />
  <Tabs>
    <TabList>
      <Tab>Payload</Tab>
      <Tab>Headers/Trailers</Tab>
      <Tab>Statistics</Tab>
    </TabList>

    <TabPanel>
      <Response model={respModel} />
    </TabPanel>

    <TabPanel>
      <HeadersTrailers {headers} {trailers} />
    </TabPanel>

    <TabPanel>
      <Statistics {stats} removed={statsRemoved} />
    </TabPanel>
  </Tabs>
</div>

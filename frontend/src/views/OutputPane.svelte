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
  
  let headers = {};
  let trailers = {};
  let rpc = {};
  let stats = [];
  let inflight = false;
  let client_stream = false;
  let server_stream = false;
  let outCount = 0;
  let inCount = 0;
  let hasPayload = false;

  const respModel = monaco.editor.createModel("", "javascript");

  // Set up event listeners with unsubscribe functions
  const unsubscribeRPC = EventsOn("wombat:rpc_started", data => {
    headers = {};
    trailers = {};
    rpc = {};
    stats = [];
    inflight = true;
    client_stream = data.client_stream;
    server_stream = data.server_stream;
    outCount = 0;
    inCount = 0;

    respModel.setValue("");
  });

  const append = (payload = "", type = "") => {
    let isEmpty = false;
    if(payload === "") {
      isEmpty = true;
      payload = "<empty>\n";
    }

    const eof = respModel.getLineCount();
    const range = new monaco.Range(eof, 0, eof, 0);
    respModel.pushEditOperations(null, [{forceMoveMarkers: true, range, text: "\n"+payload }]);

    const newEof = respModel.getLineCount();
    const payloadStart = new monaco.Range(eof+1, 0, eof+1, 0)
    const payloadEnd = new monaco.Range(newEof-1, 0, newEof-1, 0)

    const decors = [
      { range, options: { isWholeLine: true, afterContentClassName: "decor-header "+type }},
      { range: payloadStart, options: { isWholeLine: true, linesDecorationsClassName: "decor-lines "+type }},
      { range: payloadEnd, options: { isWholeLine: true, linesDecorationsClassName: "decor-lines "+type }},
    ];

    if (isEmpty) {
      decors.push({ range: payloadStart, options: { isWholeLine: true, inlineClassName: "decor-empty" }});
    }

    if (newEof - eof > 3) {
      const payloadMiddle = new monaco.Range(eof+2, 0, newEof-2, 0)
      decors.push({ range: payloadMiddle, options: { isWholeLine: true, linesDecorationsClassName: "decor-lines-block "+type }});
    }

    respModel.deltaDecorations([], decors);
  }

  const unsubscribeHeader = EventsOn("wombat:in_header_received", data => headers = data);
  const unsubscribeTrailer = EventsOn("wombat:in_trailer_received", data => trailers = data);
  const unsubscribeOutPayload = EventsOn("wombat:out_payload_received", data => {
    append(data, "out-payload");
  });
  
  const unsubscribeInPayload = EventsOn("wombat:in_payload_received", data => {
    append(data, "in-payload");
  });
  
  const unsubscribeError = EventsOn("wombat:error_received", data => {
    append(data, "error");
  });
  
  const unsubscribeRPCEnded = EventsOn("wombat:rpc_ended", data => {
    rpc = data;
    inflight = false;
  });

  const addStat = (type, data) => {
    data.type = type;
    stats = [...stats, data];
  }
  
  const unsubscribeBegin = EventsOn("wombat:stat_begin", data => addStat("begin", data));
  const unsubscribeOutHeader = EventsOn("wombat:stat_out_header", data => addStat("outHeader", data));
  const unsubscribeOutPayload2 = EventsOn("wombat:stat_out_payload", data => { 
    addStat("outPayload", data); 
    outCount = outCount + 1; 
  });
  
  const unsubscribeOutTrailer = EventsOn("wombat:stat_out_trailer", data => addStat("outTrailer", data));
  const unsubscribeInHeader = EventsOn("wombat:stat_in_header", data => addStat("inHeader", data));
  const unsubscribeInPayload2 = EventsOn("wombat:stat_in_payload", data => { 
    addStat("inPayload", data); 
    inCount = inCount + 1; 
  });
  
  const unsubscribeInTrailer = EventsOn("wombat:stat_in_trailer", data => addStat("inTrailer", data));
  const unsubscribeEnd = EventsOn("wombat:stat_end", data => addStat("end", data));

  // Clean up all subscriptions on component destroy
  onDestroy(() => {
    unsubscribeRPC();
    unsubscribeHeader();
    unsubscribeTrailer();
    unsubscribeOutPayload();
    unsubscribeInPayload();
    unsubscribeError();
    unsubscribeRPCEnded();
    unsubscribeBegin();
    unsubscribeOutHeader();
    unsubscribeOutPayload2();
    unsubscribeOutTrailer();
    unsubscribeInHeader();
    unsubscribeInPayload2();
    unsubscribeInTrailer();
    unsubscribeEnd();
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
  <OutputHeader {rpc} {inflight} {client_stream} {server_stream} {outCount} {inCount} />
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
      <Statistics {stats} />
    </TabPanel>
  </Tabs>
</div>

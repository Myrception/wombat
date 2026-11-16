<script>
  import { onDestroy } from "svelte";
  import Tab from "../controls/Tab.svelte";
  import Tabs from "../controls/Tabs.svelte";
  import TabList from "../controls/TabList.svelte";
  import TabPanel from "../controls/TabPanel.svelte";
  import MethodSelect from "./MethodSelect.svelte";
  import MethodInput from "./MethodInput.svelte";
  import RequestMetadata from "./RequestMetadata.svelte";
  import CodeEditPanel from "./CodeEditPanel.svelte";
  import { getContext, tick } from 'svelte';
  import { EventsOn, EventsEmit } from '../../wailsjs/runtime/runtime';
  import { invalidListItems } from "./invalidJSON";
  import { GetRawMessageState, GetMetadata, Send, ExportCommands } from '../../wailsjs/go/app/api';

  let methodInput = {
    full_name: "",
    fields: []
  };
  let state = {};
  let metadata = [];
  let mapItems = {};
  let methodSelected = undefined;

  const reset = all => {
    methodInput = {
      full_name: "",
      fields: [],
    }
    state = {};
    mapItems = {};
    if (all) {
      metadata = [];
    }

    return tick();
  }

  // Set up event listeners with cleanup
  // Parses a request; a broken one (e.g. saved by an older version) starts
  // the form empty instead of breaking it.
  const parseState = text => {
    try {
      const v = JSON.parse(text);
      return v !== null && typeof v === "object" && !Array.isArray(v) ? v : {};
    } catch (e) {
      console.warn("ignoring an invalid request:", e);
      return {};
    }
  };

  // Each method change gets a number; the async steps of an older one stop
  // when a newer one has started, so quick changes cannot mix their state.
  let methodChange = 0;
  const unsubscribeMethodInput = EventsOn("wombat:method_input_changed", async (data, initState, m) => {
    const change = ++methodChange;
    await reset();
    if (change !== methodChange || !data) {
      return
    }
    methodInput = data.message;
    if (Array.isArray(m)) {
      metadata = m; // an imported command's headers
    }
    if (initState) {
      state = parseState(initState);
    } else {
      const rawState = await GetRawMessageState(data.full_name);
      if (change === methodChange && rawState) {
        state = parseState(rawState);
      }
    }
  });

  const unsubscribeClientConnect = EventsOn("wombat:client_connect_started", async (addr) => {
    await reset(true);
    const m = await GetMetadata(addr);
    if (m) {
      metadata = m;
    }
  });

  // Clean up on component destroy
  onDestroy(() => {
    unsubscribeMethodInput();
    unsubscribeClientConnect();
  });

  const onSend = async ({ detail: { method } }) => {
    // An invalid list item would be sent as null, which the server rejects.
    const invalid = invalidListItems(state);
    if (invalid.length) {
      EventsEmit("wombat:error", {
        title: "Cannot send the request",
        msg: `Invalid JSON in ${invalid.join(", ")}. Fix or remove ${invalid.length > 1 ? "these items" : "this item"} first.`,
      });
      return;
    }
    try {
      await Send(method, JSON.stringify(state), metadata);
    } catch (error) {
      console.error("Error sending request:", error);
    }
  };

  const onSelected = ({ detail: { method } }) => methodSelected = method;

  const { open } = getContext('modal');
  const onEdit = async () => {
    if (methodSelected === undefined) return;
    const commands = await ExportCommands(methodSelected.value, JSON.stringify(state), metadata);
    open(CodeEditPanel, { commands });
  }
</script>

<style>
  .input-pane {
    width: 100%;
    height: 100%;
  }
</style>

<div class="input-pane">
  <MethodSelect on:send={onSend} on:selected={onSelected}/>
  <Tabs>
    <TabList>
      <Tab>Request</Tab>
      <Tab>Metadata</Tab>
    </TabList>

    <TabPanel>
      <MethodInput {methodInput} {state} {mapItems} on:edit={onEdit} />
    </TabPanel>

    <TabPanel>
      <RequestMetadata bind:metadata />
    </TabPanel>
  </Tabs>
</div>

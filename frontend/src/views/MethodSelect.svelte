<script>
  import { createEventDispatcher, onDestroy } from "svelte";
  import Button from "../controls/Button.svelte";
  import Dropdown from "../controls/Dropdown.svelte";
  import MethodSelectItem from "./MethodSelectItem.svelte";
  import { EventsOn } from '../../wailsjs/runtime/runtime';
  import { SelectMethod, Send } from '../../wailsjs/go/app/api';

  let servicesSelect = []
  let serviceOptions = [];
  let serviceSelected;
  let methodOptions = []
  let methodSelected;

  const initServiceSelection = (value) => {
    methodSelected = undefined;
    methodOptions = servicesSelect[value].methods.map(m => ({
      value: m.full_name,
      label: m.name,
      client_stream: m.client_stream,
      server_stream: m.server_stream,
    }))
  }

  // What is currently loaded, so picking the same service or method again
  // does not reset the form.
  let loadedService = -1;
  let loadedMethod = undefined;

  const loadMethod = (fullName, initState = "", metadata = []) => {
    loadedMethod = fullName;
    return SelectMethod(fullName, initState, metadata);
  }

  // Selects the first method of service `index` and loads its form.
  const selectService = (index) => {
    loadedService = index;
    serviceSelected = serviceOptions[index];
    initServiceSelection(index);
    if (methodOptions.length > 0) {
      methodSelected = methodOptions[0];
      loadMethod(methodSelected.value);
    }
  }

  // Set up event listener with cleanup
  const unsubscribeServices = EventsOn("wombat:services_select_changed", async (data = [], methodFullName, initState, metadata) => {
    reset()
    servicesSelect = data;
    serviceOptions = data.map((s, i) => ({value: i, label: s.full_name}))
    if (methodFullName) {
      // An imported command: select its method with its request data.
      serviceSelected = serviceOptions.find(it => methodFullName.startsWith(`/${it.label}/`))
      loadedService = serviceOptions.indexOf(serviceSelected)
      initServiceSelection(loadedService)
      methodSelected = methodOptions.find(it => it.value == methodFullName)
      await loadMethod(methodSelected.value, initState ?? "{}", metadata ?? []);
    } else if (serviceOptions.length > 0) {
      selectService(0);
    }
  });

  // Clean up on component destroy
  onDestroy(() => {
    unsubscribeServices();
  });

  // The dropdowns fire `select` only for choices made by the user, so the
  // methods are loaded explicitly here.
  const serviceSelectionChanged = ({ detail: { value } }) => {
    if (value !== loadedService) selectService(value);
  }
  const methodSelectionChanged = ({ detail: { value } }) => {
    if (value !== loadedMethod) loadMethod(value);
  }

  const dispatch = createEventDispatcher();
  const onSend = () => {
    dispatch("send", { method: methodSelected.value });
  }

  $: dispatch("selected", { method: methodSelected });

  const reset = () => {
    loadedService = -1;
    loadedMethod = undefined;
    serviceOptions = [];
    serviceSelected = undefined;
    methodOptions = [];
    methodSelected = undefined;
  }
</script>

<style>
  .method-select {
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: var(--border);
    flex-wrap: wrap;
    padding: 0 var(--padding);
  }
  .spacer {
    flex-grow: 1;
    min-width: var(--padding);
  }
</style>

<div class="method-select">
  <Dropdown frameless isSearchable items={serviceOptions} titleProp="label" selectedValue={serviceSelected} on:select={serviceSelectionChanged} />
  <Dropdown frameless isSearchable Item={MethodSelectItem} items={methodOptions} bind:selectedValue={methodSelected} on:select={methodSelectionChanged} />
  <div class="spacer" />
  <Button 
    text="Send"
    color="var(--primary-color)"
    on:click={onSend}
  />
</div>

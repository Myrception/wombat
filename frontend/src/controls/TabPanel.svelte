<script>
	import { getContext, onMount } from 'svelte';
	import { TABS } from './Tabs.svelte';

	const panel = {};
	const { registerPanel, selectedPanel } = getContext(TABS);

	registerPanel(panel);
	
</script>

{#if $selectedPanel === panel}
	<slot></slot>
{/if}

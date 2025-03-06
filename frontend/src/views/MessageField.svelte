<script>
  import MessageFieldRenderer from './MessageFieldRenderer.svelte';
  import FieldText from './FieldText.svelte';
  import FieldEnum from './FieldEnum.svelte';
  import FieldBool from './FieldBool.svelte';
  import FieldMessage from './FieldMessage.svelte';
  import FieldOneof from './FieldOneof.svelte';
  import RepeatedField from './RepeatedField.svelte';
  import FieldMap from './FieldMap.svelte';
  import FieldWellKnown from './FieldWellKnown.svelte';

  export let field = {};
  export let state;
  export let mapItems = {};
  export let oneof = false;
  export let idx = -1;
  export let key = undefined;

  const components = {
    FieldText,
    FieldEnum,
    FieldBool,
    FieldMessage,
    FieldOneof,
    RepeatedField,
    FieldMap,
    FieldWellKnown
  };
</script>

<MessageFieldRenderer
  {field}
  {state}
  {mapItems}
  {oneof}
  {idx}
  {key}
  {components}
  on:remove
/>

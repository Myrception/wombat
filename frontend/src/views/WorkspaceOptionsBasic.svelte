<script>
  import TextField from "../controls/TextField.svelte";
  import Checkbox from "../controls/Checkbox.svelte";
  import FileList from "../controls/FileList.svelte";

  import { FindProtoFiles, SelectDirectory } from '../../wailsjs/go/app/api';

  export let options = {
    protos: {},
  };

  const onFilesClear = () => options.protos.files = [];
  const onRootsClear = () => options.protos.roots = [];

  const onFilesAction = async () => {
    options.protos.files = options.protos.files || [];
    options.protos.files = await FindProtoFiles();
  }

  const onRootsAction = async () => {
    const dir = await SelectDirectory();
    options.protos.roots = options.protos.roots || [];
    options.protos.roots = [...options.protos.roots, dir];
  }
</script>

<style>
  .workspace-options-basic {
    flex-flow: column;
    padding: var(--padding) 0;
    width: 100%;
  }
  .protos {
    display: flex;
    flex-direction: column; 
    width: 100%;
  }
  @media (min-width: 768px) {
    .protos {
      flex-direction: row;
    }
  }
  .spacer {
    width: var(--padding);
    height: var(--padding);
  }
</style>

<div class="workspace-options-basic">
  <TextField label="gRPC server address:" bind:value={options.addr} />
  <Checkbox label="Use reflection to determine the RPC schema" rhs bind:checked={options.reflect} />
  <div class="protos">
    <FileList on:action={onFilesAction} on:clear={onFilesClear} files={options.protos.files} label="Proto source file(s):" actionText="Find *.proto files" actionColor="var(--primary-color)" />
    <div class="spacer" />
    <FileList on:action={onRootsAction} on:clear={onRootsClear} files={options.protos.roots} label="Import proto (root) path(s):" />
  </div>
</div>

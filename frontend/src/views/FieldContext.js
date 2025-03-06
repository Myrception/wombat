import { getContext, setContext } from 'svelte';

export const FIELD_RENDERER_KEY = Symbol('field-renderer');

export function setFieldRenderer(renderer) {
  setContext(FIELD_RENDERER_KEY, renderer);
}

export function getFieldRenderer() {
  return getContext(FIELD_RENDERER_KEY);
}

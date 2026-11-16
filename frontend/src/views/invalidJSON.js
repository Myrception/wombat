// A JSON-edited field (FieldStruct.svelte) whose text does not parse. It is
// kept in the request state instead of the value, so the text and the error
// stay with the field when list items move (e.g. one before it is removed).
//
// In JSON.stringify it is left out of an object (the field is not sent), and
// it becomes null in an array, so the frontend refuses to send a request
// with an invalid list item (see invalidListItems).
export class InvalidJSON {
  constructor(text, error) {
    this.text = text;
    this.error = error;
  }

  toJSON() {
    return undefined;
  }
}

/** The paths of the invalid list items in `state`, e.g. ["tree.children[1]"]. */
export function invalidListItems(value, path = "") {
  const out = [];
  if (Array.isArray(value)) {
    value.forEach((v, i) => {
      const at = `${path}[${i}]`;
      if (v instanceof InvalidJSON) out.push(at);
      else out.push(...invalidListItems(v, at));
    });
  } else if (value !== null && typeof value === "object" && !(value instanceof InvalidJSON)) {
    for (const [k, v] of Object.entries(value)) out.push(...invalidListItems(v, path ? `${path}.${k}` : k));
  }
  return out;
}

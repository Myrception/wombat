// Checks hand-written JSON for a recursive message (edited as JSON, see
// FieldStruct.svelte) against the schema, so that misspelled field names are
// reported instead of being dropped silently by the backend.
import { isJSONEdited, isWellKnown } from "./wellKnown";

/** All message types reachable from `message`, by full name. */
export function collectMessages(message, map = new Map()) {
  if (!message || message.recursive || !message.full_name || map.has(message.full_name)) return map;
  map.set(message.full_name, message);
  const visit = f => {
    if (f.message) collectMessages(f.message, map);
    if (f.map_value?.message) collectMessages(f.map_value.message, map);
    for (const m of f.oneof || []) visit(m);
  };
  for (const f of message.fields || []) visit(f);
  return map;
}

// Not checked: their JSON form does not follow their fields (Any holds
// "@type" plus the fields of the message it packs).
const UNCHECKED = new Set(["google.protobuf.Any"]);

const isObject = v => v !== null && typeof v === "object" && !Array.isArray(v);

/**
 * The paths of the fields in `value` that `message` does not have, e.g.
 * ["children[0].nmae"]. Values of the wrong type are left to the backend.
 */
export function unknownFields(value, message, messages, path = "") {
  if (message?.recursive || (message && !message.fields)) message = messages.get(message.full_name);
  if (!message || !isObject(value) || isWellKnown(message.full_name) || isJSONEdited(message) || UNCHECKED.has(message.full_name)) return [];

  const fields = new Map();
  for (const f of message.fields || []) {
    for (const m of f.kind === "oneof" ? f.oneof : [f]) {
      // The backend accepts the proto name and the JSON name (its
      // json_name option, or lowerCamelCase by default).
      fields.set(m.name, m);
      if (m.json_name) fields.set(m.json_name, m);
    }
  }

  const out = [];
  for (const [k, v] of Object.entries(value)) {
    const at = path ? `${path}.${k}` : k;
    const f = fields.get(k);
    if (!f) {
      out.push(at);
    } else if (f.kind === "map") {
      if (f.map_value?.message && isObject(v)) {
        for (const [mk, mv] of Object.entries(v)) out.push(...unknownFields(mv, f.map_value.message, messages, `${at}[${JSON.stringify(mk)}]`));
      }
    } else if (f.message) {
      if (f.repeated && Array.isArray(v)) v.forEach((item, i) => out.push(...unknownFields(item, f.message, messages, `${at}[${i}]`)));
      else if (!f.repeated) out.push(...unknownFields(v, f.message, messages, at));
    }
  }
  return out;
}

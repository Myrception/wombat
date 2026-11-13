// google.protobuf types that get a dedicated input instead of the generic
// message form (see FieldWellKnown.svelte).
const WELL_KNOWN = new Set([
  "google.protobuf.Timestamp",
  "google.protobuf.Duration",
  "google.protobuf.DoubleValue",
  "google.protobuf.FloatValue",
  "google.protobuf.Int64Value",
  "google.protobuf.UInt64Value",
  "google.protobuf.Int32Value",
  "google.protobuf.UInt32Value",
  "google.protobuf.BoolValue",
  "google.protobuf.StringValue",
  "google.protobuf.BytesValue",
  "google.protobuf.Struct",
]);

export const isWellKnown = fullName => WELL_KNOWN.has(fullName);

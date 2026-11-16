// User settings, stored in the webview's localStorage like the zoom level.
import { writable } from "svelte/store";

const MAX_MESSAGES_KEY = "wombat_max_messages";

export const DEFAULT_MAX_MESSAGES = 1000;
export const MIN_MAX_MESSAGES = 10;
export const MAX_MAX_MESSAGES = 100000;

/** A valid message limit for `v`, or undefined. */
export const parseMaxMessages = v => {
  const n = Number(v);
  if (!Number.isInteger(n) || n < MIN_MAX_MESSAGES || n > MAX_MAX_MESSAGES) return undefined;
  return n;
};

function load() {
  try {
    return parseMaxMessages(localStorage.getItem(MAX_MESSAGES_KEY) ?? "") ?? DEFAULT_MAX_MESSAGES;
  } catch (e) {
    return DEFAULT_MAX_MESSAGES; // storage unavailable
  }
}

/**
 * How many messages (sent and received) of a call the response pane keeps.
 * Older ones are removed, so a long or fast stream cannot slow the UI down
 * or use up memory.
 */
export const maxMessages = writable(load());

maxMessages.subscribe(n => {
  try {
    localStorage.setItem(MAX_MESSAGES_KEY, String(n));
  } catch (e) {
    // storage unavailable
  }
});

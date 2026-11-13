// Shared test setup: every test gets a page running the real frontend
// against the fake backend (tests/fake-backend.js).
import { test as base, expect } from "@playwright/test";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const schema = JSON.parse(readFileSync(new URL("./fixtures/schema.json", import.meta.url), "utf8"));
const fakeBackend = fileURLToPath(new URL("./fake-backend.js", import.meta.url));

export const test = base.extend({
  // Per-test fake backend configuration, e.g.
  // test.use({ fakeConfig: { saved: { "/pkg.Svc/Method": '{"a":"b"}' } } })
  fakeConfig: [{}, { option: true }],

  page: async ({ page, fakeConfig }, use) => {
    // Any uncaught error or console.error fails the test.
    const problems = [];
    page.on("pageerror", e => problems.push(`pageerror: ${e.message}`));
    page.on("console", m => {
      if (m.type() === "error") problems.push(`console.error: ${m.text()}`);
    });
    await page.addInitScript(([s, c]) => {
      window.__WOMBAT_SCHEMA__ = s;
      window.__WOMBAT_FAKE_CONFIG__ = c;
    }, [schema, fakeConfig]);
    await page.addInitScript({ path: fakeBackend });
    await use(page);
    expect(problems, "errors in the page").toEqual([]);
  },
});

export { expect };

export const METHODS = {
  AFoo: "/wombat.v1.Foobar/AFoo",
  Bar: "/wombat.v1.Foobar/Bar",
  Baz: "/wombat.v1.Foobar/Baz",
  Foo: "/wombat.v1.Foobar/Foo",
  WellKnown: "/wombat.v1.Foobar/WellKnown",
  SingleOneof: "/wombat.v1.Foobar/SingleOneof",
  GetFeature: "/wombat.v1.RouteGuide/GetFeature",
  ListFeatures: "/wombat.v1.RouteGuide/ListFeatures",
};

/** Waits until the form for method `fullName` is shown. */
export async function expectMethod(page, fullName) {
  await expect(page.locator(".method-input h2")).toHaveText(schema.methods[fullName].message.full_name);
  expect(await page.evaluate(() => window.__fake.callsTo("SelectMethod").at(-1)?.args[0])).toBe(fullName);
}

/** Opens the app and waits until the first method's form is shown. */
export async function openApp(page) {
  await page.goto("/");
  await expectMethod(page, METHODS.AFoo);
}

/** The request form. */
export const form = page => page.locator(".method-input > .fields");

/** The <input> of the text field labelled `name` inside `scope`. */
export const textInput = (scope, name) =>
  scope
    .locator(".text-field")
    .filter({ has: scope.page().locator(".input-label > span:not(.hint)", { hasText: new RegExp(`^${name}$`) }) })
    .locator("input");

/** Picks `label` in the service (0) or method (1) dropdown. */
export async function pick(page, dropdown, label) {
  await page.locator(".method-select .svelte-select").nth(dropdown).click();
  await page.locator(".svelte-select-list .list-item").filter({ hasText: new RegExp(`^\\s*${label}\\b`) }).first().click();
}

/** Selects a method through the UI, like a user would. */
export async function selectMethod(page, fullName) {
  const [, service, method] = fullName.split("/");
  const current = await page.locator(".method-select .svelte-select").nth(0).innerText();
  if (!current.includes(service)) await pick(page, 0, service);
  await pick(page, 1, method);
  await expectMethod(page, fullName);
}

/** Clicks Send and returns the request the backend received. */
export async function send(page) {
  const before = await page.evaluate(() => window.__fake.callsTo("Send").length);
  await page.locator(".method-select").getByRole("button", { name: "Send", exact: true }).click();
  await expect.poll(() => page.evaluate(() => window.__fake.callsTo("Send").length)).toBe(before + 1);
  const [method, json, metadata] = await page.evaluate(() => window.__fake.callsTo("Send").at(-1).args);
  return { method, body: JSON.parse(json), metadata };
}

// Matches the whole text, ignoring surrounding whitespace.
const exact = name => new RegExp(`^\\s*${name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\s*$`);
const label = (page, name) => page.locator(".input-label > span:not(.hint)", { hasText: exact(name) });

/** The label row of a message, repeated, map or well-known field. */
export const section = (scope, name) => scope.locator(".msg-label").filter({ has: label(scope.page(), name) });

/** The contents of that field (nested fields, list items or map pairs). */
export const sectionBody = (scope, name) =>
  section(scope, name).locator("xpath=following-sibling::*[1][contains(@class,'fields') or contains(@class,'pairs')]");

/** Ticks or unticks the checkbox of an optional message / well-known field. */
export const toggle = (scope, name) => section(scope, name).locator(".checkbox label").click();

/** Clicks the + of a repeated or map field. */
export const addItem = (scope, name) => section(scope, name).locator(":scope > button").click();

/** Clicks a bool field's checkbox. */
export const toggleBool = (scope, name) => scope.locator(".field-bool label", { hasText: exact(name) }).click();

/** Opens a dropdown and picks the option `option`. */
export async function choose(dropdown, option) {
  await dropdown.locator(".svelte-select").click();
  await dropdown.page().locator(".svelte-select-list .list-item").filter({ hasText: exact(option) }).click();
}

/** The dropdown of the enum field `name`. */
export const enumDropdown = (scope, name) => scope.locator(".dropdown").filter({ has: label(scope.page(), name) });

/** The dropdown of the oneof `name`. */
export const oneofDropdown = (scope, name) =>
  scope
    .locator(".input-label")
    .filter({ hasText: exact(`oneof ${name}`) })
    .locator("xpath=following-sibling::*[1][contains(@class,'dropdown')]");

/** The <textarea> of the bytes/multiline field `name`. */
export const textArea = (scope, name) =>
  scope.locator(".text-area").filter({ has: label(scope.page(), name) }).locator("textarea");

/** The input shown below an enabled optional field (StringValue, BytesValue, Struct). */
export const optionalInput = (scope, name) =>
  section(scope, name).locator(
    "xpath=following-sibling::*[1][contains(@class,'text-field') or contains(@class,'text-area')]//*[self::input or self::textarea]",
  );

/** Picks `option` of the radio field `name` (BoolValue). */
export const chooseRadio = (scope, name, option) =>
  scope
    .locator("div")
    .filter({ has: scope.page().locator(":scope > .input-label", { hasText: name }) })
    .locator(".radio label", { hasText: exact(option) })
    .click();

/**
 * The button with text `text`. Use this inside dialogs: svelte-simple-modal
 * 1.x marks its backdrop aria-hidden, which hides the whole dialog from
 * getByRole().
 */
export const button = (scope, text) => scope.locator("button").filter({ hasText: exact(text) });

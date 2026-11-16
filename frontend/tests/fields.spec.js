// Every field type: edit it in the form, Send, and check the request JSON.
import {
  test, expect, openApp, form, textInput, textArea, selectMethod, send, METHODS,
  sectionBody, toggle, addItem, toggleBool, choose, enumDropdown, oneofDropdown, optionalInput, chooseRadio,
} from "./app.js";

test.beforeEach(async ({ page }) => openApp(page));

test("scalars, bool, enum and bytes", async ({ page }) => {
  await selectMethod(page, METHODS.Foo);
  const f = form(page);
  await textInput(f, "type_string").fill("hello");
  await textInput(f, "type_int32").fill("-7");
  await textInput(f, "type_double").fill("1.5");
  await textInput(f, "type_uint64").fill("18446744073709551615");
  await toggleBool(f, "type_bool");
  await choose(enumDropdown(f, "type_enum"), "BAR");
  await textArea(f, "type_bytes").fill("aGk=");

  const { body } = await send(page);
  expect(body).toMatchObject({
    type_string: "hello",
    type_int32: "-7",
    type_double: "1.5",
    type_uint64: "18446744073709551615",
    type_bool: true,
    type_enum: "BAR",
    type_bytes: "aGk=",
  });
});

test("nested message can be enabled, filled and disabled", async ({ page }) => {
  await selectMethod(page, METHODS.Foo);
  const f = form(page);
  await expect(sectionBody(f, "type_message")).toHaveCount(0);

  await toggle(f, "type_message");
  await textInput(sectionBody(f, "type_message"), "id").fill("abc");
  expect((await send(page)).body.type_message).toMatchObject({ id: "abc" });

  await toggle(f, "type_message");
  await expect(sectionBody(f, "type_message")).toHaveCount(0);
  expect((await send(page)).body.type_message).toBeNull();
});

test("repeated scalars can be added and removed", async ({ page }) => {
  await selectMethod(page, METHODS.Bar);
  const f = form(page);
  await addItem(f, "type_repeated_string");
  await addItem(f, "type_repeated_string");
  const items = sectionBody(f, "type_repeated_string").locator("input");
  await expect(items).toHaveCount(2);
  await items.nth(0).fill("first");
  await items.nth(1).fill("second");
  expect((await send(page)).body.type_repeated_string).toEqual(["first", "second"]);

  // Remove the first item.
  await sectionBody(f, "type_repeated_string").locator(".text-field button").first().click();
  await expect(items).toHaveCount(1);
  await expect(items.first()).toHaveValue("second");
  expect((await send(page)).body.type_repeated_string).toEqual(["second"]);
});

test("repeated messages", async ({ page }) => {
  await selectMethod(page, METHODS.Bar);
  const f = form(page);
  await addItem(f, "type_repeated_message");
  await textInput(sectionBody(f, "type_repeated_message"), "id").fill("x1");
  expect((await send(page)).body.type_repeated_message).toEqual([expect.objectContaining({ id: "x1" })]);
});

test("map entries", async ({ page }) => {
  await openApp(page); // AFoo is selected first
  const f = form(page);
  await addItem(f, "map_string_string");
  const pairs = sectionBody(f, "map_string_string");
  await textInput(pairs, "key").fill("colour");
  await textInput(pairs, "value").fill("blue");
  expect((await send(page)).body.map_string_string).toEqual({ colour: "blue" });

  // Renaming the key moves the value.
  await textInput(pairs, "key").fill("color");
  expect((await send(page)).body.map_string_string).toEqual({ color: "blue" });
});

test("oneof: choosing a member and switching to another", async ({ page }) => {
  await selectMethod(page, METHODS.Baz);
  const f = form(page);
  await choose(oneofDropdown(f, "request"), "baz");
  await textInput(f, "baz").fill("hi");
  expect((await send(page)).body).toEqual({ baz: "hi" });

  await choose(oneofDropdown(f, "request"), "bar");
  const { body } = await send(page);
  expect(body.baz).toBeUndefined();
  expect(body.bar).toBeDefined();
});

test("well-known types get dedicated inputs", async ({ page }) => {
  await selectMethod(page, METHODS.WellKnown);
  const f = form(page);
  await expect(textInput(f, "timestamp")).toHaveAttribute("placeholder", "2006-01-02T15:04:05.000Z");
  await textInput(f, "timestamp").fill("2026-11-13T10:00:00.000Z");
  await textInput(f, "duration").fill("1.5s");
  await textInput(f, "int32_value").fill("42");
  await chooseRadio(f, "bool_value", "true");
  await toggle(f, "string_value");
  await optionalInput(f, "string_value").fill("wrapped");
  await toggle(f, "struct_value");
  await optionalInput(f, "struct_value").fill('{"a":1,"b":[true,null]}');
  await optionalInput(f, "struct_value").blur();

  const { body } = await send(page);
  expect(body).toMatchObject({
    timestamp: "2026-11-13T10:00:00.000Z",
    duration: "1.5s",
    int32_value: "42",
    bool_value: true,
    string_value: "wrapped",
    struct_value: { a: 1, b: [true, null] },
  });
  // Optional fields that were left off are not sent.
  expect(body.bytes_value).toBeUndefined();
});

test("Value and ListValue are edited as JSON", async ({ page }) => {
  await selectMethod(page, METHODS.WellKnown);
  const f = form(page);
  await toggle(f, "value");
  await optionalInput(f, "value").fill('{"x":[1,"a"]}');
  await optionalInput(f, "value").blur();
  await toggle(f, "list_value");
  await expect(optionalInput(f, "list_value")).toHaveValue("[]"); // starts as a valid list
  await optionalInput(f, "list_value").fill('[1,"two",null]');
  await optionalInput(f, "list_value").blur();

  const { body } = await send(page);
  expect(body.value).toEqual({ x: [1, "a"] });
  expect(body.list_value).toEqual([1, "two", null]);
});

test("invalid JSON is reported next to the field, without a dialog", async ({ page }) => {
  await selectMethod(page, METHODS.WellKnown);
  const f = form(page);
  await toggle(f, "struct_value");
  await optionalInput(f, "struct_value").fill("{nope");
  await optionalInput(f, "struct_value").blur();
  await expect(f.getByRole("alert")).toContainText("Invalid JSON");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect((await send(page)).body.struct_value).toBeUndefined();

  await optionalInput(f, "struct_value").fill('{"ok":true}');
  await optionalInput(f, "struct_value").blur();
  await expect(f.getByRole("alert")).toHaveCount(0);
  expect((await send(page)).body.struct_value).toEqual({ ok: true });
});

test("recursive messages: the nested occurrence is edited as JSON", async ({ page }) => {
  await selectMethod(page, METHODS.Recursive);
  const f = form(page);
  await toggle(f, "tree");
  const tree = sectionBody(f, "tree");
  await textInput(tree, "name").fill("root");
  await addItem(tree, "children");
  const child = sectionBody(tree, "children").locator("textarea");
  await child.fill('{"name":"leaf","children":[{"name":"deeper"}]}');
  await child.blur();
  await toggle(f, "self");
  await optionalInput(f, "self").fill('{"tree":{"name":"inner"}}');
  await optionalInput(f, "self").blur();

  const { body } = await send(page);
  expect(body.tree).toMatchObject({ name: "root", children: [{ name: "leaf", children: [{ name: "deeper" }] }] });
  expect(body.self).toEqual({ tree: { name: "inner" } });
});

test("timestamp picker in a list and in a map", async ({ page }) => {
  await selectMethod(page, METHODS.WellKnown);
  const f = form(page);
  // Opens on the month of the current value; noon, so the day is the same in every time zone.
  const pick = async (scope, day) => {
    await scope.locator(".timestamp > button").click();
    await scope.locator(".calendar").getByRole("button", { name: String(day), exact: true }).click();
  };

  await addItem(f, "timestamps");
  const list = sectionBody(f, "timestamps");
  await list.locator("input").fill("2026-03-01T12:00:00.000Z");
  await pick(list, 15);
  await expect(list.locator("input")).toHaveValue("2026-03-15T00:00:00.000Z");

  await addItem(f, "timestamp_map");
  const map = sectionBody(f, "timestamp_map");
  await textInput(map, "key").fill("start");
  await textInput(map, "value").fill("2026-05-02T12:00:00.000Z");
  await pick(map, 20);
  await expect(textInput(map, "value")).toHaveValue("2026-05-20T00:00:00.000Z");

  const { body } = await send(page);
  expect(body.timestamps).toEqual(["2026-03-15T00:00:00.000Z"]);
  expect(body.timestamp_map).toEqual({ start: "2026-05-20T00:00:00.000Z" });
});

test("invalid JSON in a Value field is not sent as null", async ({ page }) => {
  await selectMethod(page, METHODS.WellKnown);
  const f = form(page);
  await toggle(f, "value");
  await optionalInput(f, "value").fill("{oops");
  await optionalInput(f, "value").blur();
  await expect(f.getByRole("alert")).toContainText("not sent");
  expect("value" in (await send(page)).body).toBe(false);
});

test("JSON list items: start valid, and removing one keeps the text with its value", async ({ page }) => {
  await selectMethod(page, METHODS.Recursive);
  const f = form(page);
  await toggle(f, "tree");
  const tree = sectionBody(f, "tree");
  await addItem(tree, "children");
  await addItem(tree, "children");
  const items = sectionBody(tree, "children").locator("textarea");
  await expect(items).toHaveCount(2);
  await expect(items.nth(0)).toHaveValue("{}"); // not null
  // A list item has no checkbox: it is removed with its cross.
  await expect(sectionBody(tree, "children").locator(".checkbox")).toHaveCount(0);
  expect((await send(page)).body.tree.children).toEqual([{}, {}]);

  await items.nth(0).fill('{"name":"first"}');
  await items.nth(0).blur();
  await items.nth(1).fill('{"name":"second"}');
  await items.nth(1).blur();
  await sectionBody(tree, "children").getByRole("button", { name: "Remove" }).first().click();
  await expect(items).toHaveCount(1);
  await expect(items.first()).toHaveValue('{"name":"second"}');
  expect((await send(page)).body.tree.children).toEqual([{ name: "second" }]);
});

test("unknown field names in a recursive message's JSON are reported", async ({ page }) => {
  await selectMethod(page, METHODS.Recursive);
  const f = form(page);
  await toggle(f, "self");
  await optionalInput(f, "self").fill('{"tree":{"nmae":"x","children":[{"name":"ok","childs":[]}]},"forest":{"a":{"name":"y"}}}');
  await optionalInput(f, "self").blur();
  await expect(f.getByRole("status")).toHaveText('Unknown fields "tree.nmae", "tree.children[0].childs", not sent');

  await optionalInput(f, "self").fill('{"tree":{"name":"x","children":[]},"bar":{"id":"1"}}');
  await optionalInput(f, "self").blur();
  await expect(f.getByRole("status")).toHaveCount(0);
});

test("an invalid list item keeps its text when an item before it is removed, and blocks Send", async ({ page }) => {
  await selectMethod(page, METHODS.Recursive);
  const f = form(page);
  await toggle(f, "tree");
  const tree = sectionBody(f, "tree");
  await addItem(tree, "children");
  await addItem(tree, "children");
  const items = sectionBody(tree, "children").locator("textarea");
  await items.nth(0).fill('{"name":"first"}');
  await items.nth(0).blur();
  await items.nth(1).fill("{oops");
  await items.nth(1).blur();
  await expect(f.getByRole("alert")).toContainText("cannot be sent");

  await sectionBody(tree, "children").getByRole("button", { name: "Remove" }).first().click();
  await expect(items).toHaveCount(1);
  await expect(items.first()).toHaveValue("{oops");
  await expect(f.getByRole("alert")).toContainText("cannot be sent");

  // Send is refused in the frontend, naming the item.
  const sends = () => page.evaluate(() => window.__fake.callsTo("Send").length);
  const before = await sends();
  await page.locator(".method-select").getByRole("button", { name: "Send", exact: true }).click();
  await expect(page.locator(".error-box")).toContainText("tree.children[0]");
  expect(await sends()).toBe(before);
  await page.locator(".error-box").getByRole("button", { name: "OK" }).click();

  await items.first().fill('{"name":"fixed"}');
  await items.first().blur();
  expect((await send(page)).body.tree.children).toEqual([{ name: "fixed" }]);
});

test("the field-name check uses the real JSON name", async ({ page }) => {
  await selectMethod(page, METHODS.Recursive);
  const f = form(page);
  await toggle(f, "self");
  // display_name has [json_name = "label"]: "label" and "display_name" are accepted, "displayName" is not.
  await optionalInput(f, "self").fill('{"tree":{"label":"a","display_name":"b","displayName":"c"}}');
  await optionalInput(f, "self").blur();
  await expect(f.getByRole("status")).toHaveText('Unknown field "tree.displayName", not sent');
});

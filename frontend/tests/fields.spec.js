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

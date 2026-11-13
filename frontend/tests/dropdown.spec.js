// The dropdowns (svelte-select behind our controls/Dropdown.svelte).
import {
  test, expect, openApp, form, textInput, selectMethod, pick, expectMethod, send, choose, enumDropdown, oneofDropdown, METHODS,
} from "./app.js";

const selectMethodCalls = page => page.evaluate(() => window.__fake.callsTo("SelectMethod").map(c => c.args[0]));

for (const zoom of [1, 1.5]) {
  test(`the list opens right below the select at ${zoom * 100}% zoom`, async ({ page }) => {
    await page.addInitScript(z => localStorage.setItem("wombat_zoom_level", String(z)), zoom);
    await openApp(page);
    await selectMethod(page, METHODS.Foo);
    const dropdown = enumDropdown(form(page), "type_enum");
    await dropdown.scrollIntoViewIfNeeded();
    const select = dropdown.locator(".svelte-select");
    await select.click();
    const list = dropdown.locator(".svelte-select-list");
    await expect(list).toBeVisible();

    const s = await select.boundingBox();
    const l = await list.boundingBox();
    expect(Math.abs(l.y - (s.y + s.height))).toBeLessThan(3 * zoom);
    // Within a pixel: zoomed layouts round differently.
    expect(Math.abs(l.x - s.x)).toBeLessThanOrEqual(1);
    expect(Math.abs(l.width - s.width)).toBeLessThanOrEqual(1);
  });
}

test("picking the current method again keeps the form", async ({ page }) => {
  await openApp(page);
  await selectMethod(page, METHODS.GetFeature);
  await textInput(form(page), "latitude").fill("12");
  const before = await selectMethodCalls(page);

  await pick(page, 1, "GetFeature");
  await pick(page, 0, "wombat.v1.RouteGuide");
  await expect(textInput(form(page), "latitude")).toHaveValue("12");
  expect(await selectMethodCalls(page)).toEqual(before);
});

test("each pick loads the method exactly once", async ({ page }) => {
  await openApp(page);
  await pick(page, 0, "wombat.v1.RouteGuide");
  await expectMethod(page, METHODS.GetFeature);
  await pick(page, 1, "ListFeatures");
  await expectMethod(page, METHODS.ListFeatures);
  expect(await selectMethodCalls(page)).toEqual([METHODS.AFoo, METHODS.GetFeature, METHODS.ListFeatures]);
});

test("services can be searched", async ({ page }) => {
  await openApp(page);
  await page.locator(".method-select .svelte-select").first().click();
  await page.keyboard.type("Route");
  const items = page.locator(".svelte-select-list .list-item");
  await expect(items).toHaveCount(1);
  await expect(items).toHaveText("wombat.v1.RouteGuide");
  await page.keyboard.press("Enter");
  await expectMethod(page, METHODS.GetFeature);
});

test("a oneof member can be cleared", async ({ page }) => {
  await openApp(page);
  await selectMethod(page, METHODS.Baz);
  const oneof = oneofDropdown(form(page), "request");
  await choose(oneof, "baz");
  await textInput(form(page), "baz").fill("x");
  expect((await send(page)).body).toEqual({ baz: "x" });

  await oneof.locator(".clear-select").click();
  await expect(textInput(form(page), "baz")).toHaveCount(0);
  expect((await send(page)).body).toEqual({});
});

test("dropdowns match the form styling", async ({ page }) => {
  await openApp(page);
  await selectMethod(page, METHODS.Foo);
  const style = async (locator, props) =>
    locator.evaluate((el, props) => Object.fromEntries(props.map(p => [p, getComputedStyle(el)[p]])), props);

  const field = await style(textInput(form(page), "type_string"), ["fontSize", "height"]);
  const enumSelect = await style(enumDropdown(form(page), "type_enum").locator(".svelte-select"), ["height", "fontSize", "borderTopStyle", "borderTopWidth"]);
  // Same height as a text input (within a pixel: it depends on font metrics).
  expect(Math.abs(parseFloat(enumSelect.height) - parseFloat(field.height))).toBeLessThanOrEqual(1);
  expect(enumSelect).toMatchObject({ fontSize: field.fontSize, borderTopStyle: "solid", borderTopWidth: "1px" });
  const fieldFont = field.fontSize;

  // The service/method pickers in the toolbar are frameless.
  const service = await style(page.locator(".method-select .svelte-select").first(), ["borderTopStyle", "fontSize"]);
  expect(service).toEqual({ borderTopStyle: "none", fontSize: fieldFont });
});

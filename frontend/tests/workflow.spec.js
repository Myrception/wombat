// Method selection, metadata, the response panel and grpcurl import/export.
import { test, expect, openApp, form, textInput, selectMethod, pick, expectMethod, send, button, METHODS } from "./app.js";

const tab = (page, name) => page.locator(".input-pane").getByRole("button", { name, exact: true });

test.beforeEach(async ({ page }) => openApp(page));

test("picking a service selects its first method", async ({ page }) => {
  // svelte-select 3 fires `select` for programmatic changes, which this
  // relies on; svelte-select 5+ does not (see the Phase B plan).
  await pick(page, 0, "wombat.v1.RouteGuide");
  await expectMethod(page, METHODS.GetFeature);
  await expect(textInput(form(page), "latitude")).toBeVisible();
});

test("picking a method shows its form", async ({ page }) => {
  await selectMethod(page, METHODS.Foo);
  await expect(textInput(form(page), "type_string")).toBeVisible();
  await selectMethod(page, METHODS.Bar);
  await expect(textInput(form(page), "type_string")).toHaveCount(0);
});

test("request metadata is sent", async ({ page }) => {
  await selectMethod(page, METHODS.GetFeature);
  await tab(page, "Metadata").click();
  const meta = page.locator(".request-metadata");
  await meta.getByPlaceholder("key").first().fill("authorization");
  await meta.getByPlaceholder("value").first().fill("Bearer abc");
  await tab(page, "Request").click();

  const { metadata } = await send(page);
  expect(metadata.filter(m => m.key)).toEqual([{ key: "authorization", val: "Bearer abc" }]);
});

test.describe("with saved metadata", () => {
  test.use({ fakeConfig: { metadata: [{ key: "x-team", val: "core" }] } });

  test("it is restored on connect and sent", async ({ page }) => {
    await tab(page, "Metadata").click();
    await expect(page.locator(".request-metadata").getByPlaceholder("key").first()).toHaveValue("x-team");
    await tab(page, "Request").click();
    const { metadata } = await send(page);
    expect(metadata.filter(m => m.key)).toEqual([{ key: "x-team", val: "core" }]);
  });
});

test("the response and status are shown", async ({ page }) => {
  await selectMethod(page, METHODS.GetFeature);
  await textInput(form(page), "latitude").fill("7");
  await send(page);
  await expect(page.locator(".response .view-lines")).toContainText('echo: {"latitude":"7"');
  await expect(page.locator(".output-header")).toContainText("OK");
});

test("Edit as grpcurl shows the command, and Import loads it back", async ({ page }) => {
  await selectMethod(page, METHODS.GetFeature);
  await textInput(form(page), "latitude").fill("5");
  await page.locator(".method-input").getByRole("button", { name: "Edit", exact: true }).click();

  const panel = page.locator(".code-edit-panel");
  await expect(panel.locator("h1")).toHaveText("Edit as GRPCURL");
  await expect(panel.locator(".view-lines")).toContainText("grpcurl -d");
  await expect(panel.locator(".view-lines")).toContainText("wombat.v1.RouteGuide/GetFeature");

  await button(panel, "Import").click();
  await expect(panel).toHaveCount(0);
  const [kind, command] = await page.evaluate(() => window.__fake.callsTo("ImportCommand").at(-1).args);
  expect(kind).toBe("grpcurl");
  expect(command).toContain('"latitude":"5"');
  await expectMethod(page, METHODS.GetFeature);
  await expect(textInput(form(page), "latitude")).toHaveValue("5");
});

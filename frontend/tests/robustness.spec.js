// Edge cases: no connection, deleting workspaces, imports, quick changes.
import { test, expect, openApp, form, textInput, selectMethod, expectMethod, send, button, METHODS } from "./app.js";

const sendButton = page => page.locator(".method-select").getByRole("button", { name: "Send", exact: true });
const tab = (page, name) => page.locator(".input-pane").getByRole("button", { name, exact: true });

test.describe("not connected", () => {
  test.use({ fakeConfig: { addr: "" } });

  test("Send is disabled without a method", async ({ page }) => {
    await page.goto("/");
    await expect(sendButton(page)).toBeDisabled();
  });
});

test.describe("with another workspace", () => {
  test.use({ fakeConfig: { workspaces: [{ id: "wksp_other", addr: "other:1" }] } });

  test("deleting a workspace asks first", async ({ page }) => {
    await openApp(page);
    const deletes = () => page.evaluate(() => window.__fake.callsTo("DeleteWorkspace").length);
    await page.locator(".workspace-select").click();
    const overlay = page.locator(".overlay");
    const row = overlay.locator("tr", { hasText: "other:1" });

    await row.getByRole("button", { name: "Delete workspace" }).click();
    await expect(row).toContainText("Delete other:1?");
    await button(row, "Keep").click();
    await expect(row.getByRole("button", { name: "Delete workspace" })).toBeVisible();
    await expect(overlay).toBeVisible(); // nothing was selected
    expect(await deletes()).toBe(0);

    await row.getByRole("button", { name: "Delete workspace" }).click();
    await button(row, "Delete").click();
    await expect.poll(deletes).toBe(1);
    expect(await page.evaluate(() => window.__fake.callsTo("DeleteWorkspace")[0].args)).toEqual(["wksp_other"]);
    expect(await page.evaluate(() => window.__fake.callsTo("SelectWorkspace").length)).toBe(0);
  });
});

test.describe("with a saved request", () => {
  test.use({ fakeConfig: { saved: { "/wombat.v1.RouteGuide/GetFeature": '{"latitude":9}' } } });

  test("an imported command without -d is an empty request with the command's headers", async ({ page }) => {
    await openApp(page);
    await page.evaluate(() =>
      window.go.app.api.ImportCommand("grpcurl", "grpcurl -H 'x-team: core' -plaintext localhost:1 wombat.v1.RouteGuide/GetFeature"));
    await expectMethod(page, METHODS.GetFeature);
    await expect(textInput(form(page), "latitude")).toHaveValue(""); // not the saved 9
    await tab(page, "Metadata").click();
    await expect(page.locator(".request-metadata").getByPlaceholder("key").first()).toHaveValue("x-team");
  });

  test("a normal method change keeps the request headers", async ({ page }) => {
    await openApp(page);
    await tab(page, "Metadata").click();
    await page.locator(".request-metadata").getByPlaceholder("key").first().fill("x-keep");
    await tab(page, "Request").click();
    await selectMethod(page, METHODS.GetFeature);
    await expect(textInput(form(page), "latitude")).toHaveValue("9");
    const { metadata } = await send(page);
    expect(metadata.filter(m => m.key).map(m => m.key)).toEqual(["x-keep"]);
  });
});

test.describe("slow saved state", () => {
  test.use({
    fakeConfig: {
      saved: { "/wombat.v1.Foobar/Foo": '{"type_string":"from Foo"}', "/wombat.v1.Foobar/Bar": '{"type_repeated_string":["b"]}' },
      rawStateDelay: { "/wombat.v1.Foobar/Foo": 400 },
    },
  });

  test("a quick second method change wins", async ({ page }) => {
    await openApp(page);
    await selectMethod(page, METHODS.Foo); // its saved state takes 400 ms
    await selectMethod(page, METHODS.Bar);
    await page.waitForTimeout(600);
    await expectMethod(page, METHODS.Bar);
    const { body } = await send(page);
    expect(body.type_repeated_string).toEqual(["b"]);
    expect(body.type_string).toBeUndefined(); // Foo's late state did not land
  });
});

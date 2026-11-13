import { test, expect, openApp, form, textInput, selectMethod, send, METHODS } from "./app.js";

test("startup selects the first service and method", async ({ page }) => {
  await openApp(page);
  await expect(page.locator(".connection")).toContainText("localhost:5001");
  await expect(page.locator(".connection h3")).toHaveText("ready");
  const selects = await page.evaluate(() => window.__fake.callsTo("SelectMethod").map(c => c.args[0]));
  expect(selects).toEqual([METHODS.AFoo]);
});

test("typing into a text field is sent", async ({ page }) => {
  await openApp(page);
  await selectMethod(page, METHODS.GetFeature);
  await textInput(form(page), "latitude").fill("123");
  await textInput(form(page), "longitude").pressSequentially("-45");
  const { method, body } = await send(page);
  expect(method).toBe(METHODS.GetFeature);
  expect(body).toEqual({ latitude: "123", longitude: "-45" });
});

test.describe("with a saved request", () => {
  test.use({ fakeConfig: { saved: { "/wombat.v1.RouteGuide/GetFeature": '{"latitude":"1","longitude":"2"}' } } });

  test("the saved request is shown and edits to it are sent", async ({ page }) => {
    await openApp(page);
    await selectMethod(page, METHODS.GetFeature);
    const lat = textInput(form(page), "latitude");
    await expect(lat).toHaveValue("1");
    await expect(textInput(form(page), "longitude")).toHaveValue("2");

    await lat.fill("99");
    const { body } = await send(page);
    expect(body).toEqual({ latitude: "99", longitude: "2" });
    // The field keeps the edit after sending.
    await expect(lat).toHaveValue("99");
  });
});

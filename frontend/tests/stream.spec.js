// Long streams: only the last N messages are kept (Settings).
import { test, expect, openApp, selectMethod, send, METHODS, button } from "./app.js";

test.use({ fakeConfig: { streamCount: 1500 } });
test.beforeEach(async ({ page }) => openApp(page));

const removedNote = page => page.locator(".output-header .removed");
const lines = page => page.locator(".response .view-lines");
const openStats = page => page.locator(".output-pane").getByRole("button", { name: "Statistics", exact: true }).click();

test("a long stream keeps the last 1000 messages", async ({ page }) => {
  await selectMethod(page, METHODS.ListFeatures);
  await send(page);
  // 1 sent + 1500 received: the sent one and the first 500 received are gone.
  await expect(removedNote(page)).toHaveText("501 older messages removed");
  await expect(lines(page)).toContainText("n: 501");
  await expect(lines(page)).not.toContainText(/n: 500(?!\d)/);
  await expect(page.locator(".output-header")).toContainText("OK");

  await openStats(page);
  const stats = page.locator(".stats");
  await expect(stats.locator(".removed")).toHaveText("501 older payload statistics removed");
  await expect(stats.getByText(/^« Payload:/)).toHaveCount(1000);
  await expect(stats.getByText(/^« Payload:/).first()).toHaveText("« Payload: n: 501");
  await expect(stats.getByText("Begin time:", { exact: false })).toHaveCount(1); // kept
});

test("the limit is set in Settings, applies at once and is remembered", async ({ page }) => {
  await selectMethod(page, METHODS.ListFeatures);
  await send(page);
  await expect(removedNote(page)).toHaveText("501 older messages removed");

  await page.getByRole("button", { name: "Settings" }).click();
  const dialog = page.getByRole("dialog", { name: "Settings" });
  const input = dialog.locator("input");
  await expect(input).toHaveValue("1000");
  await input.fill("5");
  await expect(button(dialog, "Save")).toBeDisabled();
  await input.fill("10");
  await button(dialog, "Save").click();
  await expect(dialog).toHaveCount(0);

  // Applies to the current response right away.
  await expect(removedNote(page)).toHaveText("1,491 older messages removed");
  await expect(lines(page)).toContainText("n: 1491");
  await expect(lines(page)).not.toContainText(/n: 1490(?!\d)/);

  await page.reload();
  await page.getByRole("button", { name: "Settings" }).click();
  await expect(page.getByRole("dialog", { name: "Settings" }).locator("input")).toHaveValue("10");
  await button(page.getByRole("dialog", { name: "Settings" }), "Default").click();
  await expect(page.getByRole("dialog", { name: "Settings" }).locator("input")).toHaveValue("1000");
});

test.describe("a short response", () => {
  test.use({ fakeConfig: { streamCount: 1 } });

  test("removes nothing", async ({ page }) => {
    await selectMethod(page, METHODS.GetFeature);
    await send(page);
    await expect(lines(page)).toContainText("echo:");
    await expect(removedNote(page)).toHaveCount(0);
  });
});

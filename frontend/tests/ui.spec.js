// Dialogs, keyboard access and zoom.
import { test, expect, openApp, button } from "./app.js";

test.beforeEach(async ({ page }) => openApp(page));

const insideViewport = async (page, locator) => {
  const box = await locator.boundingBox();
  const vp = page.viewportSize();
  expect(box.x).toBeGreaterThanOrEqual(0);
  expect(box.y).toBeGreaterThanOrEqual(0);
  expect(box.x + box.width).toBeLessThanOrEqual(vp.width + 1);
  expect(box.y + box.height).toBeLessThanOrEqual(vp.height + 1);
};

test("workspace dialog opens, shows TLS options side by side, and closes", async ({ page }) => {
  await page.getByRole("button", { name: "Workspace", exact: true }).click();
  const dialog = page.locator(".workspace-options");
  await expect(dialog.locator("h1")).toHaveText("Workspace");
  await insideViewport(page, page.locator(".window"));

  await button(dialog, "TLS").click();
  const certs = dialog.locator(".client-certs > *");
  await expect(certs).toHaveCount(2);
  const [a, b] = [await certs.nth(0).boundingBox(), await certs.nth(1).boundingBox()];
  expect(Math.abs(a.y - b.y)).toBeLessThan(1); // same row
  expect(b.x).toBeGreaterThan(a.x + a.width); // with a gap

  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
});

test("workspace switcher works with mouse and keyboard", async ({ page }) => {
  const header = page.locator(".workspace-select");
  await expect(header).toHaveAttribute("role", "button");
  await header.click();
  await expect(page.locator(".overlay")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.locator(".overlay")).toHaveCount(0);

  await header.focus();
  await page.keyboard.press("Enter");
  await expect(page.locator(".overlay")).toBeVisible();
});

test("zoom with the keyboard, persisted across restarts", async ({ page }) => {
  const level = page.locator(".zoom-level");
  await expect(level).toHaveText("100%");
  await page.keyboard.press("Control+=");
  await page.keyboard.press("Control+=");
  await expect(level).toHaveText("120%");
  expect(await page.evaluate(() => document.documentElement.style.zoom)).toBe("1.2");

  await page.reload();
  await expect(level).toHaveText("120%");

  await page.keyboard.press("Control+0");
  await expect(level).toHaveText("100%");
});

test("dialogs stay on screen when zoomed in", async ({ page }) => {
  for (let i = 0; i < 5; i++) await page.keyboard.press("Control+=");
  await expect(page.locator(".zoom-level")).toHaveText("150%");
  await page.getByRole("button", { name: "Workspace", exact: true }).click();
  await expect(page.locator(".workspace-options h1")).toBeVisible();
  await insideViewport(page, page.locator(".window"));
});

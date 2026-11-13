// The Monaco editors under UI zoom: Monaco handles CSS zoom itself since
// 0.5x (0.21 needed a workaround and still put clicks in the wrong place).
import { test, expect, openApp, form, textInput, selectMethod, button, METHODS } from "./app.js";

// Monaco re-renders its line elements (e.g. once fonts are loaded), so a
// line can be replaced between finding and measuring it. Retry until it
// holds still; the replacement sits in the same place.
async function measure(locator) {
  let box = null;
  await expect.poll(async () => (box = await locator.boundingBox())).not.toBeNull();
  return box;
}

for (const zoom of [1, 1.5, 2]) {
  test(`clicking in the grpcurl editor puts the cursor at the clicked character at ${zoom * 100}% zoom`, async ({ page }) => {
    await page.addInitScript(z => localStorage.setItem("wombat_zoom_level", String(z)), zoom);
    await openApp(page);
    await selectMethod(page, METHODS.GetFeature);
    await textInput(form(page), "latitude").fill("5");
    await page.locator(".method-input").getByRole("button", { name: "Edit", exact: true }).click();
    const panel = page.locator(".code-edit-panel");
    const firstLine = panel.locator(".view-line").first();
    await expect(firstLine).toContainText('"latitude":"5"');

    // Monaco uses a monospace font: the clicked column follows from the
    // rendered line's width. Click on the left edge of the "5".
    const content = firstLine.locator(":scope > span");
    const text = (await content.textContent()).replace(/ /g, " ");
    const box = await measure(content);
    const charWidth = box.width / text.length;
    const col = text.indexOf('"latitude":"5"') + '"latitude":"'.length;
    await page.mouse.click(box.x + charWidth * (col + 0.2), box.y + box.height / 2);
    await page.keyboard.type("1");
    await expect(firstLine).toContainText('"latitude":"15"');

    await button(panel, "Import").click();
    await expect(textInput(form(page), "latitude")).toHaveValue("15");
  });
}

test("the editor text grows with the zoom", async ({ page }) => {
  const lineHeight = async zoom => {
    await page.evaluate(z => localStorage.setItem("wombat_zoom_level", String(z)), zoom);
    await page.reload();
    await expect(page.locator(".method-input h2")).toBeVisible();
    await selectMethod(page, METHODS.GetFeature);
    await page.locator(".method-input").getByRole("button", { name: "Edit", exact: true }).click();
    const line = page.locator(".code-edit-panel .view-line").first();
    await expect(line).toContainText("grpcurl");
    const box = await measure(line);
    await page.keyboard.press("Escape");
    return box.height;
  };
  await openApp(page);
  const normal = await lineHeight(1);
  const zoomed = await lineHeight(1.5);
  expect(zoomed / normal).toBeGreaterThan(1.4);
  expect(zoomed / normal).toBeLessThan(1.6);
});

for (const zoom of [1.5, 2]) {
  test(`the grpcurl dialog fits on screen at ${zoom * 100}% zoom`, async ({ page }) => {
    await page.addInitScript(z => localStorage.setItem("wombat_zoom_level", String(z)), zoom);
    await openApp(page);
    await selectMethod(page, METHODS.GetFeature);
    await page.locator(".method-input").getByRole("button", { name: "Edit", exact: true }).click();
    const panel = page.locator(".code-edit-panel");
    await expect(panel.locator(".view-line").first()).toContainText("grpcurl");

    // No inner scrollbar, and the buttons are fully inside the dialog.
    const content = page.locator(".window .content");
    expect(await content.evaluate(el => el.scrollHeight <= el.clientHeight)).toBe(true);
    const win = await measure(page.locator(".window"));
    for (const name of ["Close", "Import"]) {
      const b = await measure(button(panel, name));
      expect(b.y + b.height).toBeLessThanOrEqual(win.y + win.height);
    }
    expect(win.y).toBeGreaterThanOrEqual(0);
    expect(win.y + win.height).toBeLessThanOrEqual(page.viewportSize().height + 1);
  });
}

const LONG = "name: \"" + "Berkshire Valley Management Area Trail, Jefferson, NJ, USA ".repeat(3) + "\"\n";

for (const zoom of [1, 1.5]) {
  test.describe(`the response panel at ${zoom * 100}% zoom`, () => {
    test.use({ fakeConfig: { responses: { "/wombat.v1.RouteGuide/GetFeature": LONG } } });

    test("stays inside the window and wraps long lines", async ({ page }) => {
      await page.addInitScript(z => localStorage.setItem("wombat_zoom_level", String(z)), zoom);
      await openApp(page);
      await selectMethod(page, METHODS.GetFeature);
      await page.locator(".method-select").getByRole("button", { name: "Send", exact: true }).click();
      const editor = page.locator(".response .monaco-editor");
      await expect(editor).toContainText("Jefferson");

      const vp = page.viewportSize();
      for (const el of [page.locator(".output-pane"), page.locator(".response"), editor]) {
        const box = await measure(el);
        expect(box.x + box.width).toBeLessThanOrEqual(vp.width);
        expect(box.y + box.height).toBeLessThanOrEqual(vp.height);
      }
      // Every rendered line ends inside the editor (the long one wraps).
      const right = (await measure(editor)).x + (await measure(editor)).width;
      const lines = page.locator(".response .view-line > span");
      for (let i = 0; i < (await lines.count()); i++) {
        const b = await lines.nth(i).boundingBox();
        if (b) expect(b.x + b.width).toBeLessThanOrEqual(right + 1);
      }
    });
  });
}

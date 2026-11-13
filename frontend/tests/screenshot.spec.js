// Renders assets/gh/screenshot.png for the README. Not part of the normal
// test run; regenerate it with `npm run screenshot` (WebKit, in Docker).
import { test, expect, openApp, form, textInput, selectMethod, send, METHODS } from "./app.js";

test.skip(!process.env.SCREENSHOT, "only with SCREENSHOT=1 (npm run screenshot)");

test.use({
  viewport: { width: 1200, height: 760 },
  deviceScaleFactor: 2,
  fakeConfig: {
    // What the built-in test server answers, formatted like the app does.
    responses: {
      "/wombat.v1.RouteGuide/GetFeature":
        'name: "Berkshire Valley Management Area Trail, Jefferson, NJ, USA"\n' +
        "location: {\n  latitude: 409146138\n  longitude: -746188906\n}\n",
    },
  },
});

test("README screenshot", async ({ page }) => {
  await openApp(page);
  await selectMethod(page, METHODS.GetFeature);
  await textInput(form(page), "latitude").fill("409146138");
  await textInput(form(page), "longitude").fill("-746188906");
  await send(page);
  await expect(page.locator(".response .view-lines")).toContainText("Berkshire Valley");
  await page.mouse.move(0, 0);
  await page.waitForTimeout(300);
  await page.screenshot({ path: "../assets/gh/screenshot.png" });
});

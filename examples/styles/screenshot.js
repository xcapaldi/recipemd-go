// Captures screenshots of every preview page into screenshots/.
// Usage (from examples/styles, after ./build.sh):  node screenshot.js
const { chromium } = require("playwright");
const { readdirSync, mkdirSync } = require("node:fs");
const path = require("node:path");

(async () => {

const shots = [
  { suffix: "desktop", viewport: { width: 1280, height: 900 }, scheme: "light" },
  { suffix: "mobile", viewport: { width: 390, height: 844 }, scheme: "light", scale: 2 },
];
// Themes that react to prefers-color-scheme also get dark captures.
const darkAware = ["04-modern", "05-complete"];

process.chdir(__dirname);
mkdirSync("screenshots", { recursive: true });
const browser = await chromium.launch();
for (const file of readdirSync("preview").filter((f) => f.endsWith(".html"))) {
  const name = file.replace(/\.html$/, "");
  const variants = [...shots];
  if (darkAware.includes(name)) {
    variants.push(...shots.map((s) => ({ ...s, scheme: "dark", suffix: `${s.suffix}-dark` })));
  }
  for (const v of variants) {
    const page = await browser.newPage({
      viewport: v.viewport,
      deviceScaleFactor: v.scale ?? 1,
      colorScheme: v.scheme,
    });
    await page.goto("file://" + path.resolve(__dirname, "preview", file));
    const out = `screenshots/${name}-${v.suffix}.png`;
    await page.screenshot({ path: out, fullPage: true });
    console.log("wrote", out);
    await page.close();
  }
}
await browser.close();
})();

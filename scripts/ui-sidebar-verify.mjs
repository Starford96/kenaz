import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { chromium } from "playwright";

const URL = process.env.KENAZ_URL ?? "http://localhost:5173/";
const OUT_DIR = path.resolve(process.cwd(), "ui-screenshots");

function ensureDir(p) {
  fs.mkdirSync(p, { recursive: true });
}

async function waitForTree(page) {
  // Sidebar can be empty depending on vault contents.
  const tree = page.locator(".kenaz-sidebar-tree");
  const empty = page.locator(".kenaz-sidebar").getByText("No notes yet");
  await Promise.race([
    tree.waitFor({ state: "visible", timeout: 15_000 }),
    empty.waitFor({ state: "visible", timeout: 15_000 }),
  ]);
  return { hasTree: await tree.count().then((c) => c > 0) };
}

async function desktopPass(browser) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  await page.goto(URL, { waitUntil: "domcontentloaded" });

  const { hasTree } = await waitForTree(page);
  let metrics = {
    hasTree,
    rowHeightPx: null,
    rowPadding: null,
    hoverBg: null,
    selectedBg: null,
    titleEllipsis: null,
    fullRowClickable: null,
  };

  if (hasTree) {
    const row = page.locator(".kenaz-sidebar-tree .ant-tree-treenode").first();
    const wrapper = row.locator(".ant-tree-node-content-wrapper").first();
    await wrapper.scrollIntoViewIfNeeded();

    const box = await wrapper.boundingBox();
    if (box) {
      metrics.rowHeightPx = Math.round(box.height);
    }

    metrics.rowPadding = await wrapper.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return {
        paddingTop: cs.paddingTop,
        paddingBottom: cs.paddingBottom,
        paddingLeft: cs.paddingLeft,
        paddingRight: cs.paddingRight,
        minHeight: cs.minHeight,
        lineHeight: cs.lineHeight,
        borderRadius: cs.borderRadius,
      };
    });

    // Hover state.
    await wrapper.hover();
    metrics.hoverBg = await wrapper.evaluate((el) => window.getComputedStyle(el).backgroundColor);

    // Selected state.
    await wrapper.click({ position: { x: 8, y: 8 } });
    await page.waitForTimeout(50);
    metrics.selectedBg = await wrapper.evaluate((el) => window.getComputedStyle(el).backgroundColor);

    // Full-row clickability: click at far right inside the wrapper and ensure it's still selected.
    const afterClickSelected = async () =>
      wrapper.evaluate((el) => el.classList.contains("ant-tree-node-selected"));

    if (box) {
      await page.mouse.click(box.x + box.width - 2, box.y + box.height / 2);
      await page.waitForTimeout(50);
      metrics.fullRowClickable = await afterClickSelected();
    }

    // Ellipsis: titleRender uses a nested span with overflow hidden + textOverflow ellipsis.
    metrics.titleEllipsis = await row.evaluate((node) => {
      const el = node.querySelector(".ant-tree-title span[style*='text-overflow']");
      if (!el) return { found: false };
      const cs = window.getComputedStyle(el);
      return {
        found: true,
        textOverflow: cs.textOverflow,
        overflow: cs.overflow,
        whiteSpace: cs.whiteSpace,
        clientWidth: el.clientWidth,
        scrollWidth: el.scrollWidth,
        isActuallyClipped: el.scrollWidth > el.clientWidth + 1,
      };
    });
  }

  ensureDir(OUT_DIR);
  const screenshotPath = path.join(OUT_DIR, "desktop-sidebar.png");
  await page.screenshot({ path: screenshotPath, fullPage: false });

  await context.close();
  return { screenshotPath, metrics };
}

async function mobilePass(browser) {
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
    hasTouch: true,
    deviceScaleFactor: 2,
  });
  const page = await context.newPage();
  await page.goto(URL, { waitUntil: "domcontentloaded" });

  await page.locator(".mobile-header").waitFor({ state: "visible", timeout: 15_000 });
  await page.locator('button[aria-label="Open sidebar"]').click();

  // Drawer animation can take a moment.
  await page.locator(".kenaz-drawer .ant-drawer-content-wrapper").waitFor({
    state: "visible",
    timeout: 15_000,
  });

  const { hasTree } = await waitForTree(page);
  let metrics = { hasTree, touchTargetHeightPx: null, rowPadding: null };

  if (hasTree) {
    const wrapper = page
      .locator(".kenaz-drawer .kenaz-sidebar-tree .ant-tree-treenode")
      .first()
      .locator(".ant-tree-node-content-wrapper")
      .first();
    await wrapper.scrollIntoViewIfNeeded();
    const box = await wrapper.boundingBox();
    if (box) metrics.touchTargetHeightPx = Math.round(box.height);
    metrics.rowPadding = await wrapper.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return { paddingTop: cs.paddingTop, paddingBottom: cs.paddingBottom, minHeight: cs.minHeight };
    });
  }

  ensureDir(OUT_DIR);
  const screenshotPath = path.join(OUT_DIR, "mobile-sidebar-open.png");
  await page.screenshot({ path: screenshotPath, fullPage: false });

  await context.close();
  return { screenshotPath, metrics };
}

async function main() {
  ensureDir(OUT_DIR);

  const browser = await chromium.launch();
  try {
    const desktop = await desktopPass(browser);
    const mobile = await mobilePass(browser);

    // Print a compact summary for the caller (useful in CI or terminal output).
    // eslint-disable-next-line no-console
    console.log(
      JSON.stringify(
        {
          url: URL,
          outDir: OUT_DIR,
          desktop: { ...desktop, screenshotPath: path.relative(process.cwd(), desktop.screenshotPath) },
          mobile: { ...mobile, screenshotPath: path.relative(process.cwd(), mobile.screenshotPath) },
        },
        null,
        2,
      ),
    );
  } finally {
    await browser.close();
  }
}

await main();

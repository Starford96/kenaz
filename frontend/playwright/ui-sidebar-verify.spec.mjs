import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { test, expect } from "@playwright/test";

const URL = process.env.KENAZ_URL ?? "http://localhost:5173/";
const REPO_ROOT = path.resolve(process.cwd(), "..");
const OUT_DIR = path.resolve(REPO_ROOT, "ui-screenshots");

function ensureDir(p) {
  fs.mkdirSync(p, { recursive: true });
}

// Use the locally installed Google Chrome instead of Playwright's bundled
// chromium/headless-shell, which can be flaky in some sandboxed environments.
test.use({ channel: "chrome" });

async function waitForTreeOrEmpty(page) {
  const tree = page.locator(".kenaz-sidebar-tree");
  const empty = page.locator(".kenaz-sidebar").getByText("No notes yet");
  await Promise.race([
    tree.waitFor({ state: "visible", timeout: 15_000 }),
    empty.waitFor({ state: "visible", timeout: 15_000 }),
  ]);
  return { hasTree: (await tree.count()) > 0 };
}

test("desktop: sidebar tree states + screenshot", async ({ page }) => {
  test.setTimeout(120_000);
  ensureDir(OUT_DIR);
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto(URL, { waitUntil: "domcontentloaded" });

  await page.locator(".kenaz-sidebar").waitFor({ state: "visible", timeout: 20_000 });

  // Give note list query time to resolve.
  await page.waitForTimeout(250);

  // Prefer waiting for at least one treenode; otherwise accept "No notes yet".
  const firstRowWrapper = page.locator(
    ".kenaz-sidebar .ant-tree-treenode .ant-tree-node-content-wrapper",
  ).first();
  const empty = page.locator(".kenaz-sidebar").getByText("No notes yet");
  await Promise.race([
    firstRowWrapper.waitFor({ state: "visible", timeout: 20_000 }),
    empty.waitFor({ state: "visible", timeout: 20_000 }),
  ]);

  const hasTree = (await firstRowWrapper.count()) > 0;
  let metrics = { hasTree };

  if (hasTree) {
    const wrapper = firstRowWrapper;
    const treenode = page.locator(".kenaz-sidebar .ant-tree-treenode").first();
    await wrapper.scrollIntoViewIfNeeded();

    const box = await wrapper.boundingBox();
    expect(box, "expected first tree row to have a bounding box").toBeTruthy();

    metrics = {
      ...metrics,
      rowHeightPx: Math.round(box.height),
      treenodeHeightPx: Math.round((await treenode.boundingBox())?.height ?? 0),
      base: await wrapper.evaluate((el) => {
        const cs = window.getComputedStyle(el);
        return {
          paddingTop: cs.paddingTop,
          paddingBottom: cs.paddingBottom,
          paddingLeft: cs.paddingLeft,
          paddingRight: cs.paddingRight,
          minHeight: cs.minHeight,
          lineHeight: cs.lineHeight,
          borderRadius: cs.borderRadius,
          bg: cs.backgroundColor,
        };
      }),
    };

    // Hover
    await wrapper.hover();
    metrics.hoverBg = await wrapper.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return { background: cs.background, backgroundColor: cs.backgroundColor };
    });
    metrics.hoverState = await wrapper.evaluate((el) => ({
      matchesHover: el.matches(":hover"),
      className: el.className,
    }));

    // Full-row clickability: click near far-right inside wrapper; it should select the node.
    await page.mouse.click(box.x + box.width - 2, box.y + box.height / 2);
    await page.waitForTimeout(50);
    metrics.selected = {
      isSelectedClass: await wrapper.evaluate((el) => el.classList.contains("ant-tree-node-selected")),
      bg: await wrapper.evaluate((el) => window.getComputedStyle(el).backgroundColor),
    };
    metrics.fullRowClickable = metrics.selected.isSelectedClass;

    // Ellipsis: title span inside titleRender.
    metrics.titleEllipsis = await wrapper.evaluate((wrapperEl) => {
      const node = wrapperEl.closest(".ant-tree-treenode");
      if (!node) return { found: false };
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

  await page.screenshot({ path: path.join(OUT_DIR, "desktop-sidebar.png"), fullPage: false });
  // eslint-disable-next-line no-console
  console.log("[desktop metrics]", JSON.stringify(metrics, null, 2));
});

test("mobile: open sidebar drawer + screenshot", async ({ page }) => {
  test.setTimeout(120_000);
  ensureDir(OUT_DIR);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(URL, { waitUntil: "domcontentloaded" });

  await page.locator(".mobile-header").waitFor({ state: "visible", timeout: 15_000 });
  await page.locator('button[aria-label="Open sidebar"]').click();

  const drawerSidebar = page.locator(".ant-drawer-content-wrapper .kenaz-sidebar").first();
  await drawerSidebar.waitFor({ state: "visible", timeout: 20_000 });

  // Drawer opening may autofocus the sidebar search input, which opens the command palette.
  // Close it so the screenshot shows the actual sidebar note tree.
  const palette = page.locator(".kenaz-palette");
  if (await palette.isVisible().catch(() => false)) {
    await page.keyboard.press("Escape");
    await palette.waitFor({ state: "hidden", timeout: 10_000 }).catch(() => {});
  }

  // Same as desktop: wait for a tree row if present, else accept empty vault.
  const firstRowWrapper = page.locator(
    ".ant-drawer-content-wrapper .kenaz-sidebar .ant-tree-treenode .ant-tree-node-content-wrapper",
  ).first();
  const empty = page.locator(".ant-drawer-content-wrapper .kenaz-sidebar").getByText("No notes yet");
  await Promise.race([
    firstRowWrapper.waitFor({ state: "visible", timeout: 20_000 }),
    empty.waitFor({ state: "visible", timeout: 20_000 }),
  ]);

  const hasTree = (await firstRowWrapper.count()) > 0;
  let metrics = { hasTree };

  if (hasTree) {
    const treenode = page
      .locator(".ant-drawer-content-wrapper .kenaz-sidebar .ant-tree-treenode")
      .first();
    const box = await firstRowWrapper.boundingBox();
    if (box) metrics.touchTargetHeightPx = Math.round(box.height);
    metrics.treenodeHeightPx = Math.round((await treenode.boundingBox())?.height ?? 0);
    metrics.row = await firstRowWrapper.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return { paddingTop: cs.paddingTop, paddingBottom: cs.paddingBottom, minHeight: cs.minHeight };
    });
  }

  await page.screenshot({
    path: path.join(OUT_DIR, "mobile-sidebar-open.png"),
    fullPage: false,
  });
  // eslint-disable-next-line no-console
  console.log("[mobile metrics]", JSON.stringify(metrics, null, 2));
});


/**
 * Scroll to a heading by its ID within the nearest scrollable container,
 * without triggering parent frame / iframe scrolling.
 *
 * Unlike `scrollIntoView`, this only scrolls the content container
 * (the `overflow: auto` div wrapping `.md-preview`), which prevents
 * the Home Assistant iframe or any other parent from jumping.
 */
export function scrollToHeading(id: string, smooth = true): boolean {
  const el = document.getElementById(id);
  if (!el) return false;

  const container = findScrollParent(el);
  if (!container) {
    el.scrollIntoView({ behavior: smooth ? "smooth" : "instant", block: "start" });
    return true;
  }

  const containerRect = container.getBoundingClientRect();
  const elRect = el.getBoundingClientRect();
  const offset = elRect.top - containerRect.top + container.scrollTop;

  container.scrollTo({
    top: Math.max(0, offset - 8),
    behavior: smooth ? "smooth" : "instant",
  });

  return true;
}

/**
 * Retry-based variant: waits for the target element to appear in the DOM
 * (useful after async markdown rendering).
 */
export function scrollToHeadingWithRetry(
  id: string,
  { maxAttempts = 15, intervalMs = 150, smooth = true } = {},
) {
  let attempt = 0;
  const tryScroll = () => {
    if (scrollToHeading(id, smooth)) return;
    attempt++;
    if (attempt < maxAttempts) {
      setTimeout(tryScroll, intervalMs);
    }
  };
  tryScroll();
}

function findScrollParent(el: HTMLElement): HTMLElement | null {
  let node: HTMLElement | null = el.parentElement;
  while (node && node !== document.documentElement) {
    const { overflowY } = getComputedStyle(node);
    if (
      (overflowY === "auto" || overflowY === "scroll") &&
      node.scrollHeight > node.clientHeight
    ) {
      return node;
    }
    node = node.parentElement;
  }
  return null;
}

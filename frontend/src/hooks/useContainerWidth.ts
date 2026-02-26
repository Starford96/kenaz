import { useSyncExternalStore } from "react";

const MOBILE_BREAKPOINT = 768;
const CONTEXT_PANEL_BREAKPOINT = 1024;

type Listener = () => void;

let currentWidth =
  typeof window !== "undefined"
    ? document.getElementById("root")?.clientWidth ?? window.innerWidth
    : MOBILE_BREAKPOINT;

const listeners = new Set<Listener>();

function emit() {
  listeners.forEach((cb) => cb());
}

if (typeof window !== "undefined") {
  const root = document.getElementById("root");
  if (root) {
    const ro = new ResizeObserver(([entry]) => {
      const w = Math.round(entry.contentRect.width);
      if (w !== currentWidth) {
        currentWidth = w;
        emit();
      }
    });
    ro.observe(root);
  }
}

function subscribe(cb: Listener) {
  listeners.add(cb);
  return () => listeners.delete(cb);
}

function getSnapshot() {
  return currentWidth;
}

function getServerSnapshot() {
  return MOBILE_BREAKPOINT;
}

export function useContainerWidth(): number {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

export { MOBILE_BREAKPOINT, CONTEXT_PANEL_BREAKPOINT };

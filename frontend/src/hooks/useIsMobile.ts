import { useSyncExternalStore } from "react";

const MOBILE_MQ = "(max-width: 767px)";

function subscribe(cb: () => void) {
  const mql = window.matchMedia(MOBILE_MQ);
  mql.addEventListener("change", cb);
  return () => mql.removeEventListener("change", cb);
}

function getSnapshot() {
  return window.matchMedia(MOBILE_MQ).matches;
}

function getServerSnapshot() {
  return false;
}

export function useIsMobile(): boolean {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

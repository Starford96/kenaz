import { useContainerWidth, MOBILE_BREAKPOINT } from "./useContainerWidth";

export function useIsMobile(): boolean {
  return useContainerWidth() < MOBILE_BREAKPOINT;
}

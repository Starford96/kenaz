import { useEffect, useRef } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useUIStore } from "../store/ui";
import { scrollToHeadingWithRetry } from "../utils/scrollToHeading";

/**
 * Bidirectional sync between the URL path and the active note tab.
 *
 * - On mount / popstate: opens the note from the URL path and scrolls to #hash.
 * - On activeTab change: updates the URL to reflect the current note.
 */
export function useUrlSync() {
  const location = useLocation();
  const navigate = useNavigate();
  const suppressUrlUpdate = useRef(false);

  // URL -> state: when the URL changes (mount, back/forward), open the tab.
  useEffect(() => {
    const notePath = decodeURIComponent(location.pathname.replace(/^\//, ""));
    if (!notePath) return;

    const { activeTab, openTab } = useUIStore.getState();
    if (notePath !== activeTab) {
      suppressUrlUpdate.current = true;
      openTab(notePath, notePath);
      suppressUrlUpdate.current = false;
    }
  }, [location.pathname]);

  // Scroll to #hash anchor after the note content has rendered.
  useEffect(() => {
    const hash = location.hash.replace(/^#/, "");
    if (!hash) return;

    scrollToHeadingWithRetry(hash);
  }, [location.hash, location.pathname]);

  // State -> URL: when activeTab changes, update the URL (strip old hash).
  useEffect(() => {
    return useUIStore.subscribe((state, prev) => {
      if (state.activeTab === prev.activeTab) return;
      if (suppressUrlUpdate.current) return;

      const currentPath = decodeURIComponent(location.pathname.replace(/^\//, ""));
      const target = state.activeTab ? `/${state.activeTab}` : "/";

      if (state.activeTab !== currentPath) {
        navigate(target, { replace: true });
      }
    });
  }, [navigate, location.pathname]);
}

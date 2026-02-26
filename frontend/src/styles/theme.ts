import type { ThemeConfig } from "antd";
import { theme } from "antd";

/** Obsidian-inspired dark theme for Ant Design. */
export const darkTheme: ThemeConfig = {
  algorithm: [theme.darkAlgorithm, theme.compactAlgorithm],
  token: {
    colorPrimary: "#7c3aed",      // Purple accent.
    colorBgBase: "#1e1e2e",       // Dark background.
    colorBgContainer: "#252536",  // Card / panel background.
    colorBgElevated: "#2a2a3c",   // Dropdown / modal background.
    colorBorder: "#3a3a4e",       // Subtle borders.
    colorText: "#cdd6f4",         // Light text.
    colorTextSecondary: "#a6adc8",
    borderRadius: 6,
    fontFamily:
      "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
  },
  components: {
    Layout: {
      siderBg: "#181825",
      headerBg: "#1e1e2e",
      bodyBg: "#1e1e2e",
    },
    Menu: {
      darkItemBg: "#181825",
      darkSubMenuItemBg: "#181825",
    },
    Drawer: {
      colorBgElevated: "#181825",
      colorBgMask: "rgba(0, 0, 0, 0.45)",
    },
  },
};

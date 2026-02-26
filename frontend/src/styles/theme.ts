import type { ThemeConfig } from "antd";
import { theme } from "antd";
import { c } from "./colors";

export const darkTheme: ThemeConfig = {
  algorithm: [theme.darkAlgorithm, theme.compactAlgorithm],
  token: {
    colorPrimary: c.accent,
    colorBgBase: c.bgBase,
    colorBgContainer: c.bgSurface,
    colorBgElevated: c.bgElevated,
    colorBorder: c.border,
    colorText: c.textPrimary,
    colorTextSecondary: c.textSecondary,
    borderRadius: 6,
    fontFamily:
      "'DM Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
  },
  components: {
    Layout: {
      siderBg: c.bgDeepest,
      headerBg: c.bgBase,
      bodyBg: c.bgBase,
    },
    Menu: {
      darkItemBg: c.bgDeepest,
      darkSubMenuItemBg: c.bgDeepest,
    },
    Drawer: {
      colorBgElevated: c.bgDeepest,
      colorBgMask: c.maskLight,
    },
  },
};

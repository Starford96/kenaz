import { Typography } from "antd";
import { MenuOutlined } from "@ant-design/icons";
import { useUIStore } from "../store/ui";

const { Text } = Typography;

export default function MobileHeader() {
  const { activeTab, tabs, setMobileDrawer } = useUIStore();

  const activeTitle =
    tabs.find((t) => t.path === activeTab)?.title ?? "Kenaz";

  return (
    <div className="mobile-header">
      <button
        className="mobile-header__btn"
        onClick={() => setMobileDrawer("sidebar")}
        aria-label="Open sidebar"
      >
        <MenuOutlined />
      </button>

      <Text
        strong
        ellipsis
        className="mobile-header__title"
      >
        {activeTitle}
      </Text>

      <div style={{ width: 36 }} />
    </div>
  );
}

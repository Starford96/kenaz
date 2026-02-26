import {
  FolderOutlined,
  SearchOutlined,
  NodeIndexOutlined,
} from "@ant-design/icons";
import { useUIStore } from "../store/ui";

export default function BottomActionBar() {
  const { setMobileDrawer, setSearchOpen } = useUIStore();

  return (
    <nav className="bottom-bar">
      <button
        className="bottom-bar__btn"
        onClick={() => setMobileDrawer("sidebar")}
        aria-label="Sidebar"
      >
        <FolderOutlined />
        <span>Files</span>
      </button>

      <button
        className="bottom-bar__btn"
        onClick={() => setSearchOpen(true)}
        aria-label="Search"
      >
        <SearchOutlined />
        <span>Search</span>
      </button>

      <button
        className="bottom-bar__btn"
        onClick={() => setMobileDrawer("context")}
        aria-label="Context panel"
      >
        <NodeIndexOutlined />
        <span>Context</span>
      </button>
    </nav>
  );
}

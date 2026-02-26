import {
  FolderOutlined,
  SearchOutlined,
  PlusOutlined,
  NodeIndexOutlined,
} from "@ant-design/icons";
import { useUIStore } from "../store/ui";

export default function BottomActionBar() {
  const { setMobileDrawer, setSearchOpen } = useUIStore();

  const handleCreate = () => {
    window.dispatchEvent(new CustomEvent("kenaz:create-note"));
  };

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
        className="bottom-bar__btn bottom-bar__btn--accent"
        onClick={handleCreate}
        aria-label="New note"
      >
        <PlusOutlined />
        <span>New</span>
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

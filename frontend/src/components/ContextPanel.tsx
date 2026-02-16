import { useQuery } from "@tanstack/react-query";
import { Typography, List, Divider, Spin } from "antd";
import { LinkOutlined, OrderedListOutlined } from "@ant-design/icons";
import { getNote } from "../api/notes";
import { useUIStore } from "../store/ui";

const { Text } = Typography;

/** Right context panel showing backlinks and outline for the active note. */
export default function ContextPanel() {
  const { activeTab, openTab } = useUIStore();

  const { data: note, isLoading } = useQuery({
    queryKey: ["note", activeTab],
    queryFn: () => getNote(activeTab!),
    enabled: !!activeTab,
  });

  if (!activeTab) {
    return (
      <div style={{ padding: 16, color: "#6c7086" }}>
        Select a note to see context
      </div>
    );
  }

  if (isLoading) return <Spin size="small" style={{ margin: 16 }} />;

  // Extract headings for outline.
  const headings = (note?.content ?? "")
    .split("\n")
    .filter((l) => /^#{1,3}\s/.test(l))
    .map((l) => {
      const level = l.match(/^#+/)![0].length;
      const text = l.replace(/^#+\s*/, "");
      return { level, text };
    });

  return (
    <div style={{ padding: 12, height: "100%", overflow: "auto" }}>
      {/* Backlinks */}
      <Divider orientationMargin={0} plain style={{ marginTop: 0 }}>
        <LinkOutlined /> Backlinks
      </Divider>
      {note?.backlinks && note.backlinks.length > 0 ? (
        <List
          size="small"
          dataSource={note.backlinks}
          renderItem={(bl) => (
            <List.Item
              style={{ cursor: "pointer", padding: "2px 0", border: "none" }}
              onClick={() => openTab(bl, bl)}
            >
              <Text style={{ color: "#7c3aed", fontSize: 13 }}>{bl}</Text>
            </List.Item>
          )}
        />
      ) : (
        <Text type="secondary" style={{ fontSize: 12 }}>
          No backlinks
        </Text>
      )}

      {/* Outline */}
      <Divider orientationMargin={0} plain>
        <OrderedListOutlined /> Outline
      </Divider>
      {headings.length > 0 ? (
        <div>
          {headings.map((h, i) => (
            <div
              key={i}
              style={{
                paddingLeft: (h.level - 1) * 12,
                fontSize: 13,
                lineHeight: "22px",
                color: "#a6adc8",
              }}
            >
              {h.text}
            </div>
          ))}
        </div>
      ) : (
        <Text type="secondary" style={{ fontSize: 12 }}>
          No headings
        </Text>
      )}
    </div>
  );
}

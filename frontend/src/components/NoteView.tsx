import { useQuery } from "@tanstack/react-query";
import { Spin, Typography, Tag, Divider, List } from "antd";
import { LinkOutlined } from "@ant-design/icons";
import { getNote } from "../api/notes";
import { useUIStore } from "../store/ui";

const { Title, Text, Paragraph } = Typography;

interface Props {
  path: string;
}

/** Renders a note's content with metadata and backlinks. */
export default function NoteView({ path }: Props) {
  const { openTab } = useUIStore();

  const { data: note, isLoading, error } = useQuery({
    queryKey: ["note", path],
    queryFn: () => getNote(path),
    enabled: !!path,
  });

  if (isLoading) return <Spin style={{ marginTop: 48 }} />;
  if (error || !note)
    return <Text type="danger">Failed to load {path}</Text>;

  // Simple wikilink rendering: replace [[target]] and [[target|alias]]
  // with clickable spans.
  const renderContent = (raw: string) => {
    const parts = raw.split(/(\[\[.*?\]\])/g);
    return parts.map((part, i) => {
      const match = part.match(/^\[\[(.*?)\]\]$/);
      if (!match) return <span key={i}>{part}</span>;
      const inner = match[1];
      const pipeIdx = inner.indexOf("|");
      const target = pipeIdx >= 0 ? inner.slice(0, pipeIdx) : inner;
      const label = pipeIdx >= 0 ? inner.slice(pipeIdx + 1) : inner;
      return (
        <span
          key={i}
          className="wikilink"
          onClick={() => openTab(target.trim(), label.trim())}
        >
          {label}
        </span>
      );
    });
  };

  return (
    <div style={{ padding: "16px 24px", maxWidth: 800, overflow: "auto" }}>
      <Title level={3}>{note.title || path}</Title>

      {note.tags.length > 0 && (
        <div style={{ marginBottom: 12 }}>
          {note.tags.map((t) => (
            <Tag key={t} color="purple">
              {t}
            </Tag>
          ))}
        </div>
      )}

      <Paragraph style={{ whiteSpace: "pre-wrap", lineHeight: 1.8 }}>
        {renderContent(note.content)}
      </Paragraph>

      {note.backlinks.length > 0 && (
        <>
          <Divider orientationMargin={0} plain>
            <LinkOutlined /> Backlinks
          </Divider>
          <List
            size="small"
            dataSource={note.backlinks}
            renderItem={(bl) => (
              <List.Item
                style={{ cursor: "pointer", padding: "4px 0" }}
                onClick={() => openTab(bl, bl)}
              >
                <Text style={{ color: "#7c3aed" }}>{bl}</Text>
              </List.Item>
            )}
          />
        </>
      )}
    </div>
  );
}

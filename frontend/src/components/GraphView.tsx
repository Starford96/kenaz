import { useRef, useCallback, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { Spin, Typography } from "antd";
import ForceGraph2D from "react-force-graph-2d";
import { getGraph } from "../api/notes";
import { useUIStore } from "../store/ui";
import { c } from "../styles/colors";

const { Text } = Typography;

interface GraphNode {
  id: string;
  title?: string;
  x?: number;
  y?: number;
}

interface GraphLink {
  source: string | GraphNode;
  target: string | GraphNode;
}

/** Interactive 2D force-directed graph of notes and their links. */
export default function GraphView() {
  const { openTab } = useUIStore();
  const containerRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const fgRef = useRef<any>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ["graph"],
    queryFn: getGraph,
  });

  // Fit graph to container on data change.
  useEffect(() => {
    if (fgRef.current && data) {
      setTimeout(() => fgRef.current?.zoomToFit(300, 40), 200);
    }
  }, [data]);

  const handleNodeClick = useCallback(
    (node: GraphNode) => {
      if (node.id) {
        openTab(node.id, node.title || node.id);
      }
    },
    [openTab],
  );

  const nodeLabel = useCallback(
    (node: GraphNode) => node.title || node.id,
    [],
  );

  const nodeCanvasObject = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (node: any, ctx: CanvasRenderingContext2D, globalScale: number) => {
      const label = node.title || node.id || "";
      const fontSize = Math.max(12 / globalScale, 3);
      ctx.font = `${fontSize}px Inter, sans-serif`;

      // Node dot.
      ctx.beginPath();
      ctx.arc(node.x!, node.y!, 4 / globalScale, 0, 2 * Math.PI);
      ctx.fillStyle = c.accent;
      ctx.fill();

      // Label.
      ctx.fillStyle = c.textPrimary;
      ctx.textAlign = "center";
      ctx.textBaseline = "top";
      ctx.fillText(label, node.x!, node.y! + 6 / globalScale);
    },
    [],
  );

  if (isLoading) return <Spin style={{ marginTop: 48 }} />;
  if (error) return <Text type="danger">Failed to load graph</Text>;
  if (!data || data.nodes.length === 0)
    return (
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          height: "100%",
          color: c.textTertiary,
        }}
      >
        No notes to graph yet
      </div>
    );

  const graphData = {
    nodes: data.nodes.map((n) => ({ id: n.id, title: n.title })) as GraphNode[],
    links: data.links.map((l) => ({
      source: l.source,
      target: l.target,
    })) as GraphLink[],
  };

  return (
    <div
      ref={containerRef}
      style={{ width: "100%", height: "100%", background: c.bgBase }}
    >
      <ForceGraph2D
        ref={fgRef}
        graphData={graphData}
        nodeId="id"
        nodeLabel={nodeLabel as never}
        nodeCanvasObject={nodeCanvasObject as never}
        onNodeClick={handleNodeClick as never}
        linkColor={() => c.border}
        linkWidth={1}
        backgroundColor={c.bgBase}
        width={containerRef.current?.clientWidth ?? 800}
        height={containerRef.current?.clientHeight ?? 600}
      />
    </div>
  );
}

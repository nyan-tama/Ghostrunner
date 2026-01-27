"use client";

import { useEffect, useState } from "react";

interface Props {
  chart: string;
}

export default function MermaidRenderer({ chart }: Props) {
  const [svg, setSvg] = useState<string>("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    async function renderMermaid() {
      try {
        const mermaid = (await import("mermaid")).default;

        mermaid.initialize({
          startOnLoad: false,
          theme: "default",
          securityLevel: "loose",
        });

        const id = `mermaid-${Date.now()}-${Math.random().toString(36).slice(2)}`;
        const { svg } = await mermaid.render(id, chart);

        if (isMounted) {
          setSvg(svg);
          setError(null);
        }
      } catch (err) {
        if (isMounted) {
          setError(err instanceof Error ? err.message : "Mermaid render error");
        }
      }
    }

    renderMermaid();

    return () => {
      isMounted = false;
    };
  }, [chart]);

  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded my-4">
        <p className="text-red-600 text-sm">Mermaid Error: {error}</p>
        <pre className="mt-2 text-xs bg-gray-100 p-2 rounded overflow-x-auto">
          {chart}
        </pre>
      </div>
    );
  }

  if (!svg) {
    return (
      <div className="p-4 bg-gray-50 animate-pulse rounded my-4">
        Loading diagram...
      </div>
    );
  }

  return (
    <div
      className="my-4 overflow-x-auto"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  );
}

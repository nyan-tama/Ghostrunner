"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeRaw from "rehype-raw";
import rehypeHighlight from "rehype-highlight";
import MermaidRenderer from "./MermaidRenderer";
import type { Components } from "react-markdown";

interface Props {
  content: string;
  title: string;
}

export default function MarkdownViewer({ content, title }: Props) {
  const components: Components = {
    code({ className, children, ...props }) {
      const match = /language-(\w+)/.exec(className || "");
      const language = match ? match[1] : "";
      const codeString = String(children).trim();

      // Mermaidコードブロックの場合
      if (language === "mermaid") {
        return <MermaidRenderer chart={codeString} />;
      }

      // インラインコードかブロックコードかを判定
      const isInline = !className && !codeString.includes("\n");

      if (isInline) {
        return (
          <code className="bg-gray-100 px-1 py-0.5 rounded text-sm" {...props}>
            {children}
          </code>
        );
      }

      // 通常のコードブロック
      return (
        <code className={className} {...props}>
          {children}
        </code>
      );
    },
  };

  return (
    <article className="prose prose-lg max-w-none prose-headings:text-gray-800 prose-a:text-blue-600 prose-code:before:content-none prose-code:after:content-none">
      <h1>{title.replace(/_/g, " ")}</h1>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw, rehypeHighlight]}
        components={components}
      >
        {content}
      </ReactMarkdown>
    </article>
  );
}

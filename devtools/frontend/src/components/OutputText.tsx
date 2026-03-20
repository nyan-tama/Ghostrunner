"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import type { Components } from "react-markdown";

interface OutputTextProps {
  text: string;
}

const components: Components = {
  code({ className, children, ...props }) {
    const codeString = String(children).trim();
    const isInline = !className && !codeString.includes("\n");

    if (isInline) {
      return (
        <code className="bg-gray-100 px-1 py-0.5 rounded text-sm" {...props}>
          {children}
        </code>
      );
    }

    return (
      <code className={className} {...props}>
        {children}
      </code>
    );
  },
};

export default function OutputText({ text }: OutputTextProps) {
  return (
    <article className="prose prose-sm max-w-none prose-code:before:content-none prose-code:after:content-none">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        components={components}
      >
        {text}
      </ReactMarkdown>
    </article>
  );
}

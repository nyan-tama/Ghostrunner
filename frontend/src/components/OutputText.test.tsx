import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import OutputText from "./OutputText";

// react-markdown はESMのみ対応のため、テスト環境用にモックする。
// 実際のMarkdown変換はreact-markdownライブラリの責務であり、
// ここではOutputTextコンポーネントが正しくpropsを渡しているかを検証する。
vi.mock("react-markdown", () => ({
  default: ({
    children,
    remarkPlugins,
    rehypePlugins,
    components,
  }: {
    children: string;
    remarkPlugins?: unknown[];
    rehypePlugins?: unknown[];
    components?: Record<string, unknown>;
  }) => {
    // Markdownの基本的なHTMLへの変換をシミュレート
    let html = children;
    // 太字: **text** -> <strong>text</strong>
    html = html.replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>");
    // インラインコード: `code` -> <code>code</code>
    html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
    // リスト: 行頭の "- " をリスト要素に変換
    const lines = html.split("\n");
    const listItems: string[] = [];
    const result: string[] = [];
    for (const line of lines) {
      if (line.startsWith("- ")) {
        listItems.push(`<li>${line.slice(2)}</li>`);
      } else {
        if (listItems.length > 0) {
          result.push(`<ul>${listItems.join("")}</ul>`);
          listItems.length = 0;
        }
        result.push(line);
      }
    }
    if (listItems.length > 0) {
      result.push(`<ul>${listItems.join("")}</ul>`);
    }
    html = result.join("");

    return (
      <div
        data-testid="react-markdown"
        data-remark-plugins={remarkPlugins ? "true" : "false"}
        data-rehype-plugins={rehypePlugins ? "true" : "false"}
        data-has-components={components ? "true" : "false"}
        dangerouslySetInnerHTML={{ __html: html }}
      />
    );
  },
}));

vi.mock("remark-gfm", () => ({ default: {} }));
vi.mock("rehype-highlight", () => ({ default: {} }));

describe("OutputText", () => {
  it("renders the provided text", () => {
    render(<OutputText text="Hello World" />);
    expect(screen.getByText("Hello World")).toBeInTheDocument();
  });

  it("converts bold Markdown syntax to strong tags", () => {
    render(<OutputText text="This is **bold** text" />);
    const strong = screen.getByText("bold");
    expect(strong.tagName).toBe("STRONG");
  });

  it("converts inline code Markdown syntax to code tags", () => {
    render(<OutputText text="Use `npm install` command" />);
    const code = screen.getByText("npm install");
    expect(code.tagName).toBe("CODE");
  });

  it("converts list Markdown syntax to li tags", () => {
    render(<OutputText text={"- item one\n- item two"} />);
    const items = screen.getAllByRole("listitem");
    expect(items).toHaveLength(2);
    expect(items[0]).toHaveTextContent("item one");
    expect(items[1]).toHaveTextContent("item two");
  });

  it("applies prose prose-sm classes to the article element", () => {
    const { container } = render(<OutputText text="test" />);
    const article = container.querySelector("article");
    expect(article).not.toBeNull();
    expect(article!.classList.contains("prose")).toBe(true);
    expect(article!.classList.contains("prose-sm")).toBe(true);
  });

  it("does not throw an error when empty string is provided", () => {
    expect(() => render(<OutputText text="" />)).not.toThrow();
    const { container } = render(<OutputText text="" />);
    const article = container.querySelector("article");
    expect(article).toBeInTheDocument();
  });

  it("passes remarkPlugins and rehypePlugins to ReactMarkdown", () => {
    render(<OutputText text="test" />);
    const markdown = screen.getByTestId("react-markdown");
    expect(markdown.getAttribute("data-remark-plugins")).toBe("true");
    expect(markdown.getAttribute("data-rehype-plugins")).toBe("true");
  });

  it("passes custom components to ReactMarkdown", () => {
    render(<OutputText text="test" />);
    const markdown = screen.getByTestId("react-markdown");
    expect(markdown.getAttribute("data-has-components")).toBe("true");
  });
});

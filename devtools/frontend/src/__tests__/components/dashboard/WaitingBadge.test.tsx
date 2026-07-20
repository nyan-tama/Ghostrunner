import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import WaitingBadge from "@/components/dashboard/WaitingBadge";
import type { IdleState } from "@/types/dashboard";

const NOW = Date.parse("2026-07-20T00:00:00Z");

function makeIdle(overrides: Partial<IdleState> = {}): IdleState {
  return {
    timestamp: new Date(NOW).toISOString(),
    preview: "何かを待っています",
    sessionCount: 1,
    summary: "",
    summarizedAt: "",
    ...overrides,
  };
}

function tsMinutesAgo(min: number): string {
  return new Date(NOW - min * 60000).toISOString();
}

describe("WaitingBadge", () => {
  describe("N分算出", () => {
    it("timestamp が 12 分前なら [質問待ち 12分] を表示", () => {
      render(<WaitingBadge idle={makeIdle({ timestamp: tsMinutesAgo(12) })} now={NOW} />);
      expect(screen.getByText(/\[質問待ち 12分\]/)).toBeInTheDocument();
    });

    it("経過 0 分（境界）は 0分", () => {
      render(<WaitingBadge idle={makeIdle({ timestamp: tsMinutesAgo(0) })} now={NOW} />);
      expect(screen.getByText(/\[質問待ち 0分\]/)).toBeInTheDocument();
    });

    it("未来の timestamp（負の差分）は 0分にクランプ", () => {
      render(<WaitingBadge idle={makeIdle({ timestamp: tsMinutesAgo(-30) })} now={NOW} />);
      expect(screen.getByText(/\[質問待ち 0分\]/)).toBeInTheDocument();
    });

    it("不正な timestamp（NaN）は 0分にフォールバック", () => {
      render(<WaitingBadge idle={makeIdle({ timestamp: "not-a-date" })} now={NOW} />);
      expect(screen.getByText(/\[質問待ち 0分\]/)).toBeInTheDocument();
    });
  });

  describe("詳細行の表示分岐", () => {
    it("summary があれば summary を表示（要約中…は出さない）", () => {
      render(
        <WaitingBadge
          idle={makeIdle({ summary: "認証情報の確認を待っています", preview: "旧プレビュー" })}
          now={NOW}
        />
      );
      expect(screen.getByText("認証情報の確認を待っています")).toBeInTheDocument();
      expect(screen.queryByText("(要約中…)")).not.toBeInTheDocument();
      // summary 優先で preview は表示しない
      expect(screen.queryByText("旧プレビュー")).not.toBeInTheDocument();
    });

    it("summary が空なら (要約中…) を表示し preview を暫定表示", () => {
      render(
        <WaitingBadge idle={makeIdle({ summary: "", preview: "ファイル名を聞いています" })} now={NOW} />
      );
      expect(screen.getByText("(要約中…)")).toBeInTheDocument();
      expect(screen.getByText("ファイル名を聞いています")).toBeInTheDocument();
    });

    it("summary も preview も空なら (プレビューなし) を表示", () => {
      render(<WaitingBadge idle={makeIdle({ summary: "", preview: "" })} now={NOW} />);
      expect(screen.getByText("(要約中…)")).toBeInTheDocument();
      expect(screen.getByText("(プレビューなし)")).toBeInTheDocument();
    });

    it("空白のみの summary/preview も空扱い（trim）でフォールバック", () => {
      render(<WaitingBadge idle={makeIdle({ summary: "   ", preview: "  " })} now={NOW} />);
      expect(screen.getByText("(要約中…)")).toBeInTheDocument();
      expect(screen.getByText("(プレビューなし)")).toBeInTheDocument();
    });
  });
});

"use client";

import type { PatrolProject, PatrolProjectState } from "@/types/patrol";
import AnswerForm from "@/components/patrol/AnswerForm";

interface ProjectCardProps {
  project: PatrolProject;
  state: PatrolProjectState | undefined;
  onRemove: (path: string) => void;
  onAnswer: (projectPath: string, answer: string) => void;
  isAnswerSubmitting: boolean;
}

function StatusBadge({ status }: { status: PatrolProjectState["status"] | "unknown" }) {
  const config: Record<string, { bg: string; text: string; label: string }> = {
    idle: { bg: "bg-gray-100", text: "text-gray-600", label: "待機中" },
    running: { bg: "bg-blue-100", text: "text-blue-700", label: "実行中" },
    waiting_approval: { bg: "bg-yellow-100", text: "text-yellow-700", label: "承認待ち" },
    queued: { bg: "bg-gray-100", text: "text-gray-500", label: "キュー待ち" },
    completed: { bg: "bg-green-100", text: "text-green-700", label: "完了" },
    error: { bg: "bg-red-100", text: "text-red-700", label: "エラー" },
    unknown: { bg: "bg-gray-50", text: "text-gray-400", label: "不明" },
  };

  const c = config[status] || config.unknown;

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${c.bg} ${c.text}`}>
      {status === "running" && (
        <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse mr-1.5" />
      )}
      {status === "queued" && (
        <span className="w-1.5 h-1.5 rounded-full border border-gray-400 border-dashed mr-1.5" />
      )}
      {c.label}
    </span>
  );
}

export default function ProjectCard({
  project,
  state,
  onRemove,
  onAnswer,
  isAnswerSubmitting,
}: ProjectCardProps) {
  const status = state?.status || "unknown";

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
      {/* ヘッダー */}
      <div className="flex items-start justify-between mb-3">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold text-gray-800 truncate">{project.name}</h3>
          <p className="text-xs text-gray-400 truncate mt-0.5">{project.path}</p>
        </div>
        <div className="flex items-center gap-2 ml-2 shrink-0">
          <StatusBadge status={status} />
          <button
            type="button"
            onClick={() => onRemove(project.path)}
            className="text-xs text-gray-400 hover:text-red-500 transition-colors"
            title="巡回対象から解除"
          >
            解除
          </button>
        </div>
      </div>

      {/* コミット情報 */}
      {state?.recent_commits && state.recent_commits.length > 0 && (
        <div className="mb-3">
          <div className="text-xs text-gray-500 mb-1">最近のコミット</div>
          <ul className="space-y-0.5">
            {state.recent_commits.slice(0, 5).map((commit, i) => (
              <li key={i} className="text-xs text-gray-600 truncate font-mono">
                {commit}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* タスク数 */}
      {state?.pending_tasks !== undefined && state.pending_tasks > 0 && (
        <div className="text-xs text-gray-500 mb-3">
          実装待ちタスク: <span className="font-medium text-gray-700">{state.pending_tasks}</span>
        </div>
      )}

      {/* エラー表示 */}
      {state?.error && (
        <div className="p-2 bg-red-50 border border-red-200 rounded text-xs text-red-700 mb-3">
          {state.error}
        </div>
      )}

      {/* 承認待ちの質問 */}
      {status === "waiting_approval" && state?.question && (
        <AnswerForm
          projectPath={project.path}
          question={state.question}
          isSubmitting={isAnswerSubmitting}
          onSubmit={onAnswer}
        />
      )}
    </div>
  );
}

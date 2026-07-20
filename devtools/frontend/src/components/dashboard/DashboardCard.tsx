"use client";

import AccentBar from "./AccentBar";
import DevSummary from "./DevSummary";
import OpsEntryComponent from "./OpsEntryComponent";
import UnansweredList from "./UnansweredList";
import WaitingBadge from "./WaitingBadge";
import type { ProjectCardData } from "@/types/dashboard";

interface DashboardCardProps {
  project: ProjectCardData;
  onAnswered: () => void;
}

export default function DashboardCard({ project, onAnswered }: DashboardCardProps) {
  // 質問待ちは独立軸（fe-W6）。undefined/null 安全に判定する
  const hasWaiting = !!project.idle;

  return (
    <div
      className={`relative pl-3 border rounded-lg p-3 bg-white shadow-sm ${
        hasWaiting ? "border-red-300 ring-1 ring-red-300" : "border-gray-200"
      }`}
    >
      <AccentBar
        attention={project.attention}
        hasUnanswered={project.unanswered.length > 0}
        hasWaiting={hasWaiting}
      />

      <div className="flex items-center justify-between mb-2">
        <h3 className="font-semibold text-gray-900">
          {project.name}
          {project.isSelf && (
            <span className="ml-1 text-xs text-gray-400">(self)</span>
          )}
        </h3>
      </div>

      {project.idle && <WaitingBadge idle={project.idle} />}

      <DevSummary kanban={project.kanban} />

      {project.warnings.length > 0 && (
        <div className="mt-2 space-y-1">
          {project.warnings.map((w, i) => (
            <div key={i} className="text-xs text-orange-600">
              {w}
            </div>
          ))}
        </div>
      )}

      {project.opsOptedIn && project.ops.length > 0 && (
        <div className="mt-2 space-y-1">
          {project.ops.map((entry) => (
            <OpsEntryComponent
              key={`${entry.kind}-${entry.account}`}
              entry={entry}
            />
          ))}
        </div>
      )}

      <UnansweredList
        projectPath={project.path}
        items={project.unanswered}
        onAnswered={onAnswered}
      />
    </div>
  );
}

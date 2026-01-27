import Link from "next/link";
import type { FileSystemEntry } from "@/lib/docs/fileSystem";

interface Props {
  entries: FileSystemEntry[];
}

export default function FolderList({ entries }: Props) {
  if (entries.length === 0) {
    return (
      <p className="text-gray-500 text-center py-8">このフォルダは空です</p>
    );
  }

  return (
    <ul className="space-y-2">
      {entries.map((entry) => (
        <li key={entry.path}>
          <Link
            href={`/docs/${entry.path}`}
            className="flex items-center gap-3 p-3 rounded-lg border border-gray-200 hover:border-blue-300 hover:bg-blue-50 transition-colors"
          >
            {entry.type === "directory" ? (
              <span className="text-blue-500 text-lg">📁</span>
            ) : (
              <span className="text-gray-400 text-lg">📄</span>
            )}
            <span className="flex-1 truncate">
              {entry.name.replace(/_/g, " ")}
            </span>
            {entry.type === "directory" && (
              <span className="text-gray-400">→</span>
            )}
          </Link>
        </li>
      ))}
    </ul>
  );
}

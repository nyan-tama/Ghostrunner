import fs from "fs/promises";
import path from "path";

export interface FileSystemEntry {
  name: string;
  path: string;
  type: "file" | "directory";
}

// プロジェクトパスから開発フォルダのルートを解決する
function resolveDocsRoot(projectPath?: string): string {
  if (projectPath) {
    return path.join(projectPath, "開発");
  }
  return path.join(process.cwd(), "..", "開発");
}

// パストラバーサル防止: 解決後のパスが docsRoot 配下であることを検証
function validatePath(absolutePath: string, docsRoot: string): void {
  const resolved = path.resolve(absolutePath);
  const resolvedRoot = path.resolve(docsRoot);
  if (!resolved.startsWith(resolvedRoot + path.sep) && resolved !== resolvedRoot) {
    throw new Error("Access denied: path traversal detected");
  }
}

// ディレクトリの内容を取得
export async function getDirectoryContents(
  relativePath: string = "",
  projectPath?: string
): Promise<FileSystemEntry[]> {
  const docsRoot = resolveDocsRoot(projectPath);
  const absolutePath = path.join(docsRoot, relativePath);

  try {
    validatePath(absolutePath, docsRoot);
  } catch {
    return [];
  }

  try {
    const entries = await fs.readdir(absolutePath, { withFileTypes: true });

    return entries
      .filter((entry) => !entry.name.startsWith("."))
      .map((entry) => ({
        name: entry.name.replace(/\.md$/, ""),
        path: path.join(relativePath, entry.name).replace(/\.md$/, ""),
        type: entry.isDirectory() ? ("directory" as const) : ("file" as const),
      }))
      .sort((a, b) => {
        // ディレクトリを先に、ファイルは日付降順
        if (a.type !== b.type) return a.type === "directory" ? -1 : 1;
        return b.name.localeCompare(a.name);
      });
  } catch {
    return [];
  }
}

// ファイルの内容を取得
export async function getFileContent(
  relativePath: string,
  projectPath?: string
): Promise<string | null> {
  const docsRoot = resolveDocsRoot(projectPath);
  const absolutePath = path.join(docsRoot, relativePath + ".md");

  try {
    validatePath(absolutePath, docsRoot);
  } catch {
    return null;
  }

  try {
    return await fs.readFile(absolutePath, "utf-8");
  } catch {
    return null;
  }
}

// パスがディレクトリかファイルか判定
export async function getPathType(
  relativePath: string,
  projectPath?: string
): Promise<"directory" | "file" | "not_found"> {
  const docsRoot = resolveDocsRoot(projectPath);
  const dirPath = path.join(docsRoot, relativePath);
  const filePath = path.join(docsRoot, relativePath + ".md");

  try {
    validatePath(dirPath, docsRoot);
    validatePath(filePath, docsRoot);
  } catch {
    return "not_found";
  }

  try {
    const stat = await fs.stat(dirPath);
    if (stat.isDirectory()) return "directory";
  } catch {
    // ディレクトリではない
  }

  try {
    await fs.access(filePath);
    return "file";
  } catch {
    // ファイルでもない
  }

  return "not_found";
}

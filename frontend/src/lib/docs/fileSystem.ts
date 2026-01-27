import fs from "fs/promises";
import path from "path";

// 開発フォルダの絶対パス
const DOCS_ROOT = path.join(process.cwd(), "..", "開発");

export interface FileSystemEntry {
  name: string;
  path: string;
  type: "file" | "directory";
}

// ディレクトリの内容を取得
export async function getDirectoryContents(
  relativePath: string = ""
): Promise<FileSystemEntry[]> {
  const absolutePath = path.join(DOCS_ROOT, relativePath);

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
  relativePath: string
): Promise<string | null> {
  const absolutePath = path.join(DOCS_ROOT, relativePath + ".md");
  try {
    return await fs.readFile(absolutePath, "utf-8");
  } catch {
    return null;
  }
}

// パスがディレクトリかファイルか判定
export async function getPathType(
  relativePath: string
): Promise<"directory" | "file" | "not_found"> {
  const dirPath = path.join(DOCS_ROOT, relativePath);
  const filePath = path.join(DOCS_ROOT, relativePath + ".md");

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

import { spawn } from "child_process";
import path from "path";

export async function POST() {
  if (process.env.NODE_ENV !== "development") {
    return new Response(null, { status: 403 });
  }

  // プロジェクトルートを明示的に指定（frontend から一つ上のディレクトリ）
  const projectRoot = path.resolve(process.cwd(), "..");

  spawn("make", ["restart-frontend"], {
    cwd: projectRoot,
    detached: true,
    stdio: "ignore",
  }).unref();

  return new Response(null, { status: 202 });
}

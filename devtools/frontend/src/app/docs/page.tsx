import { getDirectoryContents } from "@/lib/docs/fileSystem";
import FolderList from "@/components/docs/FolderList";
import Link from "next/link";

export const metadata = {
  title: "Docs - Ghost Runner",
};

interface Props {
  searchParams: Promise<{ project?: string }>;
}

export default async function DocsPage({ searchParams }: Props) {
  const { project } = await searchParams;
  const entries = await getDirectoryContents("", project);

  return (
    <div className="max-w-[900px] mx-auto px-5 py-5 bg-gray-100 min-h-screen">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-800">開発ドキュメント</h1>
        <Link
          href="/"
          className="px-3 py-1 text-xs bg-gray-200 text-gray-600 rounded hover:bg-gray-300 transition-colors"
        >
          Home
        </Link>
      </div>

      <FolderList entries={entries} project={project} />
    </div>
  );
}

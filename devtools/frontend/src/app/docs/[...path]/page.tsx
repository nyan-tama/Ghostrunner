import { notFound } from "next/navigation";
import {
  getDirectoryContents,
  getFileContent,
  getPathType,
} from "@/lib/docs/fileSystem";
import FolderList from "@/components/docs/FolderList";
import Breadcrumb from "@/components/docs/Breadcrumb";
import MarkdownViewer from "@/components/docs/MarkdownViewer";
import Link from "next/link";

interface Props {
  params: Promise<{ path: string[] }>;
  searchParams: Promise<{ project?: string }>;
}

export default async function DocsPathPage({ params, searchParams }: Props) {
  const { path: pathSegments } = await params;
  const { project } = await searchParams;
  const relativePath = pathSegments.map(decodeURIComponent).join("/");
  const pathType = await getPathType(relativePath, project);

  if (pathType === "not_found") {
    notFound();
  }

  if (pathType === "directory") {
    const entries = await getDirectoryContents(relativePath, project);
    return (
      <div className="max-w-[900px] mx-auto px-5 py-5 bg-gray-100 min-h-screen">
        <div className="flex justify-between items-center mb-4">
          <Link
            href="/"
            className="px-3 py-1 text-xs bg-gray-200 text-gray-600 rounded hover:bg-gray-300 transition-colors"
          >
            Home
          </Link>
        </div>
        <Breadcrumb path={relativePath} project={project} />
        <FolderList entries={entries} project={project} />
      </div>
    );
  }

  // ファイル表示
  const content = await getFileContent(relativePath, project);
  if (!content) {
    notFound();
  }

  const fileName = pathSegments[pathSegments.length - 1];

  return (
    <div className="max-w-[900px] mx-auto px-5 py-5 bg-gray-100 min-h-screen">
      <div className="flex justify-between items-center mb-4">
        <Link
          href="/"
          className="px-3 py-1 text-xs bg-gray-200 text-gray-600 rounded hover:bg-gray-300 transition-colors"
        >
          Home
        </Link>
      </div>
      <Breadcrumb path={relativePath} project={project} />
      <div className="bg-white rounded-lg p-6 shadow-sm">
        <MarkdownViewer content={content} title={decodeURIComponent(fileName)} />
      </div>
    </div>
  );
}

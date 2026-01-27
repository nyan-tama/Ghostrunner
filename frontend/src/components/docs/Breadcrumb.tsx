import Link from "next/link";

interface BreadcrumbProps {
  path: string;
  project?: string;
}

function buildHref(basePath: string, project?: string): string {
  if (project) {
    return `${basePath}?project=${encodeURIComponent(project)}`;
  }
  return basePath;
}

export default function Breadcrumb({ path, project }: BreadcrumbProps) {
  const segments = path.split("/").filter(Boolean);

  return (
    <nav className="flex items-center gap-2 mb-6 text-sm text-gray-600">
      <Link href={buildHref("/docs", project)} className="hover:text-blue-600">
        開発
      </Link>
      {segments.map((segment, index) => {
        const href = "/docs/" + segments.slice(0, index + 1).join("/");
        const isLast = index === segments.length - 1;

        return (
          <span key={href} className="flex items-center gap-2">
            <span>/</span>
            {isLast ? (
              <span className="text-gray-900 font-medium">
                {decodeURIComponent(segment)}
              </span>
            ) : (
              <Link href={buildHref(href, project)} className="hover:text-blue-600">
                {decodeURIComponent(segment)}
              </Link>
            )}
          </span>
        );
      })}
    </nav>
  );
}

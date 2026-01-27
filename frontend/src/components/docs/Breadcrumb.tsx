import Link from "next/link";

interface Props {
  path: string;
}

export default function Breadcrumb({ path }: Props) {
  const segments = path.split("/").filter(Boolean);

  return (
    <nav className="flex items-center gap-2 mb-6 text-sm text-gray-600">
      <Link href="/docs" className="hover:text-blue-600">
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
              <Link href={href} className="hover:text-blue-600">
                {decodeURIComponent(segment)}
              </Link>
            )}
          </span>
        );
      })}
    </nav>
  );
}

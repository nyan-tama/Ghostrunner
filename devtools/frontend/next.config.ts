import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async redirects() {
    return [
      {
        source: "/patrol",
        destination: "/dashboard",
        permanent: true,
      },
    ];
  },
  async rewrites() {
    return [
      {
        source: "/api/prompt",
        destination: "http://localhost:3456/api/prompt",
      },
      {
        source: "/api/events",
        destination: "http://localhost:3456/api/events",
      },
      {
        source: "/api/status",
        destination: "http://localhost:3456/api/status",
      },
      {
        source: "/api/sessions/:path*",
        destination: "http://localhost:3456/api/sessions/:path*",
      },
      {
        source: "/api/sessions",
        destination: "http://localhost:3456/api/sessions",
      },
      {
        source: "/api/tts",
        destination: "http://localhost:8888/api/tts",
      },
      {
        source: "/api/:path*",
        destination: "http://localhost:8888/api/:path*",
      },
    ];
  },
  devIndicators: false,
  allowedDevOrigins: ["usermac-mini.tail85f9ea.ts.net", "100.68.245.31", "100.104.204.15"],
  experimental: {
    // Next.js 16 の global-error プリレンダリング問題を回避
    preloadEntriesOnStart: false,
  },
};

export default nextConfig;

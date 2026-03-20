import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
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

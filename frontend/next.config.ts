import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:8080/api/:path*",
      },
    ];
  },
  devIndicators: false,
  experimental: {
    // Next.js 16 の global-error プリレンダリング問題を回避
    preloadEntriesOnStart: false,
  },
};

export default nextConfig;

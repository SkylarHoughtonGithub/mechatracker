import type { NextConfig } from "next";
import path from "path";

// BACKEND_URL is a server-side env var (not NEXT_PUBLIC_*) because browsers
// never see it — Next.js's rewrite forwards /api/backend/* to the real backend
// at request time. Defaults to the local dev port.
const backendUrl = process.env.BACKEND_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  turbopack: {
    root: path.resolve(__dirname, '..'),
  },
  async rewrites() {
    return [
      {
        source: '/api/backend/:path*',
        destination: `${backendUrl}/:path*`,
      },
    ];
  },
};

export default nextConfig;

import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Emits .next/standalone — a self-contained server with only the deps it
  // actually uses. Keeps the Dokploy image small and removes the need to ship
  // node_modules into the runtime stage.
  output: "standalone",
  // Another lockfile exists higher up on this machine; pin tracing to web/ so
  // the standalone bundle keeps server.js at its root (the Dockerfile relies
  // on that layout).
  outputFileTracingRoot: __dirname,
  reactStrictMode: true,
};

export default nextConfig;

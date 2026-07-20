import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
  images: {
    unoptimized: true,
  },
  // Static assets are served by the Go embed FileServer from /
  trailingSlash: false,
};

export default nextConfig;

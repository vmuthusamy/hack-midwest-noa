import { fileURLToPath } from "node:url";
const config = {
  turbopack: { root: fileURLToPath(new URL(".", import.meta.url)) },
  poweredByHeader: false,
  devIndicators: false,
  async rewrites() {
    return [{ source: '/api/:path*', destination: 'http://127.0.0.1:8087/api/:path*' }];
  },
};
export default config;

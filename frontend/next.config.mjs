import { fileURLToPath } from 'node:url';
const config = {
  output: 'standalone',
  turbopack: { root: fileURLToPath(new URL('.', import.meta.url)) },
  poweredByHeader: false,
  devIndicators: false,
};
export default config;

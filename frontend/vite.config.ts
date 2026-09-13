import path from 'path';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';
// import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite';
import tanstackRouter from '@tanstack/router-plugin/vite';

// https://vite.dev/config/
const basePath = process.env.VITE_BASE_PATH || '/';
const base = basePath === '/' ? '/' : basePath.endsWith('/') ? basePath : `${basePath}/`;

export default defineConfig({
  base,
  define: {
    __API_BASE_PATH__: JSON.stringify(process.env.VITE_API_BASE_PATH || ''),
  },
  plugins: [
    tanstackRouter({
      target: 'react',
      autoCodeSplitting: true,
    }),
    react(),
    // React table does not work with react-compiler, disable for now.
    // react({
    //   babel: {
    //     plugins: ['babel-plugin-react-compiler'],
    //   },
    // }),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),

      // fix loading all icon chunks in dev mode
      // https://github.com/tabler/tabler-icons/issues/1233
      '@tabler/icons-react': '@tabler/icons-react/dist/esm/icons/index.mjs',
    },
  },
  server: {
    port: process.env.VITE_PORT ? parseInt(process.env.VITE_PORT) : 5173,
    proxy: {
      '/admin': {
        target: process.env.VITE_API_URL || 'http://localhost:8090',
        changeOrigin: true,
      },
      '/oauth': {
        target: process.env.VITE_API_URL || 'http://localhost:8090',
        changeOrigin: true,
        bypass: (req) => {
          if (req.url?.includes('idp-callback')) {
            return req.url;
          }
        },
      },
      '/v1': {
        target: process.env.VITE_API_URL || 'http://localhost:8090',
        changeOrigin: true,
      },
    },
  },
});

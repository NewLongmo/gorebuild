import vue from '@vitejs/plugin-vue';
import { fileURLToPath, URL } from 'node:url';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  return {
    plugins: [vue()],
    server: {
      port: 5173,
      proxy: {
        '/api': {
          target: env.VITE_API_BASE_URL || 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    build: {
      chunkSizeWarningLimit: 750,
      rollupOptions: {
        output: {
          manualChunks(id) {
            const normalizedId = id.replace(/\\/g, '/');
            if (!normalizedId.includes('node_modules')) {
              return undefined;
            }
            if (
              normalizedId.includes('/vue/') ||
              normalizedId.includes('/vue-router/') ||
              normalizedId.includes('/pinia/')
            ) {
              return 'vue';
            }
            if (normalizedId.includes('/axios/')) {
              return 'http';
            }
            if (normalizedId.includes('@ant-design/icons-vue')) {
              return 'antd-icons';
            }
            if (normalizedId.includes('ant-design-vue')) {
              return 'antd';
            }
            if (
              normalizedId.includes('@ant-design') ||
              normalizedId.includes('/rc-') ||
              normalizedId.includes('/@rc-component/')
            ) {
              return 'antd-runtime';
            }
            return 'vendor';
          },
        },
      },
    },
  };
});

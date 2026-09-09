import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.svg"],
      manifest: {
        name: "Cal",
        short_name: "Cal",
        description: "Self-hosted minimal planner",
        theme_color: "#101316",
        background_color: "#f5f2eb",
        display: "standalone",
        start_url: "/",
        icons: [
          { src: "/pwa-192.svg", sizes: "192x192", type: "image/svg+xml" },
          { src: "/pwa-512.svg", sizes: "512x512", type: "image/svg+xml" }
        ]
      },
      workbox: {
        runtimeCaching: [
          {
            urlPattern: ({ url }) => url.pathname.startsWith("/api/entries"),
            handler: "NetworkFirst",
            options: { cacheName: "cal-api" }
          }
        ]
      }
    })
  ],
  server: {
    proxy: {
      "/api": "http://localhost:8080"
    }
  }
});

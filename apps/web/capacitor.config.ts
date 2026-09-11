import type { CapacitorConfig } from "@capacitor/cli";

// The Android shell wraps the built PWA. The API URL is runtime-relative so
// the same bundle works against any self-hosted origin; set server.url during
// development to point at a live dev server if needed.
const config: CapacitorConfig = {
  appId: "dev.cal.app",
  appName: "Cal",
  webDir: "dist",
  server: {
    // http origin so the app can talk to plain-HTTP self-hosted servers —
    // with the default https scheme the webview blocks those fetches as
    // mixed content.
    androidScheme: "http",
  },
  android: {
    allowMixedContent: false,
  },
  plugins: {
    SplashScreen: { launchAutoHide: true },
  },
};

export default config;

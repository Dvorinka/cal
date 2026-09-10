import type { CapacitorConfig } from "@capacitor/cli";

// The Android shell wraps the built PWA. The API URL is runtime-relative so
// the same bundle works against any self-hosted origin; set server.url during
// development to point at a live dev server if needed.
const config: CapacitorConfig = {
  appId: "dev.cal.app",
  appName: "Cal",
  webDir: "dist",
  android: {
    allowMixedContent: false,
  },
  plugins: {
    SplashScreen: { launchAutoHide: true },
  },
};

export default config;

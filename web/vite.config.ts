import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Vite outputs directly into the Go binary's embed directory so a single
// `go build ./cmd/mafia-game` packages the freshest frontend (NFR-U4-B3).
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../cmd/mafia-game/web/dist",
    emptyOutDir: true,
    sourcemap: false,
  },
  server: {
    proxy: {
      "/ws": { target: "ws://localhost:8080", ws: true },
      "/api": { target: "http://localhost:8080" },
    },
  },
});

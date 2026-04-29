import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/tests/setup.ts"],
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      // Coverage targets the core modules called out by NFR-U5-M3
      // (reducer, hooks, validation). UI presentation components are
      // covered by manual + future integration tests instead — see
      // construction/u5-web-frontend/nfr-requirements/nfr-requirements.md.
      include: [
        "src/context/**/*.ts",
        "src/hooks/**/*.ts",
        "src/components/NicknameForm.tsx",
      ],
      exclude: ["src/**/*.test.{ts,tsx}", "src/tests/**"],
    },
  },
});

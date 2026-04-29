import type { Severity } from "../../types/wire";

interface Props {
  ann?: { subtitle: string; severity: Severity };
}

const COLOR: Record<Severity, string> = {
  INFO: "var(--paper-2)",
  EMPHASIS: "var(--gold)",
  WARN: "var(--red)",
};

export function SubtitleArea({ ann }: Props) {
  if (!ann) return null;
  const isWarn = ann.severity === "WARN";
  return (
    <div
      data-severity={ann.severity}
      role="status"
      aria-live="polite"
      style={{
        textAlign: "center",
        fontFamily: "var(--font-serif)",
        fontStyle: "italic",
        fontSize: "1.6rem",
        color: COLOR[ann.severity],
        padding: "1rem 1.5rem",
        minHeight: "3rem",
        textShadow: ann.severity === "EMPHASIS" ? "0 0 18px var(--gold-glow)" : isWarn ? "0 0 18px var(--red-glow)" : undefined,
        letterSpacing: "0.04em",
      }}
    >
      “{ann.subtitle}”
    </div>
  );
}

import type { Severity } from "../../types/wire";

interface Props {
  ann?: { subtitle: string; severity: Severity };
}

const COLOR: Record<Severity, string> = {
  INFO: "var(--info)",
  EMPHASIS: "var(--emphasis)",
  WARN: "var(--warn)",
};

export function SubtitleArea({ ann }: Props) {
  if (!ann) return null;
  return (
    <div
      data-severity={ann.severity}
      role="status"
      aria-live="polite"
      style={{
        textAlign: "center",
        fontSize: "2rem",
        fontWeight: 500,
        color: COLOR[ann.severity],
        padding: "1rem",
        minHeight: "4rem",
      }}
    >
      {ann.subtitle}
    </div>
  );
}

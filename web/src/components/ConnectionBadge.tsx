import type { ConnectionStatus } from "../context/reducer";

const LABEL: Record<ConnectionStatus, { dot: string; text: string; color: string }> = {
  connecting: { dot: "🔄", text: "연결 중…", color: "var(--fg-muted)" },
  connected: { dot: "🟢", text: "연결됨", color: "var(--alive)" },
  reconnecting: { dot: "⚠️", text: "재연결 중…", color: "var(--emphasis)" },
  closed: { dot: "🔴", text: "연결 끊김", color: "var(--warn)" },
};

interface Props {
  status: ConnectionStatus;
}

export function ConnectionBadge({ status }: Props) {
  const cfg = LABEL[status];
  return (
    <div
      role="status"
      aria-live="polite"
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: "0.5rem",
        padding: "0.25rem 0.75rem",
        background: "var(--card)",
        border: `1px solid ${cfg.color}`,
        borderRadius: "9999px",
        color: cfg.color,
        fontSize: "0.875rem",
      }}
    >
      <span aria-hidden>{cfg.dot}</span>
      <span>{cfg.text}</span>
    </div>
  );
}

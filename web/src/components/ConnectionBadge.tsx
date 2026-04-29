import type { ConnectionStatus } from "../context/reducer";

const LABEL: Record<ConnectionStatus, { dot: string; text: string; cls: string }> = {
  connecting: { dot: "◌", text: "연결 중", cls: "dim" },
  connected: { dot: "●", text: "연결됨", cls: "alive" },
  reconnecting: { dot: "◐", text: "재연결 중", cls: "warn" },
  closed: { dot: "✕", text: "연결 끊김", cls: "red" },
};

interface Props {
  status: ConnectionStatus;
}

export function ConnectionBadge({ status }: Props) {
  const cfg = LABEL[status];
  return (
    <span
      role="status"
      aria-live="polite"
      className={"tag " + cfg.cls}
    >
      <span aria-hidden>{cfg.dot}</span>
      <span>{cfg.text}</span>
    </span>
  );
}

interface Props {
  paused: boolean;
}

// PauseBadge is a fixed-position banner shown while the host has frozen
// the active timer (Iteration 5 R4). It overlays the rest of the public
// view with a soft tint so spectators immediately notice the freeze
// while still being able to read the underlying state.
export function PauseBadge({ paused }: Props) {
  if (!paused) return null;
  return (
    <div
      role="status"
      aria-live="polite"
      style={{
        position: "fixed",
        top: 0,
        left: 0,
        right: 0,
        padding: "0.5rem 1rem",
        textAlign: "center",
        background: "rgba(255, 200, 0, 0.85)",
        color: "#222",
        fontWeight: 600,
        fontSize: "1.1rem",
        zIndex: 100,
        letterSpacing: "0.05em",
      }}
    >
      ⏸ 진행이 일시정지되었습니다
    </div>
  );
}

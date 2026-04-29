interface Props {
  paused: boolean;
}

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
        padding: "0.65rem 1rem",
        textAlign: "center",
        background: "linear-gradient(180deg, rgba(20,14,10,0.95), rgba(10,8,7,0.92))",
        color: "var(--gold)",
        borderBottom: "1px solid var(--gold)",
        fontFamily: "var(--font-display)",
        fontWeight: 700,
        fontSize: "0.95rem",
        letterSpacing: "0.32em",
        textTransform: "uppercase",
        zIndex: 100,
        boxShadow: "0 4px 24px rgba(0,0,0,0.5), inset 0 -1px 0 var(--gold-glow)",
        animation: "pulse-soft 2.4s ease-in-out infinite",
      }}
    >
      ⏸&nbsp;&nbsp;진행이 일시정지되었습니다
    </div>
  );
}

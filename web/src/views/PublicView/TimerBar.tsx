import { useEffect, useState } from "react";

interface Props {
  deadline?: string;
  // Iteration 5 — when paused, freeze the displayed value. The host has
  // suspended the active timer and the deadline will be shifted forward
  // on resume; until then we render the residual seconds at the moment
  // the pause was first observed.
  paused?: boolean;
  // Optional label to clarify which window is counting down. Used for
  // NightStep ("마피아의 시간" etc.) so the public screen carries clear
  // context next to the seconds.
  label?: string;
}

function formatClock(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

export function TimerBar({ deadline, paused, label }: Props) {
  const [now, setNow] = useState(() => Date.now());
  const [frozenAt, setFrozenAt] = useState<number | null>(null);

  useEffect(() => {
    if (!deadline || paused) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [deadline, paused]);

  useEffect(() => {
    if (paused) {
      setFrozenAt((prev) => prev ?? Date.now());
    } else {
      setFrozenAt(null);
    }
  }, [paused]);

  if (!deadline) return null;

  const target = new Date(deadline).getTime();
  if (Number.isNaN(target)) return null;

  const sampleNow = paused && frozenAt != null ? frozenAt : now;
  const remainingMs = Math.max(0, target - sampleNow);
  const seconds = Math.ceil(remainingMs / 1000);
  const danger = seconds <= 10;
  const color = paused ? "var(--paper-dim)" : danger ? "var(--red)" : "var(--gold)";

  return (
    <div
      role="timer"
      aria-live="off"
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        gap: "1rem",
        margin: "0.75rem 0 1.25rem",
        opacity: paused ? 0.7 : 1,
      }}
    >
      {label ? (
        <span className="eyebrow" style={{ color: "var(--paper-2)" }}>
          {label}
        </span>
      ) : null}
      <span
        className="mono"
        style={{
          fontSize: "2rem",
          letterSpacing: "0.25em",
          color,
          textShadow: danger && !paused ? "0 0 16px var(--red-glow)" : undefined,
        }}
      >
        {formatClock(seconds)}
      </span>
      {paused && (
        <span className="tag warn" style={{ borderColor: "var(--warn)", color: "var(--warn)" }}>
          PAUSED
        </span>
      )}
    </div>
  );
}

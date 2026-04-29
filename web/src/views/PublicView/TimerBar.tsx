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

// TimerBar renders the seconds remaining until `deadline` and ticks once
// per second. We update via setInterval rather than per-frame to keep
// the React re-render cost minimal (NFR-U5-P5).
export function TimerBar({ deadline, paused, label }: Props) {
  const [now, setNow] = useState(() => Date.now());
  const [frozenAt, setFrozenAt] = useState<number | null>(null);

  useEffect(() => {
    if (!deadline || paused) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [deadline, paused]);

  // Capture the wall-clock instant at which the pause started so the
  // frozen countdown matches what the host saw the moment they hit
  // Pause. Reset when paused returns to false.
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

  return (
    <div
      role="timer"
      aria-live="off"
      style={{
        textAlign: "center",
        fontSize: "1.5rem",
        color: paused ? "var(--fg-muted)" : (danger ? "var(--warn)" : "var(--fg-muted)"),
        margin: "0.5rem 0",
        opacity: paused ? 0.7 : 1,
      }}
    >
      {label ? <span style={{ marginRight: "0.5rem" }}>{label}</span> : null}
      남은 시간: {seconds}초{paused ? " (일시정지)" : ""}
    </div>
  );
}

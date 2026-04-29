import type { Phase, Player } from "../../types/wire";

interface Props {
  players: Player[];
  phase: Phase;
}

const ROLE_KR: Record<string, string> = {
  MAFIA: "마피아",
  CITIZEN: "시민",
  DOCTOR: "의사",
  POLICE: "경찰",
};

// PlayersGrid is the public-screen grid of player cards. Roles stay
// hidden until PhaseEnd, when GameEnded delivers the full reveal. During
// LOBBY each card carries a "대기 중" status line so spectators can see
// who has joined ahead of host:start.
export function PlayersGrid({ players, phase }: Props) {
  const reveal = phase === "END";
  const lobby = phase === "LOBBY";
  return (
    <div
      className="vote-tile-grid"
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fit, minmax(9rem, 1fr))",
        gap: "0.85rem",
        padding: "0.5rem 1.25rem 1rem",
      }}
    >
      {players.map((p, i) => (
        <div
          key={p.id}
          className={"vote-tile" + (!p.alive ? " dead" : "")}
          style={{ cursor: "default" }}
        >
          <span className="vt-meta">{String(i + 1).padStart(2, "0")}</span>
          <div className={"avatar" + (!p.alive ? " dead" : "")}>{p.name.slice(0, 1)}</div>
          <div className="vt-name" style={{ color: p.alive ? "var(--paper)" : "var(--dead)" }}>
            {p.name}
            {!p.alive && <span aria-label="사망" style={{ marginLeft: "0.25rem" }}>✕</span>}
          </div>
          {reveal && p.role && (
            <span
              className="mono"
              style={{
                fontSize: "0.7rem",
                color: p.role === "MAFIA" ? "var(--red)" : "var(--gold)",
                letterSpacing: "0.18em",
              }}
            >
              {ROLE_KR[p.role] ?? p.role}
            </span>
          )}
          {lobby && (
            <span className="vt-meta" style={{ color: "var(--paper-dim)" }}>
              READY
            </span>
          )}
          {!p.alive && (
            <span
              aria-hidden
              style={{
                position: "absolute",
                inset: 0,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontFamily: "var(--font-display)",
                color: "var(--red-deep)",
                fontSize: "2rem",
                letterSpacing: "0.3em",
                opacity: 0.45,
                pointerEvents: "none",
              }}
            >
              ×
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

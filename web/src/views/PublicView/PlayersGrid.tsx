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
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fit, minmax(8rem, 1fr))",
        gap: "1rem",
        padding: "1rem",
      }}
    >
      {players.map((p) => (
        <div
          key={p.id}
          style={{
            background: "var(--card)",
            border: `2px solid ${p.alive ? "var(--alive)" : "var(--dead)"}`,
            borderRadius: "0.5rem",
            padding: "1rem",
            textAlign: "center",
            opacity: p.alive ? 1 : 0.6,
          }}
        >
          <div style={{ fontSize: "1.5rem", fontWeight: 600 }}>
            {p.name} {!p.alive && <span aria-label="사망">✕</span>}
          </div>
          {reveal && p.role && (
            <div style={{ marginTop: "0.5rem", color: "var(--fg-muted)" }}>
              {ROLE_KR[p.role] ?? p.role}
            </div>
          )}
          {lobby && (
            <div style={{ marginTop: "0.5rem", color: "var(--fg-muted)" }}>
              대기 중
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

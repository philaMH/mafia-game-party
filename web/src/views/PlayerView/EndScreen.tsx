import type { State } from "../../types/wire";

interface Props {
  state: State;
}

const ROLE_KR: Record<string, string> = {
  MAFIA: "마피아",
  CITIZEN: "시민",
  DOCTOR: "의사",
  POLICE: "경찰",
};

const WINNER_KR: Record<string, string> = {
  MAFIA: "마피아의 승리",
  CITIZEN: "시민의 승리",
};

const REASON_KR: Record<string, string> = {
  MAFIA_WIN: "마피아 승리",
  CITIZEN_WIN: "시민 승리",
  HOST_FORCE_END: "진행자 강제 종료",
};

export function EndScreen({ state }: Props) {
  const winner = state.winner ? WINNER_KR[state.winner] : "게임 종료";
  const reason = state.endReason ? REASON_KR[state.endReason] ?? state.endReason : "";

  return (
    <section style={{ padding: "1rem", textAlign: "center" }}>
      <h2 style={{ fontSize: "2rem", color: "var(--emphasis)" }}>{winner}</h2>
      {reason && <p style={{ color: "var(--fg-muted)" }}>{reason}</p>}
      <h3 style={{ marginTop: "1.5rem" }}>최종 정체 공개</h3>
      <ul
        style={{
          listStyle: "none",
          padding: 0,
          margin: "0 auto",
          maxWidth: "20rem",
        }}
      >
        {state.players.map((p) => (
          <li
            key={p.id}
            style={{
              display: "flex",
              justifyContent: "space-between",
              padding: "0.5rem 0",
              borderBottom: "1px solid var(--border)",
            }}
          >
            <span>{p.name}</span>
            <span style={{ color: "var(--fg-muted)" }}>
              {p.role ? ROLE_KR[p.role] ?? p.role : "?"}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

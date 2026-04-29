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

const REASON_KR: Record<string, string> = {
  MAFIA_WIN: "마피아 승리",
  CITIZEN_WIN: "시민 승리",
  HOST_FORCE_END: "진행자 강제 종료",
};

export function EndScreen({ state }: Props) {
  const isMafiaWin = state.winner === "MAFIA";
  const winnerTitle = isMafiaWin ? "MAFIA WINS" : state.winner === "CITIZEN" ? "CITIZENS WIN" : "GAME OVER";
  const winnerSub = isMafiaWin ? "마 피 아 의 승 리" : state.winner === "CITIZEN" ? "시 민 의 승 리" : "게 임 종 료";
  const flavor = isMafiaWin
    ? "도시는 다시 그들의 손에 떨어졌다. 시민들은 끝내 진실을 보지 못했다."
    : state.winner === "CITIZEN"
      ? "동이 트고, 그림자 속의 적들은 모두 모습을 드러냈다."
      : "막은 내렸다. 모두 자리를 떠났다.";
  const reason = state.endReason ? REASON_KR[state.endReason] ?? state.endReason : "";

  return (
    <section style={{ padding: "0.5rem 0 1rem", textAlign: "center" }}>
      <div className="eyebrow red">GAME OVER · 게 임 종 료</div>
      <h1 className="mafia-title stone sm" style={{ marginTop: "0.5rem", fontSize: "2.6rem" }}>
        {winnerTitle}
      </h1>
      <div className="mafia-sub" style={{ fontSize: "0.95rem", marginTop: "0.4rem" }}>
        {winnerSub}
      </div>
      <p
        className="serif"
        style={{
          fontStyle: "italic",
          color: "var(--paper-dim)",
          marginTop: "0.85rem",
          fontSize: "0.95rem",
          maxWidth: "26rem",
          marginInline: "auto",
          lineHeight: 1.5,
        }}
      >
        “{flavor}”
      </p>
      {reason && (
        <p className="mono" style={{ marginTop: "0.5rem", color: "var(--paper-dim)", fontSize: "0.8rem", letterSpacing: "0.18em" }}>
          {reason}
        </p>
      )}

      <div
        className="gold-corners"
        style={{
          marginTop: "1.5rem",
          padding: "1rem",
          background: "rgba(8,6,5,0.7)",
          border: "1px solid rgba(201,169,97,0.3)",
        }}
      >
        <span className="corner-tl" />
        <span className="corner-br" />
        <div className="eyebrow" style={{ marginBottom: "0.75rem" }}>
          FINAL DOSSIER · 최종 명단
        </div>
        <ul
          style={{
            listStyle: "none",
            padding: 0,
            margin: 0,
            display: "grid",
            gridTemplateColumns: "1fr 1fr",
            gap: "0.5rem",
          }}
        >
          {state.players.map((p) => {
            const isMafia = p.role === "MAFIA";
            return (
              <li
                key={p.id}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "0.6rem",
                  padding: "0.5rem 0.65rem",
                  border: "1px solid " + (isMafia ? "var(--red)" : "rgba(201,169,97,0.25)"),
                  background: "rgba(0,0,0,0.45)",
                }}
              >
                <span className={"avatar sm" + (isMafia ? " target" : "")}>{p.name.slice(0, 1)}</span>
                <div style={{ flex: 1, textAlign: "left", overflow: "hidden" }}>
                  <div style={{ fontSize: "0.85rem", color: "var(--paper)", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                    {p.name}
                  </div>
                  <div
                    className="mono"
                    style={{
                      fontSize: "0.7rem",
                      color: isMafia ? "var(--red)" : "var(--gold)",
                      letterSpacing: "0.18em",
                    }}
                  >
                    {p.role ? ROLE_KR[p.role] ?? p.role : "?"}
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      </div>
    </section>
  );
}

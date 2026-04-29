import { PlayerPicker } from "../../components/PlayerPicker";
import type { OutgoingMsg, PlayerID, State } from "../../types/wire";

interface Props {
  state: State;
  me: PlayerID;
  send: (msg: OutgoingMsg) => void;
}

const TEAM_KR: Record<string, string> = {
  MAFIA: "마피아",
  CITIZEN: "시민",
};

export function PolicePicker({ state, me, send }: Props) {
  const isMyTurn = state.nightStep === "POLICE";
  const candidates = state.players.filter((p) => p.alive && p.id !== me);
  const checked = !!state.policeCheckedThisNight;
  const history = state.policeHistory ?? [];
  const nameOf = (id: PlayerID) =>
    state.players.find((p) => p.id === id)?.name ?? id;

  return (
    <section style={{ padding: "0.5rem 0 1rem" }}>
      <div className="eyebrow">POLICE · 경찰 조사</div>
      <h3
        className="h-display"
        style={{ fontSize: "1.2rem", color: "var(--paper)", margin: "0.5rem 0 0.75rem", letterSpacing: "0.16em" }}
      >
        의심스러운 자를 조사하라
      </h3>
      <p
        className="serif"
        style={{ color: "var(--paper-dim)", fontStyle: "italic", lineHeight: 1.6, fontSize: "0.95rem", marginBottom: "0.85rem" }}
      >
        {!isMyTurn
          ? "아직 경찰 차례가 아닙니다. 사회자의 진행을 기다리세요."
          : checked
            ? "이번 밤에는 조사를 완료했습니다."
            : "조사할 대상을 선택하세요. 한 밤에 한 번 가능합니다."}
      </p>
      <PlayerPicker
        players={candidates}
        disabled={!isMyTurn || checked}
        onChange={(target) => send({ type: "submit:police-check", target })}
      />
      {history.length > 0 && (
        <div
          className="gold-frame"
          style={{ marginTop: "1rem", padding: "0.85rem 1rem" }}
        >
          <div className="eyebrow" style={{ marginBottom: "0.5rem" }}>
            DOSSIER · 조사 기록
          </div>
          <ul
            style={{
              listStyle: "none",
              margin: 0,
              padding: 0,
              display: "flex",
              flexDirection: "column",
              gap: "0.4rem",
            }}
          >
            {history.map((rec, idx) => (
              <li
                key={`${rec.day}-${rec.target}-${idx}`}
                className="serif"
                style={{
                  fontSize: "0.92rem",
                  color: "var(--paper)",
                  paddingBottom: "0.4rem",
                  borderBottom: "1px dashed rgba(201,169,97,0.15)",
                }}
              >
                <span className="mono" style={{ color: "var(--paper-dim)", marginRight: "0.5rem" }}>
                  D{rec.day}
                </span>
                <strong style={{ color: "var(--paper)" }}>{nameOf(rec.target)}</strong>
                <span style={{ color: "var(--paper-dim)" }}> 은(는) </span>
                <strong style={{ color: rec.team === "MAFIA" ? "var(--red)" : "var(--alive)" }}>
                  {TEAM_KR[rec.team]}
                </strong>{" "}
                <span style={{ color: "var(--paper-dim)" }}>진영</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}

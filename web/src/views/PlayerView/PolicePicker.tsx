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

// PolicePicker is the once-per-night investigation input. After the
// engine flips `policeCheckedThisNight`, the picker disables itself.
// The full investigation history (every prior night) is rendered below
// the picker so a returning officer can review past findings — backed
// by State.policeHistory which the server snapshots privately.
export function PolicePicker({ state, me, send }: Props) {
  const isMyTurn = state.nightStep === "POLICE";
  const candidates = state.players.filter((p) => p.alive && p.id !== me);
  const checked = !!state.policeCheckedThisNight;
  const history = state.policeHistory ?? [];
  const nameOf = (id: PlayerID) =>
    state.players.find((p) => p.id === id)?.name ?? id;

  return (
    <section style={{ padding: "1rem" }}>
      <h3 style={{ marginTop: 0 }}>경찰 조사</h3>
      {!isMyTurn ? (
        <p style={{ color: "var(--fg-muted)" }}>
          아직 경찰 차례가 아닙니다. 사회자의 진행을 기다리세요.
        </p>
      ) : checked ? (
        <p style={{ color: "var(--fg-muted)" }}>
          이번 밤에는 조사를 완료했습니다.
        </p>
      ) : (
        <p style={{ color: "var(--fg-muted)" }}>
          조사할 대상을 선택하세요. 한 밤에 한 번 가능합니다.
        </p>
      )}
      <PlayerPicker
        players={candidates}
        disabled={!isMyTurn || checked}
        onChange={(target) => send({ type: "submit:police-check", target })}
      />
      {history.length > 0 && (
        <div style={{ marginTop: "1rem" }}>
          <h4 style={{ marginBottom: "0.25rem" }}>조사 기록</h4>
          <ul
            style={{
              margin: 0,
              paddingLeft: "1.25rem",
              color: "var(--accent)",
              display: "flex",
              flexDirection: "column",
              gap: "0.25rem",
            }}
          >
            {history.map((rec, idx) => (
              <li key={`${rec.day}-${rec.target}-${idx}`}>
                {rec.day}일째 밤: <strong>{nameOf(rec.target)}</strong>은(는){" "}
                <strong>{TEAM_KR[rec.team]}</strong> 진영
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}

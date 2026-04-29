import { PlayerPicker } from "../../components/PlayerPicker";
import type { OutgoingMsg, PlayerID, State } from "../../types/wire";

interface Props {
  state: State;
  me: PlayerID;
  send: (msg: OutgoingMsg) => void;
}

export function DoctorPicker({ state, me, send }: Props) {
  const allowSelf = state.settings.doctorSelfHealAllowed;
  const candidates = state.players.filter(
    (p) => p.alive && (allowSelf || p.id !== me),
  );
  const isMyTurn = state.nightStep === "DOCTOR";

  return (
    <section style={{ padding: "0.5rem 0 1rem" }}>
      <div className="eyebrow" style={{ color: "var(--alive)" }}>
        DOCTOR · 보호 대상
      </div>
      <h3
        className="h-display"
        style={{ fontSize: "1.2rem", color: "var(--paper)", margin: "0.5rem 0 0.75rem", letterSpacing: "0.16em" }}
      >
        한 사람을 골라 보호하라
      </h3>
      <p
        className="serif"
        style={{ color: "var(--paper-dim)", fontStyle: "italic", lineHeight: 1.6, fontSize: "0.95rem", marginBottom: "0.85rem" }}
      >
        {isMyTurn
          ? `오늘 밤 보호할 대상을 선택하세요.${!allowSelf ? " (자가 보호 불가)" : ""}`
          : "아직 의사 차례가 아닙니다. 사회자의 진행을 기다리세요."}
      </p>
      <PlayerPicker
        players={candidates}
        value={state.pendingDoctorTarget}
        disabled={!isMyTurn}
        onChange={(target) => send({ type: "submit:doctor-heal", target })}
      />
    </section>
  );
}

import { PlayerPicker } from "../../components/PlayerPicker";
import type { OutgoingMsg, PlayerID, State } from "../../types/wire";

interface Props {
  state: State;
  me: PlayerID;
  send: (msg: OutgoingMsg) => void;
}

// DoctorPicker is the night-time heal input. Self-heal is included only
// when the host enabled it via Options.doctorSelfHealAllowed (FR-4.4).
// The picker stays disabled until the doctor's sub-step (after MAFIA and
// POLICE submit their actions).
export function DoctorPicker({ state, me, send }: Props) {
  const allowSelf = state.settings.doctorSelfHealAllowed;
  const candidates = state.players.filter(
    (p) => p.alive && (allowSelf || p.id !== me),
  );
  const isMyTurn = state.nightStep === "DOCTOR";

  return (
    <section style={{ padding: "1rem" }}>
      <h3 style={{ marginTop: 0 }}>보호 대상</h3>
      <p style={{ color: "var(--fg-muted)" }}>
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

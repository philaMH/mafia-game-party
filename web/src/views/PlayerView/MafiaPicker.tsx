import { PlayerPicker } from "../../components/PlayerPicker";
import type { OutgoingMsg, PlayerID, State, YourInfo } from "../../types/wire";

interface Props {
  state: State;
  your: YourInfo;
  me: PlayerID;
  send: (msg: OutgoingMsg) => void;
}

// MafiaPicker is the night-time kill input. Only the current
// representative (`state.mafiaRepresentativeId === me`) can submit; the
// rest of the cohort sees their selection mirrored via
// `pendingMafiaTarget` (BR-U5-PLAYER-5, FR-4.3).
export function MafiaPicker({ state, your, me, send }: Props) {
  const cohort = new Set(your.mafiaCohort ?? []);
  const candidates = state.players.filter((p) => p.alive && !cohort.has(p.id));
  const isRep = state.mafiaRepresentativeId === me;
  const isMyTurn = state.nightStep === "MAFIA";
  const pending = state.pendingMafiaTarget;
  const pendingName = pending
    ? state.players.find((p) => p.id === pending)?.name
    : undefined;

  return (
    <section style={{ padding: "1rem" }}>
      <h3 style={{ marginTop: 0 }}>마피아 살해 대상</h3>
      <p style={{ color: "var(--fg-muted)" }}>
        {!isMyTurn
          ? "마피아 차례가 끝났습니다. 사회자의 진행을 기다리세요."
          : isRep
            ? "대표자입니다. 살해 대상을 선택하세요."
            : "대기 중 — 대표자가 결정합니다."}
      </p>
      <PlayerPicker
        players={candidates}
        value={pending}
        disabled={!isMyTurn || !isRep}
        onChange={(target) => send({ type: "submit:mafia-kill", target })}
      />
      {!isRep && pendingName && (
        <p style={{ marginTop: "0.75rem" }}>
          대표자가 선택한 대상: <strong>{pendingName}</strong>
        </p>
      )}
    </section>
  );
}

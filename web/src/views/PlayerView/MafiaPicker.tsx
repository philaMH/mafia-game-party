import { PlayerPicker } from "../../components/PlayerPicker";
import type { OutgoingMsg, PlayerID, State, YourInfo } from "../../types/wire";

interface Props {
  state: State;
  your: YourInfo;
  me: PlayerID;
  send: (msg: OutgoingMsg) => void;
}

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
    <section style={{ padding: "0.5rem 0 1rem" }}>
      <div className="eyebrow red">MAFIA · 살해 대상</div>
      <h3
        className="h-display"
        style={{ fontSize: "1.2rem", color: "var(--paper)", margin: "0.5rem 0 0.75rem", letterSpacing: "0.16em" }}
      >
        오늘 밤 — 누구를 처단할 것인가?
      </h3>
      <p
        className="serif"
        style={{ color: "var(--paper-dim)", fontStyle: "italic", lineHeight: 1.6, fontSize: "0.95rem", marginBottom: "0.85rem" }}
      >
        {!isMyTurn
          ? "마피아 차례가 끝났습니다. 사회자의 진행을 기다리세요."
          : isRep
            ? "당신이 대표자입니다. 살해 대상을 선택하세요."
            : "대기 중 — 대표자가 결정합니다."}
      </p>
      <PlayerPicker
        players={candidates}
        value={pending}
        disabled={!isMyTurn || !isRep}
        onChange={(target) => send({ type: "submit:mafia-kill", target })}
      />
      {!isRep && pendingName && (
        <p
          className="serif"
          style={{ marginTop: "0.85rem", color: "var(--paper)", fontStyle: "italic" }}
        >
          대표자가 선택한 대상: <span style={{ color: "var(--red)" }}>{pendingName}</span>
        </p>
      )}
    </section>
  );
}

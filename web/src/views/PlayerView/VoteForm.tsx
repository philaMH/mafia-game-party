import { PlayerPicker } from "../../components/PlayerPicker";
import type { OutgoingMsg, Player, PlayerID, State } from "../../types/wire";

interface Props {
  state: State;
  me: PlayerID;
  meRow?: Player;
  send: (msg: OutgoingMsg) => void;
}

// VoteForm is reused for VOTE and RECOUNT. Selection sends immediately
// — the backend implements last-write-wins so revoting just overwrites
// the previous ballot (BR-U5-INPUT-3).
export function VoteForm({ state, me, meRow, send }: Props) {
  if (meRow && !meRow.alive) {
    return (
      <section style={{ padding: "1rem" }}>
        <p style={{ color: "var(--fg-muted)" }}>
          당신은 사망하여 투표할 수 없습니다.
        </p>
      </section>
    );
  }
  const candidates = state.players.filter((p) => p.alive && p.id !== me);
  const myVote = state.votes?.[me];
  const abstained = myVote === "";

  return (
    <section style={{ padding: "1rem" }}>
      <h3 style={{ marginTop: 0 }}>
        {state.phase === "RECOUNT" ? "재투표" : "투표"}
      </h3>
      <p style={{ color: "var(--fg-muted)" }}>
        처형할 대상을 선택하거나 기권하세요. 투표는 마지막 선택이 반영됩니다.
      </p>
      <PlayerPicker
        players={candidates}
        value={abstained ? undefined : myVote}
        onChange={(target) => send({ type: "submit:vote", target })}
      />
      <button
        type="button"
        onClick={() => send({ type: "submit:vote", target: "" })}
        aria-pressed={abstained}
        style={{
          marginTop: "0.75rem",
          width: "100%",
          padding: "0.5rem",
          borderColor: abstained ? "var(--emphasis)" : "var(--border)",
          background: abstained ? "var(--emphasis)" : "transparent",
          color: abstained ? "var(--bg)" : "var(--fg)",
        }}
      >
        기권
      </button>
    </section>
  );
}

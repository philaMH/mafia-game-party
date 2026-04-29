import { PlayerPicker } from "../../components/PlayerPicker";
import type { OutgoingMsg, Player, PlayerID, State } from "../../types/wire";

interface Props {
  state: State;
  me: PlayerID;
  meRow?: Player;
  send: (msg: OutgoingMsg) => void;
}

export function VoteForm({ state, me, meRow, send }: Props) {
  if (meRow && !meRow.alive) {
    return (
      <section
        className="center-card"
        style={{ padding: "1.25rem 1.5rem", textAlign: "center", margin: "1rem 0" }}
      >
        <div className="eyebrow red">SILENCED · 사망</div>
        <p
          className="serif"
          style={{ color: "var(--paper-dim)", fontStyle: "italic", marginTop: "0.5rem", lineHeight: 1.6 }}
        >
          당신은 사망하여 투표할 수 없습니다.
        </p>
      </section>
    );
  }
  const candidates = state.players.filter((p) => p.alive && p.id !== me);
  const myVote = state.votes?.[me];
  const abstained = myVote === "";
  const isRecount = state.phase === "RECOUNT";

  return (
    <section style={{ padding: "0.5rem 0 1rem" }}>
      <div className="eyebrow red">{isRecount ? "RECOUNT · 재투표" : "VERDICT · 투표"}</div>
      <h3
        className="h-display"
        style={{ fontSize: "1.3rem", color: "var(--paper)", margin: "0.5rem 0 0.75rem", letterSpacing: "0.16em" }}
      >
        {isRecount ? "재투표" : "한 사람을 지목하라"}
      </h3>
      <p
        className="serif"
        style={{ color: "var(--paper-dim)", fontStyle: "italic", lineHeight: 1.6, fontSize: "0.95rem", marginBottom: "0.85rem" }}
      >
        처형할 대상을 선택하거나 기권하세요. 투표는 마지막 선택이 반영됩니다.
      </p>
      <PlayerPicker
        players={candidates}
        value={abstained ? undefined : myVote}
        onChange={(target) => send({ type: "submit:vote", target })}
      />
      <button
        type="button"
        className={"btn-noir " + (abstained ? "primary" : "ghost")}
        onClick={() => send({ type: "submit:vote", target: "" })}
        aria-pressed={abstained}
        style={{ marginTop: "0.85rem", width: "100%" }}
      >
        기권
      </button>
    </section>
  );
}

import type { State } from "../../types/wire";

import { TimerBar } from "../PublicView/TimerBar";

interface Props {
  state: State;
}

export function DiscussionView({ state }: Props) {
  return (
    <section style={{ padding: "0.5rem 0 1rem", textAlign: "center" }}>
      <div className="eyebrow">DAY DISCUSSION · 토론</div>
      <h3
        className="h-display"
        style={{ fontSize: "1.4rem", color: "var(--paper)", margin: "0.5rem 0 0.75rem", letterSpacing: "0.16em" }}
      >
        토론 시간
      </h3>
      <p
        className="serif"
        style={{ color: "var(--paper-dim)", fontStyle: "italic", lineHeight: 1.6, fontSize: "0.95rem" }}
      >
        의견을 자유롭게 나누세요. 곧 투표가 시작됩니다.
      </p>
      <TimerBar deadline={state.deadline} paused={state.paused} />
    </section>
  );
}

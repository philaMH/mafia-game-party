import type { State } from "../../types/wire";

import { TimerBar } from "../PublicView/TimerBar";

interface Props {
  state: State;
}

export function DiscussionView({ state }: Props) {
  return (
    <section style={{ padding: "1rem" }}>
      <h3 style={{ marginTop: 0 }}>토론 시간</h3>
      <p style={{ color: "var(--fg-muted)" }}>
        의견을 자유롭게 나누세요. 곧 투표가 시작됩니다.
      </p>
      <TimerBar deadline={state.deadline} />
    </section>
  );
}

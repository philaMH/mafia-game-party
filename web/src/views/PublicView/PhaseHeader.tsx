import type { Phase } from "../../types/wire";

const EYEBROW: Record<Phase, string> = {
  LOBBY: "WAITING ROOM · 대기실",
  INTRO: "INTRODUCTIONS · 자기소개",
  NIGHT: "NIGHT FALLS · 밤",
  DAY: "DAY DISCUSSION · 토론",
  VOTE: "THE VERDICT · 투표",
  RECOUNT: "RECOUNT · 재투표",
  END: "GAME OVER · 게임 종료",
};

const TITLE: Record<Phase, (day: number) => string> = {
  LOBBY: () => "참가자 모집 중",
  INTRO: (d) => `${d}일째 — 자기소개`,
  NIGHT: (d) => `${d}일째 — 밤`,
  DAY: (d) => `${d}일째 — 낮`,
  VOTE: (d) => `${d}일째 — 투표`,
  RECOUNT: (d) => `${d}일째 — 재투표`,
  END: () => "최종 결과",
};

interface Props {
  phase: Phase;
  day: number;
}

export function PhaseHeader({ phase, day }: Props) {
  return (
    <header style={{ textAlign: "center", margin: "1.25rem 0 0.5rem" }}>
      <div className="eyebrow" style={{ marginBottom: "0.5rem" }}>
        {EYEBROW[phase]}
      </div>
      <h1
        className="h-display"
        style={{ fontSize: "2.4rem", color: "var(--paper)", margin: 0, letterSpacing: "0.16em" }}
      >
        {TITLE[phase](day)}
      </h1>
      <div className="divider-gold" style={{ width: "60%", margin: "0.75rem auto 0" }} />
    </header>
  );
}

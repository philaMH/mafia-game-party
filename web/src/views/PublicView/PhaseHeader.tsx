import type { Phase } from "../../types/wire";

const TEXT: Record<Phase, (day: number) => string> = {
  LOBBY: () => "참가자 모집 중",
  INTRO: (d) => `${d}일째 — 자기소개`,
  NIGHT: (d) => `${d}일째 — 밤`,
  DAY: (d) => `${d}일째 — 낮`,
  VOTE: (d) => `${d}일째 — 투표`,
  RECOUNT: (d) => `${d}일째 — 재투표`,
  END: () => "게임 종료",
};

interface Props {
  phase: Phase;
  day: number;
}

export function PhaseHeader({ phase, day }: Props) {
  return (
    <h1 style={{ fontSize: "3rem", textAlign: "center", margin: "1rem 0" }}>
      {TEXT[phase](day)}
    </h1>
  );
}

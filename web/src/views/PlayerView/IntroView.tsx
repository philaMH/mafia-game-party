import type { OutgoingMsg, PlayerID, State, YourInfo } from "../../types/wire";

interface Props {
  state: State;
  me: PlayerID;
  your: YourInfo;
  send?: (msg: OutgoingMsg) => void;
}

export function IntroView({ state, me, your, send }: Props) {
  const idx = state.introSpeakerIdx ?? 0;
  const speaker = state.players[idx];
  const isMyTurn = speaker?.id === me;

  return (
    <section style={{ padding: "0.5rem 0 1rem", textAlign: "center" }}>
      <div className="eyebrow">INTRODUCTIONS · 자기소개</div>
      <h2
        className="h-display"
        style={{ fontSize: "1.4rem", color: "var(--paper)", margin: "0.5rem 0 1rem", letterSpacing: "0.16em" }}
      >
        자기소개 단계
      </h2>
      {isMyTurn ? (
        <div
          className="center-card"
          style={{ padding: "1.5rem 1.5rem", border: "1px solid var(--gold)", boxShadow: "0 0 24px var(--gold-glow)" }}
        >
          <div className="eyebrow" style={{ marginBottom: "0.6rem" }}>
            NOW SPEAKING · 당신의 차례
          </div>
          <p
            className="h-display"
            style={{ fontSize: "1.25rem", color: "var(--paper)", margin: 0, letterSpacing: "0.14em" }}
          >
            지금 자기소개를 시작하세요.
          </p>
          {your.keyword && (
            <p
              className="mono"
              style={{
                marginTop: "0.85rem",
                color: "var(--gold)",
                fontSize: "1.1rem",
                letterSpacing: "0.25em",
              }}
            >
              {your.keyword}
            </p>
          )}
          {send && (
            <button
              type="button"
              className="btn-noir primary"
              onClick={() => send({ type: "player:end-self-intro" })}
              style={{ marginTop: "1.25rem" }}
            >
              내 자기소개 종료
            </button>
          )}
        </div>
      ) : (
        <p className="serif" style={{ color: "var(--paper-dim)", fontStyle: "italic", lineHeight: 1.6 }}>
          {speaker
            ? `「${speaker.name}」이(가) 자기소개 중입니다.`
            : "다음 발언자를 기다리는 중…"}
        </p>
      )}
    </section>
  );
}

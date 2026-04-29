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
    <section style={{ padding: "1rem", textAlign: "center" }}>
      <h2 style={{ fontSize: "1.75rem" }}>자기소개 단계</h2>
      {isMyTurn ? (
        <div
          style={{
            marginTop: "1rem",
            padding: "1rem",
            background: "var(--accent)",
            color: "#fff",
            borderRadius: "0.5rem",
          }}
        >
          <p style={{ fontSize: "1.25rem", margin: 0 }}>지금 자기소개를 시작하세요.</p>
          {your.keyword && (
            <p style={{ marginTop: "0.5rem" }}>
              키워드: <strong>{your.keyword}</strong>
            </p>
          )}
          {send && (
            <button
              type="button"
              onClick={() => send({ type: "player:end-self-intro" })}
              style={{
                marginTop: "1rem",
                padding: "0.5rem 1.5rem",
                fontSize: "1rem",
                background: "#fff",
                color: "var(--accent)",
                border: "none",
                borderRadius: "0.25rem",
                cursor: "pointer",
              }}
            >
              내 자기소개 종료
            </button>
          )}
        </div>
      ) : (
        <p style={{ color: "var(--fg-muted)" }}>
          {speaker ? `${speaker.name}이(가) 자기소개 중입니다.` : "다음 발언자를 기다리는 중…"}
        </p>
      )}
    </section>
  );
}

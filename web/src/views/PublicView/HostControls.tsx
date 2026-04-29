import type { OutgoingMsg, State } from "../../types/wire";

interface Props {
  state: State;
  send: (msg: OutgoingMsg) => void;
}

const MIN_PLAYERS = 6;

// HostControls exposes phase-appropriate buttons. Permission checks live
// in the SessionManager (BR-U4-AUTH-2); this component only hides
// buttons that don't make sense in the current phase.
//
// Iteration 5 — the manual "야간 마감" button is removed because the
// NightStep timer now drives all transitions. A Pause/Resume toggle is
// added across INTRO/DAY/NIGHT (Q4=A / Q5=B).
export function HostControls({ state, send }: Props) {
  const { phase, players, paused } = state;
  const canStart = phase === "LOBBY" && players.length >= MIN_PLAYERS;
  const canPause = phase === "INTRO" || phase === "DAY" || phase === "NIGHT";

  const onForceEnd = () => {
    if (window.confirm("정말로 게임을 강제 종료하시겠습니까?")) {
      send({ type: "host:force-end" });
    }
  };

  return (
    <div
      className="gold-frame"
      style={{
        display: "flex",
        flexWrap: "wrap",
        gap: "0.6rem",
        justifyContent: "center",
        alignItems: "center",
        padding: "0.85rem 1rem",
        margin: "1rem 1.25rem 0",
      }}
    >
      <span className="eyebrow" style={{ marginRight: "0.5rem" }}>
        HOST CONTROLS
      </span>
      {phase === "LOBBY" && (
        <button
          type="button"
          className="btn-noir primary sm"
          disabled={!canStart}
          onClick={() => send({ type: "host:start-room" })}
        >
          게임 시작
          {!canStart && ` (${MIN_PLAYERS}명 이상 필요)`}
        </button>
      )}
      {phase === "INTRO" && (
        <button
          type="button"
          className="btn-noir sm"
          onClick={() => send({ type: "submit:advance-intro" })}
        >
          다음 발언자
        </button>
      )}
      {phase === "DAY" && (
        <button
          type="button"
          className="btn-noir sm"
          onClick={() => send({ type: "submit:end-discussion" })}
        >
          토론 조기 종료
        </button>
      )}
      {canPause &&
        (paused ? (
          <button
            type="button"
            className="btn-noir sm"
            onClick={() => send({ type: "host:resume" })}
            style={{ borderColor: "var(--alive)", color: "var(--alive)" }}
          >
            ▶ 재개
          </button>
        ) : (
          <button type="button" className="btn-noir sm ghost" onClick={() => send({ type: "host:pause" })}>
            ⏸ 일시정지
          </button>
        ))}
      {phase !== "END" && (
        <button type="button" className="btn-noir sm warn" onClick={onForceEnd}>
          ⚠ 강제 종료
        </button>
      )}
      {phase === "END" && (
        <button
          type="button"
          className="btn-noir sm"
          onClick={() => {
            if (window.confirm("현재 방을 종료하고 새 방을 개설할 준비를 할까요?")) {
              send({ type: "host:close-room" });
            }
          }}
        >
          방 종료
        </button>
      )}
    </div>
  );
}

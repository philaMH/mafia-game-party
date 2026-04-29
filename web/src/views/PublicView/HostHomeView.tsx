import { useNavigate } from "react-router-dom";

import { useGameContext } from "../../context/GameContext";

export function HostHomeView() {
  const ctx = useGameContext();
  const navigate = useNavigate();

  const onStart = () => {
    ctx.send({ type: "host:open-room", options: ctx.hostOptions });
  };

  const onSettings = () => {
    navigate("/public/settings");
  };

  return (
    <section
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        gap: "1.5rem",
        padding: "2rem 2.5rem",
        maxWidth: "44rem",
        margin: "0 auto",
        position: "relative",
        zIndex: 10,
      }}
    >
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          gap: "0.85rem",
          minWidth: "20rem",
        }}
      >
        <button
          type="button"
          className="btn-noir primary lg"
          onClick={onStart}
        >
          ♠ 게임 시작
        </button>
        <button
          type="button"
          className="btn-noir lg"
          onClick={onSettings}
        >
          ⚙ 설정
        </button>
      </div>
      <p
        className="serif"
        style={{
          color: "var(--paper-dim)",
          fontStyle: "italic",
          fontSize: "0.85rem",
          marginTop: "0.5rem",
        }}
      >
        설정에서 마피아·플레이어 수와 단계별 시간을 조정할 수 있습니다.
      </p>
    </section>
  );
}

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
      <div className="eyebrow">HOST MAIN · 메인 메뉴</div>
      <h1 className="mafia-title stone sm">MAFIA</h1>
      <div className="mafia-sub" style={{ fontSize: "0.95rem", marginTop: 0 }}>
        마 피 아 게 임
      </div>
      <div
        className="gold-frame"
        style={{
          padding: "1.75rem 2.25rem",
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

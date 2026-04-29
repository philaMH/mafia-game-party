import { useMemo, useState } from "react";

import { ConnectionBadge } from "../../components/ConnectionBadge";
import { NicknameForm } from "../../components/NicknameForm";
import { ToastList } from "../../components/ToastList";
import { useGameContext } from "../../context/GameContext";
import { useToken } from "../../hooks/useToken";
import type { PlayerID } from "../../types/wire";

import { PhaseInputs } from "./PhaseInputs";
import { YourInfoCard } from "./YourInfoCard";

const BG_URL = "/assets/background.jpg";

export function PlayerView() {
  const ctx = useGameContext();
  const tokenIO = useToken();
  const [resumingToken] = useState<string | null>(() => tokenIO.get());
  const isResuming = resumingToken !== null && !ctx.playerId && ctx.errors.length === 0;

  const cohortNames = useMemo(() => {
    const map = new Map<PlayerID, string>();
    if (ctx.state) {
      for (const p of ctx.state.players) map.set(p.id, p.name);
    }
    return map;
  }, [ctx.state]);

  return (
    <main
      className="noir"
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <div className="noir-bg dim crop-table" style={{ backgroundImage: `url(${BG_URL})` }} />
      <div className="scrim" />

      <div
        className="noir-content"
        style={{
          flex: 1,
          maxWidth: "32rem",
          width: "100%",
          margin: "0 auto",
          padding: "1rem 1.25rem 2rem",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <header
          style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: "0.5rem" }}
        >
          <span className="tag" style={{ borderColor: "var(--gold-dim)", color: "var(--paper-2)" }}>
            PLAYER · 플레이어
          </span>
          <ConnectionBadge status={ctx.status} />
        </header>

        {isResuming ? (
          <section
            className="center-card"
            style={{ marginTop: "2rem", padding: "1.5rem 1.75rem", textAlign: "center" }}
          >
            <div className="eyebrow">RESUMING SESSION · 재접속</div>
            <h2
              className="h-display"
              style={{ fontSize: "1.5rem", color: "var(--paper)", marginTop: "0.75rem" }}
            >
              재접속 중…
            </h2>
            <p
              className="serif"
              style={{ color: "var(--paper-dim)", fontStyle: "italic", marginTop: "0.5rem" }}
            >
              이전 세션을 복구하고 있습니다. 잠시만 기다려 주세요.
            </p>
          </section>
        ) : !ctx.roomOpened && !ctx.playerId ? (
          <section
            className="center-card"
            style={{ marginTop: "2rem", padding: "1.5rem 1.75rem", textAlign: "center" }}
          >
            <div className="eyebrow red">CLOSED · 방 없음</div>
            <h2
              className="h-display"
              style={{ fontSize: "1.5rem", color: "var(--paper)", marginTop: "0.75rem" }}
            >
              방이 아직 없습니다
            </h2>
            <p
              className="serif"
              style={{ color: "var(--paper-dim)", fontStyle: "italic", marginTop: "0.5rem" }}
            >
              호스트가 방을 개설할 때까지 기다려 주세요. 방이 열리면 자동으로 참가 화면이 표시됩니다.
            </p>
          </section>
        ) : !ctx.playerId ? (
          <section
            style={{ marginTop: "1.5rem", display: "flex", flexDirection: "column", alignItems: "center" }}
          >
            <h1 className="mafia-title stone sm" style={{ fontSize: "3rem" }}>
              MAFIA
            </h1>
            <div className="mafia-sub" style={{ fontSize: "0.9rem", marginTop: "0.4rem" }}>
              마 피 아 게 임
            </div>
            <div
              className="center-card"
              style={{ marginTop: "1.5rem", padding: "1.5rem 1.75rem", width: "100%" }}
            >
              <div className="eyebrow">JOIN · 입장</div>
              <h2
                className="h-display"
                style={{ fontSize: "1.25rem", color: "var(--paper)", margin: "0.5rem 0 1rem" }}
              >
                닉네임을 입력하고 입장하세요
              </h2>
              <NicknameForm
                prompt="닉네임"
                onSubmit={(name) => ctx.send({ type: "join", name })}
              />
            </div>
          </section>
        ) : (
          <>
            <YourInfoCard
              your={ctx.your}
              me={ctx.state?.players.find((p) => p.id === ctx.playerId)}
              cohortNames={cohortNames}
            />
            {ctx.state && (
              <PhaseInputs
                state={ctx.state}
                your={ctx.your}
                me={ctx.playerId}
                send={ctx.send}
              />
            )}
          </>
        )}
      </div>

      <ToastList errors={ctx.errors} onDismiss={ctx.ackError} />
    </main>
  );
}

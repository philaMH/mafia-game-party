import { useMemo, useState } from "react";

import { ConnectionBadge } from "../../components/ConnectionBadge";
import { NicknameForm } from "../../components/NicknameForm";
import { ToastList } from "../../components/ToastList";
import { useGameContext } from "../../context/GameContext";
import { useToken } from "../../hooks/useToken";
import type { PlayerID } from "../../types/wire";

import { PhaseInputs } from "./PhaseInputs";
import { YourInfoCard } from "./YourInfoCard";

// PlayerView is the per-player input surface. It blocks until the
// player has joined (no playerId yet), then routes each phase to its
// dedicated component via PhaseInputs.
export function PlayerView() {
  const ctx = useGameContext();
  const tokenIO = useToken();
  // Snapshot the saved token at mount time. If a token exists on a
  // fresh page load we are mid-resume; render a "재접속 중" notice
  // instead of the nickname form so a refreshing player doesn't see
  // the form flash between room:opened and joined and accidentally
  // submit a second join request. Once the resume settles (joined
  // arrives → playerId set, or UNKNOWN_PLAYER_ERROR clears the token
  // and dispatches logout), the normal branches take over.
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
      style={{
        minHeight: "100vh",
        padding: "1rem",
        maxWidth: "32rem",
        margin: "0 auto",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <header style={{ display: "flex", justifyContent: "space-between" }}>
        <ConnectionBadge status={ctx.status} />
      </header>

      {isResuming ? (
        <section style={{ marginTop: "2rem" }}>
          <h2>재접속 중…</h2>
          <p style={{ color: "var(--fg-muted)" }}>
            이전 세션을 복구하고 있습니다. 잠시만 기다려 주세요.
          </p>
        </section>
      ) : !ctx.roomOpened && !ctx.playerId ? (
        <section style={{ marginTop: "2rem" }}>
          <h2>방이 아직 없습니다</h2>
          <p style={{ color: "var(--fg-muted)" }}>
            호스트가 방을 개설할 때까지 기다려 주세요. 방이 열리면 자동으로
            참가 화면이 표시됩니다.
          </p>
        </section>
      ) : !ctx.playerId ? (
        <section style={{ marginTop: "2rem" }}>
          <h2>닉네임을 입력하고 입장하세요</h2>
          <NicknameForm
            prompt="닉네임"
            onSubmit={(name) => ctx.send({ type: "join", name })}
          />
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

      <ToastList errors={ctx.errors} onDismiss={ctx.ackError} />
    </main>
  );
}

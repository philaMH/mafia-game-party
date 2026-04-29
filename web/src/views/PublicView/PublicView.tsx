import { useEffect, useState } from "react";

import { ConnectionBadge } from "../../components/ConnectionBadge";
import { ToastList } from "../../components/ToastList";
import { useGameContext } from "../../context/GameContext";
import { defaultOptions, type Options } from "../../types/wire";

import { HostControls } from "./HostControls";
import { PauseBadge } from "./PauseBadge";
import { PhaseHeader } from "./PhaseHeader";
import { PlayersGrid } from "./PlayersGrid";
import { SubtitleArea } from "./SubtitleArea";
import { TimerBar } from "./TimerBar";
import { VoiceToggle } from "./VoiceToggle";

// Korean labels for NightStep used by the public TimerBar. Iteration 5 R5:
// the public view shows the current step's name so spectators understand
// what the countdown represents.
const NIGHT_STEP_LABEL: Record<string, string> = {
  MAFIA: "마피아의 시간",
  POLICE: "경찰의 시간",
  DOCTOR: "의사의 시간",
};

// PublicView is the host-PC + spectator screen. It owns subtitle display,
// TTS playback (via GameContext), and the host control panel. Player-
// specific input lives in PlayerView.
//
// Iteration 2 flow: the public client immediately attempts to claim the
// GM seat. If granted, the host enters game settings and explicitly opens
// the room before players may join. If the seat is already taken, a
// blocking screen is shown.
export function PublicView() {
  const ctx = useGameContext();
  const [opts, setOpts] = useState<Options>(() => defaultOptions(8));
  const [claimSent, setClaimSent] = useState(false);

  useEffect(() => {
    if (ctx.status === "connected" && !claimSent) {
      ctx.send({ type: "subscribe-public" });
      ctx.send({ type: "host:claim" });
      setClaimSent(true);
    }
  }, [ctx, claimSent]);

  return (
    <main
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <header style={{ padding: "0.75rem 1rem" }}>
        <ConnectionBadge status={ctx.status} />
      </header>

      {ctx.hostOccupied ? (
        <section
          style={{
            flex: 1,
            display: "flex",
            flexDirection: "column",
            justifyContent: "center",
            alignItems: "center",
            gap: "1rem",
            padding: "2rem",
          }}
        >
          <h2 style={{ fontSize: "1.75rem" }}>이미 호스트가 방을 운영 중입니다</h2>
          <p style={{ color: "var(--fg-muted)" }}>
            한 서버는 동시에 한 개의 방만 운영합니다. 호스트가 게임을 종료한 뒤 다시
            접속해 주세요.
          </p>
        </section>
      ) : ctx.hostToken && !ctx.roomOpened ? (
        <section
          style={{
            flex: 1,
            display: "flex",
            flexDirection: "column",
            justifyContent: "center",
            alignItems: "center",
            gap: "1rem",
            padding: "2rem",
          }}
        >
          <h2 style={{ fontSize: "1.75rem" }}>방을 개설합니다</h2>
          <p style={{ color: "var(--fg-muted)" }}>
            게임 설정을 마치면 참가자를 받습니다.
          </p>
          <label style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
            최대 참여 인원
            <input
              type="number"
              min={6}
              max={12}
              value={opts.maxPlayers}
              onChange={(e) =>
                setOpts({ ...opts, maxPlayers: Number(e.target.value) })
              }
              style={{ width: "5rem" }}
            />
          </label>
          <label style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
            마피아 수
            <input
              type="number"
              min={1}
              max={Math.max(1, opts.maxPlayers - 3)}
              value={opts.mafiaCount}
              onChange={(e) =>
                setOpts({ ...opts, mafiaCount: Number(e.target.value) })
              }
              style={{ width: "5rem" }}
            />
            {Math.abs(opts.mafiaCount - defaultOptions(opts.maxPlayers).mafiaCount) > 1 && (
              <span style={{ color: "orange", fontSize: "0.875rem" }}>
                권장하지 않는 설정입니다
              </span>
            )}
          </label>
          <button
            type="button"
            onClick={() => ctx.send({ type: "host:open-room", options: opts })}
            style={{ padding: "0.5rem 1.5rem", fontSize: "1rem" }}
          >
            방 개설
          </button>
        </section>
      ) : !ctx.state ? (
        <section
          style={{
            flex: 1,
            display: "flex",
            flexDirection: "column",
            justifyContent: "center",
            alignItems: "center",
            gap: "1rem",
          }}
        >
          <p style={{ fontSize: "1.5rem", color: "var(--fg-muted)" }}>
            참가자를 받습니다…
          </p>
        </section>
      ) : (
        <section style={{ flex: 1 }}>
          <PauseBadge paused={!!ctx.state.paused} />
          <PhaseHeader phase={ctx.state.phase} day={ctx.state.day} />
          {ctx.state.phase === "NIGHT" ? (
            <TimerBar
              deadline={ctx.state.nightStepDeadline}
              paused={ctx.state.paused}
              label={ctx.state.nightStep ? NIGHT_STEP_LABEL[ctx.state.nightStep] : undefined}
            />
          ) : (
            <TimerBar deadline={ctx.state.deadline} paused={ctx.state.paused} />
          )}
          <PlayersGrid players={ctx.state.players} phase={ctx.state.phase} />
          <SubtitleArea ann={ctx.lastAnnounce} />
          {ctx.isHost && <HostControls state={ctx.state} send={ctx.send} />}
        </section>
      )}

      <footer
        style={{
          padding: "1rem",
          display: "flex",
          justifyContent: "flex-end",
          gap: "0.5rem",
          borderTop: "1px solid var(--border)",
        }}
      >
        {!ctx.ttsAvailable && (
          <span style={{ color: "var(--fg-muted)", fontSize: "0.875rem" }}>
            이 브라우저는 음성 안내를 지원하지 않습니다. 자막으로 대체합니다.
          </span>
        )}
        <VoiceToggle
          on={ctx.voiceOn}
          available={ctx.ttsAvailable}
          onChange={ctx.toggleVoice}
        />
      </footer>

      <ToastList errors={ctx.errors} onDismiss={ctx.ackError} />
    </main>
  );
}

import { useEffect, useState } from "react";

import { ConnectionBadge } from "../../components/ConnectionBadge";
import { ToastList } from "../../components/ToastList";
import { useGameContext } from "../../context/GameContext";

import { HostControls } from "./HostControls";
import { HostHomeView } from "./HostHomeView";
import { PauseBadge } from "./PauseBadge";
import { PhaseHeader } from "./PhaseHeader";
import { PlayersGrid } from "./PlayersGrid";
import { SubtitleArea } from "./SubtitleArea";
import { TimerBar } from "./TimerBar";
import { VoiceToggle } from "./VoiceToggle";

const NIGHT_STEP_LABEL: Record<string, string> = {
  INTRO: "밤이 시작됩니다",
  MAFIA: "마피아의 시간",
  POLICE: "경찰의 시간",
  DOCTOR: "의사의 시간",
};

const BG_URL = "/assets/background.jpg";

const HostBadge = () => (
  <div
    style={{
      position: "absolute",
      top: "1rem",
      left: "1.5rem",
      zIndex: 20,
    }}
  >
    <span
      className="tag"
      style={{ borderColor: "var(--gold)", color: "var(--gold)", letterSpacing: "0.25em" }}
    >
      ♣ HOST CONSOLE · 진행자 화면
    </span>
  </div>
);

const FullScreenSection = ({ children }: { children: React.ReactNode }) => (
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
    {children}
  </section>
);

export function PublicView() {
  const ctx = useGameContext();
  const [claimSent, setClaimSent] = useState(false);

  useEffect(() => {
    // Iteration 7: PublicView remounts when the host navigates back from
    // /public/settings. Only claim once per WS lifetime — if we already
    // hold a hostToken (or the seat is otherwise occupied) skip the
    // re-claim to avoid the room:host-occupied false-positive that would
    // boot the rightful host into the ACCESS DENIED screen.
    if (ctx.status === "connected" && !claimSent && !ctx.hostToken && !ctx.hostOccupied) {
      ctx.send({ type: "subscribe-public" });
      ctx.send({ type: "host:claim" });
      setClaimSent(true);
    }
  }, [ctx, claimSent]);

  const isHost = ctx.isHost;
  const bgClass = ctx.state?.phase === "VOTE" || ctx.state?.phase === "RECOUNT"
    ? "noir-bg bloody crop-table"
    : ctx.state?.phase === "NIGHT"
      ? "noir-bg deep crop-table"
      : ctx.state?.phase === "END"
        ? "noir-bg deep"
        : "noir-bg dim crop-table";

  return (
    <main
      className="noir"
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <div
        className={bgClass}
        style={{ backgroundImage: `url(${BG_URL})` }}
      />
      <div className="scrim" />
      <PauseBadge paused={!!ctx.state?.paused} />

      <header
        style={{
          padding: "0.85rem 1.5rem",
          display: "flex",
          justifyContent: "flex-end",
          position: "relative",
          zIndex: 20,
        }}
      >
        <ConnectionBadge status={ctx.status} />
      </header>

      {isHost && <HostBadge />}

      <div
        className="noir-content"
        style={{ flex: 1, display: "flex", flexDirection: "column" }}
      >
        {ctx.hostOccupied ? (
          <FullScreenSection>
            <div className="eyebrow red">ACCESS DENIED · 입장 불가</div>
            <div className="center-card" style={{ padding: "2.5rem 3rem", textAlign: "center" }}>
              <h2 className="h-display" style={{ fontSize: "1.75rem", color: "var(--paper)", margin: 0 }}>
                이미 호스트가 방을 운영 중입니다
              </h2>
              <p
                className="serif"
                style={{ marginTop: "1rem", color: "var(--paper-dim)", fontStyle: "italic", lineHeight: 1.6 }}
              >
                한 서버는 동시에 한 개의 방만 운영합니다.<br />
                호스트가 게임을 종료한 뒤 다시 접속해 주세요.
              </p>
            </div>
          </FullScreenSection>
        ) : ctx.hostToken && !ctx.roomOpened ? (
          <HostHomeView />
        ) : !ctx.state ? (
          <FullScreenSection>
            <div className="eyebrow">STANDBY · 대기</div>
            <h1 className="mafia-title stone sm">MAFIA</h1>
            <p
              className="serif"
              style={{ fontStyle: "italic", color: "var(--paper-dim)", fontSize: "1.1rem" }}
            >
              참가자를 받습니다…
            </p>
          </FullScreenSection>
        ) : (
          <section
            style={{ flex: 1, padding: "0 0 1rem", display: "flex", flexDirection: "column" }}
          >
            <PhaseHeader phase={ctx.state.phase} day={ctx.state.day} />
            {ctx.state.phase === "NIGHT" ? (
              <TimerBar
                deadline={ctx.state.nightStepDeadline}
                paused={ctx.state.paused}
                label={
                  ctx.state.nightStep ? NIGHT_STEP_LABEL[ctx.state.nightStep] : undefined
                }
              />
            ) : (
              <TimerBar deadline={ctx.state.deadline} paused={ctx.state.paused} />
            )}
            <SubtitleArea ann={ctx.lastAnnounce} />
            <PlayersGrid players={ctx.state.players} phase={ctx.state.phase} />
            {isHost && <HostControls state={ctx.state} send={ctx.send} />}
          </section>
        )}
      </div>

      <footer
        style={{
          padding: "0.85rem 1.5rem",
          display: "flex",
          justifyContent: "flex-end",
          alignItems: "center",
          gap: "1rem",
          borderTop: "1px solid var(--gold-dim)",
          background: "rgba(0,0,0,0.5)",
          position: "relative",
          zIndex: 20,
        }}
      >
        {isHost && !ctx.audioAvailable && (
          <span
            className="serif"
            style={{ color: "var(--paper-dim)", fontSize: "0.85rem", fontStyle: "italic" }}
          >
            이 브라우저는 음성 안내를 지원하지 않습니다. 자막으로 대체합니다.
          </span>
        )}
        {isHost && (
          <VoiceToggle
            on={ctx.voiceOn}
            available={ctx.audioAvailable}
            onChange={ctx.toggleVoice}
          />
        )}
      </footer>

      <ToastList errors={ctx.errors} onDismiss={ctx.ackError} />
    </main>
  );
}

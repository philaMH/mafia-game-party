import type { OutgoingMsg, PlayerID, State, YourInfo } from "../../types/wire";

import { DiscussionView } from "./DiscussionView";
import { EndScreen } from "./EndScreen";
import { IntroView } from "./IntroView";
import { LobbyView } from "./LobbyView";
import { NightInputs } from "./NightInputs";
import { VoteForm } from "./VoteForm";

interface Props {
  state: State;
  your: YourInfo;
  me: PlayerID;
  send: (msg: OutgoingMsg) => void;
}

// PhaseInputs is the single dispatch point for phase-specific PlayerView
// content. It keeps Phase fan-out logic out of PlayerView.tsx so each
// branch is small and independently testable.
export function PhaseInputs({ state, your, me, send }: Props) {
  const meRow = state.players.find((p) => p.id === me);

  switch (state.phase) {
    case "LOBBY":
      return <LobbyView players={state.players} />;
    case "INTRO":
      return <IntroView state={state} me={me} your={your} send={send} />;
    case "NIGHT":
      return (
        <NightInputs
          state={state}
          your={your}
          me={me}
          meRow={meRow}
          send={send}
        />
      );
    case "DAY":
      return <DiscussionView state={state} />;
    case "VOTE":
    case "RECOUNT":
      return <VoteForm state={state} me={me} meRow={meRow} send={send} />;
    case "END":
      return <EndScreen state={state} />;
    default:
      return null;
  }
}

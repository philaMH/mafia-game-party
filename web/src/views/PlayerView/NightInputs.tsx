import type { OutgoingMsg, Player, PlayerID, State, YourInfo } from "../../types/wire";

import { DoctorPicker } from "./DoctorPicker";
import { MafiaPicker } from "./MafiaPicker";
import { PolicePicker } from "./PolicePicker";

interface Props {
  state: State;
  your: YourInfo;
  me: PlayerID;
  meRow?: Player;
  send: (msg: OutgoingMsg) => void;
}

// NightInputs branches on the viewer's role. Dead players see only a
// notice — the engine ignores their submissions anyway, so we suppress
// the form to avoid confusing UI.
export function NightInputs({ state, your, me, meRow, send }: Props) {
  if (meRow && !meRow.alive) {
    return (
      <section style={{ padding: "1rem" }}>
        <p style={{ color: "var(--fg-muted)" }}>
          당신은 사망했습니다. 야간 진행을 관전하세요.
        </p>
      </section>
    );
  }

  switch (your.role) {
    case "MAFIA":
      return <MafiaPicker state={state} your={your} me={me} send={send} />;
    case "DOCTOR":
      return <DoctorPicker state={state} me={me} send={send} />;
    case "POLICE":
      return <PolicePicker state={state} me={me} send={send} />;
    case "CITIZEN":
    default:
      return (
        <section style={{ padding: "1rem" }}>
          <p>밤이 지나가길 기다리세요.</p>
        </section>
      );
  }
}

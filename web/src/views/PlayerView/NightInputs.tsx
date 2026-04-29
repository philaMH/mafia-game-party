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
      <section
        className="center-card"
        style={{ padding: "1.25rem 1.5rem", textAlign: "center", margin: "1rem 0" }}
      >
        <div className="eyebrow red">SILENCED · 사망</div>
        <p
          className="serif"
          style={{ color: "var(--paper-dim)", fontStyle: "italic", marginTop: "0.5rem", lineHeight: 1.6 }}
        >
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
        <section
          style={{ padding: "1rem 0", textAlign: "center" }}
        >
          <div className="eyebrow">CITIZEN · 시민</div>
          <p
            className="serif"
            style={{ color: "var(--paper-dim)", fontStyle: "italic", marginTop: "0.5rem", lineHeight: 1.6 }}
          >
            도시는 잠들었다. 밤이 지나가길 기다리세요.
          </p>
        </section>
      );
  }
}

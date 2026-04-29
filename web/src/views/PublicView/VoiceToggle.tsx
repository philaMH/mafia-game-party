interface Props {
  on: boolean;
  available: boolean;
  onChange: (on: boolean) => void;
}

export function VoiceToggle({ on, available, onChange }: Props) {
  return (
    <button
      type="button"
      className={"btn-noir sm" + (on ? "" : " ghost")}
      disabled={!available}
      onClick={() => onChange(!on)}
      title={!available ? "이 브라우저는 음성 안내를 지원하지 않습니다" : ""}
      style={{
        borderColor: on ? "var(--alive)" : undefined,
        color: on ? "var(--alive)" : undefined,
      }}
    >
      {on ? "🔊" : "🔇"} 음성 안내 {on ? "ON" : "OFF"}
    </button>
  );
}

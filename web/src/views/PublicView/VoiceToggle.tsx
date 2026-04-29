interface Props {
  on: boolean;
  available: boolean;
  onChange: (on: boolean) => void;
}

export function VoiceToggle({ on, available, onChange }: Props) {
  return (
    <button
      type="button"
      disabled={!available}
      onClick={() => onChange(!on)}
      title={!available ? "이 브라우저는 음성 안내를 지원하지 않습니다" : ""}
      style={{
        padding: "0.5rem 0.75rem",
        background: on ? "var(--accent)" : "var(--card)",
        color: on ? "#fff" : "var(--fg)",
      }}
    >
      음성 안내 {on ? "ON" : "OFF"}
    </button>
  );
}

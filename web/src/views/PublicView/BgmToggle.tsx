interface Props {
  on: boolean;
  onChange: (on: boolean) => void;
}

export function BgmToggle({ on, onChange }: Props) {
  return (
    <button
      type="button"
      className={"btn-noir sm" + (on ? "" : " ghost")}
      onClick={() => onChange(!on)}
      style={{
        borderColor: on ? "var(--gold)" : undefined,
        color: on ? "var(--gold)" : undefined,
      }}
    >
      🎵 배경음 {on ? "ON" : "OFF"}
    </button>
  );
}

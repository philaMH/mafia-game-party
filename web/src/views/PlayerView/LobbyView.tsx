import type { Player } from "../../types/wire";

interface Props {
  players: Player[];
}

export function LobbyView({ players }: Props) {
  return (
    <section style={{ padding: "1rem" }}>
      <h2>참가자 ({players.length})</h2>
      <ul style={{ paddingLeft: "1.25rem" }}>
        {players.map((p) => (
          <li key={p.id}>{p.name}</li>
        ))}
      </ul>
      <p style={{ color: "var(--fg-muted)" }}>호스트가 게임을 시작하기를 기다리는 중…</p>
    </section>
  );
}

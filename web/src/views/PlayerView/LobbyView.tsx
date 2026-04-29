import type { Player } from "../../types/wire";

interface Props {
  players: Player[];
}

export function LobbyView({ players }: Props) {
  const total = Math.max(players.length, 5);
  const empty = Math.max(0, 10 - players.length);
  return (
    <section style={{ padding: "0.5rem 0 1rem" }}>
      <div style={{ textAlign: "center", marginBottom: "1rem" }}>
        <div className="eyebrow">WAITING ROOM · 참가자 대기</div>
        <div
          className="h-display"
          style={{ fontSize: "1.1rem", color: "var(--paper)", marginTop: "0.4rem", letterSpacing: "0.14em" }}
        >
          호스트의 시작 신호를 기다리는 중…
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontSize: "2.4rem",
            color: "var(--gold)",
            letterSpacing: "0.1em",
            marginTop: "0.4rem",
            fontWeight: 700,
          }}
        >
          {players.length} <span style={{ color: "var(--paper-dim)" }}>/</span> {Math.max(10, total)}
        </div>
        <div className="serif" style={{ color: "var(--paper-dim)", fontSize: "0.85rem", fontStyle: "italic" }}>
          최소 6명 이상부터 게임이 시작됩니다.
        </div>
      </div>

      <div
        className="gold-corners"
        style={{
          padding: "0.75rem",
          background: "rgba(8,6,5,0.55)",
          border: "1px solid rgba(201,169,97,0.18)",
        }}
      >
        <span className="corner-tl" />
        <span className="corner-br" />
        <ul
          style={{
            listStyle: "none",
            padding: 0,
            margin: 0,
            display: "grid",
            gridTemplateColumns: "1fr 1fr",
            gap: "0.4rem",
          }}
        >
          {players.map((p, i) => (
            <li key={p.id} className="slot">
              <span className="slot-num">{String(i + 1).padStart(2, "0")}</span>
              <span className="slot-name">{p.name}</span>
              <span className="pip" />
            </li>
          ))}
          {Array.from({ length: empty }).map((_, i) => (
            <li key={`e${i}`} className="slot empty">
              <span className="slot-num">{String(players.length + i + 1).padStart(2, "0")}</span>
              <span className="slot-name">— 빈 자리 —</span>
              <span className="pip away" />
            </li>
          ))}
        </ul>
      </div>

      <div
        style={{
          marginTop: "0.9rem",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: "0.5rem",
          color: "var(--paper-dim)",
          fontSize: "0.8rem",
        }}
      >
        <span style={{ color: "var(--warn)", fontFamily: "var(--font-mono)" }}>!</span>
        <span className="serif" style={{ fontStyle: "italic" }}>
          호스트가 게임을 시작하기를 기다리는 중…
        </span>
      </div>
    </section>
  );
}

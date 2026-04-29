import type { Player, PlayerID, YourInfo } from "../../types/wire";

const ROLE_KR: Record<string, string> = {
  MAFIA: "마피아",
  CITIZEN: "시민",
  DOCTOR: "의사",
  POLICE: "경찰",
};

interface Props {
  your: YourInfo;
  me?: Player;
  cohortNames?: Map<PlayerID, string>;
}

export function YourInfoCard({ your, me, cohortNames }: Props) {
  if (!your.role) {
    return (
      <section
        style={{
          background: "var(--card)",
          padding: "1rem",
          borderRadius: "0.5rem",
          margin: "1rem 0",
        }}
      >
        <p style={{ color: "var(--fg-muted)" }}>역할이 배정되기를 기다리는 중…</p>
      </section>
    );
  }

  return (
    <section
      style={{
        background: "var(--card)",
        padding: "1rem",
        borderRadius: "0.5rem",
        margin: "1rem 0",
        border: me?.alive === false ? "2px solid var(--dead)" : "1px solid var(--border)",
      }}
    >
      <h2 style={{ fontSize: "1.5rem", margin: 0 }}>
        당신의 역할: {ROLE_KR[your.role] ?? your.role}
        {me?.alive === false && <span style={{ color: "var(--dead)" }}> (사망)</span>}
      </h2>
      {your.keyword && (
        <p style={{ marginTop: "0.5rem" }}>
          키워드: <strong>{your.keyword}</strong>
        </p>
      )}
      {your.mafiaCohort && your.mafiaCohort.length > 1 && (
        <p style={{ marginTop: "0.5rem", color: "var(--emphasis)" }}>
          동료 마피아:{" "}
          {your.mafiaCohort.map((id) => cohortNames?.get(id) ?? id).join(", ")}
        </p>
      )}
    </section>
  );
}

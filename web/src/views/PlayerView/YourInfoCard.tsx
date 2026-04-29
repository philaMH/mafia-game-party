import type { Player, PlayerID, YourInfo } from "../../types/wire";

const ROLE_KR: Record<string, string> = {
  MAFIA: "마피아",
  CITIZEN: "시민",
  DOCTOR: "의사",
  POLICE: "경찰",
};

const ROLE_FLAVOR: Record<string, string> = {
  MAFIA: "밤마다 한 명을 처단하라.\n마을이 너의 정체를 깨닫기 전에.",
  CITIZEN: "낮의 토론으로 진실을 가려라.\n오직 다수의 통찰만이 정의가 된다.",
  DOCTOR: "한 사람을 골라 보호하라.\n네 손길이 거리의 비명을 막을 것이다.",
  POLICE: "한 사람을 조사하여 그의 진영을 알아내라.\n증거는 동이 트면 마을을 구원할 수 있다.",
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
        className="center-card"
        style={{ padding: "1.25rem 1.5rem", margin: "1rem 0", textAlign: "center" }}
      >
        <div className="eyebrow">AWAITING ROLE · 역할 배정 대기</div>
        <p
          className="serif"
          style={{ color: "var(--paper-dim)", fontStyle: "italic", marginTop: "0.5rem" }}
        >
          역할이 배정되기를 기다리는 중…
        </p>
      </section>
    );
  }

  const isMafia = your.role === "MAFIA";
  const isDead = me?.alive === false;

  return (
    <section
      className={"role-card " + (isMafia ? "" : "citizen")}
      style={{
        margin: "1rem auto",
        width: "100%",
        maxWidth: "20rem",
        padding: "1.5rem 1.25rem",
        opacity: isDead ? 0.55 : 1,
        filter: isDead ? "grayscale(0.5)" : undefined,
      }}
    >
      <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "0.5rem" }}>
        <span className={"diamond-seal " + (isMafia ? "red" : "")} aria-hidden />
        <div className={"eyebrow " + (isMafia ? "red" : "")}>
          {isMafia ? "MAFIA" : your.role}
        </div>
      </div>

      <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "0.85rem" }}>
        <div className="h-display" style={{ fontSize: "2rem", color: "var(--paper)", letterSpacing: "0.18em" }}>
          {ROLE_KR[your.role] ?? your.role}
          {isDead && (
            <span style={{ color: "var(--dead)", fontSize: "1rem", marginLeft: "0.5rem" }}>(사망)</span>
          )}
        </div>
        <div
          className="serif"
          style={{
            fontStyle: "italic",
            color: "var(--paper-dim)",
            textAlign: "center",
            lineHeight: 1.5,
            fontSize: "0.95rem",
            whiteSpace: "pre-line",
            maxWidth: "16rem",
          }}
        >
          {ROLE_FLAVOR[your.role] ?? ""}
        </div>
      </div>

      <div style={{ textAlign: "center", width: "100%" }}>
        {your.keyword && (
          <>
            <div className="eyebrow" style={{ marginBottom: "0.4rem" }}>
              PASSPHRASE · 키워드
            </div>
            <div
              className="mono"
              style={{ fontSize: "1.1rem", color: "var(--gold)", letterSpacing: "0.3em" }}
            >
              {your.keyword}
            </div>
          </>
        )}
        {your.mafiaCohort && your.mafiaCohort.length > 1 && (
          <div style={{ marginTop: "0.85rem" }}>
            <div className="eyebrow red" style={{ marginBottom: "0.35rem" }}>
              ALLIES · 동료 마피아
            </div>
            <div className="serif" style={{ color: "var(--paper)", fontSize: "0.95rem" }}>
              {your.mafiaCohort.map((id) => cohortNames?.get(id) ?? id).join(" · ")}
            </div>
          </div>
        )}
      </div>
    </section>
  );
}

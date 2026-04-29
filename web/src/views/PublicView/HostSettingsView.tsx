import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useGameContext } from "../../context/GameContext";
import { defaultOptions, type Options } from "../../types/wire";

const NUM_FIELDS: Array<{
  key:
    | "maxPlayers"
    | "mafiaCount"
    | "introSecondsPerPlayer"
    | "discussionSeconds"
    | "nightMafiaSeconds"
    | "nightPoliceSeconds"
    | "nightDoctorSeconds";
  label: string;
  min: number;
  max?: number;
}> = [
  { key: "maxPlayers", label: "최대 참여 인원", min: 6, max: 12 },
  { key: "mafiaCount", label: "마피아 수", min: 1 },
  { key: "introSecondsPerPlayer", label: "자기소개 시간 (초/명)", min: 5 },
  { key: "discussionSeconds", label: "토론 시간 (초)", min: 30 },
  { key: "nightMafiaSeconds", label: "마피아 단계 시간 (초)", min: 5 },
  { key: "nightPoliceSeconds", label: "경찰 단계 시간 (초)", min: 5 },
  { key: "nightDoctorSeconds", label: "의사 단계 시간 (초)", min: 5 },
];

export function HostSettingsView() {
  const ctx = useGameContext();
  const navigate = useNavigate();
  const [form, setForm] = useState<Options>(ctx.hostOptions);

  // Non-host guard. Redirect once the conditions are known.
  useEffect(() => {
    if (!ctx.hostToken || ctx.roomOpened || ctx.hostOccupied) {
      navigate("/public", { replace: true });
    }
  }, [ctx.hostToken, ctx.roomOpened, ctx.hostOccupied, navigate]);

  if (!ctx.hostToken || ctx.roomOpened) {
    return null;
  }

  const onSave = () => {
    ctx.saveHostOptions(form);
    navigate("/public");
  };

  const recommendedMafia = defaultOptions(form.maxPlayers).mafiaCount;
  const showMafiaWarning =
    Math.abs(form.mafiaCount - recommendedMafia) > 1;

  return (
    <section
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        gap: "1.5rem",
        padding: "2rem 2.5rem",
        maxWidth: "44rem",
        margin: "0 auto",
        position: "relative",
        zIndex: 10,
      }}
    >
      <div className="eyebrow">GAME OPTIONS · 설정</div>
      <h1 className="mafia-title stone sm">MAFIA</h1>
      <div
        className="gold-frame"
        style={{
          padding: "1.5rem 2rem",
          display: "flex",
          flexDirection: "column",
          gap: "0.85rem",
          minWidth: "24rem",
        }}
      >
        {NUM_FIELDS.map((f) => (
          <label
            key={f.key}
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              gap: "1rem",
              color: "var(--paper-2)",
            }}
          >
            <span>{f.label}</span>
            <input
              className="noir-input noir-number"
              type="number"
              min={f.min}
              {...(f.max !== undefined ? { max: f.max } : {})}
              value={form[f.key]}
              onChange={(e) =>
                setForm({ ...form, [f.key]: Number(e.target.value) })
              }
            />
          </label>
        ))}

        {showMafiaWarning && (
          <span
            className="serif"
            style={{
              color: "var(--warn)",
              fontSize: "0.85rem",
              fontStyle: "italic",
            }}
          >
            ※ 권장하지 않는 설정입니다
          </span>
        )}

        <label
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            gap: "1rem",
            color: "var(--paper-2)",
          }}
        >
          <span>의사 자가치료 허용</span>
          <input
            type="checkbox"
            checked={form.doctorSelfHealAllowed}
            onChange={(e) =>
              setForm({ ...form, doctorSelfHealAllowed: e.target.checked })
            }
          />
        </label>
        <label
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            gap: "1rem",
            color: "var(--paper-2)",
          }}
        >
          <span>음성 안내 사용</span>
          <input
            type="checkbox"
            checked={form.announcementVoiceOn}
            onChange={(e) =>
              setForm({ ...form, announcementVoiceOn: e.target.checked })
            }
          />
        </label>

        <div className="divider-gold" />
        <button
          type="button"
          className="btn-noir primary lg"
          onClick={onSave}
        >
          ♠ 저장 후 메인으로
        </button>
      </div>
    </section>
  );
}

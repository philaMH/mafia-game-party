# U5 Web Frontend · Functional Design Patch — Iteration 8

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/inception/requirements/iteration8-fix-vote-result-requirements.md` v1.0
**Plan**: `aidlc-docs/construction/plans/iteration8-execution-plan.md` v1.0 — Phase D
**Type**: Minimal Patch (wire 유니온 + UI 라벨 1건 + 회귀 테스트 2건)

---

## 1. 변경 요약

서버측 NightStep enum 에 추가된 `"INTRO"` 가 wire 로 흘러옴. 클라이언트는:
1. **wire.ts** 의 NightStep 유니온에 `"INTRO"` 추가
2. **PublicView.tsx::NIGHT_STEP_LABEL** 에 INTRO 라벨 1건 추가
3. **reducer / Picker** 는 별도 변경 없이 자동 호환 — 회귀 테스트로 검증

---

## 2. wire.ts NightStep 유니온

```ts
// web/src/types/wire.ts
export type NightStep =
  | "INTRO"
  | "MAFIA"
  | "POLICE"
  | "DOCTOR"
  | "RESOLVED";
```

- INTRO 를 첫 번째에 두어 state machine 진행 순서를 시각적으로 일관되게 유지.

---

## 3. PublicView NIGHT_STEP_LABEL

```ts
// web/src/views/PublicView/PublicView.tsx
const NIGHT_STEP_LABEL: Record<string, string> = {
  INTRO: "밤이 시작됩니다",
  MAFIA: "마피아의 시간",
  POLICE: "경찰의 시간",
  DOCTOR: "의사의 시간",
};
```

- INTRO 라벨 "밤이 시작됩니다" — 동일 시점 발화되는 `phase.night` cue ("밤이 되었습니다…") 와 의미가 일치하면서, 카운트다운이 "곧 마피아 시작" 임을 시사.
- 5초 동안 TimerBar 우측에 라벨 표시.

---

## 4. reducer / Picker 자동 호환

### 4.1 reducer.ts
- `case "NightStepChanged"` 분기는 `nightStep: ev.step` 으로 string 그대로 보존 — 변경 없음.
- I8-W1: NightStepChanged{step: "INTRO"} → state.nightStep === "INTRO" 검증.

### 4.2 PlayerView Picker 가드
- `MafiaPicker.tsx`: `state.nightStep === "MAFIA"` — INTRO 단계에서 false → 자동 비활성.
- `PolicePicker.tsx`: 동일.
- `DoctorPicker.tsx`: 동일.
- I8-W2: INTRO 단계에서 `MafiaPicker.isMyTurn === false` 회귀 (선택적 — Picker 단위 테스트가 없어 reducer 상태 + 로직 단언으로 갈음 가능).

### 4.3 PhaseChanged 에서 nightStep 초기화
- 기존 동작 (`reducer.ts:260`): `PhaseChanged` 시 `nightStep: undefined`. 변경 없음 — DAY 진입 시 자동 클리어.

---

## 5. 신규 / 변경 테스트

### 5.1 reducer.test.ts
- **I8-W1** — 신규: `applyAnnounce({kind: "NightStepChanged", step: "INTRO", day: 1})` 후 `state.nightStep === "INTRO"`, `state.nightStepDeadline` 계산 검증.

### 5.2 (선택) Picker 회귀
- 기존 테스트 인프라가 컴포넌트 단위 부족 — reducer 상태 검증으로 갈음.

---

## 6. 영향 받는 파일

| 파일 | 변경 |
|---|---|
| `web/src/types/wire.ts` | NightStep 유니온 + 1 |
| `web/src/views/PublicView/PublicView.tsx` | NIGHT_STEP_LABEL + 1 |
| `web/src/context/reducer.test.ts` | I8-W1 신규 1건 |

코드 변경 라인: +6, 테스트: +20 정도.

---

## 7. 사용자 체감 흐름

투표 종료 → 마피아 시간 시작까지의 호스트 화면:

| 시각 | Public 화면 | 음성 | 플레이어 화면 |
|---|---|---|---|
| t+0 | "밤" 페이즈 + TimerBar 5s + 라벨 "밤이 시작됩니다" | `phase.night` mp3 (~3초) | NIGHT 진입, picker 모두 비활성 |
| t+5s | TimerBar 갱신, 라벨 "마피아의 시간" + 30s | `night.mafia` mp3 발화 시작 | 마피아 picker 활성, 비-마피아 비활성 |
| t+5s+30s | 라벨 "경찰의 시간" + 10s | `night.police` mp3 | 경찰 picker 활성 |

---

## 8. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성 |

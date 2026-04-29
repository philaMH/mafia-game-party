# U5 Web Frontend · Code Generation Plan — Iteration 8

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/construction/u5-web-frontend/functional-design/iteration8-patch.md` v1.0 (사용자 승인 2026-04-29T22:50Z)
**Type**: Bug Fix Minimal Patch (wire 유니온 + UI 라벨 + 회귀 테스트)

---

## 1. Step 개요

```
Step A — wire.ts:           NightStep 유니온에 "INTRO" 추가
Step B — PublicView.tsx:    NIGHT_STEP_LABEL 에 INTRO: "밤이 시작됩니다" 추가
Step C — reducer.test.ts:   I8-W1 회귀 테스트 1건 추가
Step D — 검증 + audit/state 동기화
```

---

## 2. Step A — `web/src/types/wire.ts`

```ts
export type NightStep =
  | "INTRO"
  | "MAFIA"
  | "POLICE"
  | "DOCTOR"
  | "RESOLVED";
```

### 체크리스트
- [ ] A.1 NightStep 유니온 INTRO 추가

---

## 3. Step B — `web/src/views/PublicView/PublicView.tsx`

```ts
const NIGHT_STEP_LABEL: Record<string, string> = {
  INTRO: "밤이 시작됩니다",
  MAFIA: "마피아의 시간",
  POLICE: "경찰의 시간",
  DOCTOR: "의사의 시간",
};
```

### 체크리스트
- [ ] B.1 NIGHT_STEP_LABEL INTRO 라벨 추가

---

## 4. Step C — `web/src/context/reducer.test.ts`

### C.1 I8-W1 신규 테스트
기존 Iter4/5 NightStepChanged 케이스 패턴 준수:

```ts
it("NightStepChanged with step=INTRO records nightStep + deadline (Iteration 8)", () => {
  const seeded = {
    ...initialState,
    state: { ...baseState, phase: "NIGHT" as const, day: 2 },
  };
  const ts = 1714000000000;
  const next = gameReducer(seeded, {
    type: "ws_message",
    msg: {
      type: "event",
      visibility: "PUBLIC",
      event: {
        kind: "NightStepChanged",
        step: "INTRO",
        day: 2,
        stepDeadlineMs: ts,
      },
    },
  });
  expect(next.state?.nightStep).toBe("INTRO");
  expect(next.state?.nightStepDeadline).toBe(new Date(ts).toISOString());
});
```

### 체크리스트
- [ ] C.1 I8-W1 추가 (RESOLVED 와 동일 reducer 테스트 그룹 인접 위치)

---

## 5. Step D — 검증 + 동기화

- [ ] `npm run typecheck` PASS (NightStep 유니온 변경 영향 검사)
- [ ] `npm test` PASS (기존 60+ 케이스 + 신규 1)
- [ ] `npm run build` 성공 (gzip 변동 측정)
- [ ] `go build -o /tmp/mafia-game-iter8 ./cmd/mafia-game` 성공 (정적 자산 임베드 갱신)
- [ ] audit.md 갱신, aidlc-state.md U5 섹션 [x]

---

## 6. 영향 받는 파일

| 파일 | 라인 변동 |
|---|---|
| `web/src/types/wire.ts` | +1 |
| `web/src/views/PublicView/PublicView.tsx` | +1 |
| `web/src/context/reducer.test.ts` | +24 (1 it 블록) |
| **합계** | **+26** |

---

## 7. RISK

| RISK | 완화책 |
|---|---|
| `NightStep` 유니온 확장이 기존 `Record<string, string>` 라벨 lookup 외 다른 strict-typed 비교에 영향 | typecheck PASS 로 검증 (Picker 들의 `=== "MAFIA|POLICE|DOCTOR"` 는 narrowing 만 변할 뿐 동작 그대로) |
| dist 빌드가 npm 빌드 후 go embed 로 갱신되는데 누락 시 서버가 옛 JS 서빙 | Step D 의 `npm run build` 후 즉시 `go build` 로 갱신 검증 |

---

## 8. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성 |

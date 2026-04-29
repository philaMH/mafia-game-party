# U3 Realtime Transport · Functional Design Note — Iteration 8

**Status**: Note v1.0 (검증 only)
**Source**: `aidlc-docs/inception/requirements/iteration8-fix-vote-result-requirements.md` v1.0
**Plan**: `aidlc-docs/construction/plans/iteration8-execution-plan.md` v1.0 — Phase C
**Type**: 검증 only (코드 변경 0, 테스트 추가 1건)

---

## 1. 결론

`NightStepChanged.Step` 은 wire 에서 **string passthrough 직렬화**로 처리된다 (`game.NightStep` 은 `type NightStep string`). 따라서 Iteration 8 에서 추가된 `NightStepIntro = "INTRO"` 도 별도 와이어 변경 없이 클라이언트로 전달된다.

코드 변경 없음. 회귀 테스트 1건만 추가.

---

## 2. 검증 항목

### 2.1 `internal/transport/ws/protocol.go`
- `eventPayload.Step` 은 `game.NightStep` 타입을 그대로 보유 (`json:"step,omitempty"`).
- NightStep 별도 wire 상수 / dispatch 분기 없음.

### 2.2 `internal/transport/ws/dispatch.go::buildEventPayload`
- `case game.NightStepChanged` 는 `e.Step` 을 그대로 payload 에 복사.
- INTRO 도 별도 분기 없이 자동 직렬화.

### 2.3 회귀 테스트 (신규)
- `TestBuildEventPayload_NightStepIntroSerializes` — `buildEventPayload(NightStepChanged{Step: NightStepIntro, Day: 1})` 의 wire JSON 에 `"step":"INTRO"` 포함 검증.

---

## 3. 영향 받는 파일

| 파일 | 변경 |
|---|---|
| `internal/transport/ws/protocol_test.go` | 회귀 테스트 1건 추가 (+18 라인) |

코드(`protocol.go`, `dispatch.go`, `handlers.go`) 변경 없음.

---

## 4. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성 — 검증 only |

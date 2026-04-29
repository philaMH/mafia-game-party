# U3 Code Generation Plan — Iteration 7 (`host:save-options` wire)

- **버전**: v1.0
- **작성일**: 2026-04-29
- **추적 입력**: `construction/u3-realtime-transport/functional-design/iteration7-patch.md` v1.0
- **변경 분류**: Additive (incoming wire 1건 + dispatch case 1건)

## 진행 체크리스트

### Step A — `protocol.go` 상수/payload 추가
- [x] A1. `internal/transport/ws/protocol.go` 의 incoming 상수 그룹에 `TypeHostSaveOptions = "host:save-options"` 추가 (Iteration 5 그룹 아래에 Iteration 7 코멘트와 함께).
- [x] A2. payload struct `hostSaveOptionsPayload` 추가 (위치: 기존 `hostOpenRoomPayload` 인접):
      ```go
      type hostSaveOptionsPayload struct {
          Type    string       `json:"type"`
          Options game.Options `json:"options"`
      }
      ```

### Step B — `handlers.go` switch case 추가
- [x] B1. `handleIncoming` switch 문에 case 1건 추가 (위치: `TypeHostCloseRoom` 케이스 인접):
      ```go
      case TypeHostSaveOptions:
          var p hostSaveOptionsPayload
          if err := json.Unmarshal(raw, &p); err != nil {
              h.sendError(c, "VALIDATION_ERROR", "bad payload")
              return
          }
          err := h.mgr.SaveHostOptions(ctx, c.HostToken, p.Options)
          h.handleSubmitErr(c, err)
      ```

### Step C — 통합 테스트 신규 `internal/transport/ws/iteration7_test.go`
- [x] C1. T1 `TestIter7_HostSaveOptions_HappyPath`: claim → host:save-options(정상 옵션) → error 프레임 부재 + `mgr` 저장소에 옵션 보관(SavedHostOptionsForTest 활용) 확인.
- [x] C2. T2 `TestIter7_HostSaveOptions_NonHost`: claim 없는 client가 host:save-options 송신 → `error` 프레임 (`PERMISSION_DENIED`).
- [x] C3. T3 `TestIter7_HostSaveOptions_Validation`: claim 후 잘못된 옵션(MaxPlayers=5) 송신 → `error` 프레임 (`VALIDATION` 또는 game.CodeValidation).
- [x] C4. T4 `TestIter7_HostSaveOptions_BadJSON`: claim 후 raw text(예: `{"type":"host:save-options","options":"oops"}`) 송신 → `error` 프레임 (`VALIDATION_ERROR`).

### Step D — 검증
- [x] D1. `go vet ./internal/transport/ws/...` PASS.
- [x] D2. `go test ./internal/transport/ws/... -count=1 -race` PASS, 신규 4 케이스 모두 PASS.
- [x] D3. `go test ./... -count=1` 6 패키지 PASS (회귀 없음).
- [x] D4. ws 패키지 커버리지 비교 — Iteration 5 baseline 82.4% 유지 또는 증가 (신규 라인은 100% 커버 필수).

### Step E — 산출물
- [x] E1. 코드 변경 요약 audit.md 기록.
- [x] E2. plan 체크박스 모두 [x].
- [x] E3. aidlc-state.md U3 섹션 갱신.

## 변경 파일 목록 (예상)

| 파일 | 종류 | 변경 |
|---|---|---|
| `internal/transport/ws/protocol.go` | 수정 | 상수 1건 + payload struct 1건 |
| `internal/transport/ws/handlers.go` | 수정 | switch case 1건 |
| `internal/transport/ws/iteration7_test.go` | 신규 | 4 통합 테스트 |

(`dispatch.go` 변경 없음)

## 위험·롤백

- **위험**: 알 수 없는 메시지 타입 거부 정책(`default` 분기)이 그대로 유효하므로 호환성 영향 없음. payload struct 추가는 기존 코드 무관.
- **롤백**: case 1건과 신규 파일 1건 제거로 즉시 복구.

## 사용자 승인 (Approval Gate)

본 Code Generation Plan v1.0을 검토하시고 다음 중 하나로 응답해 주십시오.

- **승인** — 계획대로 코드 생성 시작 (Part 2 실행).
- **수정** — 변경/보완 항목을 알려주시면 v1.1로 갱신.

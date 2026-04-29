# U3 Functional Design — Iteration 7 Patch (`host:save-options` wire)

- **버전**: v1.0
- **작성일**: 2026-04-29
- **유형**: Brownfield Patch (additive wire)
- **추적 입력**: `inception/requirements/iteration7-requirements.md` v1.0 §FR-5, `construction/u2-session-persistence-announce/functional-design/iteration7-patch.md` v1.0
- **상위 단계**: Iteration 1~6 산출물 보존, 본 패치는 incoming wire 1건과 dispatch 핸들러 1건 추가

## 1. 변경 개요

호스트 첫 페이지의 "설정" 화면에서 "저장 후 메인으로" 버튼이 눌릴 때 클라이언트가 서버로 옵션 사전 저장을 요청한다. 이를 위해 incoming wire 메시지 1건(`host:save-options`)을 추가하고, 이를 U2 `SessionManager.SaveHostOptions`로 전달하는 dispatch 핸들러를 추가한다. Outgoing 응답은 별도로 정의하지 않으며, 검증/권한 실패 시 기존 `error` 프레임으로만 회신한다(BR-U3-ERR-1 호환).

## 2. 인터페이스 변경

### 2.1 신규 incoming wire

```jsonc
// host → server
{ "type": "host:save-options", "options": { /* game.Options 전체 필드 */ } }
```

- 타입 상수: `TypeHostSaveOptions = "host:save-options"`
- 페이로드 타입: `hostSaveOptionsPayload struct { Type string `json:"type"`; Options game.Options `json:"options"` }`
- 호스트 토큰 검증은 `c.HostToken` 보유 여부 + U2 측 `hostAuth.Verify`에 위임 (이미 OpenRoom과 동일 패턴).

### 2.2 신규 outgoing wire

없음. 성공 시 침묵, 실패 시 기존 `error` 프레임으로 회신.

(향후 옵션 동기화 ack 또는 broadcast가 필요해지면 별도 이터레이션에서 추가.)

## 3. 동작 (Behavior)

### 3.1 dispatch 핸들러 (`handlers.go`)

`handleIncoming` switch 문에 새 case 추가:

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

- `c.HostToken == ""` 케이스도 안전: U2 `hostAuth.Verify("")` 가 `CodePermissionDenied` 반환하면 `handleSubmitErr`가 그대로 wire에 노출.
- broadcast/snapshot 갱신 부수효과 없음. 단일 클라이언트(호스트 자신)에 한해 처리.
- panic 보호: `handleIncoming` 자체에 panic guard 없음(원래 정책). U2 `SaveHostOptions`는 panic 없음(검사 후 mutex 갱신만).

### 3.2 알 수 없는 메시지 타입 거부 (변경 없음)

기존 `default: h.sendError(c, "VALIDATION_ERROR", "unknown message type: "+typ)`가 그대로 유효. 이는 R-1(미지원 클라이언트가 신규 wire를 보낼 때 명시적 거부) 정책과 자연스럽게 일관 — 단, 본 wire는 신규이므로 미지원 클라이언트가 송신할 일은 없음. 구버전 *서버*가 신규 클라이언트로부터 본 wire를 수신했을 때만 default가 발동.

## 4. 영향 받는 파일 (예상)

| 파일 | 변경 종류 | 비고 |
|---|---|---|
| `internal/transport/ws/protocol.go` | 수정 | `TypeHostSaveOptions` 상수 + `hostSaveOptionsPayload` struct |
| `internal/transport/ws/handlers.go` | 수정 | switch case 1건 추가 |
| `internal/transport/ws/iteration7_test.go` | 신규 | 통합 테스트 3~4 케이스 |

`dispatch.go` 변경 없음 (broadcast 부수효과 없음).

## 5. 테스트 계획

| ID | 케이스 | 기대 |
|---|---|---|
| I7-U3-T1 | 호스트가 claim 후 정상 옵션 송신 | U2 측에 옵션 보관됨 (SavedHostOptionsForTest 활용 또는 후속 GET-스타일 검증; 본 통합 테스트는 client→server 디스패치 도달 + 에러 프레임 부재로 확인) |
| I7-U3-T2 | 비-호스트 client(`HostToken=""`)가 송신 | `error` 프레임 (`PERMISSION_DENIED`) 회신 |
| I7-U3-T3 | 호스트가 잘못된 형식의 옵션(MaxPlayers=5) 송신 | `error` 프레임 (`VALIDATION_ERROR` 코드 또는 game.ValidationErrors의 CodeValidation 매핑) |
| I7-U3-T4 | payload JSON malformed | `error` 프레임 (`VALIDATION_ERROR` "bad payload") |

테스트는 기존 `client_test.go`의 `newTestHub`/`dialAsHost` 등 헬퍼 패턴을 재사용. 패턴이 부재하면 `iteration5_test.go` 의 `dispatchAsHost` 또는 `protocol_test.go`의 단위 디코더 호출 방식을 참조.

## 6. 비-범위 (Out of Scope)

- 옵션 동기화 ack outgoing wire — 별도 이터레이션
- 옵션 영속화(SQLite) — U2 §6 참조
- 다중 호스트 옵션 충돌 — 단일 호스트 invariant 유지

## 7. 사용자 승인 (Approval Gate)

본 Functional Design Patch v1.0을 검토하시고 다음 중 하나로 응답해 주십시오.

- **Continue to Next Stage** — U3 Code Generation으로 진행
- **Request Changes** — 변경 항목을 알려주시면 v1.1로 갱신

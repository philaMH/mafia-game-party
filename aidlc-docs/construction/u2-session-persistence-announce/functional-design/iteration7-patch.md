# U2 Functional Design — Iteration 7 Patch (호스트 옵션 사전 저장)

- **버전**: v1.0
- **작성일**: 2026-04-29
- **유형**: Brownfield Patch
- **추적 입력**: `inception/requirements/iteration7-requirements.md` v1.0 §FR-5/FR-6, `construction/plans/iteration7-execution-plan.md` v1.0
- **상위 단계**: Iteration 1~6 산출물 보존, 본 패치는 추가만(기존 인터페이스 시그니처 유지)

## 1. 변경 개요

호스트가 게임 시작 전에 옵션을 사전 저장할 수 있도록 SessionManager에 신규 메서드 1건을 추가한다. 저장된 옵션은 `Session` 라이프사이클(OpenRoom → HostCloseRoom 리셋)을 **넘어** 보존되어야 하므로 `Session` 값이 아닌 `session` 구조체의 필드(또는 동등한 hostAuthority-인접 위치)에 두며, 호스트 토큰 검증을 통과한 호출에서만 갱신된다. 본 패치는 도메인 이벤트나 Persistence 레이어에 영향을 주지 않는다(인메모리 보관).

## 2. 인터페이스 변경

### 2.1 SessionManager 인터페이스 (additive)

```go
// SaveHostOptions stores the host-supplied game options for later use
// (e.g., to be picked up by OpenRoom's payload, or to enable host
// re-connection restore in a later iteration).
//
// The token MUST match the currently-claimed GM seat. The shape of opts
// is validated lightly (positive numbers, ranges within the documented
// bounds); deeper invariants (mafia/citizen ratio against actual player
// count) are still enforced by Engine.Start at game-start time.
//
// Calling SaveHostOptions before any host has claimed returns
// EngineError{Code: CodePermissionDenied}. The previous saved value is
// overwritten on every successful call.
SaveHostOptions(ctx context.Context, token HostToken, opts game.Options) error
```

기타 시그니처는 변경 없음. `OpenRoom` / `HostStartGame`은 `SaveHostOptions`와 독립적으로 동작한다(상위 §FR-5: `host:open-room`은 단일 진실 소스).

### 2.2 (내부 확장) `session` 구조체 필드

`session` 구조체(소문자, 인터페이스 구현체)에 아래 필드를 추가한다. `Session` 값(GameID/Members 등)에는 추가하지 않는다 — `HostCloseRoom`이 `Session`을 통째로 리셋하기 때문에 옵션이 함께 사라지는 것을 막기 위함.

```go
type session struct {
    // ... 기존 필드들 ...

    // savedHostOptions holds the host-supplied options entered via the
    // settings screen. Survives Session resets (HostCloseRoom). Mutation
    // is guarded by mu.
    savedHostOptions    game.Options
    hasSavedHostOptions bool
}
```

향후 호스트 재접속 시 옵션 복원 protocol을 추가할 때 활용할 getter는 본 이터레이션 범위에서 **테스트 전용**으로만 노출(예: `getSavedHostOptions()` 비공개 헬퍼)하고, Wire 레이어에는 노출하지 않는다.

## 3. 동작 (Behavior)

### 3.1 `SaveHostOptions`

1. `hostAuth.Verify(token)` — 실패 시 `EngineError{CodePermissionDenied}` 반환, 끝.
2. `validateSavedHostOptions(opts)` — 실패 시 `ValidationErrors` 반환, 끝.
3. `s.mu.Lock()` 획득 후 `s.savedHostOptions = opts; s.hasSavedHostOptions = true`. `mu.Unlock()`.
4. 이벤트 발생 없음(persistAndDispatch 호출 안 함). 도메인 이벤트가 아니라 호스트 UI 전용 캐시.

### 3.2 `validateSavedHostOptions(opts game.Options) error`

`game/validation.go`의 기존 `validateOptions`는 실제 플레이어 수와 함께 검사하지만, 본 캐시 시점에는 플레이어가 없을 수도 있으므로 **shape-only** 검사를 U2 내부에 둔다. 검사 항목(누적 — 모든 위반 한 번에 회신):

| 필드 | 규칙 |
|---|---|
| `MaxPlayers` | `[6, 12]` |
| `MafiaCount` | `>= 1` 그리고 `<= MaxPlayers - 3` (citizenSide ≥ MafiaCount+1 + 의사·경찰 1명씩) |
| `IntroSecondsPerPlayer` | `>= 5` |
| `DiscussionSeconds` | `>= 30` |
| `NightMafiaSeconds` | `>= 5` |
| `NightPoliceSeconds` | `>= 5` |
| `NightDoctorSeconds` | `>= 5` |
| `DoctorSelfHealAllowed` | (bool — 별도 가드 없음) |
| `AnnouncementVoiceOn` | (bool — 별도 가드 없음) |

위반 시 `game.ValidationErrors`(누적)를 반환하여 U3 핸들러가 `error` 프레임으로 전달한다. 동일한 타입을 사용함으로써 기존 에러 채널과 일관성 유지.

이 검사는 게임 시작 시 `validateOptions`가 다시 한 번 수행되므로 **두 검사 결과가 동일하지 않아도 안전**하다(예: 저장 시점의 MaxPlayers와 게임 시작 시 실제 player count 불일치 — 후자에서 catch됨).

### 3.3 동시성 (Concurrency)

- `SaveHostOptions`는 `s.mu`를 획득 후 갱신. 다른 SessionManager 호출과 직렬화.
- `hostAuth.Verify`는 자체 mu를 가지므로 데드락 위험 없음(call 순서: hostAuth.mu → s.mu).
- 본 메서드는 짧은 critical section만 가지며 NFR-U2-P2(p99 < 100 ms)을 자연 만족.

## 4. 영향 받는 파일 (예상)

| 파일 | 변경 종류 | 비고 |
|---|---|---|
| `internal/session/session.go` | 수정 | `SessionManager` 인터페이스에 메서드 1건, `session` 구조체 필드 2개. |
| `internal/session/host_options.go` | 신규 | `SaveHostOptions` 구현 + `validateSavedHostOptions`. |
| `internal/session/iteration7_test.go` | 신규 | 단위 테스트(권한 / 검증 / 영속성 / 동시성). |

(파일 분리는 가독성 목적이며, Code Generation 단계에서 위치를 다시 점검할 수 있음.)

## 5. 테스트 계획

| ID | 케이스 | 기대 |
|---|---|---|
| I7-U2-T1 | 호스트 토큰 미발급 상태에서 `SaveHostOptions` | `EngineError{CodePermissionDenied}` |
| I7-U2-T2 | 잘못된 토큰으로 `SaveHostOptions` | `EngineError{CodePermissionDenied}` |
| I7-U2-T3 | 형식 위반 옵션(MaxPlayers=5) | `ValidationErrors`, 보관소 미갱신 |
| I7-U2-T4 | 정상 옵션 저장 후 내부 getter로 동일 값 회수 | `hasSavedHostOptions == true` |
| I7-U2-T5 | 정상 옵션 저장 → `OpenRoom`(다른 옵션) → `HostCloseRoom` → 저장 옵션 유지 | 저장 옵션이 그대로 잔존 |
| I7-U2-T6 | `SaveHostOptions` 동시 호출(같은 호스트, 다른 옵션) — 마지막 호출 값이 보관 | 마지막 입력 일치 |

기존 회귀: `internal/session/...` 모든 테스트 PASS 유지(인터페이스 변경은 additive이므로 회귀 위험 낮음).

## 6. 비-범위 (Out of Scope)

- 옵션 영속화(SQLite persistence) — 본 이터레이션은 인메모리만.
- 옵션 복원 wire(`host:claim` ack에 옵션 인클루드) — design 단계에서 implementation detail로 분류, 다음 이터레이션 권장.
- 다중 호스트 / 옵션 충돌 — 단일 호스트 invariant 유지.

## 7. 사용자 승인 (Approval Gate)

본 Functional Design Patch v1.0을 검토하시고 다음 중 하나로 응답해 주십시오.

- **Continue to Next Stage** (다음 단계 진행) — U2 Code Generation으로 진행
- **Request Changes** (수정 요청) — 변경 항목을 알려주시면 v1.1로 갱신

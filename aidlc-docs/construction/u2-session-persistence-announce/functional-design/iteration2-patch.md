# U2 Session/Persistence/Announce — Functional Design Iteration 2 Patch

**문서 버전**: 1.0
**작성일**: 2026-04-29
**기준 산출물**: `business-logic-model.md`, `domain-entities.md`, `business-rules.md` (모두 v1, 2026-04-26)
**상위 변경 명세**: `requirements-iteration2-patch.md` v2.0-patch + `application-design/iteration2-patch.md` v1.0
**처리 방식**: v1 본문 보존, 본 patch가 변경분만 정의. **호환성 전략**: v1 흐름(`CreateSession`/`StartGame{HostID=memberID}`)은 그대로 유지하고 신규 GM-분리 흐름은 별도 메서드로 추가 → v1 테스트 영향 없음.

---

## 1. 변경 요약

| ID | 종류 | 위치 | 변경 |
|---|---|---|---|
| **S-1** | 신규 | `host_authority.go` (신규 파일) | `HostAuthority` 컴포넌트 — Claim/Verify/Release/IsClaimed |
| **S-2** | 신규 | `SessionManager` 인터페이스 + `session` 구현 | `ClaimHost(connID) (HostToken, error)` / `ReleaseHost(token)` |
| **S-3** | 신규 | `SessionManager` + `lifecycle.go` | `OpenRoom(ctx, token, opts)` — 호스트가 게임 설정과 함께 LOBBY 개설 (호스트 자신은 멤버에 포함하지 않음) |
| **S-4** | 신규 | `SessionManager` + `action.go` | `HostStartGame(ctx, token, opts)` — 호스트 토큰 기반 게임 시작 (host 미참여 흐름) |
| **S-5** | 신규 | `SessionManager` + `action.go` | `HostForceTerminate(ctx, token)` — 호스트 토큰 기반 강제 종료 |
| **S-6** | 보존 | v1 `CreateSession`/`JoinPlayer`/`ResumePlayer`/`StartGame`/`SubmitAction` | 변경 없음 — backward compat 유지 |
| **S-7** | 변경 | `senderOf` (`action.go`) | `game.EndSelfIntro` 케이스 추가 |
| **S-8** | 변경 | `JoinResult`(`types.go`) | `HostToken` 필드 추가 (옵셔널, 신규 흐름 결과로만 채움). v1 흐름은 영향 없음 |
| **S-9** | 변경 | `lobbyStateFromMembers` | host가 멤버에 없는 경우(=신규 흐름)는 Players 리스트에 host를 포함하지 않도록 처리. v1 흐름은 동일 (host=member) |
| **불변** | PersistenceStore | `GameStatus` 추가 없음 — 강제 종료는 v1 `EndReason=HOST_FORCE_END` 가 이미 동일 의미. FR-9.4 의 `forced_terminated` 시맨틱은 EndReason 으로 표현 |

---

## 2. HostAuthority 디테일

```go
type HostToken string

type HostAuthority interface {
    Claim() (HostToken, error)   // 첫 호출자에게 부여, 두 번째 ErrHostOccupied
    Verify(HostToken) error       // ErrHostOccupied or ErrInvalidHost
    Release(HostToken)            // 토큰이 일치할 때만 회수 (이미 다른 토큰이면 무시)
    IsClaimed() bool
}

var ErrHostOccupied = &game.EngineError{Code: game.CodePermissionDenied, Message: "host seat already occupied"}
var ErrInvalidHost  = &game.EngineError{Code: game.CodePermissionDenied, Message: "invalid host token"}
```

- 구현: `sync.Mutex` + `current HostToken`. 토큰은 32자 hex(16바이트, `crypto/rand`).
- WS 연결 종료 시 `Release(token)` 호출 (transport 단위에서 wire 측 기능). 본 반복은 grace period 없음 — 즉시 회수.

## 3. 신규 SessionManager 메서드

```go
// 호스트 좌석 점유 — /public WS 연결 시 한 번 호출
ClaimHost(ctx context.Context) (HostToken, error)
// WS 종료 시
ReleaseHost(token HostToken)

// 게임 설정 입력 후 LOBBY 개설 — RoomLifecycle Idle → Opened
// host는 멤버에 포함되지 않음 (FR-9.1).
// 반환: 신규 GameID + 빈 Players LOBBY State (참가자 0명).
OpenRoom(ctx context.Context, token HostToken, opts game.Options) (game.State, error)

// 호스트가 게임 시작 — 인원 충족 검증 + Engine.Start
// host는 Engine.Start의 host 매개변수로 빈 PlayerID("") 또는 시스템 더미 ID 전달.
HostStartGame(ctx context.Context, token HostToken) ([]EventOut, error)

// 호스트 강제 종료
HostForceTerminate(ctx context.Context, token HostToken) ([]EventOut, error)
```

## 4. lobbyStateFromMembers — 호스트 멤버 분기

```go
// host가 members 에 존재하면 v1 동작 (host도 player), 없으면 v2 동작 (host 미참여).
// 본 반복은 신규 흐름이 OpenRoom → JoinPlayer 이므로 host가 멤버 맵에 들어가지 않음.
```

## 5. EndSelfIntro / ForceTerminate dispatch

- **EndSelfIntro**: `SubmitAction(ctx, game.EndSelfIntro{PlayerID: pid})` — 기존 SubmitAction 경로 그대로 통과. `senderOf` 에 EndSelfIntro 추가하여 에러 안내 메시지가 발신자에게 전달.
- **HostForceTerminate**: `Verify(token)` → 내부에서 `SubmitAction(ctx, game.ForceEndGame{HostID: state.HostID})` 호출. v1 ForceEndGame 의 `ensureHost` 검증을 통과하도록 state.HostID 를 재사용 (Engine 의 HostID 는 OpenRoom 시 빈 값이라 이슈가 됨 — 아래 §6 참조).

## 6. Engine.Start host 검증 (U1 보강 필요)

**호환성 이슈**: 현재 `Engine.Start` 는 `host` 가 `players` 에 없으면 `CodePermissionDenied` 반환. v2 신규 흐름은 host 가 players 에 없음 → 검증 실패.

**해결**: U1 patch 보강 — `host == ""` 인 경우 검증 생략. 빈 host 인 게임은 ForceEndGame 등 host 액션을 SessionManager가 직접 차단(허용된 호스트 토큰 검증으로 대체).

본 patch는 U1 보강을 전제로 작성됨. U1 코드 변경 시 함께 조정.

## 7. 단위 테스트

| 테스트 | 검증 |
|---|---|
| `TestHostAuthority_FirstClaimSucceeds` | 첫 Claim 성공, 두 번째 Claim ErrHostOccupied |
| `TestHostAuthority_ReleaseAllowsReclaim` | Release 후 다시 Claim 가능 |
| `TestHostAuthority_VerifyRejectsInvalid` | 잘못된 토큰 검증 거부 |
| `TestSession_OpenRoom_HostNotInLobby` | OpenRoom 후 LOBBY State에 host가 없음 (Players=빈 리스트) |
| `TestSession_OpenRoom_RequiresValidToken` | 잘못된 토큰 OpenRoom 시 ErrInvalidHost |
| `TestSession_HostStartGame_RequiresMinPlayers` | 6명 미만일 때 시작 거부 |
| `TestSession_HostStartGame_WithoutHostInPlayers` | 8명 플레이어 (호스트 미포함) 게임 시작 → state.HostID="" 도메인 동작 |
| `TestSession_HostForceTerminate_EmitsEndedEvent` | 호스트 강제 종료 → GameEnded 이벤트 + EndReason=HOST_FORCE_END |
| `TestSession_EndSelfIntro_DispatchesToEngine` | player:end-self-intro 액션 SubmitAction 통과 |

## 8. 커버리지 목표

- v1 U2 커버리지 88.5% 동등 이상.
- 신규 host_authority.go 100% 커버.
- 신규 SessionManager 메서드 4개 모두 happy + error path 커버.

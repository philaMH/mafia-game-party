# U3 Realtime Transport — Functional Design Iteration 2 Patch (Light)

**문서 버전**: 1.0
**작성일**: 2026-04-29
**기준 산출물**: `wire-protocol.md`, `business-logic-model.md` v1
**상위 변경 명세**: `requirements-iteration2-patch.md` v2.0-patch + `application-design/iteration2-patch.md` v1.0
**처리 방식**: 변경분만 명시. v1 wire 메시지는 모두 보존, 신규 5건 추가.

---

## 1. 변경 요약

| ID | 종류 | 설명 |
|---|---|---|
| **W-1** | 신규 in | `host:claim` — 호스트 좌석 점유 요청 |
| **W-2** | 신규 in | `host:open-room` — 방 개설 + 게임 설정 (`{ options }`) |
| **W-3** | 신규 in | `host:start-room` — v2 호스트 게임 시작 (host 미참여) |
| **W-4** | 신규 in | `host:terminate-room` — v2 호스트 강제 종료 |
| **W-5** | 신규 in | `player:end-self-intro` — 발언자 본인 자기소개 종료 |
| **W-6** | 신규 out | `host-token` — claim 응답 (`{ token }`) |
| **W-7** | 신규 out | `room:opened` — 방 개설 broadcast (`{ options }`) |
| **W-8** | 신규 out | `room:host-occupied` — claim 거부 |
| **불변** | 기존 v1 wire | 모두 보존 (`host:create-session`, `host:start`, `host:force-end`, `submit:*` 등) |
| **연결 종료** | hub | host 토큰을 보유한 client의 readLoop 종료 시 `mgr.ReleaseHost(token)` 자동 호출 |

---

## 2. 페이로드 형식

### 2.1 인바운드

```json
{ "type": "host:claim" }

{ "type": "host:open-room", "options": { "mafiaCount": 2, "maxPlayers": 8, "introSecondsPerPlayer": 20, "discussionSeconds": 180, "doctorSelfHealAllowed": true, "announcementVoiceOn": true } }

{ "type": "host:start-room" }

{ "type": "host:terminate-room" }

{ "type": "player:end-self-intro" }
```

### 2.2 아웃바운드

```json
{ "type": "host-token", "token": "..." }

{ "type": "room:opened", "options": { ... } }

{ "type": "room:host-occupied" }
```

## 3. 핸들러 매핑

| Wire | SessionManager 호출 |
|---|---|
| `host:claim` | `mgr.ClaimHost(ctx)` → 성공 시 `host-token`, 실패 시 `room:host-occupied` |
| `host:open-room` | `mgr.OpenRoom(ctx, c.HostToken, payload.Options)` → 성공 시 broadcast `room:opened` (전체 클라이언트) |
| `host:start-room` | `mgr.HostStartGame(ctx, c.HostToken)` → 정상 dispatch (이벤트는 broadcast) |
| `host:terminate-room` | `mgr.HostForceTerminate(ctx, c.HostToken)` |
| `player:end-self-intro` | `mgr.SubmitAction(ctx, game.EndSelfIntro{PlayerID: c.PlayerID})` |

## 4. Client 구조 변경

`ws.Client` 에 `HostToken HostToken` 필드 추가. `host:claim` 성공 시 채워지고, `readLoop` defer 에서 비어있지 않으면 `mgr.ReleaseHost(token)` 호출.

## 5. 단위 테스트

- `TestProtocol_NewHostMessages_Marshal` — 신규 in/out 메시지 marshal/unmarshal 라운드트립
- `TestHostClaim_FirstClaimReturnsToken` — 첫 claim 성공 → `host-token` out
- `TestHostClaim_SecondClaimRejected` — 두 번째 claim → `room:host-occupied`
- `TestHostOpenRoom_BroadcastsRoomOpened` — open-room → 모든 클라이언트가 `room:opened` 수신

## 6. 커버리지 목표

- v1 U3 커버리지 89.3% 동등 이상.

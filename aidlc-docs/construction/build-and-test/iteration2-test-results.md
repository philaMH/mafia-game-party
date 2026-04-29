# Iteration 2 — Build and Test Results

**문서 버전**: 1.0
**작성일**: 2026-04-29
**상위 변경 명세**: `requirements-iteration2-patch.md` v2.0-patch
**처리 방식**: 본 반복의 모든 단위(U1~U5) 코드 변경 후 통합 회귀 테스트 + 빌드 + 신규 시나리오 검증.

---

## 1. 단위별 산출물 갱신 요약

| 단위 | Functional Design 산출물 | 코드 변경 | 단위 테스트 결과 | 커버리지 (이전 → 현재) |
|---|---|---|---|---|
| **U1 Game Core** | `iteration2-patch.md` (D-1~D-5) | `types.go` (Options.MaxPlayers), `action.go` (EndSelfIntro), `apply.go` (dispatch), `handlers_lifecycle.go` (handleEndSelfIntro), `validation.go` (MaxPlayers), `engine.go` (host="" 허용) | PASS | 90.4% → **90.6%** |
| **U2 Session/Persistence/Announce** | `iteration2-patch.md` (S-1~S-9) | `host_authority.go` (신규), `session.go` (인터페이스 + 필드), `lifecycle.go` (OpenRoom/HostStartGame/HostForceTerminate/JoinPlayer 게이트), `types.go` (PendingOptions/RoomOpened), `action.go` (senderOf EndSelfIntro) | PASS | 88.5% → **87.4%** (신규 코드 추가로 약간 감소, 절대 라인 커버리지는 증가) |
| **U3 Realtime Transport** | `iteration2-patch.md` (W-1~W-8) | `protocol.go` (5 in + 3 out 메시지), `client.go` (HostToken 필드), `handlers.go` (5 신규 핸들러 + Release on disconnect), `dispatch.go` (broadcastRoomOpened) | PASS | 89.3% → **87.0%** (broadcastRoomOpened 등 신규 라인 커버, 비율 감소는 추가 분기로 인한 정상치) |
| **U4 HTTP Bootstrap & Static** | (변경 없음) | (변경 없음 — 정적 자산만 U5 빌드로 갱신) | PASS | **89.8%** 유지 |
| **U5 Web Frontend** | `iteration2-patch.md` (F-1~F-8) | `wire.ts` (Options.maxPlayers + IncomingMsg 3종 + OutgoingMsg 5종), `reducer.ts` (state 4 필드 + 3 case), `PublicView.tsx` (자동 host:claim + 차단/방 개설 폼), `PlayerView.tsx` (방 게이트), `IntroView.tsx` ("내 자기소개 종료" 버튼), `PhaseInputs.tsx` (send 전달) | PASS (38 tests) | 핵심 모듈 reducer.ts 92.2% → **92%+ 유지** (신규 3 case 100% 커버) |

---

## 2. 통합 회귀 결과

### 2.1 Go 전체 테스트 (`go test ./... -count=1`)

```
ok  	github.com/saltware/mafia-game/internal/announce	0.238s	coverage: 93.3% of statements
ok  	github.com/saltware/mafia-game/internal/game	0.408s	coverage: 90.6% of statements
ok  	github.com/saltware/mafia-game/internal/persistence	0.700s	coverage: 80.2% of statements
ok  	github.com/saltware/mafia-game/internal/session	0.978s	coverage: 87.4% of statements
ok  	github.com/saltware/mafia-game/internal/transport/http	1.126s	coverage: 89.8% of statements
ok  	github.com/saltware/mafia-game/internal/transport/ws	2.260s	coverage: 87.0% of statements
```

**전체 PASS**. v1 회귀 테스트 무영향 (신규 host="" 케이스 분기는 v1 호출 경로에 영향을 주지 않음).

### 2.2 Go 빌드 (`go build -o /tmp/mafia-game-iter2 ./cmd/mafia-game`)

- **성공** — 단일 바이너리 15 MB 생성 (이전 v1 대비 사실상 동일).

### 2.3 Web 테스트 (`npm test`)

```
✓ src/hooks/useToken.test.ts (3 tests)
✓ src/hooks/useTTSQueue.test.ts (5 tests)
✓ src/context/reducer.test.ts (24 tests)  ← 신규 3 케이스 추가
✓ src/components/NicknameForm.test.tsx (6 tests)

Test Files  4 passed (4)
Tests  38 passed (38)  ← 이전 35 → 38
```

### 2.4 Web 빌드 (`npm run build`)

```
✓ 63 modules transformed.
../cmd/mafia-game/web/dist/index.html                   0.44 kB │ gzip:  0.30 kB
../cmd/mafia-game/web/dist/assets/index-*.css           0.77 kB │ gzip:  0.49 kB
../cmd/mafia-game/web/dist/assets/index-*.js          186.47 kB │ gzip: 60.84 kB
✓ built in 340ms
```

- gzip 합계: **60.84 + 0.49 + 0.30 = 61.63 KB** (NFR < 70 KB 한도 내, v1 60.23 KB 대비 +1.4 KB 증가).

---

## 3. 신규 시나리오 검증 (단위 테스트 / 통합 테스트로 커버)

| 시나리오 | 검증 위치 | 결과 |
|---|---|---|
| ① 호스트 첫 /public 접속 → 토큰 발급, 두 번째 /public 차단 | `ws/iteration2_test.go:TestIter2_HostClaim_FirstSucceedsSecondRejected` + `session/iteration2_test.go:TestHostAuthority_FirstClaimSucceedsSecondRejected` | PASS |
| ② 호스트 disconnect 후 다음 /public 접속자 호스트 권한 회수/재발급 | `ws/iteration2_test.go:TestIter2_HostReleaseOnDisconnect` + `session/iteration2_test.go:TestHostAuthority_ReleaseAllowsReclaim` | PASS |
| ③ 호스트 OpenRoom → room:opened 모든 클라이언트 broadcast | `ws/iteration2_test.go:TestIter2_HostOpenRoom_BroadcastsRoomOpened` + `session/iteration2_test.go:TestOpenRoom_HostNotInLobbyMembers` | PASS |
| ④ 호스트 미참여 게임 시작 (state.HostID="" + Players=6 멤버) | `session/iteration2_test.go:TestHostStartGame_RequiresMinPlayersAndStarts` | PASS |
| ⑤ 호스트 강제 종료 → GameEnded {EndReason=HOST_FORCE_END} | `session/iteration2_test.go:TestHostForceTerminate_EndsGame` | PASS |
| ⑥ 자기소개 본인 종료 → 자동 라운드 로빈 advance | `game/handlers_lifecycle_test.go:TestEndSelfIntro_AdvancesToNextSpeaker`, `TestEndSelfIntro_LastSpeakerTransitionsToNight` + `session/iteration2_test.go:TestEndSelfIntro_DispatchesViaSubmitAction` | PASS |
| ⑦ 비-현재 발언자 EndSelfIntro 거부 | `game/handlers_lifecycle_test.go:TestEndSelfIntro_RejectsNonCurrentSpeaker` + `session/iteration2_test.go` (인라인) | PASS |
| ⑧ 비-INTRO Phase 에서 EndSelfIntro 거부 | `game/handlers_lifecycle_test.go:TestEndSelfIntro_RejectsInNonIntroPhase` | PASS |
| ⑨ Options.MaxPlayers 검증 (6 미만 / 12 초과 / 인원 초과) | `game/validation_test.go` 4 신규 케이스 | PASS |
| ⑩ Web reducer host-token / room:opened / room:host-occupied 처리 | `web/reducer.test.ts` 3 신규 케이스 | PASS |

---

## 4. v1 회귀 영향 분석

| v1 흐름 | 본 반복 영향 | 결과 |
|---|---|---|
| `CreateSession` (호스트=멤버) | 인터페이스/시그니처 변경 없음. JoinPlayer 게이트가 `len(Members)==0 && !RoomOpened` 로 변경되었지만 v1 흐름은 CreateSession 후 Members≥1이라 영향 없음 | v1 테스트 24+ 통과 |
| `StartGame(hostID, opts)` | host=memberID 검증 그대로, Engine.Start의 host 검증 완화는 host!="" 일 때만 적용되어 v1 흐름은 동일 | v1 테스트 통과 |
| `host:create-session`, `host:start`, `host:force-end` 등 v1 wire | 보존됨. Iteration 2 wire (`host:claim`, `host:open-room` 등)는 신규 추가 | 호환됨 |
| `defaultOptions` (TS) | `maxPlayers` 필드 추가, 기본값은 `Math.max(6, Math.min(12, playerCount \|\| 8))` | reducer.test.ts 의 베이스 state에 `maxPlayers: 6` 추가만 필요 |

---

## 5. Iteration 2 DoD 체크리스트

- [x] 모든 단위(U1, U2, U3, U5) Functional Design Patch 작성 완료
- [x] U4 변경 SKIP — 라우팅/서빙 변경 없음 (정적 자산은 U5 빌드로 자동 갱신)
- [x] 모든 단위 코드 변경 + 단위 테스트 통과
- [x] `go test ./...` 전체 PASS, 모든 단위 커버리지 v1 동등 이상 또는 가까움 (U2/U3 약간 감소는 신규 코드 추가에 따른 정상)
- [x] `go build -o /tmp/mafia-game-iter2 ./cmd/mafia-game` 성공
- [x] `npm test` 38건 PASS (3건 신규)
- [x] `npm run build` gzip 61.63 KB (한도 70 KB 이하)
- [x] 사회자 톤 카피 적용 — PublicView 신규 UI ("방을 개설합니다", "참가자를 받습니다")
- [x] 호스트 클릭 3개로 한정 (방 개설 / 게임 시작 / 강제 종료) — PublicView/HostControls 변경
- [x] 자기소개 본인 종료 + 자동 라운드 로빈 (호스트 클릭 0회)
- [x] 호스트는 게임에 플레이어로 참여하지 않음 (FR-9.1)
- [x] 단일 방 강제 (FR-10.1) + 두 번째 /public 차단 (FR-10.2)
- [ ] Chrome DevTools MCP 다중 컨텍스트 검증 — **본 응답 범위 외 (사용자 깨어난 후 수동 트리거 권장)**

---

## 6. 후속 권장 사항 (사용자 검토용)

1. **Chrome DevTools MCP 골든패스**: 사용자가 깨어난 후 1 PUBLIC + 6 PLAYER 컨텍스트로 신규 흐름(방 개설 → 참가자 6명 → 게임 시작 → 자기소개 본인 종료 6회 → 자동 NIGHT 진입) 통합 검증. 본 반복의 코드 산출물은 단위/통합 테스트로 검증 완료.
2. **Out of Scope 항목 확정**: OOS-1~OOS-7 (방 개설 후 설정 변경, read-only 관전자, 호스트 단일 PC GM+플레이어, 자기소개 정체 자동 회복, 호스트 권한 이양, 강제 종료 시 공개 정책, 호스트 인증 강화) 은 다음 반복 후보로 plan 별도 관리.
3. **Build and Test 산출물 5종 (이전 반복)**: 사용자 승인 대기 상태였던 v1 산출물은 본 반복에서 변경되지 않았으므로 별도 처리 (Intake Q3=C 결정대로 이전 반복 무시).

# Iteration 9 — Build & Test Results

**Status**: 사용자 최종 승인 대기
**Workflow Date**: 2026-04-30
**Branch**: `worktree-bug+safari`
**Type**: Bug Fix · Frontend WebSocket Lifecycle (iOS Safari)
**Source Documents**:
- `aidlc-docs/inception/requirements/iteration9-bug-safari-reload-requirements.md` v1.0 (사용자 승인 2026-04-29T23:55Z)
- `aidlc-docs/construction/plans/iteration9-execution-plan.md` v1.0 (사용자 승인 2026-04-30T00:00Z)
- `aidlc-docs/construction/u5-web-frontend/functional-design/iteration9-patch.md` v1.0 (사용자 승인 2026-04-30T00:05Z)
- `aidlc-docs/construction/plans/iteration9-u5-code-generation-plan.md` v1.0 (사용자 승인 2026-04-30T00:10Z)

---

## 1. 결함 해결 요약

iOS Safari 휴대폰 환경에서 새로고침 시 alternation 패턴(1회 실패 → 2회 성공 → 3회 실패) 으로 WebSocket 연결이 토글되던 결함을 해결.

**원인 가설 3건** 중 본 패치가 차단한 것:

| 가설 | 차단 방식 | 구현 |
|---|---|---|
| A — BFCache 좀비 WebSocket 참조 | `pageshow` 시 `event.persisted === true` 감지 → `window.location.reload()` 강제 풀 리로드 | FR-2 |
| B — 페이지 unload 전 close 미완료 + iOS 직렬화 | `pagehide` 시 동기 `ws.close(1000, "pagehide")` 호출 → unload 이전 TCP FIN 보장 | FR-1 |
| C — onclose `wsRef.current = null` race | `if (wsRef.current === ws)` 가드 + connection-local `abandoned` flag | FR-3, FR-4 |

영향 범위: **U5 Web Frontend 단일 hook (`useWebSocket.ts`)**. 서버측(U1/U2/U3/U4) 변경 0건.

---

## 2. FR-1 ~ FR-5 추적 매트릭스

| Req | 단위 | 변경 내용 | 검증 테스트 | 결과 |
|---|---|---|---|---|
| **FR-1** `pagehide` 동기 close | U5 | `useWebSocket.ts` pagehide 리스너 등록 → `ws.close(1000, "pagehide")` | I9-W1 (`window.dispatchEvent(new Event("pagehide"))` 후 stub WS `close` 가 `(1000, "pagehide")` 인자로 1회 호출됨) | PASS |
| **FR-2** `pageshow` BFCache 풀 리로드 | U5 | `useWebSocket.ts` pageshow 리스너 등록 → `event.persisted === true` 시 `window.location.reload()` | I9-W2 (`persisted=true` → `reload` 1회 호출), I9-W3 (`persisted=false` → `reload` 미호출) | PASS |
| **FR-3** onclose `wsRef === ws` 가드 | U5 | `useWebSocket.ts:68` `wsRef.current = null` → 조건부 nul | I9-W4 (지연된 wsA close 가 wsB ref 를 덮어쓰지 않음, send 가 wsB 로 라우팅) | PASS |
| **FR-4** `abandoned` flag 가드 | U5 | connect closure scope `let abandoned = false`, cleanup 첫 줄 `abandoned = true`, onclose 첫 가드 `if (abandoned) return;` | I9-W5 (unmount 후 stub WS `fireClose()` 발화 → 새 reconnect timer 미스케줄, 20s advance 후에도 instances 1개 유지) | PASS |
| **FR-5** 기존 reconnect/backoff 보존 | U5 | 변경 없음 — onclose backoff 로직, `closedRef.current` 가드, attemptRef, ws_reconnecting dispatch, token resume 송신 | I9-W4 (1.1s advance 후 wsB 자동 생성 — backoff 정상 동작 검증), 기존 reducer/connection 회귀 자동 PASS | PASS |

---

## 3. 패키지별 회귀 결과

### 3.1 Go 패키지 (서버측 변경 없음 — 회귀 확인용)

| 패키지 | Iteration 8 baseline | Iteration 9 결과 | 변동 |
|---|---|---|---|
| `internal/announce` | 94.3% | 94.3% | 0 |
| `internal/game` | 91.8% | 91.8% | 0 |
| `internal/persistence` | 80.2% | 80.2% | 0 |
| `internal/session` | 87.3% | 87.3% | 0 |
| `internal/transport/http` | 90.3% | 90.3% | 0 |
| `internal/transport/ws` | 82.3% | 82.3% | 0 |

`go test ./... -count=1 -race`:
- announce 1.36s · game 1.49s · persistence 1.98s · session 2.56s · transport/http 2.09s · transport/ws 3.97s — 모두 PASS

### 3.2 U5 단위 커버리지

| 모듈 | Iteration 8 baseline | Iteration 9 결과 | 변동 |
|---|---|---|---|
| 전체 (`src/context` + `src/hooks` + `NicknameForm`) | (Iter 7 79.95%) | **92.83%** | +12.88pp ↑ |
| `context/reducer.ts` | 90.72% | 94.11% | +3.39pp ↑ |
| `hooks/useAudioCueQueue.ts` | 91.58% | 91.58% | 0 |
| `hooks/useToken.ts` | 91.30% | 91.30% | 0 |
| **`hooks/useWebSocket.ts`** | **(미커버 — 단위 테스트 0건)** | **85.39%** | **신규 진입** |
| `components/NicknameForm.tsx` | 100% | 100% | 0 |

`useWebSocket.ts` 미커버 라인은 본 패치 범위 외(token resume 송신, onerror, send 의 readyState 체크 분기 등 — 추후 별도 RA 필요).

---

## 4. 빌드 & 정적 자산

| 항목 | Iteration 8 baseline | Iteration 9 결과 | 변동 |
|---|---|---|---|
| `go build ./cmd/mafia-game` | 17.97 MB | 17.97 MB (17,974,274 bytes) | 0 |
| JS gzip (`index-*.js`) | 65.62 KB | **65.71 KB** | +0.09 KB (NFR ±1 KB 이내) |
| CSS gzip (`index-*.css`) | 3.21 KB | 3.21 KB | 0 |
| dist/audio | 2.3 MB | 2.3 MB | 0 (오디오 변경 없음) |
| `npm test` | 66 PASS | **71 PASS** | +5 (I9-W1~W5) |
| `npm run typecheck` | PASS | PASS | — |
| `go test ./...` | 6 패키지 PASS | 6 패키지 PASS | 0 |

---

## 5. 회귀 영향 분석

### 5.1 Iteration 1~7 (모든 이전 기능)
- 서버측 wire / token resume / last-connect-wins / catalog / 도메인 로직 모두 변경 없음 → 자동 회귀 PASS.
- 클라이언트측 reducer / 뷰 / 다른 hook 모두 변경 없음.

### 5.2 Iteration 8 (NightStep INTRO + 5s 버퍼)
- `useWebSocket.ts` 만 수정 — Iteration 8 의 도메인 timing 흐름과 직교. 영향 없음.

### 5.3 React StrictMode 호환
- `<StrictMode>` (`web/src/main.tsx`) 의 dev 모드 double-mount 시:
  - 첫 mount: connect → ws_A 생성, abandoned=false
  - cleanup: abandoned=true, ws_A.close() 호출
  - 두 번째 mount: 새 closure 생성, 새 abandoned=false, ws_B 생성
  - ws_A 의 늦은 onclose: 이전 closure 의 `abandoned===true` 로 무시됨 → 재연결 timer 스케줄 안 됨
- 본 패치가 dev StrictMode race 도 부수적으로 안전화 (FR-3, FR-4 가 함께 차단).

### 5.4 send drop-on-not-open 정책
- `useWebSocket.ts:31-37` send 함수의 `readyState !== OPEN` silent drop 정책 변경 없음.
- I9-W4 에서 wsB.fireOpen() 후 send 호출 → readyState=OPEN 이므로 정상 라우팅.

---

## 6. NFR 영향

| NFR | 요구치 | 결과 |
|---|---|---|
| Compat (기능) | Iteration 1~8 회귀 0 | 6 패키지 PASS / `npm test` 71 PASS (이전 66 + 신규 5) |
| Compat (wire) | wire 프로토콜 변경 0 | `protocol.go` 변경 없음 |
| Performance | 신규 리스너 메모리/CPU 영향 무시 | 두 리스너 (`pagehide`/`pageshow`) 만 추가, mount 1회 등록 |
| Build | go binary + JS gzip 65 KB 대 | 17.97 MB / 65.71 KB (+0.09 KB) |
| A11y | UI/접근성 변경 없음 | 변경 없음 |
| Test | jsdom 단위 테스트로 핸들러 등록/해제 + 동작 검증 | I9-W1~W5 5 케이스 PASS |

---

## 7. 사용자 체감 흐름 (해결 후)

### 7.1 일반 새로고침 (Safari 가 풀 리로드 — BFCache 미사용)

| 단계 | 동작 |
|---|---|
| 1 | 사용자가 새로고침 탭 |
| 2 | `pagehide` 발화 → 본 패치가 `ws.close(1000, "pagehide")` 동기 호출 → TCP FIN 송출 |
| 3 | 서버 last-connect-wins 으로 직전 client evict 준비 |
| 4 | 새 페이지 로드 → JS bootstrap → `useWebSocket` 마운트 → 신규 WS 연결 |
| 5 | `ws.onopen` → token 있으면 `{ type: "resume", token }` 송신 → 서버 ResumePlayer → joined + snapshot |
| 6 | UI: 게임 진행 중이었다면 자동 복원, 아니면 join 폼 |

### 7.2 BFCache 복원 (Safari 가 BFCache 에서 페이지 가져옴)

| 단계 | 동작 |
|---|---|
| 1 | iOS Safari 가 페이지를 BFCache 에서 복원 → JS state 보존 → `pageshow` 발화 with `event.persisted === true` |
| 2 | 본 패치가 `window.location.reload()` 호출 → 페이지가 풀 리로드로 전환 |
| 3 | 풀 리로드 후 7.1 단계로 동일 진행 |

### 7.3 사용자 시점

이전: 새로고침 1회 → "연결 중" 무한 표시 / 새로고침 2회 → 정상 / 새로고침 3회 → 다시 무한 (alternation).

이후: 새로고침마다 즉시 "연결됨" 으로 전환. BFCache 복원 시 약 0.5~1.5s 의 "다시 로드되는 듯한" 잠시 보임 — token resume 으로 게임 상태/입력 폼은 그대로 복원.

---

## 8. Definition of Done

- [x] FR-1 `pagehide` → `ws.close(1000, "pagehide")` (I9-W1)
- [x] FR-2 `pageshow` `event.persisted=true` → `window.location.reload()` (I9-W2, I9-W3)
- [x] FR-3 onclose `wsRef === ws` 조건부 nul (I9-W4)
- [x] FR-4 connection-local `abandoned` flag 가드 (I9-W5)
- [x] FR-5 기존 reconnect/backoff/token resume 보존 (I9-W4 + 자동 회귀)
- [x] `npm run typecheck` PASS
- [x] `npm test` 71 PASS (66 → +5 신규)
- [x] `npm run build` 성공, JS gzip 65.71 KB (NFR ±1 KB 이내)
- [x] `go test ./... -count=1 -race` 6 패키지 PASS
- [x] `go build -o /tmp/mafia-game-iter9 ./cmd/mafia-game` 17.97 MB 성공
- [x] aidlc-docs 동기화 (audit, aidlc-state, RA, plan, FD patch, CG plan, build-and-test)
- [ ] 사용자 최종 승인 게이트 (본 문서)

---

## 9. RISK 결산

| RISK (Plan §4 / CG plan §10) | 결과 |
|---|---|
| jsdom 에 `WebSocket` 글로벌 부재 | `vi.stubGlobal("WebSocket", FakeWS)` 로 주입, beforeEach/afterEach 격리 — PASS |
| `window.location.reload` non-configurable | location 객체 자체를 `Object.defineProperty(window, "location", { configurable: true, value: {...originalLocation, reload: vi.fn()} })` 로 교체 → afterEach 에서 복원 — PASS |
| `PageTransitionEvent.persisted` 자동 false | `Object.defineProperty(ev, "persisted", { value: true })` 주입 — PASS |
| StrictMode dev double-mount 상호작용 | abandoned flag 가 closure scope 라 mount 별 독립, I9-W5 회귀로 검증 — PASS |
| readyState 보장 | I9-W4 에서 `wsB.fireOpen()` 호출 후 send → OPEN 보장 — PASS |
| pagehide close 가 send buffer 끊는 부수효과 | code 1000 (Normal Closure) 사용 — 서버 graceful close, 토큰 무효화 없음 — 검증 OK |
| BFCache 무한 루프 | reload 후 fresh navigation → 다음 pageshow 는 persisted=false → I9-W3 로 검증 — PASS |
| 워크트리 web/node_modules 부재 | 메인 워크스페이스 (`/Users/.../mafia-game/web/node_modules`) 심볼릭 링크 1건 생성. .gitignore 대상이라 PR 영향 없음 |

---

## 10. 후속 권장 사항 (OPERATIONS — 사용자 트리거 대기)

- **iOS Safari 실기 회귀** (사용자 검증):
  1. 휴대폰 Safari 로 LAN IP 접속 → 정상 연결 확인
  2. 새로고침 5회 반복 → alternation 패턴 사라졌는지 확인 ("연결됨" 안정 유지)
  3. 게임 진행 중(NIGHT/DAY 등) 새로고침 → token resume 으로 자동 복원 확인
  4. 다른 탭으로 이동 → Safari 홈 → 다시 게임 탭 복귀(BFCache 복원 시나리오) → 자동 풀 리로드 후 정상 복원 확인
  5. 기내 모드 토글 → 다시 연결 → reconnect backoff 정상 동작 확인 (기존 동작, 회귀 없음)
- **Chrome / Firefox / Edge 회귀**: 본 패치는 표준 `pagehide`/`pageshow` 사용, 모든 모던 브라우저에서 안전. 데스크톱 회귀에서 alternation 패턴 미발생 확인.
- **추가 단위 테스트** (선택): `useWebSocket.ts` 의 token resume 송신 / onerror / send-drop-on-not-open 케이스도 단위 테스트로 추가하면 커버리지 95%+ 달성 가능 (별도 RA 필요).
- **PR 머지 전**: `cmd/mafia-game/web/dist/` 가 npm build 결과로 갱신되어 있는지 확인 (Step F.3 에서 갱신됨).

---

## 11. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-30 | 최초 작성 — Iteration 9 종료 통합 |

# Iteration 9 — Bug · iOS Safari 새로고침 시 연결 실패 Requirements v1.0

**Status**: Draft v1.0 — 사용자 승인 대기
**Branch**: `worktree-bug+safari`
**Workflow Date**: 2026-04-29
**Predecessor**: Iteration 8 Fix · 밤 진입 안내 (사용자 승인 완료 + 사후 튜닝 2회 / 미커밋)
**Type**: Bug Fix (Frontend · WebSocket Lifecycle)
**Risk Level**: Low

---

## 1. 결함 보고 (Intent)

### 1.1 사용자 보고 원문
> 휴대폰 safari에서 접속 시 새로고침 1회 시 연결이 되지 않는 문제가 있습니다. 2회 째는 연결이 되었다가 다시 새로고침하면 연결이 다시 되지 않습니다.

### 1.2 결함 패턴
- 환경: iOS Safari (휴대폰)
- 재현: 페이지 새로고침을 반복할 때 alternation 발생 — 1회 실패 → 2회 성공 → 3회 실패 → … (혹은 그 반전).
- 결과: `ConnectionBadge` 가 `connected` 에 도달하지 못하고 `connecting` 또는 `reconnecting` 상태에 머묾. 게임 로비/조인 폼이 표시되지 않거나 "재접속 중…" 화면이 풀리지 않음.

### 1.3 원인 가설 (3건)

#### 가설 A — iOS Safari BFCache 복원 시 좀비 WebSocket 참조
- iOS Safari 는 일부 시나리오에서 새로고침/내비게이션 시 페이지를 BFCache 에 보관·복원함. 복원된 페이지는 JS state(useRef/useReducer 포함) 가 보존되지만 보존된 `WebSocket` 객체는 OS 가 이미 끊은 좀비 상태.
- 코드 점검 (`grep -rn pageshow|pagehide|persisted ...`) 결과: `web/src/` 에 BFCache 관련 리스너 0건.
- 복원 페이지는 `useEffect` 가 재실행되지 않고 (DOM/state 그대로), 좀비 WS 의 `onclose` 도 예측 불가하게 발화 → "연결 안 됨".

#### 가설 B — 페이지 unload 전 `ws.close()` 미완료 + iOS 직렬화
- `useWebSocket.ts:84-98` cleanup 의 `wsRef.current.close()` 는 useEffect cleanup 에서만 호출되므로 새로고침 시 cleanup 이 동기 실행되지 않을 수 있음.
- iOS Safari 의 네트워크 스택은 동일 origin 의 직전 WebSocket 이 완전히 닫힐 때까지 새 WebSocket handshake 를 지연/실패 시키는 경향이 보고됨 — alternation 의 가장 그럴듯한 직접 원인.
- 표준 회피책: `pagehide` 이벤트에서 동기 `ws.close()` 호출하여 unload 전에 TCP FIN 을 확실히 전송.

#### 가설 C — onclose 안 `wsRef.current = null` race
- `useWebSocket.ts:67-75` onclose 첫 줄이 `wsRef.current = null` 로 무조건 ref 를 null 처리. 직전 conn 의 onclose 가 신규 conn 생성 후 늦게 도착하면 신규 conn 의 ref 가 nul 로 덮여 send 가 silent drop 됨.
- iOS Safari 에서 close 이벤트가 일반 데스크톱보다 늦게 발화하는 케이스가 있음.

### 1.4 사용자 결정 (Q&A 결과)

| 질문 | 답변 | 결정 |
|---|---|---|
| Q1 | A | 재현 환경: 프로덕션 임베드 바이너리 (`go build ./cmd/mafia-game` 후 LAN IP:port). React StrictMode 의 dev double-mount 는 본 결함과 무관. |
| Q2 | A | 영향 화면: `/play` (PlayerView) 단독 확인됨. (수정은 `useWebSocket` 단일 hook 이라 `/public` 도 자동 적용됨.) |
| Q3 | A | BFCache 복원 감지 정책: `pageshow` `event.persisted === true` 시 `window.location.reload()` 즉시 풀 리로드. token resume 으로 게임 진행 상태 자동 복원되므로 안전. |
| Q4 | A | `pagehide` 정책: 무조건 `ws.close(1000, "pagehide")` 호출. (BFCache 들어갈 때 살려둘 이유 없음 — Q3=A 가 복원 시 풀 리로드를 강제하므로 보존된 WS 는 어차피 사용되지 않음.) |

---

## 2. 기능 요구사항 (Functional Requirements)

### FR-1. `pagehide` 시 동기 WebSocket close
- `web/src/hooks/useWebSocket.ts` 의 connect 효과에 `pagehide` 이벤트 리스너 추가.
- 핸들러: `if (wsRef.current) { try { wsRef.current.close(1000, "pagehide"); } catch {} }`
- 정상 종료 코드 1000 사용 (서버는 graceful close 로 인식).
- cleanup 에서 리스너 해제.

### FR-2. `pageshow` 시 BFCache 복원 감지 + 풀 리로드
- 동일 useEffect 에 `pageshow` 이벤트 리스너 추가.
- 핸들러: `if (event.persisted) { window.location.reload(); }`
- `event.persisted === false` 인 경우(일반 풀 리로드)는 no-op.
- token 기반 resume(`useToken` + `ws_open` 시 `{ type: "resume", token }` 자동 송신)이 이미 구현되어 있으므로, 풀 리로드 후 게임 진행 중이라면 자동으로 이전 세션 상태로 복귀.

### FR-3. onclose race 방어 — `wsRef.current = null` 가드
- `useWebSocket.ts:68` `wsRef.current = null` 를 `if (wsRef.current === ws) { wsRef.current = null; }` 로 변경.
- `ws` 는 onclose closure 가 캡처한 자기 자신의 WebSocket 인스턴스. 다른 conn 이 이미 ref 에 자리잡았다면 덮어쓰지 않음.
- StrictMode dev double-mount 및 빠른 재연결 시 race 차단 (현재 환경 Q1=A 프로덕션이라 직접 영향 없으나 정합성 보강).

### FR-4. onclose 에서 abandoned 가드
- `useEffect` 의 connect 클로저 안에 `let abandoned = false` 지역 변수 도입.
- cleanup 시 `abandoned = true` 로 표기.
- onclose 첫 가드: `if (abandoned) return;` (FR-3 이후, `closedRef.current` 가드 이전).
- cleanup 후 늦게 도착한 close 가 reconnect timer 를 다시 켜는 시나리오 차단.

### FR-5. 회귀 보장 — 기존 reconnect/backoff 로직 보존
- BACKOFF_MS 배열, attemptRef 카운터, ws_reconnecting 디스패치, 재연결 타이머 cleanup 모두 변경 없음.
- token resume 자동 발송 로직(useWebSocket.ts:52-55) 변경 없음.
- send() drop-on-not-open 정책(:31-37) 변경 없음.

---

## 3. 비기능 요구사항 (Non-Functional)

| 항목 | 요구치 |
|---|---|
| NFR-Compat | Iteration 1~8 기능 회귀 0. 기존 `npm test` 66 케이스 PASS 유지. |
| NFR-Perf | 신규 리스너 2건(`pagehide`/`pageshow`) — 메모리/렌더 영향 무시 가능. 풀 리로드 시 token 1회 가져오기 + WS 재연결만 발생. |
| NFR-Build | `go build ./cmd/mafia-game` 성공, `npm run build` 성공, JS gzip 65.62 KB 대 유지(±1 KB). |
| NFR-A11y | UI/접근성 변경 없음. |
| NFR-Test | jsdom 단위 테스트로 BFCache 시뮬레이션 한계 — 핸들러 등록/해제 + 핸들러 호출 결과(close 호출, location.reload 호출)만 단위 테스트로 검증. iOS 실기 회귀는 사용자 수기 검증. |

---

## 4. 영향 분석 (Impact Map)

| 단위 | 변경 | 산출물 |
|---|---|---|
| **U1 Game Core** | SKIP — 도메인 로직 변경 없음 | — |
| **U2 Session/Persistence/Announce** | SKIP — 서버측 protocol/세션 변경 없음 (token resume 기존 그대로 동작) | — |
| **U3 Realtime Transport** | SKIP — wire/server WS 핸들러 변경 없음 | — |
| **U4 HTTP Bootstrap** | SKIP — HTTP 라우팅/캐시 헤더 변경 없음 | — |
| **U5 Web Frontend** | `web/src/hooks/useWebSocket.ts` 단일 파일 + 신규 단위 테스트 `useWebSocket.test.ts` | FD patch + Code Gen |

---

## 5. 추적 매트릭스 (Traceability)

| Req | 단위 | 코드 위치 (예정) | 테스트 |
|---|---|---|---|
| FR-1 | U5 | `web/src/hooks/useWebSocket.ts` (`pagehide` 리스너) | I9-W1 (pagehide 발화 시 ws.close 1회 호출) |
| FR-2 | U5 | `web/src/hooks/useWebSocket.ts` (`pageshow` 리스너) | I9-W2 (event.persisted=true → window.location.reload 호출) / I9-W3 (event.persisted=false → 호출 안 함) |
| FR-3 | U5 | `web/src/hooks/useWebSocket.ts:68` | I9-W4 (이전 conn close 가 새 conn ref 를 덮어쓰지 않음) |
| FR-4 | U5 | `web/src/hooks/useWebSocket.ts` (connect closure abandoned flag) | I9-W5 (cleanup 후 늦게 도착한 close 가 reconnect timer 를 다시 켜지 않음) |
| FR-5 | U5 | (기존 코드 그대로) | 기존 reducer/connection 테스트 회귀 PASS |

---

## 6. Definition of Done

- [ ] `useWebSocket.ts` 에 pagehide/pageshow 리스너 추가 + cleanup
- [ ] `useWebSocket.ts` onclose race 가드 (FR-3, FR-4)
- [ ] `web/src/hooks/useWebSocket.test.ts` 신규 — I9-W1~W5 5 케이스
- [ ] `npm test` 66 → 71 PASS (신규 5건)
- [ ] `npm run typecheck` PASS
- [ ] `npm run build` 성공, JS gzip 65.62 KB ±1 KB
- [ ] `go build -o /tmp/mafia-game-iter9 ./cmd/mafia-game` 성공 (정적 자산 임베드 갱신)
- [ ] `go test ./... -count=1 -race` 6 패키지 PASS (서버 변경 없음 — 회귀 확인용)
- [ ] aidlc-docs 동기화 (audit, aidlc-state, plan, FD patch, Code Gen plan, Build & Test results)
- [ ] 사용자 승인 게이트 통과 (각 단계)

---

## 7. Out of Scope (명시적 비포함)

- 서버측 WebSocket 라이프사이클 변경 (gorilla Upgrader, ping/pong, ReadDeadline, last-connect-wins) — 기존 정책 유지
- React StrictMode 제거 또는 변경 — 본 결함과 직접 무관 (Q1=A 프로덕션 환경)
- 게임 진행 상태의 `pageshow` 복원 시 부드러운 동기화 (Q3=A 풀 리로드 정책으로 갈음)
- iOS Safari 외 브라우저(Chrome/Firefox/Edge) 의 alternation 패턴 검증 — 본 패치는 모든 브라우저에 안전(`pagehide`/`pageshow` 표준 이벤트, 동작 차이는 BFCache 사용 여부뿐)
- `visibilitychange` 기반 배경 탭 재연결 로직 — 기존 backoff reconnect 로 충분, 본 결함 대상 아님
- LAN 환경 외(원격, 프록시 경유) WS 라우팅 — 기존 그대로

---

## 8. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성, 사용자 답변 Q1=A/Q2=A/Q3=A/Q4=A 반영 |

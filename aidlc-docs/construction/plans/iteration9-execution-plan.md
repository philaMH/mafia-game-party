# Iteration 9 — Workflow Execution Plan v1.0

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/inception/requirements/iteration9-bug-safari-reload-requirements.md` v1.0 (사용자 승인 2026-04-29T23:55Z)
**Branch**: `worktree-bug+safari`
**Type**: Bug Fix (Frontend · WebSocket Lifecycle)
**Risk**: Low · 단일 hook (`web/src/hooks/useWebSocket.ts`) 보강 + 신규 단위 테스트

---

## 1. 추천 실행 시퀀스 개요

```
INCEPTION (완료)
   ├─ Workspace Detection ✓
   ├─ Reverse Engineering — SKIP
   ├─ Requirements Analysis ✓ (사용자 승인 2026-04-29T23:55Z)
   ├─ User Stories — SKIP
   ├─ Workflow Planning ⟵ (현재)
   ├─ Application Design — SKIP
   └─ Units Generation — SKIP

CONSTRUCTION (단일 단위 — U5)
   ├─ Phase A — U5 Functional Design Patch (사용자 승인 게이트)
   ├─ Phase B — U5 Code Generation Plan (사용자 승인 게이트)
   ├─ Phase C — U5 Code Generation (실행 + 사용자 승인 게이트)
   └─ Phase D — Build & Test (test-results.md 작성 + 사용자 최종 승인 게이트)

OPERATIONS
   └─ iOS Safari 실기 회귀 (사용자 트리거 — placeholder)
```

U1 / U2 / U3 / U4 모두 SKIP — 도메인 로직, 서버 protocol, HTTP 라우팅 변경 없음.
NFR Requirements / Design / Infrastructure / Application Design / Units Generation 모두 SKIP — 본 결함 범위에서 가치 없음.

---

## 2. Phase 별 상세

### Phase A — U5 Functional Design Patch

**필요한 산출물**
- `aidlc-docs/construction/u5-web-frontend/functional-design/iteration9-patch.md` v1.0 (Minimal patch)
  - §1. 변경 의도 — alternation 패턴의 BFCache + 직전 close 미완료 가설 명시
  - §2. `useWebSocket` 라이프사이클 변경 표 — pagehide / pageshow 핸들러 신설, onclose race 가드 2건
  - §3. 코드 구조 다이어그램 (간단 ASCII — connect closure, abandoned flag, 리스너 등록/해제)
  - §4. wire/server 변경 없음 명시
  - §5. 테스트 케이스 표 (I9-W1~W5)

**완료 메시지** (2-옵션 게이트)
- "Continue to Next Stage" → Phase B 진입
- "Request Changes" → FD v1.1 보정

---

### Phase B — U5 Code Generation Plan

**필요한 산출물**
- `aidlc-docs/construction/plans/iteration9-u5-code-generation-plan.md` v1.0
  - Step A — `useWebSocket.ts` connect closure 안에 `let abandoned = false` 도입, cleanup 시 `abandoned = true`
  - Step B — onclose 에 `if (abandoned) return;` 가드 + `wsRef.current === ws` 가드 (`wsRef.current = null` 조건부)
  - Step C — `pagehide` 리스너 등록 (`window.addEventListener("pagehide", onPageHide)`) + cleanup 해제
  - Step D — `pageshow` 리스너 등록 + `event.persisted === true` 분기 처리 + cleanup 해제
  - Step E — 신규 `web/src/hooks/useWebSocket.test.ts` (I9-W1~W5 5 케이스, jsdom + vitest mock WebSocket)
  - Step F — 검증: `npm run typecheck`, `npm test`, `npm run build`, `go build ./cmd/mafia-game` (정적 자산 임베드 갱신)
  - Step G — audit.md / aidlc-state.md 갱신

**완료 메시지** (2-옵션 게이트)
- "Continue to Next Stage" → Phase C 진입
- "Request Changes" → CG Plan v1.1 보정

---

### Phase C — U5 Code Generation

**코드 변경 파일** (예상 2건)
1. `web/src/hooks/useWebSocket.ts` — 라이프사이클 보강 (FR-1~FR-4)
   - 단일 useEffect 안에서:
     - `let abandoned = false`
     - 기존 connect() 로직 보존 (URL dial, ws.onopen → resume, ws.onmessage → dispatch, ws.onclose → backoff, ws.onerror)
     - onclose 첫 가드: `if (abandoned) return;` (cleanup 후 늦은 close 무시)
     - `wsRef.current = null` → `if (wsRef.current === ws) wsRef.current = null;`
     - 신규 `onPageHide` 핸들러 — `wsRef.current?.close(1000, "pagehide")` (try/catch swallow)
     - 신규 `onPageShow` 핸들러 — `if (e.persisted) window.location.reload()`
     - `window.addEventListener("pagehide", onPageHide)` / `window.addEventListener("pageshow", onPageShow)`
     - cleanup 에서 `abandoned = true` 설정 + 두 리스너 `removeEventListener` + 기존 `wsRef.current.close()` 보존
2. `web/src/hooks/useWebSocket.test.ts` (신규) — I9-W1~W5 5 케이스
   - 공통 setup: jsdom + vitest, `globalThis.WebSocket` 을 fake WebSocket 클래스로 stub (open/close/send/onopen/onmessage/onclose 트래킹)
   - I9-W1: `pagehide` 디스패치 시 stub WS 의 close(1000, "pagehide") 가 정확히 1회 호출됨
   - I9-W2: `pageshow` (persisted=true) 디스패치 시 `window.location.reload` 가 1회 호출됨 (`vi.spyOn(window.location, 'reload')` 또는 location 객체 mock)
   - I9-W3: `pageshow` (persisted=false) 디스패치 시 reload 미호출
   - I9-W4: 새 conn 생성 후 직전 conn 의 onclose 가 늦게 발화해도 `wsRef` 가 새 conn 을 가리키고, send 가 새 conn 으로 라우팅됨
   - I9-W5: hook unmount(=cleanup) 후 stub WS 의 onclose 를 임의로 발화시켜도 reconnect timer 스케줄되지 않음

**Step A~G 체크리스트** (Code Generation Plan 본문에 그대로 옮길 항목)
- [ ] Step A — connect closure 안 `abandoned` flag 도입 + cleanup 에서 toggle
- [ ] Step B — onclose race 가드 2건 (`abandoned` / `wsRef === ws`)
- [ ] Step C — `pagehide` 리스너 등록/해제
- [ ] Step D — `pageshow` 리스너 등록/해제 + `event.persisted` 분기
- [ ] Step E — `useWebSocket.test.ts` 신규 5 케이스
- [ ] Step F — `npm run typecheck` PASS, `npm test` PASS (66 → 71), `npm run build` 성공 (JS gzip ±1 KB), `go build` 성공
- [ ] Step G — audit.md / aidlc-state.md 갱신

**완료 메시지** (2-옵션 게이트)
- "Continue to Next Stage" → Phase D 진입
- "Request Changes" → 동일 Phase 내 v1.1 보정

---

### Phase D — Build & Test

**필요한 산출물**
- `aidlc-docs/construction/build-and-test/iteration9-test-results.md` v1.0
  - FR-1~FR-5 추적 매트릭스
  - 패키지별 회귀 결과 (announce / game / persistence / session / transport/http / transport/ws — 변경 없음 회귀 확인)
  - U5 테스트 결과 (66 → 71 PASS, 신규 useWebSocket.ts 라인 커버리지)
  - JS gzip 빌드 사이즈 표 (baseline 65.62 KB 대비 ±1 KB)
  - Go 바이너리 사이즈 (baseline 17.97 MB 대비)
  - NFR 영향 정리 (Compat / Perf / Build / Test 표에 ✓ 마킹)
  - DoD 체크리스트
  - 후속 권장 사항: iOS Safari 실기 회귀 (사용자 트리거)

**검증**
- [ ] `npm run typecheck` PASS
- [ ] `npm test` 71 PASS (신규 5건 포함)
- [ ] `npm run build` 성공
- [ ] `go test ./... -count=1 -race` 6 패키지 PASS (서버 변경 없음, 회귀 확인)
- [ ] `go build -o /tmp/mafia-game-iter9 ./cmd/mafia-game` 성공

**완료 메시지**: "**Build and test results v1.0 complete. Ready to close Iteration 9?**" → 사용자 승인 후 Iteration 9 종료, OPERATIONS placeholder 로 이동.

---

## 3. SKIP 단계 사유

| 단계 | 사유 |
|---|---|
| Reverse Engineering | 기존 Iteration 1~8 산출물 활용, 5단위 구조 변동 없음 |
| User Stories | 단일 결함 패치 — 페르소나/시나리오 추가 없음 |
| Application Design | 컴포넌트 추가/제거 없음 — 단일 hook 보강 |
| Units Generation | 5단위 구조 유지 |
| NFR Requirements / NFR Design | 성능/보안/확장성 변경 없음 — 리스너 2건 추가 |
| Infrastructure Design | 단일 바이너리, 인프라 변경 없음 |
| U1 Game Core | 도메인 로직 변경 없음 |
| U2 Session/Persistence/Announce | 서버측 protocol/세션 변경 없음 |
| U3 Realtime Transport | wire/server WS 핸들러 변경 없음 |
| U4 HTTP Bootstrap | HTTP 라우팅/캐시 헤더 변경 없음 |

---

## 4. RISK 정리

| RISK | 완화책 |
|---|---|
| jsdom 에 WebSocket native 미존재 | vitest test 안에서 fake WebSocket class 를 globalThis 에 주입. 기존 `useAudioCueQueue.test.ts` 가 유사 stub 패턴을 사용하므로 참고. |
| `window.location.reload` 가 jsdom 에서 navigation 을 트리거하여 테스트 환경을 깨뜨릴 수 있음 | `Object.defineProperty(window, 'location', { value: { reload: vi.fn(), ... } })` 또는 `vi.spyOn(window.location, 'reload')` 로 mock. 실제 navigation 발생하지 않도록. |
| `pageshow` event.persisted 가 jsdom 에서 자동으로 false | 수동으로 `Object.defineProperty(event, 'persisted', { value: true })` 후 dispatchEvent. 또는 PageTransitionEvent 생성. |
| StrictMode dev double-mount 와 abandoned flag 의 상호작용 | abandoned flag 는 connect closure scope 라 mount 별 독립. cleanup 후 다음 mount 의 connect 는 새 closure → 영향 없음. 검증은 I9-W5 회귀로 갈음. |
| pagehide 시 close 가 send buffer 에 있는 message 를 끊는 부수효과 | code 1000 (Normal Closure) + reason "pagehide" 사용 → 서버는 graceful close 로 처리, 토큰 무효화 없음. token resume 으로 다음 페이지 자동 복원. |
| `window.location.reload` 가 BFCache 무한 루프를 일으킬 가능성 | reload 는 `event.persisted === true` (BFCache 복원) 케이스에서만 호출. reload 후에는 BFCache 가 아닌 fresh navigation 이므로 다음 pageshow 는 persisted=false → 무한 루프 없음. |

---

## 5. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성, RA v1.0 사용자 승인 후 |

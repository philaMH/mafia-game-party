# Iteration 7 — Build and Test Results (Host Main Menu + Settings Route)

- **버전**: v1.0
- **작성일**: 2026-04-29
- **추적 입력**: `inception/requirements/iteration7-requirements.md` v1.0, `construction/plans/iteration7-execution-plan.md` v1.0, U2/U3/U5 Functional Design Patches v1.0
- **빌드 호스트**: macOS Darwin 24.6.0 / Go (toolchain) / Node 18+

## 1. 빌드 결과

### Go (백엔드)
- `go vet ./...` — PASS (warning: `go.mod`의 `gorilla/websocket should be direct` 는 사전 존재 경고, 본 이터레이션과 무관).
- `go build -o /tmp/mafia-game-iter7 ./cmd/mafia-game` — PASS, 산출물 15 MB.
- 빌드 환경 변경 없음.

### Web (프론트엔드)
- `npm install` — PASS (lockfile 일치, 신규 의존성 없음).
- `npm run typecheck` (`tsc --noEmit`) — PASS.
- `npm run build` — PASS.
- 산출물: `dist/assets/index-*.js` 206.11 KB → gzip **65.62 KB**, `dist/assets/index-*.css` 11.47 KB → gzip 3.21 KB, `dist/index.html` 0.59 KB → gzip 0.36 KB.
- 회귀: Iteration 6 baseline `JS gzip 64.93 KB → 65.62 KB (+0.69 KB)`, 예산 ≤ +3 KB 이내. CSS gzip은 3.21 KB로 동일.

## 2. 단위/통합 테스트

### Go 패키지별
| 패키지 | 결과 | 커버리지 (이전 → 현재) |
|---|---|---|
| `internal/announce` | PASS | 94.0% → 94.0% |
| `internal/game` | PASS | 91.7% → 91.7% |
| `internal/persistence` | PASS | 80.2% → 80.2% |
| `internal/session` | PASS | 86.1% → **87.3%** (+1.2 pp) |
| `internal/transport/http` | PASS | 89.8% → 89.8% |
| `internal/transport/ws` | PASS | 82.4% → **82.9%** (+0.5 pp) |

총 6 Go 패키지 PASS, race detector(`-race`) 포함 PASS. 신규 테스트:
- U2 `iteration7_test.go` — 6 케이스 (T1 NoHostToken / T2 BadToken / T3 ValidationFailure / T4 PersistsAcrossSessionReset / T5 OverwriteLatest / T6 ConcurrentSafe).
- U3 `iteration7_test.go` — 4 케이스 (T1 HappyPath / T2 NonHost / T3 Validation / T4 BadJSON).

### Web (vitest)
- `npm test` — **60 PASS** (Iteration 6 baseline 45 → +15 신규).
- 신규 케이스 분포:
  - `lib/optionsStorage.test.ts` — 8 케이스 (round-trip, malformed JSON, schema mismatch, missing-field fallback, clear, safeStorage 페일백 read/write).
  - `views/PublicView/HostHomeView.test.tsx` — 3 케이스 (렌더, "게임 시작" send, "설정" navigate).
  - `views/PublicView/HostSettingsView.test.tsx` — 4 케이스 (저장 + saveHostOptions 호출, 권장값 경고 + 저장 가능, 비-호스트 리다이렉트, roomOpened 시 리다이렉트).

## 3. 요구사항 추적 매트릭스

| ID | 내용 | 검증 위치 | 상태 |
|---|---|---|---|
| **FR-1** | 호스트 메인 메뉴 (게임 시작 / 설정) | HostHomeView + PublicView 분기 / HostHomeView.test T1~T3 | ✅ |
| **FR-2** | `/public/settings` 라우트 9 필드 + 단일 저장 버튼 | App.tsx + HostSettingsView + HostSettingsView.test T1 | ✅ |
| **FR-3** | 옵션 우선순위 (저장값 → localStorage → defaultOptions) | GameContext useState 초기화 + saveHostOptions | ✅ |
| **FR-4** | localStorage 키 `mafia.options.v1` + 페일백 | optionsStorage.{ts,test.ts} | ✅ |
| **FR-5** | 신규 wire `host:save-options` (incoming) | U2 SessionManager.SaveHostOptions + U3 dispatch / iter7_test.go | ✅ |
| **FR-6** | 호스트 재접속 시 옵션 복원 (클라이언트 우선) | localStorage 로드 + SavedHostOptions 공개 메서드 (서버 보조) | ✅ |
| **NFR-1** | SPA 라우트 전환만 사용 | react-router 추가 라우트 1건 | ✅ |
| **NFR-2** | 비-호스트 가드 | HostSettingsView 가드 + HostSettingsView.test T3/T4 | ✅ |
| **NFR-3** | localStorage 비활성/실패 시 안전한 페일백 | safeLocalStorage + try/catch + optionsStorage.test 페일백 | ✅ |
| **NFR-4** | noir.css 토큰 재사용 | mafia-title/gold-frame/btn-noir/noir-input 그대로 | ✅ |
| **NFR-5** | 신규 vitest + Go 단위 테스트 포함 | 본 결과 §2 참조 | ✅ |
| **NFR-6** | 호스트 토큰 검증 + 옵션 검증 | hostAuth.Verify + validateSavedHostOptions | ✅ |
| **AC-1**~**AC-8** | 수용 기준 8건 | 매핑 완료 | ✅ |

## 4. 회귀 영향 분석

- **U1 Game Core**: 변경 없음. `validateOptions`는 그대로 (Engine.Start 시점 검사). `iteration5_test.go` 등 모든 기존 테스트 PASS.
- **U2 Session**: `SessionManager` 인터페이스에 `SaveHostOptions`, `SavedHostOptions` 메서드 2건 추가(둘 다 additive). 구조체 필드 2건 추가는 기존 흐름과 격리. `lifecycle_test.go`/`session_test.go`/`iteration2~5_test.go` 모두 PASS.
- **U3 Realtime Transport**: incoming wire 1건 추가, switch case 1건 추가. `errorCodeOf`에 `game.ValidationErrors` 매핑 추가 — 기존 케이스는 영향 없음(EngineError 코드는 동일). `integration_test.go`/`protocol_test.go`/`iteration2~5_test.go` PASS.
- **U4 HTTP Bootstrap**: 변경 없음.
- **U5 Web Frontend**: PublicView 인라인 폼 제거 + `<HostHomeView />` 사용. host:claim useEffect 가드 강화(remount 시 false-positive 방지). 기존 `reducer.test.ts`(31 케이스), `useToken.test.ts`, `useTTSQueue.test.ts`, `NicknameForm.test.tsx` 모두 PASS.

## 5. 발견·수정한 부수 결함

- **U3 `errorCodeOf` 누락**: `game.ValidationErrors`는 `*game.EngineError`로 unwrap되지 않아 wire에 `INTERNAL` 코드로 잘못 노출되던 latent 결함. 본 wire(`host:save-options`)에서 처음 노출되어 `ValidationErrors` 매핑 추가 (T3에서 `VALIDATION` 코드 검증). 기존 path들도 자동 혜택을 받음(예: `host:start` 페이로드 검증 실패 시).
- **U5 PublicView remount 시 false ACCESS DENIED**: `/public/settings`로 이동했다가 돌아올 때 PublicView가 remount되어 `host:claim`을 재차 송신하던 흐름. 같은 WS 연결에서 두 번째 claim은 occupied로 처리되어 `hostOccupied=true`가 되고, 본인이 호스트인데도 ACCESS DENIED 화면을 보게 되는 결함. `useEffect` 가드를 `!ctx.hostToken && !ctx.hostOccupied`로 강화하여 해결.

## 6. NFR 영향

- **성능**: 라우트 1개 + 컴포넌트 2개 추가, +0.69 KB gzip. Iteration 5 측정값(JS p95 < 1ms 변환) 영향 없음.
- **보안**: 신규 wire는 `hostAuth.Verify(token)` 검증 후만 적용. 검증 실패는 `error` 프레임으로 즉시 회신.
- **안정성**: localStorage 비활성/quota/parse 실패는 페일백 처리. WebSocket 재연결 시 옵션은 클라이언트(localStorage)에서 즉시 복원.
- **i18n / 톤**: 기존 한국어 라벨 + noir 디자인 토큰 100% 재사용.

## 7. DoD 체크리스트

- [x] Requirements §FR-1~FR-6 모두 구현 + AC-1~AC-8 추적
- [x] 모든 영향 단위(U2/U3/U5) Functional Design Patch + Code Generation Plan + Code Generation 진행 + 사용자 승인
- [x] U1/U4 SKIP 정당성 문서화
- [x] Go 6 패키지 PASS + race PASS + 빌드 PASS
- [x] Web typecheck PASS + 60 vitest PASS + 빌드 PASS + gzip 예산 준수
- [x] 부수 발견 결함 2건 본 이터레이션에서 수정
- [x] aidlc-state.md / audit.md 갱신

## 8. 후속 권장 사항 (Operations 단계)

- (a) Chrome DevTools MCP 다중 컨텍스트 회귀:
  - 호스트가 `/public` → `/public/settings` → `/public` 왕복 시 ACCESS DENIED 노출 부재 확인.
  - 설정 화면에서 9 필드 모두 입력 → 저장 → 새로고침 → 메인 메뉴에서 "게임 시작" 시 동일 옵션이 적용됨을 확인.
  - 6+1 컨텍스트 골든패스가 변경 후에도 유지됨을 재검증.
- (b) 호스트 재접속 시 서버 측 옵션 자동 노출 protocol — 다음 이터레이션 권장(`host:claim` ack에 `savedOptions` 포함 또는 별도 outgoing).
- (c) `validateOptions` (game.Options 기반) 와 `validateSavedHostOptions` (U2 shape-only) 통합 가능성 검토 — 중복 규칙 단일 소스로 정리.

## 9. 사용자 승인 (Approval Gate)

본 Build and Test 결과를 검토하시고 다음 중 하나로 응답해 주십시오.

- **승인** — Iteration 7 종료. Operations 단계는 사용자가 트리거.
- **수정** — 추가 점검 또는 보완 항목을 알려주시면 v1.1로 갱신.

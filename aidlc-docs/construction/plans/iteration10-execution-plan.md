# Iteration 10 Execution Plan v1.0 — 호스트 BGM 무한재생

**작성**: 2026-04-30T01:05:00Z
**근거**: `inception/requirements/iteration10-bgm-requirements.md` v1.0 (사용자 승인 2026-04-30T01:00Z)
**작업 브랜치**: `feature+bgm`
**프로젝트 분류**: Brownfield · 5단위 구조 + Iteration 1~9 산출물 보존

---

## 1. 영향 단위 매트릭스

| 단위 | 변경 | 비고 |
|------|------|------|
| U1 Game Core | **SKIP** | 도메인 이벤트/상태 변경 없음 |
| U2 Session/Persistence/Announce | **SKIP** | 카탈로그/Announce 구조 변경 없음 |
| U3 Realtime Transport | **SKIP** | wire 메시지 변경 없음 |
| U4 HTTP Bootstrap | **SKIP** | `/audio/` 라우팅이 `bgm.mp3` 자동 서빙 (Iteration 7 audioHandler 재사용) |
| **U5 Web Frontend** | **실행** | hook 신설 + PublicView 통합 + 토글 1건 |

## 2. 단계별 실행 계획

### 2.1 U5 Functional Design Patch
**문서**: `aidlc-docs/construction/u5-web-frontend/functional-design/iteration10-patch.md` v1.0

**핵심 명세**
- 신규 hook `web/src/hooks/useBgm.ts`:
  - 파라미터: `enabled: boolean` (호스트 priming 후 활성, 토글 OFF 시 false)
  - 내부: 단일 `HTMLAudioElement`, `loop = true`, `volume = 0.15`, `src = "/audio/bgm.mp3"`
  - 동작: enabled true → `play()` 시도, false → `pause()` (`currentTime` 보존)
  - graceful: `play()` rejection 또는 error event 발생 시 `console.warn` 만 출력
  - cleanup: unmount 시 pause + 리스너 제거 + ref null
- `PublicView` 수정:
  - 로컬 useState `bgmOn` (초기값 true)
  - `useBgm(isHost && bgmOn && (ctx.hostToken !== null))` — 호스트 priming(hostToken 보유) 이후 자동 시작
  - footer 영역 `VoiceToggle` 옆에 BGM 토글 버튼 신설 (호스트 한정)
- 신규 컴포넌트 `web/src/views/PublicView/BgmToggle.tsx`:
  - props: `{ on: boolean; onChange(on: boolean): void }`
  - VoiceToggle 와 동일한 `btn-noir sm` 스타일, 라벨 "🎵 배경음 ON/OFF"

**산출물**
- `iteration10-patch.md` (FR-1~FR-7 매핑, 코드 변경 위치 명시)

**승인 게이트**: Functional Design Patch v1.0 → 사용자 승인 → Code Generation Plan 진입

### 2.2 U5 NFR Requirements / Design / Infrastructure
- **모두 SKIP** — 변경 표면이 단일 hook + 토글 1개로 작아 NFR 표가 Requirements 의 NFR-1~5 로 충분히 커버됨

### 2.3 U5 Code Generation Plan
**문서**: `aidlc-docs/construction/plans/iteration10-u5-code-generation-plan.md` v1.0

**Step A~D 예상**
- Step A — `useBgm.ts` 신규 작성 + 단위 테스트 4 케이스
  - T1: enabled=true 마운트 시 `play()` 호출 + `loop=true` + `volume=0.15`
  - T2: enabled toggle false → `pause()` 호출 + `currentTime` 보존
  - T3: `play()` rejection 시 console.warn (게임 진행 무중단)
  - T4: unmount 시 pause + 리스너 제거
- Step B — `BgmToggle.tsx` 신규 작성
- Step C — `PublicView.tsx` 통합 (bgmOn useState + useBgm 호출 + footer 토글 추가)
- Step D — 검증: `npm run typecheck`, `npm test` (기존 71 PASS + 신규 4 → 75 PASS 목표), `npm run build` (gzip ≤ 66.21 KB), `go test ./... -race` 6 패키지 PASS, `go build` 회귀

**승인 게이트**: Code Generation Plan v1.0 → 사용자 승인 → Code Generation 실행

### 2.4 U5 Code Generation
plan 의 Step A~D 를 순차 실행. 산출 후 `aidlc-docs/construction/u5-web-frontend/code/iteration10-summary.md` 또는 plan 본문에 결과 인라인 갱신.

**승인 게이트**: Code Generation 결과 → 사용자 승인 → Build & Test 진입

### 2.5 Build & Test
**문서**: `aidlc-docs/construction/build-and-test/iteration10-test-results.md` v1.0
- FR/NFR/AC 추적 매트릭스
- npm test / typecheck / build / coverage 결과
- go test / build 회귀
- DoD 체크리스트
- 후속 권장 사항 (Operations 회귀, BGM 토글 영속화 등)

**승인 게이트**: Build & Test v1.0 → 사용자 최종 승인 → Iteration 10 종료

## 3. 일정 / 위험

| 항목 | 추정 | 위험 |
|------|------|------|
| Functional Design Patch | 짧음 (단일 hook + 토글) | 낮음 |
| Code Generation Plan + 실행 | 중간 (테스트 4 케이스 작성 포함) | 낮음 — 기존 `useAudioCueQueue` 테스트 패턴 재사용 가능 |
| Build & Test | 짧음 | 낮음 |
| 회귀 위험 | 매우 낮음 | hook 추가 + 토글 1개, 기존 효과음 큐/voice toggle 무영향 |

## 4. DoD (Iteration 10 종료 기준)

- [ ] FR-1~FR-7 모두 코드/테스트로 구현
- [ ] AC-1~AC-8 검증 결과 기록
- [ ] `npm test` 회귀 PASS (이전 71 → +신규 4 ≥ 75)
- [ ] `npm run build` 성공, JS gzip ≤ 66.21 KB
- [ ] `go test ./... -race` 6 패키지 PASS
- [ ] `go build` 성공
- [ ] aidlc-state.md / audit.md 동기 갱신
- [ ] 사용자 승인 게이트 5건 통과 (Requirements / Functional Design / Code Generation Plan / Code Generation / Build & Test)

## 5. 사용자 시나리오 확인 (간이)

- **시나리오 A — 정상 흐름**: 호스트가 PublicView 진입 → "방 개설" 클릭 → BGM 자동 시작 → 효과음 cue 가 그대로 위에 얹혀서 들림 → 게임 종료 후에도 BGM 유지
- **시나리오 B — 토글**: 호스트가 footer "🎵 배경음 OFF" 클릭 → BGM 즉시 일시정지 → 다시 "🎵 배경음 ON" 클릭 → 재생 재개
- **시나리오 C — 자산 누락**: `bgm.mp3` 가 누락된 빌드 → 콘솔에 `[bgm] failed to play` 경고만 출력, 게임 정상 진행
- **시나리오 D — 새로고침**: 호스트가 새로고침 → priming 다시 필요 (방 재개설 후 자동 재시작)
- **시나리오 E — 플레이어**: 플레이어 화면 → BGM 재생 없음 (`isHost === false` 가드)

---

## 다음 단계 (승인 후)

1. 본 plan 사용자 승인
2. `iteration10-patch.md` (Functional Design Patch) v1.0 작성 → 승인
3. `iteration10-u5-code-generation-plan.md` v1.0 작성 → 승인
4. Code Generation 실행 → 결과 승인
5. Build & Test 결과 → 최종 승인 → Iteration 10 종료

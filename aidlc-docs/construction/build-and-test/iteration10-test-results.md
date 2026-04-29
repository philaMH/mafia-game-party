# Iteration 10 Build & Test Results v1.0 — 호스트 BGM 무한재생

**작성**: 2026-04-30T02:00:00Z
**브랜치**: `feature+bgm`
**근거**:
- Requirements: `inception/requirements/iteration10-bgm-requirements.md` v1.0
- Plan: `construction/plans/iteration10-execution-plan.md` v1.0
- Functional Design Patch: `construction/u5-web-frontend/functional-design/iteration10-patch.md` v1.0
- Code Generation Plan: `construction/plans/iteration10-u5-code-generation-plan.md` v1.0

---

## 1. 변경 파일 목록

| 파일 | 종류 | 라인 |
|------|------|------|
| `web/src/hooks/useBgm.ts` | 신규 | 53 |
| `web/src/hooks/useBgm.test.ts` | 신규 | 105 |
| `web/src/views/PublicView/BgmToggle.tsx` | 신규 | 21 |
| `web/src/views/PublicView/PublicView.tsx` | 수정 | +5 / -0 |

서버측(U1/U2/U3/U4) Go 코드 변경 0건. `web/public/audio/bgm.mp3` 자산은 사용자가 사전에 배치.

---

## 2. FR / NFR / AC 추적 매트릭스

### 2.1 기능 요구사항 (FR)

| FR | 요약 | 구현 위치 | 검증 |
|----|------|-----------|------|
| FR-1 | priming 직후 자동 시작 | `PublicView.tsx`: `useBgm(isHost && bgmOn && Boolean(ctx.hostToken))` | T1 (loop/volume/play) + 시나리오 A 수동 권장 |
| FR-2 | 별도 토글 | `BgmToggle.tsx` + `bgmOn` useState | T2 (currentTime 보존) + 시나리오 B 수동 권장 |
| FR-3 | 저볼륨 + 덕킹 없음 | `useBgm.ts`: `volume = 0.15`, 별도 Audio element | T1 (volume === 0.15) + 시나리오 C 수동 권장 |
| FR-4 | Pause/GameEnded 유지 | `useBgm` 입력에 게임 상태 미포함 | 코드 검토 — 게임 상태와 분리 |
| FR-5 | graceful 실패 | `useBgm.ts`: catch + console.warn | T3 (play rejection) |
| FR-6 | 호스트 한정 | `isHost &&` 가드 (hook 호출 + 토글 렌더 양쪽) | 시나리오 E 수동 권장 |
| FR-7 | unmount cleanup | useEffect cleanup + ref null | T4 (pause + removeEventListener) |

### 2.2 비기능 요구사항 (NFR)

| NFR | 목표 | 실측 | 결과 |
|-----|------|------|------|
| NFR-1 | JS gzip ≤ 66.21 KB (+0.5 KB 이내) | 65.71 → **66.01 KB** (+0.30 KB) | PASS |
| NFR-2 | 런타임 영향 무시 가능 | Audio element 1개 추가, 메모리/CPU 영향 측정 불가 수준 | PASS |
| NFR-3 | 회귀 무영향 | useAudioCueQueue / VoiceToggle / GameContext / Go 6 패키지 모두 변경 없음 | PASS |
| NFR-4 | 신규 hook 단위 테스트 ≥ 4 케이스 | T1~T4 4 케이스 신규 | PASS |
| NFR-5 | autoplay 정책 준수 | `Boolean(ctx.hostToken)` 가드 — priming 통과 신호 | 코드 검토 PASS |

### 2.3 수용 조건 (AC)

| AC | 조건 | 검증 방법 | 결과 |
|----|------|-----------|------|
| AC-1 | 방 개설 후 BGM 자동 시작·무한반복 | 수동 (Operations 회귀) | 자동화 미가능 — Operations 트리거 권장 |
| AC-2 | 토글 OFF→ON 즉시 동작 | T2 (currentTime 보존) + 수동 | 자동 PASS · 수동 권장 |
| AC-3 | 효과음과 동시 재생 (덕킹 없음) | 코드 검토 (별도 Audio element) + 수동 | 코드 검토 PASS · 수동 권장 |
| AC-4 | volume === 0.15 | T1 단언 | PASS |
| AC-5 | Pause/GameEnded 시 BGM 유지 | 코드 검토 (입력에 게임 상태 미포함) + 수동 | 코드 검토 PASS · 수동 권장 |
| AC-6 | 새로고침 후 priming 재진입 | 수동 (Iteration 9 BFCache 풀 리로드와 호환) | Operations 회귀 권장 |
| AC-7 | 자산 누락 시 console.warn 만 | T3 (play rejection) | PASS |
| AC-8 | 플레이어 화면 BGM 재생 없음 | 코드 검토 (`isHost &&` 가드) | PASS |

---

## 3. 검증 결과

### 3.1 정적 / 단위 / 빌드

| 명령 | 결과 |
|------|------|
| `npx tsc --noEmit` | PASS (no output) |
| `npm test` | **75/75 PASS** (이전 71 + 신규 4 — useBgm.test.ts) |
| `npm run build` | 성공. JS gzip **66.01 KB** (baseline 65.71 → +0.30 KB) / CSS gzip 3.21 KB 동일 / index.html 0.36 KB |

### 3.2 Go 회귀

| 패키지 | go test -race | 비고 |
|--------|---------------|------|
| internal/announce | ok 1.386s | 변경 없음 |
| internal/game | ok 1.617s | 변경 없음 |
| internal/persistence | ok 2.076s | 변경 없음 |
| internal/session | ok 2.670s | 변경 없음 |
| internal/transport/http | ok 2.464s | 변경 없음 |
| internal/transport/ws | ok 3.925s | 변경 없음 |

`go build -o /tmp/mafia-game-iter10` 성공. 바이너리 크기 26,494,482 bytes (≈ 25.27 MB, baseline 17.97 MB 대비 +7.30 MB / +8.52 MB raw — `bgm.mp3` 8.44 MB 임베드 반영).

> 주: BGM 자산 자체는 8.44 MB 로 큼. 빌드 산출 바이너리 크기 증가는 사용자 호스팅 환경에 영향 가능 — 자산 압축 또는 외부 CDN 분리는 별도 iteration 검토 가능.

### 3.3 신규 단위 테스트 상세

| ID | 의도 | 검증 |
|----|------|------|
| useBgm T1 | enabled=true 마운트 시 loop/volume/play() | `lastAudio.loop === true`, `volume === 0.15`, `play` 1회 |
| useBgm T2 | enabled toggle false → pause(), currentTime 보존 | `pause()` 호출 + `currentTime` 12.34 그대로 |
| useBgm T3 | play() rejection 시 console.warn (graceful) | `console.warn` 호출, 첫 인자에 `[bgm] failed to play` 포함 |
| useBgm T4 | unmount 시 pause + listener 해제 | `pause()` 호출, `removeEventListener("error", ...)` 호출 |

---

## 4. 회귀 영향 분석

| 대상 | 영향 평가 | 근거 |
|------|-----------|------|
| `useAudioCueQueue` | 무영향 | 별도 Audio element, 분리된 hook, 동일 priming 게이트(hostToken) 공유만 |
| `VoiceToggle` | 무영향 | 컴포넌트 변경 없음, footer 옆에 `BgmToggle` 추가만 |
| `GameContext` | 무영향 | reducer/state 변경 없음 |
| `useWebSocket` (Iter9 BFCache) | 호환 | 새로고침 시 풀 리로드 → BGM 도 priming 다시 필요 (FR Q5=A 와 일치) |
| Go 서버 (U1~U4) | 무영향 | wire/카탈로그/HTTP 라우팅 변경 없음. `/audio/bgm.mp3` 는 기존 `audioHandler` 정적 라우팅으로 자동 서빙 |

---

## 5. NFR 영향

| 항목 | 평가 |
|------|------|
| 성능 | Audio element 1개 추가로 측정 불가 수준 |
| 빌드 | JS gzip +0.30 KB (NFR-1 ≤ +0.5 KB), 바이너리 +7.30~+8.52 MB (BGM 자산 자체 — 별도 항목) |
| 보안 | 변경 없음 (서버 라우팅 무변경, 사용자 입력 처리 무변경) |
| 호환성 | 모바일 Safari 등 autoplay 제한 환경에서 priming 후 재생 — `Boolean(hostToken)` 가드로 정책 준수 |

---

## 6. DoD 체크리스트

- [x] FR-1~FR-7 코드/테스트 구현 완료
- [x] AC-1~AC-8 검증 결과 기록 (자동 + 수동 권장 항목 분리)
- [x] `npm test` 71 → 75 PASS (신규 4)
- [x] `npm run build` 성공, JS gzip 66.01 KB (≤ 66.21 KB)
- [x] `go test ./... -race` 6 패키지 PASS
- [x] `go build` 성공
- [x] aidlc-state.md / audit.md 동기 갱신 (Step F)
- [ ] **사용자 최종 승인** (본 게이트)

---

## 7. 후속 권장 사항

- **Operations 회귀** (사용자 트리거): Chrome DevTools MCP 또는 실측 — 호스트 priming 후 BGM 무한재생 / 효과음과 동시 재생 청취감 / 새로고침 후 재priming / 모바일 Safari 자동재생 동작 / 토글 OFF→ON 위치 보존 / 플레이어 화면 BGM 부재
- **BGM 토글 영속화** (별도 iteration 검토): localStorage 에 `bgmOn` 저장 — 새로고침 후에도 마지막 선호 유지
- **BGM 자산 경량화** (별도 iteration 검토): 현재 `bgm.mp3` 8.44 MB → 비트레이트/길이 조정 또는 외부 CDN 분리 시 바이너리 크기 7.30 MB 회수 가능

---

## 8. 종료 메모

- 변경 미커밋 — 사용자의 명시적 commit 지시 후 진행 예정
- `web/node_modules` 은 메인 워크스페이스 심볼릭 링크 (Iteration 9 와 동일 패턴, `.gitignore` 대상)
- Iteration 10 변경 표면은 U5 단독, FR/NFR/AC 모두 자동 또는 코드 검토로 검증 가능한 항목까지 PASS, 잔여는 수동/Operations 회귀 항목

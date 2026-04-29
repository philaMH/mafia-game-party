# Iteration 6 — Build and Test Results

**Date**: 2026-04-29
**Scope**: U5 Web Frontend Noir UI 시각 재설계 (행동/프로토콜 변경 없음)
**Plan**: `aidlc-docs/construction/plans/iteration6-execution-plan.md` (사용자 승인 2026-04-29T08:05Z)
**Requirements**: `aidlc-docs/inception/requirements/iteration6-requirements.md` (사용자 승인 2026-04-29T07:55Z)

---

## 1. 산출물 요약

### 신규 파일
| 파일 | 크기 | 설명 |
|---|---|---|
| `web/src/styles/noir.css` | 8.5 KB | 디자인 토큰 + 32 개 유틸리티 클래스 |
| `web/public/assets/background.jpg` | 198 KB | `background.png` (1.9 MB) → JPEG q=75 1280×800 압축 (90% 감소) |

### 수정 파일 (27개)
- **PublicView 트리** (8): `PublicView.tsx`, `PhaseHeader.tsx`, `TimerBar.tsx`, `PauseBadge.tsx`, `SubtitleArea.tsx`, `PlayersGrid.tsx`, `HostControls.tsx`, `VoiceToggle.tsx`
- **PlayerView 트리** (12): `PlayerView.tsx`, `YourInfoCard.tsx`, `LobbyView.tsx`, `IntroView.tsx`, `DiscussionView.tsx`, `NightInputs.tsx`, `MafiaPicker.tsx`, `DoctorPicker.tsx`, `PolicePicker.tsx`, `VoteForm.tsx`, `EndScreen.tsx`, (`PhaseInputs.tsx` 변경 없음)
- **Components** (4): `ConnectionBadge.tsx`, `NicknameForm.tsx`, `PlayerPicker.tsx`, `ToastList.tsx`
- **Bootstrap** (3): `web/index.html`, `web/src/main.tsx`, `web/src/styles/global.css`

### 미수정 (변경 없음 — Go 백엔드 + 인프라)
- `internal/**` (6 패키지)
- `cmd/mafia-game/main.go`
- `web/src/App.tsx`, `web/src/types/wire.ts`, `web/src/context/**`, `web/src/hooks/**`, `web/vite.config.ts`, `web/tsconfig.json`
- 모든 테스트 파일 (`*.test.ts`/`*.test.tsx`) — 그대로 PASS

## 2. 빌드 결과

### `npm run typecheck`
```
> tsc --noEmit
```
**결과**: PASS (에러 없음)

### `npm test`
```
 ✓ src/hooks/useToken.test.ts (3 tests) 7ms
 ✓ src/hooks/useTTSQueue.test.ts (5 tests) 11ms
 ✓ src/context/reducer.test.ts (31 tests) 5ms
 ✓ src/components/NicknameForm.test.tsx (6 tests) 48ms

 Test Files  4 passed (4)
      Tests  45 passed (45)
   Duration  806ms
```
**결과**: 45/45 PASS — Iteration 5 와 동일.

### `npm run build` (vite + tsc)
```
✓ 65 modules transformed.
../cmd/mafia-game/web/dist/index.html                   0.59 kB │ gzip:  0.36 kB
../cmd/mafia-game/web/dist/assets/index-DlA7cNKj.css   11.47 kB │ gzip:  3.21 kB
../cmd/mafia-game/web/dist/assets/index-CFQEi_JL.js   203.06 kB │ gzip: 64.93 kB
../cmd/mafia-game/web/dist/assets/background.jpg      198.56 kB
✓ built in 348ms
```
**결과**: 성공. JS gzip 64.93 KB (Iteration 5 61.75 KB 대비 +3.18 KB) — NFR-I6-1 목표 80 KB 미만 ✓.

### `go build`
```
go build -o /tmp/mafia-game-iter6 ./cmd/mafia-game
ls -la /tmp/mafia-game-iter6 → 15,942,306 bytes (≈ 15.2 MB)
```
**결과**: 성공. 바이너리 크기 +0.7 MB (Iteration 5 ≈ 15 MB 대비, 임베드 자산/CSS 추가분).

### `go test ./... -count=1`
```
ok  	github.com/saltware/mafia-game/internal/announce         0.230s
ok  	github.com/saltware/mafia-game/internal/game             0.441s
ok  	github.com/saltware/mafia-game/internal/persistence      0.712s
ok  	github.com/saltware/mafia-game/internal/session          1.006s
ok  	github.com/saltware/mafia-game/internal/transport/http   1.085s
ok  	github.com/saltware/mafia-game/internal/transport/ws     2.859s
```
**결과**: 6/6 패키지 PASS — Go 코드 변경 없음, 회귀 0 건.

## 3. NFR 영향 분석

| NFR | 목표 | 실측 | 상태 |
|---|---|---|---|
| NFR-I6-1 빌드 사이즈 (JS gzip) | < 80 KB | 64.93 KB | ✅ |
| NFR-I6-1 배경 자산 | < 500 KB | 198 KB | ✅ |
| NFR-I6-2 vitest PASS | 45/45 | 45/45 | ✅ |
| NFR-I6-3 Go 6 패키지 PASS | 6/6 | 6/6 | ✅ |
| NFR-I6-4 폰트 fallback chain | 시스템 폰트 fallback | `Cinzel, "Noto Serif KR", serif` 등 명시 | ✅ |
| NFR-I6-5 색상 대비 | WCAG AA | paper(#e8d9b5) on ink(#0a0807) ≈ 13.5:1 (AAA) | ✅ |
| NFR-I6-6 데스크탑 + 모바일 | 1280px 우선 + 600px 단일 컬럼 | `@media (max-width: 600px)` 그리드 자동 재배치 | ✅ |

## 4. 회귀 영향 분석

| 영역 | 변경 | 결과 |
|---|---|---|
| WebSocket 프로토콜 | 변경 없음 | `transport/ws` 테스트 PASS |
| 게임 엔진 | 변경 없음 | `game` 테스트 PASS |
| 세션/지속성/방송 | 변경 없음 | `session`/`persistence`/`announce` 테스트 PASS |
| HTTP bootstrap | 변경 없음 | `transport/http` 테스트 PASS |
| 상태 reducer | 변경 없음 | `reducer.test.ts` 31 PASS |
| TTS/토큰 hooks | 변경 없음 | `useTTSQueue.test.ts` 5 + `useToken.test.ts` 3 PASS |
| `NicknameForm` 검증 | 텍스트 "입장" 보존, "♠" 는 `aria-hidden` 으로 분리 | `NicknameForm.test.tsx` 6 PASS |

## 5. 디자인 적용 추적성 (Requirements §7 vs 실측)

| Mockup | 적용 화면 | 핵심 노이르 요소 |
|---|---|---|
| Splash · Main Menu | Lobby/PlayerView 헤더에 컨셉만 | mafia-title.stone + mafia-sub |
| Player Lobby | `LobbyView` | slot 그리드, 5/10 카운터, 모더레이터 인용 |
| Role Reveal | `YourInfoCard` | role-card 5:7, diamond-seal, PASSPHRASE mono gold |
| Night | `NightInputs` + 3 picker | vote-tile + target border, 동료 마피아 분리 |
| Day Discussion | `DiscussionView` + `PublicView` | TimerBar mono gold, SubtitleArea serif italic |
| Vote | `VoteForm` + `PlayerPicker` | vote-tile selected=red, 기권 btn-noir.primary |
| End | `EndScreen` | mafia-title.stone WIN + dossier (마피아 red) |
| Host Lobby | `PublicView` host setup | gold-frame 옵션 패널 + btn-noir.primary "방 개설" |
| Host Night | `PublicView` NIGHT + `HostControls` | omniscient PlayersGrid + Pause/Resume + force-end |
| Host Day | `PublicView` DAY + `SubtitleArea` | LIVE 배지 동적 + spotlight |

## 6. DoD 체크리스트 (Requirements §9)

- [x] `web/src/styles/noir.css` 작성 + 토큰/유틸리티 정의 (32 클래스)
- [x] 13 view 컴포넌트 + 4 component 시각 적용
- [x] `background.jpg` 임베드, 198 KB (< 500 KB)
- [x] `npm test` 45 PASS 유지
- [x] `npm run build` 성공, gzip JS 64.93 KB (< 80 KB)
- [x] `go build -o /tmp/mafia-game-iter6 ./cmd/mafia-game` 성공 (15.2 MB)
- [x] `go test ./...` 6 패키지 PASS
- [x] 본 iteration6-test-results.md 작성 완료
- [ ] aidlc-state.md Iteration 6 섹션 갱신 (다음 단계)

## 7. 가정 사항 적용 결과 (Requirements §8)

- **채팅 입력창**: 디자인 mockup 의 chat 모듈은 wire 미지원이라 미구현. PlayerView 의 DiscussionView 는 안내 메시지로 대체.
- **강퇴 버튼**: HostControls 에 미포함 (wire 미지원).
- **+30초/−30초**: 미포함 (wire 미지원).
- **음성 안내 OFF**: VoiceToggle 로 기존 host:toggle-voice 매핑 유지.

## 8. 후속 권장 사항 (Operations 단계)

- **Chrome DevTools MCP 회귀**: 다음 시나리오 골든패스 수동 검증 권장 (호스트 + 6 player 컨텍스트):
  - 노이르 배경 + mafia-title 가시성 확인 (특히 모바일 < 600px 폭)
  - YourInfoCard role-card 5:7 비율 + DiamondSeal 렌더 확인
  - vote-tile target border (red) 강조 확인
  - PauseBadge gold pulse 애니메이션 확인
  - EndScreen final dossier 마피아 red 보더 확인
- **폰트 차단 환경**: `https://fonts.googleapis.com` 차단 시에도 Noto Serif KR 시스템 폰트로 한국어 가독성 유지되는지 확인.
- **이미지 추가 압축 검토**: 현재 198 KB JPEG q=75. 필요 시 q=85 (~250 KB) 로 화질 개선 또는 cwebp WebP 변환으로 추가 감소 가능.

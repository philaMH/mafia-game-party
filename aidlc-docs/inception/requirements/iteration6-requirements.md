# Iteration 6 Requirements — Noir UI 시각 재설계

**Status**: Draft — 사용자 승인 대기
**Date**: 2026-04-29
**Type**: UI/UX 시각 개편 (Brownfield, 행동/프로토콜 변경 없음)
**Scope**: U5 Web Frontend 단독 (U1~U4 변경 없음)

---

## 1. 배경 (Background)

Iteration 5까지 PoC 수준의 다크 테마(neutral gray + accent blue)로 화면이 구성되어 있다. Claude Design 핸드오프 패키지 (`https://api.anthropic.com/v1/design/h/LcS00UIyCTrT5bNQA18c0A`) 가 1929 New York · 1920~40s film noir 컨셉의 디자인 시스템과 11개 mockup 화면(8 entry/player + 3 host)을 제공한다. Iteration 6은 이 비주얼 언어를 기존 React SPA에 반영한다.

## 2. 의도 (Intent)

**기능 변경 없음** — WebSocket 프로토콜, 게임 엔진, 세션/지속성/방송, HTTP bootstrap, 라우팅 모두 그대로 둔다. 본 Iteration의 산출물은:

- 디자인 토큰 (CSS 변수)
- 재사용 가능한 노이르 유틸리티 클래스 (`web/src/styles/noir.css`)
- 11개 mockup의 시각 언어를 기존 13개 view 컴포넌트 + 4개 component에 반영
- 1개 압축 배경 자산 (`background.png` → WebP/JPEG 압축)
- 기존 45개 테스트 전부 PASS 유지

## 3. 사용자 결정 사항 (Q&A 결과)

| ID | 질문 | 답변 |
|---|---|---|
| Q1 | 자산 처리 | **D** — `background.png` 압축 후 임베드, `host.png`/`room.png` 는 CSS 그라디언트 대체 |
| Q2 | Splash/Main Menu 라우트 | **B** — 신규 라우트 없음, 시각 컨셉만 Lobby/Intro 등에 반영 |
| Q3 | 반응형 범위 | **A** — 데스크탑 1280px 우선, 플레이어 모바일은 단일 컬럼 자동 |
| Q4 | 디자인 시스템 도입 | **A** — 단일 `noir.css` 추가 + 인라인 스타일 점진 교체 |

## 4. Functional Requirements

**FR-I6-1 디자인 토큰 정의**: `web/src/styles/noir.css`(또는 `global.css` 확장)에 다음 토큰을 CSS 커스텀 프로퍼티로 정의한다:
- 색상: `--ink`, `--ink-2`, `--char`, `--leather`, `--oxblood`, `--paper`, `--paper-2`, `--paper-dim`, `--gold`, `--gold-2`, `--gold-dim`, `--gold-glow`, `--red`, `--red-deep`, `--red-glow`, `--alive`, `--dead`, `--warn`
- 타이포: `--font-display` (Cinzel), `--font-serif` (Crimson Text), `--font-kr` (Noto Serif KR), `--font-mono` (JetBrains Mono)
- 기존 `global.css` 변수(`--bg`, `--fg`, `--accent` 등)는 새 토큰으로 매핑하여 호환성 유지

**FR-I6-2 노이르 유틸리티 클래스**: `noir.css`에 다음 클래스를 정의한다 (디자인 핸드오프 `styles.css` 기반):
- 표면: `.noir`, `.noir-bg`, `.noir-bg.dim`, `.noir-bg.deep`, `.noir-bg.bloody`, `.noir-bg.crop-table`, `.noir-content`, `.scrim`, `.gold-frame`, `.gold-corners`, `.center-card`, `.wood`
- 타이포: `.mafia-title`, `.mafia-title.stone`, `.mafia-title.lg/.sm`, `.mafia-sub`, `.eyebrow` (`.red`/`.dim`), `.h-display`, `.serif`, `.mono`, `.divider-gold`
- 인터랙티브: `.btn-noir` (`.primary`/`.ghost`/`.lg`/`.sm`), `.slot` (`.empty`/`.dead`), `.vote-tile` (`.target`/`.dead`), `.avatar` (`.host`/`.dead`/`.target`/`.lg`/`.sm`), `.chat`
- 상태: `.pip` (`.dead`/`.away`/`.host`), `.tag` (`.red`/`.dim`), `.progress` (`.red`)

**FR-I6-3 폰트 로딩**: `web/index.html` 또는 `noir.css` `@import` 로 Google Fonts 4종 (Cinzel · Crimson Text · JetBrains Mono · Noto Serif KR) 을 로드한다. 오프라인 / 폐쇄 LAN 환경에서도 시스템 폰트 fallback 으로 동작해야 한다 (CSS `font-family` chain 의 마지막에 `serif` / `monospace` / `sans-serif`).

**FR-I6-4 배경 자산**: `background.png` 를 다음 절차로 처리한다:
1. macOS `sips` 또는 `cwebp` 로 1280×800 이내, q≈75 WebP/JPEG 변환
2. 목표 크기 ~150~400 KB (현재 1.9 MB → 80% 이상 감소)
3. `web/public/assets/background.webp` (또는 `.jpg`) 에 배치 → vite build 시 `cmd/mafia-game/web/dist/assets/` 에 자동 복사
4. WebP 미지원 브라우저 fallback 은 같은 dim한 그라디언트 fallback CSS 로 처리

**FR-I6-5 PublicView 시각 재설계**:
- 호스트 클레임/방 개설 화면: noir 배경 + mafia-title (작은 크기) + gold-frame settings 패널 + btn-noir.primary "방 개설"
- 호스트 점유 차단 화면: 중앙 center-card + eyebrow.red 라벨
- 게임 진행 화면: PhaseHeader 를 mafia-title 스타일로 / TimerBar 를 mono large + gold accent / PlayersGrid 를 vote-tile/avatar 그리드로 / SubtitleArea 를 serif italic + severity 색상 / HostControls 를 btn-noir 시리즈로
- 좌상단에 `♣ HOST CONSOLE · 진행자 화면` 태그 배지 (디자인 mockup 의 HostBadge)

**FR-I6-6 PlayerView 시각 재설계**:
- 닉네임 입력 화면: noir 배경 dim + mafia-title sm + center-card + btn-noir.primary "입장"
- 재접속 안내: center-card + 시네마틱 인용 톤
- LobbyView: 호스트는 별도 모더레이터 배너로 분리 (이미 wire 에서 isHost 로 식별 가능 — players 리스트에 host 포함 여부는 backend 동작 그대로 유지하되 클라이언트 측 렌더링에서 호스트 ID 만 분리). slot 컴포넌트로 2 컬럼 그리드 (모바일 1 컬럼).
- IntroView: 발언자 spotlight 카드 + serif italic 안내문 + btn-noir "내 자기소개 종료"
- NightInputs/Mafia/Doctor/PolicePicker: vote-tile/avatar 그리드. 마피아 선택 시 target border 강조. 동료 마피아는 `●` 표식.
- DiscussionView: gold-frame 채팅 박스 (현재 채팅 미구현 — 디자인의 chat UI 는 read-only 안내문으로 표현)
- VoteForm: vote-tile + 빨간 vote bar + tag/eyebrow.red
- EndScreen: mafia-title.stone "MAFIA WINS"/"CITIZENS WIN" + gold-corners final dossier (마피아는 red border)
- YourInfoCard: role-card 형식 (5:7 비율, 상단 DiamondSeal, 하단 PASSPHRASE 키워드 mono gold)

**FR-I6-7 컴포넌트 시각 재설계**:
- ConnectionBadge: tag 스타일 (gold/red/warn)
- NicknameForm: noir input + btn-noir.primary
- PlayerPicker: vote-tile 미니 (active=target border, disabled=opacity)
- ToastList: oxblood/red border + serif italic message

**FR-I6-8 PauseBadge 갱신**: 현재 yellow (`rgba(255, 200, 0, 0.85)`) 배너를 noir 톤으로 — 검정 + gold border + paused gold pulse 애니메이션. 메시지 그대로 유지: "⏸ 진행이 일시정지되었습니다".

## 5. Non-Functional Requirements

**NFR-I6-1 빌드 사이즈**: gzip 후 JS 번들 < 80 KB (현재 61.75 KB → 신규 noir.css 약 8 KB 추가 예상). 배경 이미지 fetch 는 별도. 총 페이지 first-load `< 500 KB` 목표.

**NFR-I6-2 테스트 호환성**: 기존 45개 vitest 테스트 전부 PASS. 인라인 스타일에 의존하는 테스트가 없음을 확인 — 클래스명 / 텍스트 / role 만 단언.

**NFR-I6-3 백엔드 영향 없음**: `go test ./...` 의 6 패키지 결과 변동 없음 (go 코드 변경 없음).

**NFR-I6-4 폰트 fallback**: 오프라인 LAN 환경에서 Google Fonts 차단 시에도 화면이 깨지지 않도록 시스템 fallback chain 보장. `Cinzel, "Noto Serif KR", serif` 등.

**NFR-I6-5 접근성**: 색상 대비 WCAG AA 통과 (paper #e8d9b5 on ink #0a0807 contrast ratio ≈ 13.5:1 ✓). `aria-live`, `role` 속성 모두 보존.

**NFR-I6-6 데스크탑 우선 + 플레이어 모바일 폴백**: PublicView 는 1280×800 데스크탑 가정. PlayerView 는 기존 maxWidth 32rem 단일 컬럼 유지 (모바일 자동 호환). 플레이어 화면 폭 < 600px 에서는 vote-tile 그리드 1~2 컬럼.

## 6. 산출물 및 변경 파일

### 신규
- `web/src/styles/noir.css` — 디자인 토큰 + 유틸리티 클래스 (~12~15 KB)
- `web/public/assets/background.webp` (또는 `.jpg`) — 압축 배경 (~150~400 KB)

### 수정 (시각 재설계, 행동 보존)
- `web/index.html` — Google Fonts preconnect 추가 (선택)
- `web/src/main.tsx` — `import "./styles/noir.css"` 추가
- `web/src/styles/global.css` — 기존 변수 유지/노이르 토큰으로 매핑
- `web/src/App.tsx` — 변경 없음 (라우팅 유지)
- `web/src/views/PublicView/PublicView.tsx`, `HostControls.tsx`, `PauseBadge.tsx`, `PhaseHeader.tsx`, `PlayersGrid.tsx`, `SubtitleArea.tsx`, `TimerBar.tsx`, `VoiceToggle.tsx`, `PublicView.module.css`
- `web/src/views/PlayerView/PlayerView.tsx`, `LobbyView.tsx`, `IntroView.tsx`, `DiscussionView.tsx`, `NightInputs.tsx`, `MafiaPicker.tsx`, `DoctorPicker.tsx`, `PolicePicker.tsx`, `VoteForm.tsx`, `EndScreen.tsx`, `YourInfoCard.tsx`, `PlayerView.module.css`
- `web/src/components/ConnectionBadge.tsx`, `NicknameForm.tsx`, `PlayerPicker.tsx`, `ToastList.tsx`

### 변경 없음
- `internal/**` (Go 백엔드 전체)
- `cmd/mafia-game/main.go`, `web/dist` (빌드시 재생성)
- `web/src/types/wire.ts`, `web/src/context/{GameContext,reducer}.tsx`, `web/src/hooks/**`

## 7. 추적성 매트릭스

| 디자인 mockup | 현재 화면 | 적용 방식 |
|---|---|---|
| Splash · 타이틀 | (해당 없음 — Q2=B) | 시각 reference만 채택 (Lobby 헤더 mafia-title) |
| Main Menu | (해당 없음 — Q2=B) | 시각 reference만 채택 (host setup 화면) |
| Player Lobby | `PlayerView` Lobby 분기 + `LobbyView` | slot 그리드 + ROOM ID + 모더레이터 배너 |
| Role Reveal | `YourInfoCard` | role-card (5:7 비율, DiamondSeal, PASSPHRASE) |
| Night | `NightInputs` + Mafia/Doctor/PolicePicker | vote-tile 그리드 + target border + 마피아 동료 표식 |
| Day Discussion | `DiscussionView` + `PlayersGrid` | 사망 알림 배너(red gold-corners) + 발언자 spotlight + 채팅 placeholder |
| Vote | `VoteForm` + `PlayerPicker` | vote-tile + red bar 표 시각 |
| End | `EndScreen` + `PlayersGrid(reveal)` | mafia-title.stone WIN + gold-corners dossier |
| Host Lobby | `PublicView` (호스트 LOBBY) + `HostControls` | ROOM ID 큰 mono + 옵션 패널 + HostBadge |
| Host Night | `PublicView` (호스트 NIGHT) + `PlayersGrid` | omniscient view (역할 컬러 표시) + 컨트롤 패널 |
| Host Day | `PublicView` (호스트 DAY) + `HostControls` + `SubtitleArea` | LIVE 송출 배지 + 발언자 spotlight + 시간 제어 |

## 8. 가정 (Assumptions)

- 디자인 mockup 의 "channel chat" 입력창은 시각 구조만 재현 — 실제 채팅 메시지 송수신 기능은 wire 프로토콜에 없으므로 placeholder 또는 read-only 안내문으로 처리.
- "강퇴(kick)" 버튼은 wire 에 해당 메시지가 없으므로 본 Iteration 에서는 비활성 또는 시각만 노출.
- "음성 안내 OFF" 토글은 기존 `host:toggle-voice` 와 매핑.
- "+30초/−30초" 시간 조정은 wire 미지원 — 시각만 노출 또는 생략.

## 9. 완료 기준 (Definition of Done)

- [ ] `web/src/styles/noir.css` 작성 + 토큰/유틸리티 정의
- [ ] 13개 view 컴포넌트 + 4개 component 시각 적용
- [ ] `background.webp` (또는 `.jpg`) 임베드, 크기 < 500 KB
- [ ] `npm test` 45 PASS 유지
- [ ] `npm run build` 성공, gzip JS < 80 KB
- [ ] `go build -o /tmp/mafia-game-iter6 ./cmd/mafia-game` 성공
- [ ] `go test ./...` 6 패키지 PASS (변경 없으나 회귀 확인)
- [ ] iteration6-test-results.md 작성
- [ ] aidlc-state.md Iteration 6 섹션 갱신

---

**다음 단계**: 본 Requirements 문서 사용자 승인 후 Workflow Planning (`construction/plans/iteration6-execution-plan.md`) 진행.

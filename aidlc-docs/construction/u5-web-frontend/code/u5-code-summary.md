# U5 Web Frontend — Code Summary

**작성일**: 2026-04-26
**대상 단위**: U5 (`web/*`)
**plan**: `aidlc-docs/construction/plans/u5-web-frontend-code-generation-plan.md`

---

## 1. 빌드 / 검증 결과

| 게이트 | 결과 |
|---|---|
| `npm install` | ✅ 의존 11종 + transitive 설치 |
| `npm run typecheck` (tsc --noEmit) | ✅ 0 error |
| `npm run lint` (eslint) | ✅ 0 error |
| `npm test` (vitest) | ✅ **32/32 테스트 통과** |
| `npm run build` (vite) | ✅ gzip **60.14 KB** ≪ 500 KB (NFR-U5-P4) |
| `npm run test:coverage` | ✅ 핵심 모듈 합산 **78.72%** ≥ 70% (NFR-U5-M3) |
| · NicknameForm.tsx | 100% |
| · reducer.ts | 91.13% |
| · useTTSQueue.ts | 89.9% |
| · useToken.ts | 91.3% |
| `go build ./cmd/mafia-game` | ✅ Mach-O 64-bit arm64 단일 바이너리 (15.6 MB, Vite dist 동봉) |

> 커버리지 대상은 NFR-U5-M3 명시 핵심 모듈(reducer / hooks / NicknameForm validation). UI 프레젠테이션 컴포넌트는 수동 + 향후 통합 테스트로 검증 (NFR Requirements 비-요구사항).

---

## 2. 산출 파일 인벤토리 (총 49 파일)

### 2.1 설정 (7)
| 파일 | 책임 |
|---|---|
| `web/package.json` | 의존 11종 + scripts (dev/build/lint/typecheck/test/test:coverage) |
| `web/tsconfig.json` | strict + noUncheckedIndexedAccess + DOM lib |
| `web/vite.config.ts` | outDir = ../cmd/mafia-game/web/dist + dev proxy |
| `web/vitest.config.ts` | jsdom + setup + 핵심 모듈 coverage |
| `web/.eslintrc.cjs` | TS + react-hooks 룰 |
| `web/index.html` | root mount + meta viewport (모바일) |
| `web/.gitignore` | node_modules + coverage |

### 2.2 도메인 + 스타일 (2)
| 파일 | 책임 |
|---|---|
| `src/types/wire.ts` | IncomingMsg 6종 + EventPayload 15 kind + OutgoingMsg 14종 + State/YourInfo + defaultOptions/teamOf 헬퍼 |
| `src/styles/global.css` | CSS 변수(색상/폰트) + reset + 폼 기본 스타일 |

### 2.3 Hooks (3)
| 파일 | 책임 | LC |
|---|---|---|
| `src/hooks/useToken.ts` | localStorage 격리 (TokenIO 인터페이스) | LC-U5-6 |
| `src/hooks/useWebSocket.ts` | 자동 재연결 + 지수 백오프 (1/2/4/8/16s) + 토큰 자동 resume | LC-U5-4 |
| `src/hooks/useTTSQueue.ts` | enqueue/enqueueUrgent/cancelAll + ko-KR 자동 + voiceschanged | LC-U5-5 |

### 2.4 Context + Reducer (2)
| 파일 | 책임 | LC |
|---|---|---|
| `src/context/reducer.ts` | gameReducer + applyEvent (15 kind 매핑) + GameAction | LC-U5-3 |
| `src/context/GameContext.tsx` | GameProvider + useGameContext + URGENT_KINDS 분류 + 토큰 자동 정리 | LC-U5-2 |

### 2.5 공통 컴포넌트 (4)
| 파일 | 책임 |
|---|---|
| `src/components/ConnectionBadge.tsx` | 4종 status 인디케이터 |
| `src/components/NicknameForm.tsx` | validateName (1~20자, 한글/영문/숫자) + 폼 |
| `src/components/PlayerPicker.tsx` | memo + 안정 key 라디오 그룹 |
| `src/components/ToastList.tsx` | 5초 자동 dismiss |

### 2.6 PublicView (8)
| 파일 | 책임 |
|---|---|
| `src/views/PublicView/PublicView.tsx` | 호스트 닉네임/state 조건부 렌더 + subscribe-public |
| `src/views/PublicView/PhaseHeader.tsx` | 단계 헤더 (≥ 48px, 7 phase) |
| `src/views/PublicView/TimerBar.tsx` | 1초 setInterval 카운트다운 |
| `src/views/PublicView/PlayersGrid.tsx` | auto-fit 그리드 + 사망 표시 + END reveal |
| `src/views/PublicView/SubtitleArea.tsx` | data-severity 색상 매핑 |
| `src/views/PublicView/HostControls.tsx` | 단계별 버튼 (start/advance/end-discussion/end-night/force-end) |
| `src/views/PublicView/VoiceToggle.tsx` | TTS available/disabled |
| `src/views/PublicView/PublicView.module.css` | severity pulse 애니메이션 |

### 2.7 PlayerView (13)
| 파일 | 책임 |
|---|---|
| `src/views/PlayerView/PlayerView.tsx` | 닉네임 폼 OR YourInfoCard + PhaseInputs |
| `src/views/PlayerView/YourInfoCard.tsx` | 자기 역할/키워드/마피아 동료 표시 |
| `src/views/PlayerView/PhaseInputs.tsx` | Phase 분기 (LOBBY/INTRO/NIGHT/DAY/VOTE/RECOUNT/END) |
| `src/views/PlayerView/LobbyView.tsx` | 입장 대기 |
| `src/views/PlayerView/IntroView.tsx` | 자기 차례 강조 |
| `src/views/PlayerView/NightInputs.tsx` | 역할 분기 (MAFIA/DOCTOR/POLICE/CITIZEN) |
| `src/views/PlayerView/MafiaPicker.tsx` | 대표자 권한 분기 + pendingMafiaTarget 표시 |
| `src/views/PlayerView/DoctorPicker.tsx` | doctorSelfHealAllowed 분기 |
| `src/views/PlayerView/PolicePicker.tsx` | policeCheckedThisNight 분기 + lastResult 표시 |
| `src/views/PlayerView/DiscussionView.tsx` | 토론 시간 (TimerBar 재사용) |
| `src/views/PlayerView/VoteForm.tsx` | VOTE/RECOUNT 공유 + 사망자 차단 |
| `src/views/PlayerView/EndScreen.tsx` | 결과 + 전체 reveal |
| `src/views/PlayerView/PlayerView.module.css` | alive/dead 클래스 |

### 2.8 진입점 (2)
| 파일 | 책임 |
|---|---|
| `src/App.tsx` | BrowserRouter + GameProvider + Routes (/, /public, /play) |
| `src/main.tsx` | createRoot + StrictMode |

### 2.9 단위 테스트 (5)
| 파일 | 책임 |
|---|---|
| `src/tests/setup.ts` | FakeSpeechSynthesis + FakeUtterance + WebSocket 스텁 + localStorage clear |
| `src/context/reducer.test.ts` | gameReducer 18 테스트 (모든 event kind + announce + error + logout) |
| `src/hooks/useToken.test.ts` | localStorage 격리 3 테스트 |
| `src/hooks/useTTSQueue.test.ts` | enqueue/Urgent/disable/unavailable 5 테스트 |
| `src/components/NicknameForm.test.tsx` | validateName 4종 + 폼 동작 2종 (총 6) |

### 2.10 문서 (2)
- `aidlc-docs/construction/u5-web-frontend/code/u5-code-summary.md` (본 파일)
- `aidlc-docs/construction/u5-web-frontend/code/u5-public-api.md`

---

## 3. 스토리/요구사항 ↔ 구현 매핑

| 요구사항 | 구현 위치 |
|---|---|
| FR-1.2 (재연결) | `useWebSocket.ts` 자동 resume + `reducer.ts` joined 처리 |
| FR-2.3 (역할 비공개) | `reducer.ts` applyEvent (RoleRevealedToPlayer) + `MafiaPicker` cohort 필터 |
| FR-3.2 (키워드 비공개) | `YourInfoCard` 자기만 표시 |
| FR-4.3 (마피아 대표자) | `MafiaPicker` `mafiaRepresentativeId === me` 분기 |
| FR-4.4 (의사 자가 보호) | `DoctorPicker` `doctorSelfHealAllowed` 분기 |
| FR-8.1 (Web Speech) | `useTTSQueue.ts` |
| FR-8.2 (PublicView 한정) | TTS effect는 `GameProvider`에 1회만 (PlayerView는 announce를 받지 않음 — 백엔드 필터) |
| FR-8.4 (안내 풍부) | `SubtitleArea` + `useTTSQueue` |
| FR-8.5 (ON/OFF 토글) | `VoiceToggle` + `host:toggle-voice` 송신 |
| FR-8.6 (큐잉/인터럽션) | `useTTSQueue` enqueue vs enqueueUrgent + URGENT_KINDS 분류 |
| FR-8.7 (자막 폴백) | `PublicView` `!ttsAvailable` 토스트 + 자막 항상 표시 |
| NFR-U5-S1 (토큰 미노출) | `useToken` 격리, GameProvider에서 `UNKNOWN_PLAYER_ERROR` 시 자동 clear |
| NFR-U5-U2 (자막 ≥ 32px) | `SubtitleArea` fontSize 2rem (32px) |
| NFR-U5-P4 (gzip < 500 KB) | 빌드 산출 60.14 KB |

---

## 4. 핵심 설계 결정 (재확인)

| 결정 | 위치 |
|---|---|
| 단일 GameContextValue (P-U5-1) | `GameContext.tsx` |
| direct dispatch + React 18 batching (P-U5-2) | `useWebSocket.ts` onmessage |
| React.memo + 안정 key (P-U5-3) | `PlayerPicker` Item memo |
| voiceschanged + 즉시 호출 (P-U5-4) | `useTTSQueue.ts` 음성 로드 effect |
| useToken 훅 격리 (P-U5-5) | `useToken.ts` |
| CSS Modules + data-severity (P-U5-6) | `PublicView.module.css` |
| Vitest + jsdom + SS mock (P-U5-7) | `vitest.config.ts` + `tests/setup.ts` |

---

## 5. 알려진 제한 / 후속 작업

| 항목 | 상태 |
|---|---|
| UI 컴포넌트 단위 테스트 | 핵심 폼만 작성. PlayerView/PublicView 통합 테스트는 NFR 비-요구사항 |
| Service Worker / PWA | 비-요구사항 |
| Web Speech 음성 사전 정밀 선택 (이름 일치 등) | `lang.startsWith("ko")` 자동 선택만 |
| WCAG AA 명시 | A1 자막 폴백 + 키보드 네비만 보장 (NFR-U5-A4) |

---

## 6. 변경된 모듈 메타데이터

`web/package.json`: 11종 직접 의존 (런타임 3 + 도구 5 + 테스트 4 + 린트 4).

> Go 모듈은 변경 없음. 누계 Go 직접 의존: `modernc.org/sqlite` + `gorilla/websocket` 2개.

---

## 7. 통합 빌드 명령

```bash
# 1) Frontend 빌드 → cmd/mafia-game/web/dist
cd web && npm install && npm run build

# 2) Go 단일 바이너리 (Vite dist 동봉)
cd .. && go build ./cmd/mafia-game

# 3) 실행
./mafia-game --port 8080
```

> 출력 예시:
> ```
> mafia-game listening on:
>   http://192.168.1.42:8080
> ```

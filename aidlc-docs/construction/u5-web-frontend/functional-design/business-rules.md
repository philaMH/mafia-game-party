# Business Rules — U5 Web Frontend

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `domain-entities.md`, `business-logic-model.md`

---

## 1. 공통 규칙 (BR-U5-COMMON)

| ID | 규칙 |
|---|---|
| BR-U5-COMMON-1 | 모든 비즈니스 결정은 백엔드(U1/U2)가 보유. 클라이언트는 화면 표시 + 입력만 |
| BR-U5-COMMON-2 | wire 메시지는 `web/src/types/wire.ts` 단일 진실 소스. 백엔드 protocol.go 변경 시 같이 수정 (Q-FD-U5-13=A) |
| BR-U5-COMMON-3 | 알 수 없는 wire `type` 또는 `event.kind` 수신 시 `console.warn` 후 무시. 페이지 깨짐 방지 |
| BR-U5-COMMON-4 | 에러 토스트는 자동 사라짐 (5초) — `errors[]`에서 자동 ack |
| BR-U5-COMMON-5 | 한국어 텍스트만 사용 (FR-8.3). i18n 라이브러리 없음 (Q-FD-U5-14=A) |

---

## 2. 라우팅 규칙 (BR-U5-ROUTE, Q-FD-U5-1=A)

| ID | 규칙 |
|---|---|
| BR-U5-ROUTE-1 | `/` → `/play`로 redirect (사용자가 host PC면 `/public` 수동 진입) |
| BR-U5-ROUTE-2 | `/public`은 PUBLIC 클라이언트 (TTS + 호스트 컨트롤) |
| BR-U5-ROUTE-3 | `/play`는 PLAYER 클라이언트 (입력 폼) |
| BR-U5-ROUTE-4 | 정의되지 않은 경로는 `<Navigate to="/play" />`로 fallback |

---

## 3. WebSocket 연결 규칙 (BR-U5-WS, Q-FD-U5-4=A)

| ID | 규칙 |
|---|---|
| BR-U5-WS-1 | WebSocket URL은 `ws://<window.location.host>/ws` (TLS 미지원, NFR-4 LAN 한정) |
| BR-U5-WS-2 | 연결 끊김 시 자동 재연결 — 지수 백오프 1s → 2s → 4s → 8s → 16s (상한) |
| BR-U5-WS-3 | 재연결 직후 `localStorage.mafia.token`이 있으면 `resume` 메시지 자동 송신. 없으면 사용자가 닉네임 재입력 |
| BR-U5-WS-4 | 페이지 unmount(컴포넌트 정리) 시 ws.close() — 재연결 시도 차단 |
| BR-U5-WS-5 | 재연결 진행 중 `status="reconnecting"` 표시 — 사용자가 인지 가능 |

---

## 4. 토큰 / 식별 규칙 (BR-U5-TOKEN, Q-FD-U5-3=A)

| ID | 규칙 |
|---|---|
| BR-U5-TOKEN-1 | `joined` 메시지 수신 시 `localStorage.setItem("mafia.token", token)` |
| BR-U5-TOKEN-2 | 페이지 로드 시 토큰이 있으면 자동 resume. 없으면 NicknameForm 표시 |
| BR-U5-TOKEN-3 | `error.code === "UNKNOWN_PLAYER_ERROR"` 수신 (resume 실패) → 토큰 무효화: `localStorage.removeItem` + NicknameForm 재표시 |
| BR-U5-TOKEN-4 | 게임 종료 후 (`GameEnded` 수신) 토큰을 즉시 삭제하지 않음 — 다음 LOBBY까지 유지. 호스트가 새 게임 시작 시 백엔드가 새 토큰 발급 |
| BR-U5-TOKEN-5 | 토큰을 화면에 표시 금지 — DEBUG 로그 포함 (NFR-4) |

---

## 5. TTS 규칙 (BR-U5-TTS, FR-8.1~8.7, Q-FD-U5-5/6/7=A)

| ID | 규칙 |
|---|---|
| BR-U5-TTS-1 | TTS는 `/public` PublicView 한정 (FR-8.2). PlayerView에서 발화 0 |
| BR-U5-TTS-2 | `window.speechSynthesis === undefined`이면 `ttsAvailable=false` + 1회 토스트 후 자막만 (FR-8.7) |
| BR-U5-TTS-3 | 음성 선택: `voices.find(v => v.lang.startsWith("ko"))`. 없으면 default voice |
| BR-U5-TTS-4 | 기본 발화 속성: `lang=ko-KR`, `pitch=0.9`, `rate=0.95` (근엄 톤). severity별 미세 조정 (EMPHASIS pitch 0.85, WARN rate 1.0) |
| BR-U5-TTS-5 | urgent 분류: PhaseChanged / Eliminated / DeathAnnounced / GameEnded — 큐 클리어 + 즉시 발화 (인터럽트, FR-8.6) |
| BR-U5-TTS-6 | 일반 announce: 큐에 적층 후 순차 발화 |
| BR-U5-TTS-7 | `host:toggle-voice` 또는 `VoiceToggled` 이벤트로 ON/OFF — OFF면 `speechSynthesis.cancel()` 즉시 + 큐 비움 |
| BR-U5-TTS-8 | 페이지 unload 시 `speechSynthesis.cancel()` |
| BR-U5-TTS-9 | TTS 발화 중에도 자막은 항상 표시 — 음성 청취 불가능자 접근성 |

---

## 6. PublicView 규칙 (BR-U5-PUBLIC)

| ID | 규칙 |
|---|---|
| BR-U5-PUBLIC-1 | 첫 접속 시 NicknameForm으로 호스트 닉네임 입력 → `host:create-session` |
| BR-U5-PUBLIC-2 | `joined.isHost === true`일 때만 HostControls 노출 (Q-FD-U5-8=A) |
| BR-U5-PUBLIC-3 | 큰 글씨 강조 (NFR-3 가독성) — 자막 폰트 ≥ 32px, 단계 헤더 ≥ 48px |
| BR-U5-PUBLIC-4 | 사망자 표시는 X 표식 + 회색 처리 |
| BR-U5-PUBLIC-5 | 살아있는 플레이어들의 이름과 상태(ALIVE/DEAD)만 표시. Role/Keyword 절대 표시 안 함 (PhaseEnd reveal 전까지) |
| BR-U5-PUBLIC-6 | TimerBar는 Phase에 따라 `state.deadline` 카운트다운 |

---

## 7. PlayerView 규칙 (BR-U5-PLAYER, Q-FD-U5-9/10=A)

| ID | 규칙 |
|---|---|
| BR-U5-PLAYER-1 | 미식별 상태(playerId 없음)면 NicknameForm만 표시 |
| BR-U5-PLAYER-2 | 자기 Role/Keyword/Team을 `<YourInfo>`에 표시. 마피아면 `MafiaCohort`도 표시 |
| BR-U5-PLAYER-3 | Phase 분기는 단일 `<PlayerView>` 내에서. 별도 라우트 없음 |
| BR-U5-PLAYER-4 | 사망 플레이어는 모든 NIGHT/VOTE 입력 비활성화 + "당신은 사망했습니다" 표시. PUBLIC 메시지는 계속 수신 (Q-FD-U3-5=A) |
| BR-U5-PLAYER-5 | 마피아 대표자만 SubmitMafiaKill 활성화 (`state.mafiaRepresentativeId === your.playerId`). 다른 마피아는 대기 + 대표자가 선택한 `pendingMafiaTarget` 표시 |
| BR-U5-PLAYER-6 | 의사 자가 보호 토글은 `state.settings.doctorSelfHealAllowed === true`일 때만 자기 ID를 PlayerPicker에 포함 |
| BR-U5-PLAYER-7 | 경찰 조사 한 밤 1회 — `state.policeCheckedThisNight === true`이면 입력 비활성화 |
| BR-U5-PLAYER-8 | 투표 폼 (VOTE/RECOUNT)은 모든 살아있는 플레이어 후보. 자기 자신 제외 |
| BR-U5-PLAYER-9 | PlayerView에서 TTS 발화 0 (BR-U5-TTS-1 참조) |

---

## 8. 호스트 컨트롤 규칙 (BR-U5-HOST)

| ID | 규칙 |
|---|---|
| BR-U5-HOST-1 | LOBBY 단계: "게임 시작" 버튼 (≥ 6명일 때 활성화) |
| BR-U5-HOST-2 | INTRO 단계: "다음 발언자" 버튼 |
| BR-U5-HOST-3 | DAY 단계: "토론 조기 종료" 버튼 |
| BR-U5-HOST-4 | NIGHT 단계: "야간 마감" 버튼 |
| BR-U5-HOST-5 | 모든 단계: "강제 종료" 버튼 (확인 다이얼로그 1회) |
| BR-U5-HOST-6 | "음성 안내 ON/OFF" 토글 (FR-8.5) — 백엔드 `host:toggle-voice` |
| BR-U5-HOST-7 | 호스트 컨트롤 권한 검증은 백엔드(U2/U3) 단독 — UI는 단순히 비활성화 힌트만 |

---

## 9. 마스킹 / 비공개 규칙 (BR-U5-MASK)

| ID | 규칙 |
|---|---|
| BR-U5-MASK-1 | PublicView는 `state.players[*].role`을 절대 표시 안 함 (PhaseEnd reveal 제외) |
| BR-U5-MASK-2 | PlayerView는 자기 Role/Keyword만 표시. 다른 플레이어 Role 노출 금지 (마피아 viewer는 다른 마피아 Role만 추가 노출) |
| BR-U5-MASK-3 | 백엔드가 wire 메시지에서 Role 마스킹 — 클라이언트는 보낸 그대로 신뢰. 자체 검증 없음 |
| BR-U5-MASK-4 | PhaseEnd 도달 시 `GameEnded.reveal[]`에 모든 Role 포함 → 양 화면에 공개 표시 가능 |
| BR-U5-MASK-5 | 토큰은 화면/로그/URL에 노출 금지 (BR-U5-TOKEN-5와 일치) |

---

## 10. 입력 검증 규칙 (BR-U5-INPUT)

| ID | 규칙 |
|---|---|
| BR-U5-INPUT-1 | 닉네임은 1~20자 한글/영문/숫자 — 클라이언트에서 1차 검증, 백엔드에서 최종 결정 |
| BR-U5-INPUT-2 | PlayerPicker 선택 후 즉시 송신 — 별도 "확인" 버튼 없음 (UX 간결성) |
| BR-U5-INPUT-3 | 같은 액션 중복 송신 허용 — 백엔드 last-write-wins (BR-U2-* 정책) |
| BR-U5-INPUT-4 | 백엔드 거부 (error 메시지) 시 토스트 표시. 입력은 그대로 유지 (재시도 가능) |
| BR-U5-INPUT-5 | 호스트 컨트롤 버튼은 단계 부적합 시 disabled 처리 — 클릭 자체 차단 |

---

## 11. 에러 처리 규칙 (BR-U5-ERR)

| ID | 규칙 |
|---|---|
| BR-U5-ERR-1 | wire `error` 메시지의 `message`를 한국어로 그대로 표시 — 백엔드가 announce.RenderError에서 한국어 매핑 (BR-U3-ERR-2) |
| BR-U5-ERR-2 | `error` 메시지는 5초 후 자동 사라짐 |
| BR-U5-ERR-3 | `code === "UNKNOWN_PLAYER_ERROR"`이면 토큰 무효화 + NicknameForm 재표시 (BR-U5-TOKEN-3) |
| BR-U5-ERR-4 | `code === "PERMISSION_DENIED_ERROR"`이면 호스트 권한 부족 안내 (PUBLIC에선 보통 발생 안 함) |
| BR-U5-ERR-5 | JSON.parse 실패 등 wire 형식 에러는 `console.warn`만 — 사용자 토스트 없음 |

---

## 12. 빌드 규칙 (BR-U5-BUILD, Q-FD-U5-15=A)

| ID | 규칙 |
|---|---|
| BR-U5-BUILD-1 | Vite outDir = `../cmd/mafia-game/web/dist` (U4 embed 위치와 일치) |
| BR-U5-BUILD-2 | 빌드 산출물에 source map 미포함 (운영 단순성) |
| BR-U5-BUILD-3 | 자산 파일명에 content hash (Vite 기본) — U4 immutable cache와 호환 |
| BR-U5-BUILD-4 | 개발 모드 `npm run dev`는 Vite proxy로 `/ws`를 백엔드 포트로 forward |

---

## 13. FR/NFR 추적성

| 출처 | 본 문서 규칙 |
|---|---|
| FR-1.2 (재연결) | BR-U5-WS-3, BR-U5-TOKEN-2 |
| FR-2.3 (역할 비공개) | BR-U5-MASK-1~3 |
| FR-3.2 (자기 키워드 비공개) | BR-U5-PLAYER-2, BR-U5-MASK-2 |
| FR-4.3 (마피아 대표자) | BR-U5-PLAYER-5 |
| FR-4.4 (의사 자가 보호) | BR-U5-PLAYER-6 |
| FR-5 (게임 종료 화면) | BR-U5-PLAYER-3 case END |
| FR-8.1 (Web Speech API) | BR-U5-TTS-1~9 |
| FR-8.2 (PublicView 한정) | BR-U5-TTS-1 |
| FR-8.3 (한국어, 근엄 톤) | BR-U5-TTS-4, BR-U5-COMMON-5 |
| FR-8.4 (안내 풍부) | BR-U5-TTS-1 (모든 announce 발화) |
| FR-8.5 (ON/OFF 토글) | BR-U5-HOST-6, BR-U5-TTS-7 |
| FR-8.6 (큐잉/인터럽션) | BR-U5-TTS-5, BR-U5-TTS-6 |
| FR-8.7 (자막 폴백) | BR-U5-TTS-2, BR-U5-TTS-9 |
| NFR-1 (재연결 시 화면 복원) | BR-U5-WS-3 + snapshot 메시지 처리 |
| NFR-3 (가독성) | BR-U5-PUBLIC-3 |
| NFR-4 (비공개) | BR-U5-MASK-*, BR-U5-TOKEN-5 |

---

## 14. 검증 체크리스트

- [x] Common 5개 규칙
- [x] 라우팅 4개 규칙
- [x] WS 연결·재연결 5개 규칙
- [x] 토큰 5개 규칙
- [x] TTS 9개 규칙 (FR-8.1~8.7 모두 매핑)
- [x] PublicView 6개 규칙
- [x] PlayerView 9개 규칙 (마피아 대표자 / 의사 / 경찰 분기)
- [x] 호스트 컨트롤 7개 규칙
- [x] 마스킹 5개 규칙
- [x] 입력 검증 5개 규칙
- [x] 에러 처리 5개 규칙
- [x] 빌드 4개 규칙
- [x] FR/NFR 추적성 매트릭스 (§13)

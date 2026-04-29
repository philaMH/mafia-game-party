# Iteration 7 — Build and Test Results (Voice 개편)

**작성일**: 2026-04-29
**대상**: `iteration7-execution-plan.md` Stage D
**상위 산출물**: `iteration7-requirements-patch.md` v7.0-patch / `iteration7-voice-script.md` v1.0

---

## 1. Requirements 추적 매트릭스 (R1~R8 ↔ FR-8.1~8.10)

| ID | 요구사항 (요약) | 영향 단위 | 검증 |
|---|---|---|---|
| R1 / FR-8.1 | 사전 녹음 MP3 재생, Web Speech API 의존 제거 | U5 | `useTTSQueue` + 테스트 + setup mock 삭제, `useAudioCueQueue` 신규 + 6 테스트 PASS |
| R2 / FR-8.2 | 호스트(`host:claim` 보유)만 음성 출력 | U5 | `GameContext` `useAudioCueQueue(state.voiceOn && state.isHost)` |
| R3 / FR-8.4.2 | 자막=변수 보간 / 음성=정형 멘트 + Eliminated 마피아·비마피아 분기 | U2 | `catalog_default.go` Render 분기 + 신규 테스트 `TestRender_EliminatedAudioCueSplitsOnMafia` |
| R4 / FR-8.5 | VoiceToggle 호스트 한정 노출 | U5 | `PublicView.tsx` `{isHost && <VoiceToggle />}` |
| R5 / FR-8.8 | 음성 누락 시 자막만 표시(graceful skip) | U5 | `useAudioCueQueue.ts` `play().catch` → `console.warn + playNext()`. 테스트 `survives play() rejection` PASS |
| R6 / FR-8.9 | 안정 audioId, `/audio/<id>.mp3` 매핑 | U2/U3/U5 | `Announcement.AudioID` 필드 + 27 cue 상수 + `announceMsg.audioId` JSON omitempty + 클라이언트 `/audio/${id}.mp3` |
| R7 / FR-8.10 | 호스트 식별 강화 (`isHost` flag) | U5 | GameContext 에서 `state.isHost` 게이팅 |
| R8 / FR-8.6 | 큐잉 + urgent 인터럽트 (`PhaseChanged`/`Eliminated`/`DeathAnnounced`/`GameEnded`) | U5 | `URGENT_KINDS` 상수 유지 + `enqueueUrgent` 보존 |

---

## 2. 영향 파일 일람 (실측)

### 2.1 Go (Stage A + B)

**수정**:
- `internal/announce/catalog.go` — `Announcement.Speech` → `Announcement.AudioID`, doc 갱신
- `internal/announce/catalog_default.go` — 27 audioId 상수 + `ann()` 시그니처(`text, audioID, sev`) + `Render` 27 분기 + `Eliminated` mafia/notmafia 분기
- `internal/announce/catalog_data.go` — `SystemRestore` / `SystemPersistFailure` 의 `Speech` 필드 → `AudioID`
- `internal/announce/error.go` — error 안내에서 `Speech` 필드 제거 (audioId 미발급)
- `internal/announce/catalog_test.go` — `Subtitle != Speech` 어서션 제거, audioId 검증 + 신규 3 테스트 (`TestRender_EliminatedAudioCueSplitsOnMafia` / `TestRender_AudioCueAssignmentCoversCatalog` / `TestRenderError_NoAudioID` / `TestSystemHelpers_HaveAudioCues`)
- `internal/transport/ws/protocol.go` — `announceMsg.Speech` → `announceMsg.AudioID` (`json:"audioId,omitempty"`)
- `internal/transport/ws/dispatch.go` — `Speech: out.Announcement.Speech` → `AudioID: out.Announcement.AudioID`
- `internal/transport/ws/handlers.go` — 같은 패턴

**신규/삭제**: 없음

### 2.2 Web (Stage C)

**신규**:
- `web/src/hooks/useAudioCueQueue.ts` (157 LoC) — HTMLAudioElement 기반 직렬 큐, urgent 인터럽트, graceful skip
- `web/src/hooks/useAudioCueQueue.test.ts` (6 케이스) — FIFO / urgent / disabled / 빈 audioId / cancelAll / play 실패 복원
- `web/public/audio/.gitkeep` — 외부 녹음 파일 배치 안내

**수정**:
- `web/src/types/wire.ts` — `AnnounceMsg.speech` → `AnnounceMsg.audioId?` (optional)
- `web/src/context/reducer.ts` — `ttsAvailable` → `audioAvailable`, `tts_unavailable` → `audio_unavailable`, `lastAnnounce` 에 `audioId?` 보존, initial state 가용성 검사 `typeof Audio !== "undefined"`
- `web/src/context/reducer.test.ts` — `speech` 어서션 제거, `audioId` 보존 + 미설정 graceful 케이스 추가
- `web/src/context/GameContext.tsx` — `useTTSQueue` import 제거, `useAudioCueQueue(state.voiceOn && state.isHost)`, `audio_unavailable` dispatch, `enqueue/enqueueUrgent(ann.audioId)` 빈 값은 skip
- `web/src/views/PublicView/PublicView.tsx` — VoiceToggle/안내 문구 모두 `isHost` 가드, `ctx.ttsAvailable` → `ctx.audioAvailable`
- `web/src/tests/setup.ts` — `FakeSpeechSynthesis` / `FakeUtterance` 제거, `FakeAudio` 도입(jsdom 호환)

**삭제**:
- `web/src/hooks/useTTSQueue.ts`
- `web/src/hooks/useTTSQueue.test.ts`

---

## 3. 검증 결과

### 3.1 Go

| 명령 | 결과 |
|---|---|
| `go build ./...` | ✅ |
| `go build -o /tmp/mafia-game-iter7 ./cmd/mafia-game` | ✅ 15.94 MB |
| `go test ./... -count=1` | ✅ 6 패키지 PASS |

| 패키지 | 커버리지 (Iter7) | 커버리지 (Iter6 또는 직전) | Δ |
|---|---|---|---|
| `internal/announce` | **94.3%** | 94.0% | +0.3 pp |
| `internal/game` | **91.7%** | 91.7% | 0 |
| `internal/persistence` | **80.2%** | 80.2% | 0 |
| `internal/session` | **86.1%** | 86.1% | 0 |
| `internal/transport/http` | **89.8%** | 89.8% | 0 |
| `internal/transport/ws` | **82.4%** | 82.4% | 0 |

### 3.2 Web

| 명령 | 결과 |
|---|---|
| `npm test` (vitest) | ✅ 47 PASS (Test Files 4) |
| `npm run build` (`tsc --noEmit && vite build`) | ✅ |

| 산출 | 크기 | Iter6 비교 |
|---|---|---|
| `index.html` gzip | 0.36 KB | = |
| CSS gzip | 3.21 KB | = |
| **JS gzip** | **64.83 KB** | 64.93 KB → **−0.10 KB** (useTTSQueue 제거 효과) |

| 모듈 | Stmts | Branch | Funcs | Lines |
|---|---|---|---|---|
| All files | 81.11 | 80.15 | 94.44 | 81.11 |
| `context/reducer.ts` | **90.72** | 75.9 | 100 | 90.72 |
| `hooks/useAudioCueQueue.ts` | **91.58** | 84 | 100 | 91.58 |
| `hooks/useToken.ts` | 91.3 | 87.5 | 100 | 91.3 |
| `components/NicknameForm.tsx` | 100 | 100 | 100 | 100 |

> 참고: `useWebSocket.ts` 0% 는 Iter6 와 동일(통합 테스트는 vitest 가 아닌 수동 검증). Iter7 변경 무관.

### 3.3 회귀 영향

- **자막 표시**: `lastAnnounce.subtitle` 형태 보존, SubtitleArea 변경 없음 → 기존 자막 그대로
- **Pause/Resume**: `cueGamePaused` / `cueGameResumed` audioId 매핑만 추가, Iter5 동작 보존
- **LOBBY membership**: PlayerJoined audioId 미발급(빈값 graceful skip), 동작 영향 없음
- **TTS 부재 안내 토스트**: 기존 "이 브라우저는 음성 안내를 지원하지 않습니다" 문구는 이제 호스트 한정 + audio 가용성 0인 경우만 표시 (실용상 거의 발생하지 않음). 메시지 자체는 유지.

---

## 4. NFR 영향

| NFR | 결과 |
|---|---|
| NFR-1 안정성 | ✅ 외부 의존 0, 정적 자산. 누락 graceful skip 으로 게임 무중단 (테스트 `survives play() rejection`) |
| NFR-2 성능 | ✅ JS gzip −0.10 KB. mp3 자산은 외부 준비, 본 빌드 산출 변화 없음. 첫 게임 lazy-load 시 호스트 LAN 지연 < 100 ms 예상 |
| NFR-3 사용성 | ✅ 발화자 일관성 + 정형 멘트, 자막 변수 보간 유지 |
| NFR-5 호환성 | ✅ Web Speech API 의존 제거 → Firefox/한국어 음성 부재 환경 호환성 향상 |
| NFR-6 유지보수성 | ✅ audioId 카탈로그 단일 진실 소스 (catalog_default.go cue 상수 27건 + voice-script §3 표) |
| NFR-7 운영 단순성 | ✅ 호스트는 mp3 만 `web/public/audio/` 에 배치하면 즉시 동작 |

---

## 5. 위험 결산 (plan §9)

| ID | 위험 | 결과 |
|---|---|---|
| RISK-7-1 | Chrome autoplay 정책 | 호스트 `host:open-room` 클릭이 첫 인터랙션으로 audio context 활성. 실측은 사용자 Chrome DevTools MCP 회귀 시 확인 |
| RISK-7-2 | 27 mp3 모두 누락 시 첫 가동 | graceful skip 으로 게임 무중단 검증 (테스트 + console.warn 동작) |
| RISK-7-3 | 커버리지 백분율 변동 | 정상 범위 유지. announce +0.3 pp, 다른 패키지 동일 |
| RISK-7-4 | `Announcement.Speech` 외부 사용처 | grep + 컴파일러 진단으로 8건 모두 일괄 갱신, `go build ./...` 깨끗 |
| RISK-7-5 | audio context 재초기화 | 훅 unmount cleanup 으로 Audio 엘리먼트 분리 |

---

## 6. Definition of Done 체크

- [x] U2 DoD (§3.1.5) — Speech 모두 제거, AudioID 도입, 27 매핑, Eliminated 분기, 테스트 PASS, 커버리지 ≥ 85%
- [x] U3 DoD (§3.2.4) — Speech wire 제거, AudioID 직렬화, 테스트 PASS, 커버리지 ≥ 80%
- [x] U5 DoD (§3.3.10) — useTTSQueue 제거, useAudioCueQueue + 테스트, VoiceToggle host-only, reducer/GameContext 갱신, npm test 47 PASS, 빌드 성공, gzip ±2 KB 이내(−0.10 KB), reducer.ts ≥ 90.7%
- [x] `go test ./... -count=1` 6 패키지 PASS
- [x] `go build` 성공 (15.94 MB)
- [x] `npm test` PASS (47/47)
- [x] `npm run build` 성공 (JS gzip 64.83 KB)
- [x] `iteration7-test-results.md` 작성 (본 문서)
- [ ] aidlc-state.md Iteration 7 모든 [x] (사용자 승인 게이트 후 마감)
- [x] audit.md 모든 단계 raw input 기록

---

## 7. 후속 권장 사항 (사용자 트리거)

1. **외부 녹음 작업 발주** — `iteration7-voice-script.md` §3 의 27 audioId × 음성 텍스트로 작업자에게 전달. 28~29(system.*) 는 선택.
2. **Chrome DevTools MCP 회귀** — 호스트 vs 일반 PublicView 관전자 분리 (호스트만 음성 재생, 관전자는 자막만), VoiceToggle 호스트 한정 노출, mp3 누락 시 graceful skip 동작, urgent 인터럽트(PhaseChanged 시 직전 발화 중단) 시나리오. host autoplay 가드(첫 인터랙션 후 재생) 실측.
3. **mp3 추가 후 재배포** — 파일을 `web/public/audio/<audioId>.mp3` 에 두고 `npm run build` → `go build` 재실행하면 임베드 정적 자산에 자동 포함.

---

## 8. 변경 요약 (한 줄)

브라우저 TTS 의존을 제거하고 호스트 PublicView 가 사전 녹음 MP3 (`/audio/<audioId>.mp3`) 를 재생하도록 전환했으며, 27 안내 시점에 안정 audioId 를 부여하고 Eliminated 정체 공개를 마피아/비마피아 두 음성 큐로 분리했습니다.

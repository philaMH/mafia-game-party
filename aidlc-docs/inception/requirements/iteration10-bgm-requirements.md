# Iteration 10 — 호스트 BGM 무한재생 요구사항 v1.0

**작성**: 2026-04-30T00:55:00Z
**작업 브랜치**: `feature+bgm`
**프로젝트 분류**: Brownfield · 5단위 구조 + Iteration 1~9 산출물 보존
**영향 단위**: U5 (Web Frontend) 단독
**소스 답변 파일**: `inception/requirements/iteration10-bgm-questions.md` (Q1=A / Q2=A / Q3=B / Q4=A / Q5=A / Q6=A)

---

## 1. 개요 (Intent)

호스트(`PublicView`) 화면에서 단일 BGM 음원(`web/public/audio/bgm.mp3`)을 무한재생한다. 기존 효과음(`useAudioCueQueue`) 과는 독립된 오디오 트랙으로 운용하며, 게임 흐름과 무관하게 호스트 priming 직후부터 재생을 유지한다. 플레이어 화면에서는 재생되지 않는다.

## 2. 범위

### In-Scope
- BGM 단일 트랙(`/audio/bgm.mp3`) 무한 루프 재생
- 호스트(`isHost`) 화면에서만 동작
- 별도 BGM on/off 토글 UI (VoiceToggle 옆 배치)
- graceful 실패 처리(자산 누락·디코딩 실패 시 경고만)

### Out-of-Scope
- 다중 BGM 트랙 / 트랙 전환 / 페이즈별 BGM 변경
- 덕킹(자동 음량 감소) — cue 재생 중에도 BGM 음량 변동 없음
- BGM 토글 상태 영속화(localStorage 저장 — 본 iteration 미포함; 필요 시 별도 iteration)
- Go 서버 변경(`bgm.mp3` 는 기존 `audioHandler` 정적 라우팅으로 자동 서빙)
- 플레이어 화면 BGM, TTS 변경

## 3. 기능 요구사항 (FR)

### FR-1. 호스트 priming 직후 자동 시작 (Q1=A)
PublicView 가 마운트되고 `isHost === true` 이며 BGM 토글이 ON 인 상태에서, 호스트가 priming 제스처(예: 방 개설 버튼 클릭) 후 첫 효과음 cue 가 priming 되는 시점과 함께 BGM 도 `play()` 시도한다.
- 브라우저 autoplay 정책상 priming 이전에는 재생 시도하지 않음
- priming 한 번이면 이후 무한 루프(`HTMLAudioElement.loop = true`) 로 유지

### FR-2. 별도 BGM 토글 (Q2=A)
- 기존 `VoiceToggle` 컴포넌트 옆(또는 동일 영역 내)에 BGM on/off 토글 신설
- 토글 OFF 시: BGM 즉시 일시정지(`pause()`), `currentTime` 은 보존
- 토글 ON 시: 일시정지된 위치에서 재생 재개 (priming 이미 통과한 상태일 때)
- 호스트(`isHost === true`)에게만 노출

### FR-3. 저볼륨 고정 + 덕킹 없음 (Q3=B, Q4=A)
- `HTMLAudioElement.volume = 0.15` 고정
- 효과음 cue 가 재생 중이어도 BGM 볼륨/재생 상태 변경 없음
- 효과음과 BGM 은 두 개의 독립된 `Audio` 엘리먼트로 분리 관리

### FR-4. Pause / GameEnded / 새로고침 동작 (Q5=A)
- 게임 일시정지(`Paused` 상태): BGM 재생 유지(중단 없음)
- 게임 종료(`GameEnded` 이벤트): BGM 재생 유지
- 호스트 새로고침 / BFCache 복원: BGM 엘리먼트는 새로 생성됨 → 다음 priming 제스처 후 자동 재시작 (Iteration 9 의 `useWebSocket` pageshow 풀 리로드 동작과 호환)

### FR-5. Graceful 실패 처리 (Q6=A)
- `bgm.mp3` 자산 누락(404) 또는 디코딩 오류:
  - `console.warn("[bgm] failed to play /audio/bgm.mp3", err)` 출력
  - 게임 진행에 영향 없음 (효과음 큐 / 게임 상태 무영향)
- UI 배지/안내는 본 iteration 미포함

### FR-6. 호스트 한정
- `PublicView` 내부에서 `isHost === true` 일 때만 `useBgm` 훅이 활성
- 플레이어(`PlayerView`) 화면에서는 BGM 미재생

### FR-7. 컴포넌트 unmount 시 정리
- PublicView unmount 시:
  - `pause()` 호출
  - `src` 해제 / 리스너 제거
  - 메모리 누수 방지 (Iteration 7 `useAudioCueQueue` 와 동일한 cleanup 패턴)

## 4. 비기능 요구사항 (NFR)

### NFR-1. 빌드 사이즈
- 추가 코드는 ~80~120 lines (단일 hook + 토글 통합) 예상
- JS gzip 증가 ≤ +0.5 KB (baseline 65.71 KB → 목표 ≤ 66.21 KB)
- CSS 변경 미미 또는 없음

### NFR-2. 런타임 영향
- BGM 엘리먼트 1개 추가 — 메모리/CPU 영향 무시 가능
- `bgm.mp3` 단일 파일 로딩(이미 1회) — 네트워크 추가 요청 없음

### NFR-3. 회귀 영향
- 기존 효과음 큐(`useAudioCueQueue`) 동작 변경 없음
- 기존 `VoiceToggle` 동작 변경 없음 (옆에 토글 1개 추가만)
- Go 패키지 6종 변경 없음

### NFR-4. 테스트 커버리지
- 신규 `useBgm` 훅 단위 테스트 ≥ 4 케이스 (기본 재생 / 토글 OFF / 누락 graceful / unmount cleanup)
- 기존 60 npm test 회귀 PASS 유지

### NFR-5. 브라우저 autoplay 정책 준수
- priming 전 `play()` 호출 금지
- priming 후 `play()` rejection 시 콘솔 경고 + 자동 재시도 없음 (사용자가 토글 OFF→ON 으로 명시적 재시도)

## 5. 수용 조건 (AC)

| ID | 조건 | 검증 |
|----|------|------|
| AC-1 | 호스트가 방을 열면 BGM이 자동 시작되어 끝없이 반복된다 | 실측 — 호스트 화면 priming 후 5분 이상 무중단 재생 확인 |
| AC-2 | BGM 토글 OFF 클릭 시 즉시 일시정지, ON 클릭 시 재개 | 실측 + 단위 테스트 |
| AC-3 | 효과음 cue 가 BGM 위에 그대로 들린다 (덕킹 없음) | 실측 — `phase.intro.mp3` 재생 중 BGM 볼륨 변화 없음 |
| AC-4 | BGM 기본 볼륨이 0.15 | 단위 테스트(`audio.volume === 0.15`) |
| AC-5 | Pause/GameEnded 발생해도 BGM 유지 | 실측 — pause() 호출 없음을 testify |
| AC-6 | 새로고침 후 priming 다시 거치면 자동 재시작 | 실측 — 호스트 새로고침 → 방 재개설 → BGM 재생 |
| AC-7 | `bgm.mp3` 강제 누락(이름 변경) 시 콘솔 경고만 출력, 게임 정상 진행 | 실측 — 파일명 임시 변경 후 검증 |
| AC-8 | 플레이어 화면에서 BGM 재생 없음 | 실측 — 플레이어 컨텍스트 inspect |

## 6. 영향 분석

| 단위 | 변경 | 비고 |
|------|------|------|
| U1 Game Core | 없음 | 도메인 변경 없음 |
| U2 Session/Persistence/Announce | 없음 | 카탈로그 변경 없음 |
| U3 Realtime Transport | 없음 | wire 메시지 변경 없음 |
| U4 HTTP Bootstrap | 없음 | `/audio/` 라우팅이 자동 서빙 (Iteration 7 audioHandler 재사용) |
| U5 Web Frontend | 신규 hook + PublicView 통합 + 토글 | 이번 iteration의 유일한 변경 |

## 7. 가정 / 의존성

- `web/public/audio/bgm.mp3` 자산이 빌드 시 `dist/audio/bgm.mp3` 로 복사된다 (Vite `public/` 자동 처리)
- 기존 `useAudioCueQueue` 의 priming 시점이 BGM priming 시점과 동일 (방 개설 버튼 클릭) — 별도 priming 게이트 불필요
- 호스트는 단일 PublicView 인스턴스 (다중 호스트 시나리오 없음)

## 8. 위험 / 트레이드오프

| 위험 | 완화 |
|------|------|
| iOS Safari 등 일부 브라우저에서 priming 직후에도 `play()` rejection 가능 | console.warn 만 출력, 토글 OFF→ON 으로 사용자가 재시도 가능 |
| 두 Audio 엘리먼트(`useAudioCueQueue` + `useBgm`)가 같은 cue 와 겹쳐 청취감 저하 가능 | 기본 볼륨 0.15 로 충분히 낮음 (Q4=A 사용자 결정) |
| BGM 토글 상태 미영속 — 새로고침 시 ON 으로 초기화 | 본 iteration 범위 외, 향후 별도 iteration 에서 localStorage 추가 가능 |

## 9. 다음 단계 (Workflow Planning 예고)

- User Stories: SKIP (단일 호스트 페르소나, UX 작은 추가)
- Workflow Planning: U5 단독 — Functional Design Patch → Code Generation Plan → Code Generation → Build & Test
- Application Design: SKIP (도메인 인터페이스 추가 없음)
- Units Generation: SKIP (5단위 구조 유지)
- Construction:
  - U1/U2/U3/U4 모두 SKIP
  - U5: Functional Design Patch → Code Generation Plan(Step A~D 예상) → Code Generation → 회귀 검증
- Build and Test: `iteration10-test-results.md` v1.0

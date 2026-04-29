# Mafia Game — 요구사항 정의서 Iteration 7 Patch (Voice 개편)

**문서 버전**: 7.0-patch
**작성일**: 2026-04-29
**상위 문서**: `inception/requirements/requirements.md` (v1.1) — FR-8 진행자 음성 안내
**프로젝트 유형**: Brownfield Patch
**문서 깊이**: Standard (요건 영향 범위 제한적)

---

## 0. Intent Analysis 요약

| 항목 | 내용 |
|---|---|
| **사용자 원본 요청** | "voice 기능을 개편합니다. 1) 호스트만 음성 출력 가능. 2) 진행 phase에 따라 사전 녹음된 음성파일 재생. 3) 녹음이 필요한 phase 모든 상황 선별 + 스크립트 작성, 음성 파일은 외부에서 직접 준비." |
| **요청 유형** | Brownfield Modification (FR-8 전면 개정) |
| **요청 명확도** | Clear after Round 1 (8문항 객관식 답변 수신) |
| **초기 범위** | U2 announce 카탈로그 + U3 wire eventPayload + U5 web frontend 훅·뷰 |
| **요구사항 깊이** | Standard |

---

## 1. 변경 본질

| 항목 | Before (v1.1) | After (v7.0-patch) |
|---|---|---|
| **TTS 엔진** | 클라이언트 Web Speech API (`SpeechSynthesis`) | **사전 녹음 MP3 파일 재생** (`<audio>` 또는 Web Audio API) |
| **출력 클라이언트** | PublicView( `/` ) 모든 클라이언트 | **`host:claim` 보유 PublicView 만** |
| **VoiceToggle 가시성** | PublicView footer 모두 노출 | **호스트에게만 노출** |
| **변수 보간** | 자막 = 음성 동일 (이름·일수 그대로 발화) | **자막은 변수 보간 / 음성은 변수 제거 정형 멘트** |
| **음성 누락 처리** | 한국어 음성 미가용 시 자막만 | **자막만 표시 (graceful skip)** |
| **음성 식별** | 자막 텍스트 그대로 발화 | **이벤트별 안정 `audioId` → `/audio/<audioId>.mp3`** |

### 1.1 사용자 답변 매핑 (Iteration 7 Round 1)

| Q | Answer | 결정 |
|---|---|---|
| Q1 | A | 호스트(`host:claim` 보유 PublicView) 만 음성 재생 |
| Q2 | A | MP3 단일 형식, `web/public/audio/*.mp3` |
| Q3 | Other | **자막**=변수 포함 / **음성**=변수 제거 정형 멘트. Eliminated 는 마피아 vs 비마피아 2벌 분리 |
| Q4 | A | 현재 카탈로그 27 시점 모두 (Q3 정책 적용) |
| Q5 | A | 음성 누락 시 자막만 표시, 게임 진행 무중단 |
| Q6 | A | `useTTSQueue` + 테스트 완전 제거, 새 훅 `useAudioCueQueue` 도입 |
| Q7 | A | 호스트에게만 VoiceToggle 노출 |
| Q8 | A | 카탈로그가 안정 `audioId` 발급, 클라이언트는 `/audio/<audioId>.mp3` 매핑 |

---

## 2. FR-8 (진행자 음성 안내) — 개정안 v7.0

### FR-8.1 음성 엔진 (개정)
- **사전 녹음 MP3 파일**을 클라이언트가 재생한다. Web Speech API(`SpeechSynthesis`) 의존성은 제거된다.
- 파일 포맷: **MP3 단일 형식** (Q2=A). 모든 모던 브라우저 호환, 웹 빌드 정적 자산으로 임베드.
- 파일 경로: `web/public/audio/<audioId>.mp3` — 빌드 후 `dist/audio/...` 로 복사. Go 서버는 `internal/transport/http` 의 정적 라우팅으로 그대로 서빙.

### FR-8.2 출력 위치 (강화)
- 음성 재생은 **PublicView (`/`)** 클라이언트 중 **`host:claim` 토큰을 받은 단일 호스트** 에서만 출력 (Q1=A, Q7=A).
- 다른 PublicView 관전자는 **자막만** 표시.
- PlayerView( `/play` 등 ) 는 **음성·자막 모두 미출력** (현 정책 유지).

### FR-8.3 음성 톤 (유지)
- 한국어 (`ko-KR`), **근엄·차분·고전적 진행자** 톤 유지 (Iteration 1 Q-FD-U2-8=A).
- 외부 녹음으로 일관된 발화자 1인을 권장 (Voice Script 문서 §4 가이드 참조).

### FR-8.4 안내 범위 (재정의 + 변수 정형화)

#### FR-8.4.1 카탈로그 27 시점 모두 대상
기존 v1.1 의 27 시점 (게임/단계 전환 + 밤 단계 + 발언자 + 사망/평화 + 처형 + 토론 카운트다운 + 동률/처형무 + 게임 종료 + 음성 토글 + Pause/Resume + 시스템 복원/저장 실패) 을 모두 사전 녹음 대상으로 한다.

#### FR-8.4.2 자막/음성 분리 정책 (Q3=Other 사용자 가이드)
- **자막**: 동적 변수(이름·일수·시간·역할) 를 **그대로 보간** 하여 표시. Iteration 6 노이르 SubtitleArea 변경 없음.
- **음성**: 동적 변수가 들어가지 않는 **정형 멘트** 만 발화. 변수 부분은 일반화 표현(예: `%s` → "한 사람", `%d일째` → "새로운 아침이") 으로 대체.
- **변형 분기**: 정체 공개(Eliminated) 같이 의미가 갈리는 경우 정형 음성을 **2벌 이상** 분리 녹음. 이번 카탈로그에서는 다음 1건만 분기:
  - `eliminated.mafia` (정체 = 마피아)
  - `eliminated.notmafia` (정체 ≠ 마피아 — 시민/의사/경찰 통합)

#### FR-8.4.3 시점·audioId 매핑 산출물
구체적 27 시점 × 음성/자막 매핑은 별도 산출물 `iteration7-voice-script.md` 로 분리 작성한다 (사용자 검토·외부 녹음 발주 편의 목적).

### FR-8.5 음성 ON/OFF 토글 (개정)
- **호스트에게만** VoiceToggle 노출 (Q7=A). 기본값 ON.
- 일반 PublicView 관전자에게는 토글이 화면에 노출되지 않으며, 그들의 클라이언트는 자막만 출력하므로 토글은 의미가 없다.
- 현 토글 상태 저장 정책(세션 동안 유지, 새 게임 시 ON 복원) 은 그대로 유지.

### FR-8.6 큐잉 / 인터럽션 (유지 + 단순화)
- 짧은 시간에 여러 안내가 발생하면 **순차 재생(큐잉)**.
- 단계 전환 시 **이전 발화를 중단하고 새 안내로 대체** (urgent 인터럽트). 대상 이벤트: `PhaseChanged`, `Eliminated`, `DeathAnnounced`, `GameEnded` (현 `URGENT_KINDS` 유지).
- 음성 파일 미로드/누락 시 **에러 없이 다음 큐 항목으로 진행** (FR-8.8 graceful skip).

### FR-8.7 자막 (유지)
- 모든 안내는 자막을 동시에 표시한다 (NFR-3 명확한 가이드).
- 자막 텍스트는 v1.1 형식 그대로 유지 — 변수 보간 포함.

### FR-8.8 음성 누락 / 로드 실패 처리 (신규, Q5=A)
- 클라이언트가 `/audio/<audioId>.mp3` 를 fetch 또는 재생할 때 404·decode error 가 발생하면, **콘솔 경고만 남기고 자막은 정상 표시** 한다.
- 게임 진행은 무중단. 큐는 다음 항목으로 즉시 진행.
- 필수 안내(예: GameStarted)가 누락되어도 게임 자체에 영향 없음 — 자막으로 진행 가능.

### FR-8.9 식별자 발급 정책 (신규, Q8=A)
- `internal/announce` 카탈로그는 각 이벤트 → **안정 `audioId`** (kebab/dot 표기, ASCII) 를 결정한다.
- `Announcement` 구조에 `AudioID string` 필드를 추가한다 (또는 기존 `Speech` 필드를 `AudioID` 로 의미 전환). 기존 `Speech` 필드는 폐기된다 (Q6=A 와 정합).
- U3 wire `eventPayload` 직렬화는 `audioId` 를 추가 필드로 전달.
- 클라이언트는 `audioId` 가 빈 문자열이면 음성 미재생, 비어 있지 않으면 `/audio/<audioId>.mp3` 를 재생.

### FR-8.10 호스트 식별 강화 (신규)
- Iteration 2 패치에서 도입된 `host:claim` 메커니즘과 정합: 클라이언트는 `state.isHost === true` 일 때만 음성 재생 훅을 활성화한다.
- 같은 PublicView 페이지를 여러 디바이스에서 열어도 음성 재생은 1개 디바이스(호스트)로 한정된다.

---

## 3. 영향 단위 (Brownfield Impact)

| 단위 | 영향 | 비고 |
|---|---|---|
| **U1 Game Core** | 무영향 | 도메인 이벤트 변경 없음 |
| **U2 Session/Persistence/Announce** | **변경** | `Announcement` 구조에 `AudioID` 필드 추가. `Speech` 필드 제거. 카탈로그 27 시점에 `audioId` 부여. `Eliminated` 분기(2 audioId) 처리. |
| **U3 Realtime Transport** | **변경** | wire `eventPayload` 에 `audioId` 필드 추가. 기존 `subtitle` 자막은 변수 보간 그대로 유지(=현 동작). 직렬화 테스트 갱신. |
| **U4 HTTP Bootstrap** | 무영향 | `/audio/*.mp3` 는 Iteration 4 단계에서 도입한 정적 파일 라우팅이 그대로 처리(추가 라우팅 불필요 검증 예정 — Code Generation 시) |
| **U5 Web Frontend** | **변경** | `useTTSQueue` 폐기 → `useAudioCueQueue` 도입. `VoiceToggle` 호스트 한정 렌더. `IntroView` TTS 호출 제거. `GameContext` 의 audio 큐 통합. wire 타입 갱신. 테스트 갱신. |

---

## 4. 비기능 요구사항 영향

| NFR | 영향 |
|---|---|
| **NFR-1 안정성** | 외부 인터넷 의존 0 (정적 자산). 서버 재시작 후에도 음성 정상 동작. 누락 graceful skip 으로 게임 무중단 보장. |
| **NFR-2 성능** | MP3 파일 1개당 < 200 KB 권장(가이드, Voice Script §4). 27 파일 × 200 KB ≈ 5 MB 추가 정적 자산. 빌드 결과 gzip 영향 미미(MP3 자체는 이미 압축 포맷). 첫 로드 시 모든 파일 prefetch 권고(또는 lazy load). |
| **NFR-3 사용성** | 발화자 일관성 + 정형 멘트로 청취 명료성 향상. 자막은 변수 보간 유지로 정보량 동일. |
| **NFR-4 보안** | Security Baseline 비활성 정책 유지 (LAN). |
| **NFR-5 호환성** | Web Speech API 의존 제거 → Firefox/한국어 음성 부재 환경 호환성 향상. MP3 는 모든 모던 브라우저 지원. |
| **NFR-6 유지보수성** | `audioId` 카탈로그가 단일 진실 소스. 외부 녹음 작업과 코드 변경이 분리되어 발화자/문구 교체 용이. |
| **NFR-7 운영 단순성** | 운영자(호스트) 가 mp3 파일을 `web/public/audio/` 에 두기만 하면 됨. 별도 빌드 파이프라인 불필요. |

---

## 5. 시나리오 변경

### 시나리오 1 (정상 게임 진행) — 음성 출력 부분 갱신
4단계: 호스트가 "방 개설" → 게임 시작 → **호스트 디바이스 음성**: `game.started` mp3 재생 ("마피아 게임이 시작됩니다…")
9단계: 사망자 발표 → **호스트 디바이스 음성**: `death.announced` mp3 ("전날 밤 한 사람이 사망했습니다…") + **자막**: "전날 밤 김철수님이 사망했습니다…" (변수 포함)
11단계: 시민팀 승리 → **호스트 디바이스 음성**: `end.citizen` mp3.

### 시나리오 6 (음성 부재) — 폐기/대체
v1.1 의 "한국어 TTS 음성 부재" 시나리오는 폐기. 대체:
- **시나리오 6'**: 음성 파일 누락(404) — 콘솔 경고, 자막 정상, 게임 무중단.

### 시나리오 7 (음성 OFF) — 동작 동일
- 호스트가 VoiceToggle OFF → 이후 모든 안내는 자막으로만. 동작은 v1.1 동일, 다만 **호스트가 아닌 클라이언트는 토글 자체가 보이지 않음**.

---

## 6. 가정 및 제약 (변경분)

### 6.1 가정 (추가)
- 외부 녹음은 개발자가 직접 준비하며 본 워크플로우 범위 외 (사용자 확인). 본 산출물의 Voice Script 표가 발주 명세 역할을 한다.
- 호스트 PC 가 audio 자동재생 정책(Chrome autoplay) 을 만족할 수 있도록, 첫 사용자 인터랙션(예: "방 개설" 클릭) 후 audio context 가 활성화되어 있어야 함 — Code Generation 단계에서 우회 처리 검토.

### 6.2 제약 (추가)
- MP3 파일 1개당 권장 크기 ≤ 200 KB, 길이 ≤ 12초. (NFR-2 의 5 MB 정적 자산 한도와 정합)
- 다국어 지원은 범위 외 (한국어 단일).

---

## 7. 추적 가능성 매트릭스 (개정분)

| 요구사항 ID | 출처 | 근거 |
|---|---|---|
| FR-8.1 (사전 녹음 MP3) | Iter7 Q2=A | MP3 단일 형식 |
| FR-8.2 (호스트 한정) | Iter7 Q1=A, Q7=A | host:claim 식별 + 토글 host-only |
| FR-8.4.2 (변수 정형화) | Iter7 Q3=Other (사용자 직접 가이드) | 자막=변수 / 음성=정형, Eliminated 2벌 |
| FR-8.8 (graceful skip) | Iter7 Q5=A | 누락 시 자막만 |
| FR-8.9 (audioId 식별자) | Iter7 Q8=A | 카탈로그 발급 + `/audio/<id>.mp3` |
| FR-8.10 (호스트 식별) | Iter2 host:claim 메커니즘 + Iter7 Q1=A | isHost flag 활용 |

---

## 8. Definition of Done (Requirements 단계)

- [x] 모든 명확화 질문 8건 답변 수신
- [x] 사용자 가이드(Q3=Other)를 정형 멘트 정책으로 문서화
- [x] FR-8 v1.1 → v7.0-patch 개정 매핑 표 완성
- [x] 영향 단위 추적 (U2/U3/U5)
- [x] NFR/시나리오/가정 영향 반영
- [ ] 사용자 승인 (GATE)
- [ ] 별도 산출물 `iteration7-voice-script.md` 사용자 검토

---

## 9. 미해결 추정 사항 (사용자 검토 필요)

다음 정형화 추정은 Q3 사용자 가이드를 다른 변수 멘트에도 일관 적용한 결과입니다. **사용자가 검토 후 수정 가능**:

| 시점 | 변수 자막 (유지) | 추정 정형 음성 |
|---|---|---|
| PhaseChanged → Intro | "한 사람당 %d초가 주어집니다" | "각자 차례대로 자기소개를 진행하시오." (시간 부분 음성에서 제거) |
| PhaseChanged → Day (day≥2) | "%d일째 아침이 밝았습니다" | "새로운 아침이 밝았습니다." (일수 부분 음성에서 제거) |
| IntroSpeakerChanged | "%s, 발언하시오." | "다음 차례입니다. 발언하시오." (이름 음성에서 제거) |

세부 27 시점 정형화 결과는 `iteration7-voice-script.md` §3 표 참조.

---

## 10. 다음 단계

1. **본 문서 + voice-script 문서 사용자 승인** (GATE)
2. **Workflow Planning** — Iteration 2~5 패턴(per-unit patch) 적용. U2 → U3 → U5 순차, U1/U4 SKIP.
3. **Code Generation Plan + 실행** — Stage A (U2 audioId 발급) → Stage B (U3 wire) → Stage C (U5 훅·뷰 교체) → Stage D (Build & Test)
4. **Build and Test** — `go test ./...` + `npm test` + `npm run build` + `go build`. Chrome DevTools MCP 회귀(호스트 vs 관전자 분리 검증)는 사용자 트리거.

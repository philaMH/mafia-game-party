# Iteration 10 — BGM 무한재생 명료화 질문

**작성**: 2026-04-30T00:44:00Z
**작업 브랜치**: `feature+bgm`
**대상 단위**: U5 (Web Frontend) 단독 예상
**자산**: `web/public/audio/bgm.mp3` (이미 배치됨)

호스트 화면에서 단일 BGM을 무한재생하기 위해 다음 6가지 결정이 필요합니다. 각 질문에 [Answer]: 뒤에 알파벳을 적어주세요.

---

## Question 1
BGM 재생 시작 시점은 언제로 할까요? (호스트 화면 priming 기준 — 브라우저 autoplay 정책상 호스트가 한 번 클릭한 후에만 재생 가능)

A) 호스트 메인 메뉴에서 "방 개설" 버튼을 누른 직후부터 (PublicView 진입 시 자동 시작)
B) 게임이 실제로 시작(PhaseChanged → INTRO/DAY1)된 후부터
C) 로비(LOBBY) 진입 시점부터, 게임 종료 후에도 계속 유지
D) Other (please describe after [Answer]: tag below)

[Answer]: A 

---

## Question 2
음소거/볼륨 조절 UI는 어떻게 구성할까요?

A) 별도 BGM 토글 버튼 신설 (기존 VoiceToggle 옆에 음표 아이콘 등으로 추가)
B) 기존 VoiceToggle 하나로 효과음 + BGM을 동시에 on/off
C) UI 토글 없이 항상 켜진 상태(저볼륨 고정)로 두고, 사용자는 OS/브라우저 볼륨으로만 제어
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 3
효과음(announcement cue) 재생 중 BGM "덕킹"(자동 음량 감소) 처리를 할까요?

A) 덕킹 적용 — cue 재생 중 BGM 볼륨을 절반(예: 0.2 → 0.1)으로 낮추고 cue 종료 시 원복
B) 덕킹 없음 — BGM은 항상 같은 저볼륨으로 깔리고 cue가 그대로 위에 얹힘
C) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 4
BGM 기본 볼륨은 어느 수준으로 할까요? (HTMLAudioElement.volume 값, 0.0~1.0)

A) 0.15 (매우 잔잔, 효과음 대비 충분히 낮음)
B) 0.25 (적당히 들리는 배경음)
C) 0.40 (비교적 또렷한 배경음)
D) Other (please describe after [Answer]: tag below — 예: 0.3)

[Answer]: A

---

## Question 5
호스트가 일시정지(Pause) / 게임 종료(GameEnded) / 호스트 새로고침(BFCache 복원·재접속) 시 BGM은 어떻게 동작할까요?

A) Pause: 그대로 유지 / GameEnded: 그대로 유지 / 새로고침 후: priming 다시 필요(첫 사용자 제스처 후 자동 재시작)
B) Pause: 일시정지 / GameEnded: 정지 / 새로고침 후: priming 다시 필요
C) Pause: 일시정지 / GameEnded: 그대로 유지 / 새로고침 후: priming 다시 필요
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 6
BGM 자산이 누락되거나 디코딩 실패할 경우 동작 방침은? (`useAudioCueQueue` 의 graceful skip 패턴 참고)

A) 콘솔 경고만 출력하고 게임 진행은 무중단 (효과음 큐는 영향 없음)
B) 콘솔 경고 + UI 배지로 호스트에게 BGM 비활성 상태를 안내
C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 답변 후 다음 단계
1. 이 파일에 답변을 채워주시면, AI 가 답변을 audit.md 에 기록합니다.
2. `iteration10-bgm-requirements.md` v1.0 초안을 작성하고 사용자 승인 게이트를 띄웁니다.
3. 승인 후 Workflow Planning(`iteration10-execution-plan.md`) 으로 진행합니다.

# Iteration 7 — Requirements Intake Questions (호스트 첫 페이지)

> 사용자 원본 요청 (2026-04-29T17:55Z):
>
> "호스트 화면의 첫 페이지를 만듭니다. 첫 페이지에서는 다음과 같은 메뉴가 준비되어 있습니다.
> 1. 게임 시작 (방이 개설됨)
> 2. 설정 (마피아 수, 플레이어 수 등의 게임 설정을 할 수 있습니다.)"

## 현재 구조 요약

- 호스트가 `/public`에 접속 → host claim 성공 시 곧바로 단일 화면이 노출됨.
- 그 화면에는 **옵션 입력(최대 인원·마피아 수)** + **"♠ 방 개설" 버튼**이 같이 들어 있음 (`web/src/views/PublicView/PublicView.tsx` line 134~189).
- 이번 변경은 그 단일 화면을 **메인 메뉴(2개 버튼)**로 바꾸는 것이며, 분리되는 흐름과 적용 시점을 결정해야 합니다.

## Question 1
"게임 시작" 버튼을 눌렀을 때의 동작은 무엇이어야 하나요?

A) 즉시 `host:open-room` 송신 → 현재 저장된(또는 기본) 옵션으로 LOBBY 개설, 메인 페이지 → 진행 화면 전환
B) 확인 모달("이 설정으로 시작합니다") 표시 → 확인 후 `host:open-room` 송신
C) "게임 시작" 클릭 시 즉시 LOBBY 개설하되, 만약 옵션이 한 번도 저장된 적 없으면 자동으로 설정 화면으로 우선 이동
D) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question 2
"설정" 화면에서 노출할 항목 범위는 어디까지인가요? (현재 `Options` 타입은 9개 필드를 가짐)

A) 핵심 2개만: 마피아 수(`mafiaCount`) + 최대 인원(`maxPlayers`) — 현재 화면과 동일한 범위
B) 핵심 + 토론/소개 시간: 위 2개 + 자기소개 시간(`introSecondsPerPlayer`) + 토론 시간(`discussionSeconds`)
C) 전부 노출: 위 + 야간 단계별 시간(`nightMafia/police/doctorSeconds`) + 의사 자가치료(`doctorSelfHealAllowed`) + 음성 안내(`announcementVoiceOn`)
D) Other (please describe after [Answer]: tag below)

[Answer]: C

## Question 3
"설정" 화면 진입 방식은 어떻게 할까요?

A) 같은 라우트(`/public`) 내 화면 전환(컴포넌트 스왑) — 새로 라우트를 추가하지 않음
B) 별도 라우트(`/public/settings`) — `react-router` 경로 신설
C) 모달/오버레이 — 메인 메뉴 위에 다이얼로그로 띄움
D) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question 4
"설정" 화면에서 변경한 값의 저장/적용 시점은 어떻게 할까요?

A) "저장 후 메인으로" 버튼 1개 — 명시적으로 저장해야만 메인 메뉴의 "게임 시작"이 그 값을 사용
B) 자동 저장 — 입력이 바뀌면 메모리 상 옵션이 즉시 갱신되고, "메인으로" 버튼은 단순 뒤로가기
C) "저장" + "취소" 두 버튼 — 명시적 저장/취소 분리
D) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question 5
설정값을 페이지 새로고침/재접속 후에도 유지하나요?

A) 유지하지 않음 — 새로고침하면 기본값으로 복귀 (현재와 동일, 메모리만)
B) `localStorage`에 저장 — 동일 브라우저에서는 새로고침 후에도 유지
C) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question 6
"게임 시작" 후, 방이 종료되어 호스트가 메인 메뉴로 돌아왔을 때 메뉴는 어떻게 동작하나요?

A) "게임 시작" + "설정" 그대로 표시 — 다시 시작 가능 (현재의 종료 후 흐름과 자연 연결)
B) 메뉴는 동일하지만 종료된 게임의 결과 요약(승자 등) 배너를 메인 메뉴 위에 같이 표시
C) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question 7
"설정" 화면에서 권장값 가이드(예: 현재의 "※ 권장하지 않는 설정입니다" 경고)는 어떻게 처리할까요?

A) 유지 — 권장 범위에서 벗어나면 동일한 인라인 경고를 그대로 표시
B) 강화 — 권장 범위 외 값일 때 저장 버튼을 비활성화 (잘못된 조합 차단)
C) 제거 — 호스트가 자유롭게 입력하도록 가이드 문구 삭제
D) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question 8
백엔드(Go) 변경이 필요한 범위는 어디까지인가요?

A) 변경 없음 — 첫 페이지는 순수 U5(웹 프론트) 변경, 기존 `host:open-room` 페이로드만 사용
B) `Options` 직렬화 사전 검증 강화만 — U1 validation 보강(예: 음성/시간 필드 범위 체크)
C) 신규 wire 메시지(`host:save-options` 등) 추가 — 서버에 옵션 사전 저장 후 시작 시 재사용
D) Other (please describe after [Answer]: tag below)

[Answer]: C

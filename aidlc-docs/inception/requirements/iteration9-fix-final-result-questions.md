# Iteration 9 — 최종 결과 발표 → 승리 화면 전환 결함 수정 질문지

> **사용자 결함 보고 (2026-04-30)**
>
> 투표 결과 또는 전날 밤의 결과가 호스트 화면에서 보여지고 다음 페이즈로 진행되었으면 좋겠습니다.
> 왜: 투표 또는 전날 밤의 액션으로 승리자가 결정되는 경우, 화면에서 바로 마피아 또는 시민의 승리 화면이 나옵니다. 게임의 흥미를 지속하기 위해 투표 결과 또는 전날 밤의 결과가 먼저 공지되었으면 좋겠습니다.

## 결함 컨텍스트 (코드 인용)

- `internal/game/tally.go::applyElimination` 가 `Eliminated` 이벤트 직후 같은 batch 안에서 `e.checkEnd()` 를 호출 → 승리 조건 충족 시 `GameEnded` 가 동일 batch 의 마지막 이벤트로 추가됨.
- `internal/game/resolve_night.go::resolveNight` 가 `PhaseChanged{DAY}` + `DeathAnnounced`/`PeacefulNight` 를 emit 한 직후 같은 batch 안에서 `e.checkEnd()` 호출 → 승리 조건 충족 시 `GameEnded` 가 같은 batch 마지막에 추가됨.
- U5 reducer 는 `GameEnded` 를 받자마자 `state.phase` 를 `END` 로 갱신 → `PlayerView` 가 `EndScreen` 으로 전환, `PublicView` 의 `SubtitleArea` 도 마지막 자막 갱신 직후 EndScreen 배경으로 덮임. mp3 cue (`eliminated.mafia` / `death.announced` ~3s) 도 `audioCues` FIFO 에서는 재생되지만 시각적으로는 결과를 거의 볼 수 없는 상태.

## 결정해야 할 모호점

답변은 각 질문 아래의 `[Answer]:` 태그에 알파벳 1개로 적어주세요. (해당 옵션이 없으면 마지막 X) Other 선택 후 자유 기술)

---

## Question 1 — 적용 트리거 범위
승리 발표를 지연시킬 트리거를 어디까지 포함하시겠습니까?

A) 두 경로 모두 — VOTE/RECOUNT 의 처형(`Eliminated` → 승리) 과 NIGHT → DAY 사망 발표(`DeathAnnounced`/`PeacefulNight` → 승리) 둘 다.
B) VOTE/RECOUNT 만 — 처형으로 게임이 끝날 때만 지연.
C) NIGHT → DAY 만 — 사망 발표로 게임이 끝날 때만 지연.
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 2 — 결과 발표 후 EndScreen 노출까지 대기 시간
"결과 안내 자막이 화면에 보인 뒤 → 승리 화면으로 전환" 사이에 몇 초의 버퍼를 두시겠습니까? (Iteration 8 의 `defaultDayIntroSeconds = 5` 와 일관된 정수 초 단위)

A) 5초 (Iter8 사망 발표 버퍼와 동일 — 공식 cue + 짧은 여백)
B) 8초
C) 10초
D) 15초
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 3 — `end.mafia` / `end.citizen` 음성 cue 의 위치
승리 안내 cue (`end.mafia` / `end.citizen`) 는 언제 들리도록 하시겠습니까?

A) 결과 자막 + 결과 cue (eliminated/death) → 대기 → EndScreen 전환 시 `end.*` cue 재생 (현재 흐름 유지, 자막 노출만 늘림)
B) 결과 자막 + 결과 cue → 대기 → EndScreen 전환과 동시에 `end.*` cue 재생 (대기 종료 직후 즉시)
C) 결과 cue 와 `end.*` cue 사이에 추가 짧은 무음 1~2초
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 4 — Pause/Resume 와 상호작용
신규 "결과 안내 → 승리 화면" 버퍼는 호스트 Pause 토글의 영향을 받아야 합니까?

A) 받음 — Pause 시 카운트다운 정지, Resume 시 남은 시간만큼 EndScreen 전환 지연 (Iter5 Pause 정책과 동일)
B) 받지 않음 — 게임 결과는 이미 결정되었으므로 Pause 와 무관하게 일정 시간 후 자동 전환
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 5 — 호스트 강제 종료(`HOST_FORCE_END`) 의 처리
호스트가 메뉴에서 게임을 강제 종료하여 `EndHostForceEnd` 가 발화되는 경우에도 동일한 버퍼를 적용하시겠습니까?

A) 강제 종료는 즉시 EndScreen — 자연 결판(Vote/Night)만 버퍼 적용
B) 강제 종료에도 같은 버퍼 적용 — 일관성 우선
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 6 — 신규 Phase 도입 여부 (구현 전략)
서버-주도 일관성을 위해 어떤 구현 전략을 선호하십니까?

A) 신규 Phase 도입 안 함 — `GameEnded` emit 시점만 Tick 으로 5s 지연 (State.Phase 는 이전 phase 유지, `EndAt`/`PendingEndAt` 같은 새 필드 1~2개 추가). 화면은 기존 SubtitleArea/EndScreen 그대로 동작.
B) 신규 Phase `RESULT` 도입 — VOTE/RECOUNT/NIGHT-resolve 직후 `PhaseChanged{RESULT}` 로 진입, 5s 후 Tick 이 `GameEnded` + `PhaseChanged{END}` 로 전환. 화면은 RESULT phase 동안 마지막 자막을 강조 표시.
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 7 — 호스트 vs 플레이어 화면 일관성
EndScreen 전환 지연을 적용할 화면 범위는?

A) 모든 화면 (호스트 PublicView + 모든 PlayerView + 전광판 PUBLIC) 동일 시점 전환 — 서버에서 GameEnded 시점을 미루므로 자동 일치
B) 호스트만 — 플레이어는 즉시 EndScreen, 호스트만 결과 자막 5s 유지
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 8 — Voice Off 또는 mp3 누락 환경에서의 동작
호스트가 음성을 끄거나 cue mp3 가 누락되어 graceful skip 되는 경우, 결과 자막 표시 시간은 동일해야 합니까?

A) 동일 — 자막은 항상 동일 시간 유지 (음성과 무관하게 시각적 일관성 확보)
B) 음성 길이에 맞춰 동적 조정 — cue 가 짧거나 없으면 더 짧게
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## (참고) 사용자 답변 가이드

- **모든 질문에 알파벳 1개**로 답변해 주세요. 위 질문지의 기본값(예시) 이 본인의 의도와 맞으면 그대로 두셔도 됩니다.
- 답변 완료 후 "완료" 또는 "done" 으로 알려주세요. AI 가 답변을 읽고 모호점/모순이 없는지 다시 검토 후 모호점이 남으면 추가 질문 파일을 만듭니다.
- 모순 없이 정리되면 `requirements/iteration9-fix-final-result-requirements.md` 를 작성하고 사용자 승인 게이트를 진행합니다.

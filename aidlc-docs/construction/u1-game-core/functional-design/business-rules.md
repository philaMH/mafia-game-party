# Business Rules — U1 Game Core

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `domain-entities.md`, `business-logic-model.md`

본 문서는 GameEngine의 사전조건, 검증 규칙, 권한 게이팅, 동률 처리, 에러 분류를 규칙 단위로 정리합니다. 본 규칙은 단위 테스트의 직접 명세가 됩니다.

---

## 1. 액션별 사전조건 (Preconditions)

### 1.1 모든 액션 공통

| ID | 규칙 |
|---|---|
| BR-COMMON-1 | 액션 발신자가 유효한 PlayerID여야 한다 (state.Players에 존재) |
| BR-COMMON-2 | 액션 발신자가 `Alive=true`여야 한다 (사망자 입력 거부) |
| BR-COMMON-3 | 현재 Phase가 그 액션을 수용해야 한다 (각 액션의 §1.2 표 참조) |
| BR-COMMON-4 | Phase=END에서는 어떤 액션도 수용하지 않는다 |

### 1.2 액션별 Phase 호환

| Action | 허용 Phase | 추가 조건 |
|---|---|---|
| `StartGame` | LOBBY | 발신자 = HostID |
| `AdvanceIntro` | INTRO | 발신자 = HostID |
| `SubmitMafiaKill` | NIGHT | 발신자 Role=MAFIA AND 발신자 = MafiaRepresentativeID |
| `SubmitDoctorHeal` | NIGHT | 발신자 Role=DOCTOR |
| `SubmitPoliceCheck` | NIGHT | 발신자 Role=POLICE AND PoliceCheckedThisNight=false |
| `EndNightEarly` | NIGHT | 발신자 = HostID |
| `EndDiscussionEarly` | DAY | 발신자 = HostID |
| `SubmitVote` | VOTE 또는 RECOUNT | (RECOUNT일 땐 Target ∈ VoteCandidates) |
| `ToggleVoice` | LOBBY/INTRO/NIGHT/DAY/VOTE/RECOUNT | 발신자 = HostID |
| `ForceEndGame` | LOBBY/INTRO/NIGHT/DAY/VOTE/RECOUNT | 발신자 = HostID |

---

## 2. Options 검증 규칙 (StartGame 시)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-OPT-1 | 총 인원 N: `6 ≤ N ≤ 12` | FR-1.3 |
| BR-OPT-2 | `MafiaCount ≥ 1` | Q-FD-U1-1-FU2=A |
| BR-OPT-3 | `(N − MafiaCount) ≥ MafiaCount + 1` (시민 진영 인원이 마피아보다 많아야 함) | Q-FD-U1-1-FU2=A |
| BR-OPT-4 | 의사 1명, 경찰 1명 고정 | Q-FD-U1-1-FU=A |
| BR-OPT-5 | `(N − MafiaCount − 2) ≥ 1` (시민(CITIZEN) 최소 1명) | 게임 진행 보장 |
| BR-OPT-6 | `IntroSecondsPerPlayer ≥ 5` (자기소개 최소 시간) | 합리적 하한 |
| BR-OPT-7 | `DiscussionSeconds ≥ 30` (토론 최소 시간) | 합리적 하한 |
| BR-OPT-8 | 기본값: `IntroSecondsPerPlayer=20, DiscussionSeconds=180, DoctorSelfHealAllowed=true, AnnouncementVoiceOn=true` | Q-FD-U1-2/3/7, FR-8.5 |

> BR-OPT-1~5 위반은 `ValidationError(code: "INVALID_OPTIONS", details: ...)` 반환.

---

## 3. 권한 게이팅 규칙

### 3.1 호스트 한정 액션

다음 액션은 발신자가 `state.HostID`와 일치하지 않으면 `PermissionDeniedError`:
- `StartGame`, `AdvanceIntro`, `EndNightEarly`, `EndDiscussionEarly`, `ToggleVoice`, `ForceEndGame`

### 3.2 역할 한정 액션

| 액션 | 필요한 Role | 위반 시 에러 |
|---|---|---|
| `SubmitMafiaKill` | MAFIA + 대표자 | RoleMismatchError 또는 NotRepresentativeError |
| `SubmitDoctorHeal` | DOCTOR | RoleMismatchError |
| `SubmitPoliceCheck` | POLICE | RoleMismatchError |

### 3.3 마피아 대표자 규칙 (Q-FD-U1-4-FU=C, FU2=A)

| ID | 규칙 |
|---|---|
| BR-REP-1 | 게임 시작 시 마피아 ≥ 2이면, 시스템이 마피아 중 1명을 무작위로 선정하여 `MafiaRepresentativeID`에 저장 |
| BR-REP-2 | 게임 시작 시 마피아 = 1이면, 그 1명이 자동 대표자 |
| BR-REP-3 | 대표자만 `SubmitMafiaKill`을 발행할 수 있다 |
| BR-REP-4 | 대표자가 사망(살해 또는 처형)하면, 즉시 살아있는 마피아 중 무작위 1명을 새 대표자로 지정하고 `MafiaRepresentativeReassigned` 이벤트 발행 |
| BR-REP-5 | 마피아 전원 사망 시 `MafiaRepresentativeID = ""` (빈 값). 종료 조건 검사에서 게임 종료. |

---

## 4. 자기소개 (INTRO) 규칙

| ID | 규칙 | 출처 |
|---|---|---|
| BR-INTRO-1 | INTRO는 Day=1에서만 진입 (이후 NIGHT/DAY 사이클은 INTRO 미진입) | 단계 머신 |
| BR-INTRO-2 | 발화자 순서는 `state.Players` 배열 순서 (가입 순) | 안정성 |
| BR-INTRO-3 | 1인당 발화 시간은 `Settings.IntroSecondsPerPlayer` 초 (기본 20) | Q-FD-U1-2=A |
| BR-INTRO-4 | 시간 만료 시 자동으로 다음 발화자 진행 (Q-AD-4=B 자동) | Q-AD-4 |
| BR-INTRO-5 | 호스트는 `AdvanceIntro`로 강제 진행 가능 (Q-AD-4 보조) | Q-AD-4 |
| BR-INTRO-6 | 마지막 발화자 종료 시 NIGHT으로 전이 | 단계 머신 |
| BR-INTRO-7 | 자기소개 중에는 야간 행동·투표 액션 거부 | BR-COMMON-3 |

---

## 5. 야간(NIGHT) 행동 규칙

### 5.1 일반

| ID | 규칙 |
|---|---|
| BR-NIGHT-1 | NIGHT 진입 시 `PendingMafiaTarget = PendingDoctorTarget = PendingPoliceTarget = nil`, `PoliceCheckedThisNight = false`로 초기화 |
| BR-NIGHT-2 | NIGHT은 시간 마감이 없다 (Q-FD-U1-12=B). 모든 야간 행동 입력 또는 호스트의 `EndNightEarly`로 종료 |
| BR-NIGHT-3 | "모든 야간 행동 입력"의 정의: 마피아 대표자가 살해 대상 입력 + 살아있는 의사가 보호 대상 입력 + 살아있는 경찰이 조사 대상 입력 |
| BR-NIGHT-4 | 의사 또는 경찰이 사망한 상태에서는 그 입력을 기다리지 않음 (`hasLivingDoctor()`/`hasLivingPolice()` 체크) |

### 5.2 마피아 살해

| ID | 규칙 |
|---|---|
| BR-MAFIA-1 | `Target`은 살아있어야 함 (`Alive=true`) |
| BR-MAFIA-2 | `Target`은 마피아가 아니어야 함 (마피아끼리 못 죽임) |
| BR-MAFIA-3 | 대표자는 같은 NIGHT 내에 여러 번 입력 가능 — 마지막 입력이 최종(Q-AD-7=Other "마지막 입력 채택"이 합의 모드와 결합된 형태). 매 입력마다 `MafiaTargetSelected` 이벤트 발행 (마피아 비공개) |
| BR-MAFIA-4 | 입력은 Pending에 저장되며, NIGHT 종료 시점에 `resolveNight()`에서 적용 |

### 5.3 의사 보호

| ID | 규칙 |
|---|---|
| BR-DOC-1 | `Target`은 살아있어야 함 |
| BR-DOC-2 | `Doctor == Target`(자가 보호)는 `Settings.DoctorSelfHealAllowed=true`일 때만 허용 (Q-AD-8=A 기본 true) |
| BR-DOC-3 | 같은 NIGHT 내 여러 번 입력 가능 — 마지막 입력이 최종 |
| BR-DOC-4 | 의사의 선택은 비공개 — 이벤트 발행 안 함. Pending에만 저장. |
| BR-DOC-5 | resolveNight에서 `PendingDoctorTarget == PendingMafiaTarget`이면 살해 무효 (보호 성공) |

### 5.4 경찰 조사

| ID | 규칙 |
|---|---|
| BR-POL-1 | `Target`은 살아있어야 함 |
| BR-POL-2 | `Police == Target`(자기 조사) 금지 |
| BR-POL-3 | NIGHT당 1회만 가능 (`PoliceCheckedThisNight=true`로 잠금) |
| BR-POL-4 | 결과 `PoliceResult{Police, Target, Team}`은 즉시 발행 (경찰 비공개). `Team`은 `MAFIA`(Target.Role==MAFIA) 또는 `CITIZEN`(그 외) (Q-FD-U1-6=A 진영만) |

### 5.5 NIGHT → DAY 전이 (resolveNight)

| ID | 규칙 |
|---|---|
| BR-RESOLVE-1 | `PendingMafiaTarget == nil`이면 `PeacefulNight` 발행, victim 없음 |
| BR-RESOLVE-2 | `PendingMafiaTarget != nil` AND `PendingDoctorTarget == PendingMafiaTarget`이면 보호 성공 → `PeacefulNight` |
| BR-RESOLVE-3 | 그 외에는 `victim = PendingMafiaTarget`, `Players[victim].Alive = false`, `DeathAnnounced{victim}` 발행 |
| BR-RESOLVE-4 | victim이 마피아 대표자인 경우 BR-REP-4 적용 (재지정 + 이벤트) |
| BR-RESOLVE-5 | resolveNight 종료 시 `Day++`, `Phase=DAY`, `Deadline = now + DiscussionSeconds`, `PhaseChanged{DAY, Day, Deadline}` 발행 |
| BR-RESOLVE-6 | 종료 조건 검사 (BR-END-*) 후 충족 시 GameEnded 추가 발행 + `Phase=END` |

---

## 6. 낮(DAY) 토론 규칙

| ID | 규칙 | 출처 |
|---|---|---|
| BR-DAY-1 | DAY 진입 시 `Deadline = now + Settings.DiscussionSeconds` (기본 180초) | Q-FD-U1-3=C |
| BR-DAY-2 | Deadline 만료 시 VOTE 진입 (자동, Tick에서 트리거) | 단계 머신 |
| BR-DAY-3 | 호스트는 `EndDiscussionEarly`로 즉시 VOTE 진입 가능 | Q-AD-5=C |
| BR-DAY-4 | DAY 단계는 시간 안내 임계 (30, 10, 0초)에 `DiscussionTimerTick{secondsLeft}` 발행 | NFR-3 사용성 |

---

## 7. 투표(VOTE / RECOUNT) 규칙

| ID | 규칙 | 출처 |
|---|---|---|
| BR-VOTE-1 | 투표권: `Alive=true`인 모든 Player 1표씩 | FR-4.6 |
| BR-VOTE-2 | 같은 NIGHT/VOTE 라운드 내 여러 번 입력 시 마지막 입력이 최종 (`Votes[voter] = target`) | 사용성 |
| BR-VOTE-3 | `Target`은 살아있어야 함 | BR-COMMON |
| BR-VOTE-4 | RECOUNT에서는 `Target ∈ VoteCandidates`만 허용 (Q-FD-U1-5=A 동률 후보자만 대상) | Q-FD-U1-5=A |
| BR-VOTE-5 | 모든 살아있는 Player가 입력 완료 시 즉시 `tally()` 실행 | 단계 머신 |
| BR-VOTE-6 | tally 결과 단일 최다 → 처형, 동률 → RECOUNT(VoteRound=2) 또는 무처형(VoteRound=2였던 경우) | FR-4.6 |
| BR-VOTE-7 | 처형 시 `Players[eliminated].Alive=false`, `Eliminated` 이벤트 발행 (역할 공개) | 사용성 |
| BR-VOTE-8 | 처형 후 종료 조건 검사 (BR-END) → 미충족 시 NIGHT 진입 | 단계 머신 |
| BR-VOTE-9 | 무처형 (재투표 동률) 후 종료 조건 미충족 시 NIGHT 진입 | 단계 머신 |
| BR-VOTE-10 | VOTE/RECOUNT는 시간 마감 없음 (모두 입력 또는 호스트 `ForceEndGame`) | 단순화 |

---

## 8. 종료 조건 (BR-END)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-END-1 | `liveMafiaCount == 0` → 시민 승, `Phase=END`, `Winner=CITIZEN`, `EndReason=CITIZEN_WIN` | FR-5.1 |
| BR-END-2 | `liveMafiaCount >= liveCitizenSideCount` → 마피아 승, `EndReason=MAFIA_WIN` | FR-5.2 |
| BR-END-3 | `ForceEndGame` 액션 → `Phase=END`, `Winner=nil`, `EndReason=HOST_FORCE_END` | 호스트 통제 |
| BR-END-4 | END 진입 시 `GameEnded{Winner, EndReason, Reveal: Players}` 발행 (모든 플레이어 역할 공개) | 사용성 |
| BR-END-5 | END 진입 후 어떤 액션도 수용하지 않음 (BR-COMMON-4) | 종착 |

> 주의: 종료 검사는 매 NIGHT/VOTE 처리 직후에 수행 (NIGHT은 살해 발생 후, VOTE은 처형 후).

---

## 9. 키워드 부여 규칙 (BR-KW)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-KW-1 | RoleAssigner는 역할별로 KeywordPool.Pick을 1번씩만 호출 | Q-FD-U1-8=A |
| BR-KW-2 | 같은 역할의 모든 Player는 같은 Keyword를 받음 | Q-FD-U1-8=A |
| BR-KW-3 | KeywordPool은 외부화 가능한 인터페이스로 추상화 (FR-7.1) | FR-7.1 |
| BR-KW-4 | 기본 풀(`internal/game/keyword.go`)이 부재한 역할(예: 비활성화된 역할 — 본 시스템에선 없음)이 있으면 빈 문자열 부여 | 견고성 |

---

## 10. 에러 분류

GameEngine의 Apply는 다음 에러를 반환할 수 있다 (분류 표):

| 에러 코드 | 의미 | 대표 발생 상황 |
|---|---|---|
| `VALIDATION_ERROR` | 입력 검증 실패 | StartGame Options 위반 |
| `WRONG_PHASE_ERROR` | 현재 단계에서 허용되지 않는 액션 | INTRO 중 SubmitVote |
| `PERMISSION_DENIED_ERROR` | 호스트 한정 액션을 비호스트가 발신 | 비호스트가 ForceEndGame |
| `ROLE_MISMATCH_ERROR` | 역할 불일치 | 시민이 SubmitMafiaKill |
| `NOT_REPRESENTATIVE_ERROR` | 마피아지만 대표자가 아님 | 비대표 마피아가 SubmitMafiaKill |
| `DEAD_PLAYER_ERROR` | 사망자 입력 또는 사망자 대상 | Alive=false인 플레이어가 SubmitVote / 사망자에게 투표 |
| `ALREADY_DONE_ERROR` | 야간 1회 제한 위반 | 경찰이 야간 2번 조사 시도 |
| `INVALID_TARGET_ERROR` | 대상이 규칙 위반 | 마피아가 마피아를 살해 시도, 경찰이 자기 조사 시도, RECOUNT에서 후보 외 투표 |
| `UNKNOWN_PLAYER_ERROR` | PlayerID가 state.Players에 없음 | 알 수 없는 voter |

> 모든 에러는 **상태 변경 없이** 즉시 반환되어야 함 (불변식: Apply가 에러를 반환할 때 state는 호출 직전과 동일).

---

## 11. 결정성·동시성 규칙

| ID | 규칙 |
|---|---|
| BR-CONC-1 | GameEngine은 단일 스레드 가정 — 동시 호출 안전성은 U2(SessionManager 단일 GM 락)이 보장 |
| BR-CONC-2 | Tick은 멱등 — 동일 `now`로 N번 호출되어도 첫 호출 외에는 no-op |
| BR-CONC-3 | Apply는 단일 액션 처리 — 한 번에 한 액션 |
| BR-CONC-4 | 무작위성은 생성자 주입 `rng io.Reader`만 사용 — 단위 테스트 시드 가능 |

---

## 12. FR/NFR 추적성

| 요구사항 | 본 문서 규칙 |
|---|---|
| FR-1.3 (인원 6~12) | BR-OPT-1 |
| FR-2.1 (역할 배분) | BR-OPT-2~5, RoleAssigner 알고리즘 |
| FR-2.2 (무작위 키워드) | BR-KW-1, BR-CONC-4 |
| FR-2.3 (본인 비공개) | (가시성 정책 — domain-entities.md §5) |
| FR-3.1 (키워드 풀) | BR-KW-1~4 |
| FR-3.3 (자기소개 시간 자동) | BR-INTRO-1~7 |
| FR-4.1 (단계 전이) | business-logic-model.md §1, §4, §5 |
| FR-4.2 (밤 행동) | BR-NIGHT-*, BR-MAFIA-*, BR-DOC-*, BR-POL-* |
| FR-4.3 (마피아 합의) | BR-REP-1~5 (Q-FD-U1-4-FU=C 시스템 무작위 고정) |
| FR-4.4 (의사 자가 보호) | BR-DOC-2 |
| FR-4.5 (토론 + 호스트 조기 종료) | BR-DAY-1~4 |
| FR-4.6 (동률 처리) | BR-VOTE-1~10 |
| FR-5.1 (시민 승) | BR-END-1 |
| FR-5.2 (마피아 승) | BR-END-2 |
| FR-7.1 (외부화 인터페이스) | BR-KW-3 |
| NFR-1 (안정성·복원) | business-logic-model.md §9 (Snapshot/Restore) |
| NFR-6 (도메인 분리) | BR-CONC-1, 외부 I/O 0 |

---

## 13. 검증 체크리스트

- [x] 모든 액션에 사전조건이 명시됨 (§1)
- [x] StartGame Options가 코드 가능 수준의 검증식으로 표현됨 (§2)
- [x] 권한 게이팅이 액션마다 명확 (§3)
- [x] 마피아 대표자 결정·재지정 규칙이 모든 시나리오 커버 (§3.3)
- [x] 동률 처리가 라운드별로 명확 (§7)
- [x] 종료 조건이 enum으로 분류됨 (§8)
- [x] 에러 코드가 분류표로 정리됨 (§10)
- [x] 모든 Primary FR/NFR이 규칙으로 매핑됨 (§12)

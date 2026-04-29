# Domain Entities — U1 Game Core

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1, `application-design/component-methods.md` §공용 타입, `plans/u1-game-core-functional-design-plan.md` §3-4

본 문서는 U1 Game Core의 도메인 엔티티(타입)를 기술 비종속(technology-agnostic) 관점에서 정의합니다. 코드 시그니처는 Generation 단계에서 확정되며, 본 문서는 **개념 모델·필드 의미·불변식**에 집중합니다.

> 모든 도메인 엔티티는 U1이 단일 정의처입니다 (Q-UG-5=A). 다른 단위(U2~U5)는 import만 합니다.

---

## 1. 식별자 / 열거형

### 1.1 PlayerID
- **타입**: 불투명 문자열 (UUID 또는 안정적 토큰)
- **생성**: PlayerID는 U2(Session) 책임으로 발급. U1은 받기만 함.
- **수명**: 게임 한 판 동안 불변. 재연결 시에도 동일 PlayerID로 식별.

### 1.2 Role
- **값**: `MAFIA`, `CITIZEN`, `DOCTOR`, `POLICE`
- **의미**: 게임 시작 시 RoleAssigner가 부여, 게임 종료까지 변경 불가.

### 1.3 Team (진영)
- **값**: `MAFIA`, `CITIZEN`
- **매핑**: `MAFIA → MAFIA 진영`, `DOCTOR/POLICE/CITIZEN → CITIZEN 진영` (Q-FD-U1-6=A 결정 — 경찰 조사 결과는 진영만 노출)

### 1.4 Phase (단계)
- **값**: `LOBBY`, `INTRO`, `NIGHT`, `DAY`, `VOTE`, `RECOUNT`, `END`
- **순서**: `LOBBY → INTRO → NIGHT → DAY → VOTE [→ RECOUNT] → NIGHT → ...` (END로 종료)

### 1.5 EndReason
- **값**: `MAFIA_WIN`, `CITIZEN_WIN`, `HOST_FORCE_END`
- **의미**: 게임 종료 사유. `HOST_FORCE_END`는 호스트가 강제 종료한 경우(요구사항: 호스트 통제 명령 게이팅).

---

## 2. Player

| 필드 | 타입 | 의미 | 비고 |
|---|---|---|---|
| `ID` | PlayerID | 고유 식별자 | 게임 한 판 불변 |
| `Name` | string | 닉네임 | 표시용. 게임 내 중복 금지(U2가 강제) |
| `Alive` | bool | 생존 여부 | 처형/살해 시 false |
| `Role` | Role | 역할 | **비공개**(본인·동맹 마피아만) |
| `Keyword` | string | 자기소개 키워드 | **비공개**(본인만) |

### 불변식 (Invariants)
- `Alive=false`인 Player는 모든 액션 입력 거부 (사망자 입력 금지)
- `Role`은 게임 시작 후 변경 불가
- `Keyword`는 게임 시작 후 변경 불가
- 같은 게임 내에서 동일 역할의 Player들은 **같은 Keyword**를 가짐 (Q-FD-U1-8=A)

---

## 3. Options (게임 시작 옵션)

호스트가 게임 시작 시 결정. State.Settings로 저장.

| 필드 | 타입 | 기본값 | 의미 | 출처 |
|---|---|---|---|---|
| `MafiaCount` | int | 인원에 따라 표준안 (6→1, 7~9→2, 10~12→3) | 마피아 수 | Q-FD-U1-1-FU=A |
| `IntroSecondsPerPlayer` | int | **20** | 자기소개 1인당 초 | Q-FD-U1-2=A |
| `DiscussionSeconds` | int | **180** | 토론 단계 기본 시간 | Q-FD-U1-3=C |
| `DoctorSelfHealAllowed` | bool | **true** | 의사 자가 보호 허용 | Q-AD-8=A, Q-FD-U1-7=A (제약 없음) |
| `AnnouncementVoiceOn` | bool | **true** | TTS ON 기본값 | FR-8.5 |

### 검증 규칙 (Q-FD-U1-1-FU2=A 고정 하한)
- 총 인원 N: `6 ≤ N ≤ 12`
- `MafiaCount ≥ 1`
- `시민 진영 인원 = N − MafiaCount ≥ MafiaCount + 1` (= 시민 진영이 마피아보다 항상 많음)
- DOCTOR 1명, POLICE 1명 (고정)
- CITIZEN 인원 = `N − MafiaCount − 2`

#### 인원별 허용 MafiaCount 범위

| 총 인원 N | 표준 기본값 | 허용 범위 (시민 진영 ≥ 마피아+1) |
|---|---|---|
| 6 | 1 | 1 ~ 2 (2일 때 시민3, 3≥3 ❌ → 1만 허용) — **결과: 1 only** |
| 7 | 2 | 1 ~ 2 |
| 8 | 2 | 1 ~ 3 |
| 9 | 2 | 1 ~ 3 |
| 10 | 3 | 1 ~ 4 |
| 11 | 3 | 1 ~ 4 |
| 12 | 3 | 1 ~ 5 |

> 검증: N=6, M=2 → 시민 진영 = 4, 마피아+1 = 3 → 4 ≥ 3 ✅. 그러나 의사+경찰 2명 차감 후 시민=4-2=2 < 마피아 2 → 위험. 본 검증은 **진영 단위**이므로 OK. 그러나 게임 밸런스상 N=6, M=1로 권장(기본값).

---

## 4. Action (외부 입력)

GameEngine.Apply의 입력. `interface{}` sealed via type switch (Go 의사 코드 기준).

| Action | 발행자 | 사전조건 (Phase) | 페이로드 |
|---|---|---|---|
| `StartGame` | HostID | `LOBBY` | `Options` |
| `AdvanceIntro` | HostID | `INTRO` | (없음) — 호스트 강제 진행 (보조 수단) |
| `SubmitMafiaKill` | 마피아 대표자 | `NIGHT` | `Mafia: PlayerID, Target: PlayerID` |
| `SubmitDoctorHeal` | DOCTOR | `NIGHT` | `Doctor: PlayerID, Target: PlayerID` |
| `SubmitPoliceCheck` | POLICE | `NIGHT` | `Police: PlayerID, Target: PlayerID` |
| `EndDiscussionEarly` | HostID | `DAY` | (없음) — Q-AD-5=C |
| `EndNightEarly` | HostID | `NIGHT` | (없음) — **Q-FD-U1-12=B** 야간 마감 없는 대신 호스트 강제 종료 |
| `SubmitVote` | 살아있는 모든 PlayerID | `VOTE` 또는 `RECOUNT` | `Voter: PlayerID, Target: PlayerID` |
| `ToggleVoice` | HostID | any | `On: bool` (FR-8.5) |
| `ForceEndGame` | HostID | any (END 제외) | (없음) — 호스트 강제 종료 |

### 권한 게이팅 규칙
- `HostID`만 발행 가능한 액션은 발신자가 `State.HostID`와 일치하지 않으면 거부
- 역할 한정 액션(`SubmitMafiaKill`, `SubmitDoctorHeal`, `SubmitPoliceCheck`)은 발신자 Role 확인
- 모든 액션은 발신자가 `Alive=true`여야 함 (사망자 입력 거부)
- `SubmitMafiaKill`은 발신자가 **그 게임의 마피아 대표자**(아래 §6)와 일치해야 함

---

## 5. Event (도메인 이벤트, 출력)

GameEngine.Apply / Tick의 결과로 발생. U2의 AnnouncementService가 안내 메시지로 변환, U3의 WSHub가 라우팅.

| Event | 페이로드 | 가시성 |
|---|---|---|
| `GameStarted` | `State` (마스킹 없는 초기 상태) | 공용 (단, Player.Role/Keyword는 클라이언트 송신 시 마스킹) |
| `PhaseChanged` | `Phase, Day, Deadline` | 공용 |
| `RoleRevealedToPlayer` | `PlayerID, Role, Keyword` | **비공개** (해당 PlayerID 1인) |
| `MafiaCohortRevealed` | `PlayerIDs` (마피아 전원), `RepresentativeID` | **비공개** (모든 마피아) — Q-FD-U1-4-FU=C 대표자 통지 |
| `IntroSpeakerChanged` | `PlayerID, SecondsLeft` | 공용 |
| `MafiaTargetSelected` | `RepresentativeID, Target: PlayerID` | **비공개** (모든 마피아) — 현재 대표자의 선택 표시 |
| `PoliceResult` | `Police, Target, Team` | **비공개** (POLICE 1인) |
| `DeathAnnounced` | `Victim: PlayerID` | 공용 |
| `PeacefulNight` | (없음) | 공용 |
| `DiscussionTimerTick` | `SecondsLeft` | 공용 (특히 SecondsLeft=30, 10, 0에서 안내) |
| `VoteTallied` | `Counts: map[PlayerID]int, Eliminated: *PlayerID, Recount: bool` | 공용 |
| `Eliminated` | `PlayerID, Role` (사후 공개 정책) | 공용 (역할은 잠정 공개; FD 결정: **공개**) |
| `MafiaRepresentativeReassigned` | `OldID, NewID` | **비공개** (모든 마피아) — 대표자 사망 시 |
| `GameEnded` | `Winner: Team, EndReason, Reveal: []Player` (전원 역할 공개) | 공용 |
| `VoiceToggled` | `On: bool` | 공용 |

### 가시성 정책 정리
- **공용**: 모든 PublicView + 살아있는 모든 PlayerView
- **비공개(개인)**: 특정 PlayerID 1인
- **비공개(역할군)**: 특정 Role 보유 모든 살아있는 Player

> 가시성 라우팅은 U3가 책임. U1은 이벤트에 가시성 메타데이터를 부여만 함 (Generation 단계에서 메타데이터 전달 방식 확정 — 가능한 형태: `EventEnvelope{Event, Visibility}` 또는 인터페이스 메서드).

---

## 6. State (게임 한 판의 전체 상태)

GameEngine이 보유. U2가 Snapshot/Restore로 영속화.

```
State {
  GameID                  string         // 게임 식별자 (U2 발급)
  Phase                   Phase
  Day                     int            // 1부터 증가
  Players                 []Player       // 가입 순서 보존
  HostID                  PlayerID       // Q-AD-6=B
  Settings                Options
  StartedAt               time.Time
  Deadline                time.Time      // 현재 단계 종료 예정 시각 (없으면 zero)

  // 진행 상태
  IntroSpeakerIdx         int            // 현재 자기소개 발화자 인덱스 (Phase=INTRO일 때)
  IntroSpeakerStartedAt   time.Time      // 현재 발화자 시작 시각

  // 야간 행동 누적 (현재 야간만)
  MafiaRepresentativeID   PlayerID       // 현재 마피아 대표자 (Q-FD-U1-4-FU=C)
  PendingMafiaTarget      *PlayerID      // 대표자가 선택한 살해 대상 (없으면 nil)
  PendingDoctorTarget     *PlayerID      // 의사가 선택한 보호 대상
  PendingPoliceTarget     *PlayerID      // 경찰이 선택한 조사 대상
  PoliceCheckedThisNight  bool           // 경찰이 조사 완료했는지 (재조사 방지)

  // 투표 누적 (현재 투표 라운드만)
  Votes                   map[PlayerID]PlayerID  // voter → target
  VoteRound               int            // 1=초기 투표, 2=재투표(RECOUNT)
  VoteCandidates          []PlayerID     // RECOUNT 시 동률 후보 (Q-FD-U1-5=A)

  // 종료
  Winner                  *Team          // 종료 시 set
  EndReason               *EndReason
}
```

### 불변식 (Invariants)
- `Phase=END`는 종착 상태 — 어떤 액션으로도 벗어나지 않음
- `Phase=NIGHT` 시작 시 `Pending* = nil`, `PoliceCheckedThisNight = false`로 초기화
- `Phase=VOTE` 시작 시 `Votes = {}`, `VoteRound = 1`로 초기화
- `Phase=RECOUNT` 시작 시 `Votes = {}`, `VoteRound = 2`, `VoteCandidates`는 직전 Tally의 동률 후보로 set
- `Day` 증가 시점: `NIGHT → DAY` 전이 시 (NIGHT은 같은 Day의 후반부)
- `MafiaRepresentativeID`가 사망 처리될 때, 즉시 살아있는 마피아 중 무작위 1명으로 재지정 (Q-FD-U1-4-FU2=A)
- `Players` 배열은 한 번 정해진 순서를 유지 (자기소개 순서 안정성)

### 클라이언트 송신 시 마스킹 (U2/U3 공동 책임)
U1은 마스킹 책임이 없으나, State 직렬화 시 가시성 정책을 위한 메타데이터를 제공해야 함.
- 공용 화면: `Players[*].Role = ""`, `Players[*].Keyword = ""`로 마스킹
- 본인 화면: 본인의 Role/Keyword만 노출
- 마피아 동맹: 다른 마피아의 Role도 노출 (Keyword는 같으므로 동일)

---

## 7. PendingActions (요약 뷰)

`State`의 `Pending*` 필드를 묶어 표현하는 보조 타입(코드 단계에서 명명):

```
PendingActions {
  MafiaTarget   *PlayerID
  DoctorTarget  *PlayerID
  PoliceTarget  *PlayerID
}
```

> 관측자 화면(다른 마피아의 PlayerView)에서 "현재 대표자의 선택"을 표시하기 위한 의도. 의사/경찰의 Pending은 본인에게만 비공개 표시.

---

## 8. KeywordPool (FR-7.1 외부화 인터페이스)

```
KeywordPool {
  Pick(role Role, rng) string
}
```

### 정책
- 역할별 풀에서 무작위 1개 추출.
- 같은 게임 내 같은 역할은 같은 Keyword (Q-FD-U1-8=A) → `Pick`은 RoleAssigner가 역할당 1번만 호출, 그 결과를 모든 동일 역할 Player에게 부여.
- 초기 콘텐츠는 **AI 자동 생성**(Q-FD-U1-9=A) — 본 문서 §10에 초안.

---

## 9. RoleAssigner (인터페이스 의미)

```
RoleAssigner.Assign(playerIDs, options, rng) → Assignments
  Assignments {
    PlayerRoles    map[PlayerID]Role
    PlayerKeywords map[PlayerID]string
    MafiaIDs       []PlayerID            // 통지용
    Representative PlayerID              // 마피아 대표자 (마피아 ≥ 2일 때)
  }
```

### 알고리즘
1. `Options` 검증 (§3 검증 규칙)
2. PlayerIDs 무작위 셔플
3. 앞에서부터: MafiaCount명 → MAFIA, 그 다음 1명 → DOCTOR, 그 다음 1명 → POLICE, 나머지 → CITIZEN
4. 각 역할마다 `KeywordPool.Pick(role, rng)`로 1개 추출 → 모든 동일 역할 Player에게 동일 부여
5. MafiaCount ≥ 2일 때, MAFIA들 중 무작위 1명을 `Representative`로 지정 (Q-FD-U1-4-FU=C)
6. MafiaCount = 1일 때, 그 1명이 자동 Representative

> 무작위성: `rng`는 `crypto/rand` 기반 (Q-FD-U1-10=A). 단위 테스트에서는 시드 가능한 `io.Reader`로 주입.

---

## 10. 기본 키워드 풀 초안 (FR-7.1, Q-FD-U1-9=A)

본 풀은 **`internal/game/keyword.go`의 default constant**로 임베드됩니다. 운영자는 추후 외부 파일(YAML/JSON)로 교체 가능.

### MAFIA (어둡고 비밀스러운 분위기 — 자기소개 시 의심을 부르는 단어)

```
그림자, 침묵, 가면, 어둠, 속삭임, 숨결, 안개, 비밀, 거울, 골목,
새벽, 밀실, 늪, 약속, 잔, 등불, 발자국, 자물쇠, 외투, 망토,
서리, 차가움, 묘비, 회색, 우물, 유리창, 풍경, 칼날, 옷깃, 그늘,
계단, 한숨, 상자, 깃털, 벽지, 늦가을, 휘파람, 잠언, 조각, 카드
```

### CITIZEN (밝고 일상적인 분위기 — 무해해 보이는 단어)

```
햇살, 빵, 우산, 시계, 주전자, 의자, 책상, 신문, 사과, 빗자루,
양말, 안경, 우체통, 화분, 텃밭, 고양이, 강아지, 빨래, 모자, 단추,
연필, 공책, 물병, 손수건, 자전거, 모래, 낙엽, 종이배, 유치원, 바람개비,
쿠키, 차주전자, 밀짚, 손전등, 지도, 사다리, 양동이, 정원, 베란다, 풍선
```

### DOCTOR (보호·치유의 이미지)

```
실, 약병, 청진기, 솜, 붕대, 나비, 노을, 깃발, 등대, 생명선,
우유, 별빛, 따뜻함, 조약돌, 풀잎, 흰색, 거즈, 라일락, 버섯, 토끼풀,
연못, 솜털, 호숫물, 비누, 손난로, 기도, 자장가, 배냇저고리, 우편엽서, 풀무
```

### POLICE (탐색·관찰의 이미지)

```
나침반, 망원경, 발자국, 단서, 지도책, 묶음끈, 호각, 손전등, 수첩, 펜,
인장, 안경테, 문고리, 캐비닛, 서류, 스탬프, 모래시계, 나뭇결, 자, 격자,
교차로, 가로등, 모자, 외투, 상자, 비둘기, 거울, 벽돌, 굴뚝, 광장
```

> 개수: MAFIA·CITIZEN 각 40개, DOCTOR·POLICE 각 30개 (역할별 30~50개 범위).
> 본 풀은 게임당 1개씩 추출되므로 총 40판 이상 반복 없이 진행 가능 (역할별).
> Generation 단계에서 풀 외부화 인터페이스(파일 경로 옵션)를 함께 코드화.

---

## 11. 검증 체크리스트

- [x] 모든 도메인 타입이 외부 의존 없이 정의 가능 (FR-7.1 외부화 인터페이스만 추상화)
- [x] State는 직렬화 가능한 데이터만 포함 (스냅샷·복원 가능, NFR-1)
- [x] 비공개 정보(Role, Keyword) 표시 정책이 명확
- [x] 사망자 입력 금지·권한 게이팅 규칙이 명확
- [x] 마피아 대표자 결정/재지정 규칙이 명확 (Q-FD-U1-4-FU=C, FU2=A)
- [x] 검증 규칙이 코드 가능 수준으로 구체화 (Options 검증, 인원별 허용 범위 표)

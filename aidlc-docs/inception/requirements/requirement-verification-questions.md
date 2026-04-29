# 요구사항 명확화 질문지 (Requirement Verification Questions)

마피아 게임 프로젝트의 요구사항을 명확히 하기 위한 질문입니다.
각 질문 아래의 `[Answer]:` 태그 옆에 선택한 **알파벳(A/B/C/D...)** 을 적어주세요.
보기 중 어느 것도 정확히 맞지 않으면 마지막 보기인 **Other** 를 선택하고 `[Answer]:` 옆에 직접 설명을 적어주시면 됩니다.

모든 질문에 답변 후 "완료" 또는 "done" 이라고 알려주세요.

---

## Question 1
이 마피아 게임은 어떤 **플랫폼/실행 환경**에서 동작하길 원하시나요?

A) 웹 브라우저 (반응형 웹앱 — PC/모바일 공용)
B) 모바일 네이티브 앱 (iOS/Android)
C) 데스크톱 애플리케이션 (Electron 등)
D) CLI / 터미널 기반
E) 메신저 봇 (Discord, Slack, 카카오톡 등)
F) Other (please describe after [Answer]: tag below)

[Answer]: 웹 브라우저에서 접속하여 유저들이 게임 운영을 따를 수 있고, 실제 게임 플레이는 현실에서 한다. 

---

## Question 2
게임은 어떤 **진행 방식**을 가정하나요?

A) 오프라인 모임 보조 도구 (한 명이 진행자로서 단일 기기 사용 — 역할 배정·진행 가이드)
B) 온라인 실시간 멀티플레이 (각자 자기 기기로 접속해 함께 플레이)
C) 비동기 턴제 (각 플레이어가 자기 시간에 행동, 카카오톡 마피아류)
D) Other (please describe after [Answer]: tag below)

[Answer]: A)이고 모두가 플레이어임.

---

## Question 3
한 게임에 동시에 참여 가능한 **플레이어 수** 범위는 어느 정도인가요?

A) 4–8명 (소규모 팀)
B) 6–12명 (일반적 마피아 인원)
C) 8–20명 (대규모 모임 가능)
D) 가변형 (4명부터 30명 이상까지 광범위 지원)
E) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 4
지원할 **역할(role) 범위**는 어느 정도인가요?

A) 기본형 4종 (마피아, 시민, 의사, 경찰)
B) 표준 확장 6–8종 (기본형 + 도둑, 군인, 변호사, 기자 등)
C) 풍부한 확장 10종 이상 + 사용자 커스터마이징 (역할 추가/규칙 설정)
D) Other (please describe after [Answer]: tag below)

[Answer]: A)

---

## Question 5
**진행자(Game Master)** 역할은 어떻게 처리되나요?

A) 시스템이 자동 진행 (사람 진행자 불필요 — 시스템이 밤/낮 전환, 투표 집계 등 처리)
B) 사람 진행자가 시스템 보조를 받아 진행 (시스템은 역할 배정과 기록 보조)
C) 두 모드 모두 지원 (자동 진행 + 수동 진행 선택 가능)
D) Other (please describe after [Answer]: tag below)

[Answer]: A)

---

## Question 6
플레이어 간 **소통 수단**은 어떻게 설계할까요?

A) 인앱 텍스트 채팅만 (낮 토론, 마피아끼리 비밀 채팅 등)
B) 인앱 음성 채팅 (실시간 마이크 통신)
C) 외부 도구 사용 (Discord/Zoom 등에서 직접 대화, 시스템은 게임 진행만 담당)
D) 인앱 텍스트 + 음성 모두 지원
E) 직접 만나서 대화 (오프라인 — 별도 통신 기능 불필요)
F) Other (please describe after [Answer]: tag below)

[Answer]: E

---

## Question 7
플레이어 **인증/접속 방식**은 어떻게 가져갈까요?

A) 인증 없음 — 닉네임만 입력하고 바로 입장 (가장 간편)
B) 방 비밀번호 / 초대 코드 방식 (호스트가 코드 공유)
C) 회원가입 (이메일 또는 소셜 로그인)
D) 사내 SSO 연동
E) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 8
**게임 기록/통계** 보존 수준은 어느 정도가 필요한가요?

A) 저장 없음 (한 판이 끝나면 데이터 폐기)
B) 결과 요약만 저장 (승팀, 역할별 생존 여부 정도)
C) 상세 로그 + 플레이어 통계 (개인 전적, 승률, 역할별 성과)
D) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 9
**배포 환경(어디서 실행될 것인지)** 은 어떻게 가정하나요?

A) 로컬 실행 (각자 PC/사내망에서 실행)
B) 퍼블릭 클라우드 (AWS / GCP / Azure)
C) 사내 서버 (온프레미스)
D) 무료 호스팅 / PaaS (Vercel, Netlify, Cloudflare 등)
E) 아직 정하지 않음 — AI 추천을 받고 싶음
F) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 10
**기술 스택** 선호도가 있나요?

A) 특별한 선호 없음 — AI가 적절히 추천
B) Python 기반 (FastAPI, Django, Flask 등)
C) JavaScript/TypeScript 기반 (Node.js + React/Next.js 등)
D) Java / Kotlin 기반 (Spring Boot 등)
E) Go 기반
F) Other (please describe after [Answer]: tag below)

[Answer]: E

---

## Question 11
이 게임의 **주된 사용 맥락**은 무엇인가요?

A) 회사 팀빌딩 / 워크숍 (부정기적, 큰 모임)
B) 정기적인 팀 친목 활동 (주간·월간 모임 등)
C) 원격 근무팀의 온라인 친목 (지리적으로 분산)
D) 사내 동호회 활동
E) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 12
이 프로젝트에서 **가장 중요한 한 가지 가치**는 무엇인가요? (트레이드오프 결정 시 우선시할 기준)

A) 빠른 시작과 간편함 (5분 안에 게임 시작 가능)
B) 풍부한 게임 경험 (다양한 역할, 전략적 깊이)
C) 공정성/부정 방지 (역할 비공개, 투표 무결성, 외부 정보 유출 방지)
D) 즐거운 분위기 (애니메이션, 사운드, 시각 효과)
E) 안정성과 신뢰성 (끊김 없이 매끄러운 진행)
F) Other (please describe after [Answer]: tag below)

[Answer]: E

---

## Question 13
**향후 확장 계획**이 있나요?

A) 일회성 — 마피아 게임 한 가지로 충분
B) 점진적 확장 — 마피아 게임 내에서 기능/역할 지속 추가
C) 다른 파티 게임으로 확장 — 마피아 외 다른 게임도 같은 플랫폼에서 지원
D) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 14: Security Extensions
이 프로젝트에 **보안 확장(Security Baseline) 규칙**을 적용할까요?

A) Yes — 모든 SECURITY 규칙을 차단 제약으로 강제 적용 (운영 등급 애플리케이션 권장)
B) No — 모든 SECURITY 규칙 생략 (PoC, 프로토타입, 실험 프로젝트에 적합)
X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## (선택) 추가 의견
질문지에 없지만 미리 알려주고 싶은 요구사항이나 제약, 영감을 받은 레퍼런스 게임 등이 있다면 자유롭게 적어주세요.

[Free-form notes]: 첫째날 자기소개 때 직업 별로 반드시 말해야 하는 키워드가 있으면 좋겠어.


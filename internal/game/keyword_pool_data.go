package game

// defaultMafiaWords is the built-in Korean keyword pool for the mafia role.
// Tone: dark, suggestive of secrets — supports the natural-introduction game
// loop where mafia must drop the keyword without alerting the citizens.
var defaultMafiaWords = []string{
	"그림자", "침묵", "가면", "어둠", "속삭임", "숨결", "안개", "비밀", "거울", "골목",
	"새벽", "밀실", "늪", "약속", "잔", "등불", "발자국", "자물쇠", "외투", "망토",
	"서리", "차가움", "묘비", "회색", "우물", "유리창", "풍경", "칼날", "옷깃", "그늘",
	"계단", "한숨", "상자", "깃털", "벽지", "늦가을", "휘파람", "잠언", "조각", "카드",
}

// defaultCitizenWords is the built-in Korean keyword pool for plain citizens.
// Tone: bright and ordinary — innocuous everyday objects.
var defaultCitizenWords = []string{
	"햇살", "빵", "우산", "시계", "주전자", "의자", "책상", "신문", "사과", "빗자루",
	"양말", "안경", "우체통", "화분", "텃밭", "고양이", "강아지", "빨래", "모자", "단추",
	"연필", "공책", "물병", "손수건", "자전거", "모래", "낙엽", "종이배", "유치원", "바람개비",
	"쿠키", "차주전자", "밀짚", "손전등", "지도", "사다리", "양동이", "정원", "베란다", "풍선",
}

// defaultDoctorWords is the built-in Korean keyword pool for the doctor role.
// Tone: protective, healing imagery.
var defaultDoctorWords = []string{
	"실", "약병", "청진기", "솜", "붕대", "나비", "노을", "깃발", "등대", "생명선",
	"우유", "별빛", "따뜻함", "조약돌", "풀잎", "흰색", "거즈", "라일락", "버섯", "토끼풀",
	"연못", "솜털", "호숫물", "비누", "손난로", "기도", "자장가", "배냇저고리", "우편엽서", "풀무",
}

// defaultPoliceWords is the built-in Korean keyword pool for the police role.
// Tone: investigative, observational imagery.
var defaultPoliceWords = []string{
	"나침반", "망원경", "발자국", "단서", "지도책", "묶음끈", "호각", "손전등", "수첩", "펜",
	"인장", "안경테", "문고리", "캐비닛", "서류", "스탬프", "모래시계", "나뭇결", "자", "격자",
	"교차로", "가로등", "모자", "외투", "상자", "비둘기", "거울", "벽돌", "굴뚝", "광장",
}

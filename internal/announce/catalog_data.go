package announce

import "github.com/saltware/mafia-game/internal/game"

// Korean string constants for the catalog. Centralized so future
// localization or wording tweaks happen in one file.
//
// "근엄·차분·고전적 진행자" 톤 (Q-FD-U2-8=A).
const (
	msgGameStarted     = "마피아 게임이 시작됩니다. 모든 시민은 침묵 속에서 운명을 받아들이시오."
	msgPhaseIntro      = "각자 차례대로 자기소개를 진행하시오. 한 사람당 %d초가 주어집니다."
	msgPhaseNight      = "이제 밤이 깊어졌습니다. 모두 눈을 감으시오."
	msgPhaseDayFirst   = "첫째 날 아침이 밝았습니다. 어떤 자가 마을을 위협하는지, 토론을 시작하시오."
	msgPhaseDay        = "%d일째 아침이 밝았습니다. 마을은 어떤 운명을 맞이했는가."
	msgPhaseVote       = "토론은 끝났습니다. 이제 의심스러운 자에게 표를 던지시오."
	msgPhaseRecount    = "결과가 같습니다. 마지막 한 번 더, 신중히 선택하시오."
	msgIntroSpeaker    = "%s, 발언하시오."
	msgDeath           = "전날 밤 %s이(가) 사망했습니다. 마을의 슬픔이 깊어집니다."
	msgPeacefulNight   = "전날 밤에는 아무도 사망하지 않았습니다."
	msgEliminated      = "%s이(가) 마을의 결정으로 처형되었습니다. 그의 정체는 %s이었습니다."
	msgNightStepMafia  = "마피아는 눈을 뜨고, 처단할 자를 지목하시오."
	msgNightStepPolice = "경찰은 눈을 뜨고, 의심스러운 자를 조사하시오."
	msgNightStepDoctor = "의사는 눈을 뜨고, 지킬 자를 선택하시오."
	msgTimer30        = "토론 종료까지 30초 남았습니다."
	msgTimer10        = "토론 종료까지 10초 남았습니다. 마음을 정하시오."
	msgTimer0         = "토론이 종료되었습니다."
	msgVoteNoElim     = "재투표 또한 동률이었습니다. 오늘은 처형이 없습니다."
	msgEndMafia       = "마피아의 승리. 어둠이 마을을 삼켰습니다."
	msgEndCitizen     = "시민의 승리. 정의가 어둠을 몰아냈습니다."
	msgEndForce       = "진행자의 결정으로 게임이 종료되었습니다."
	msgVoiceOn        = "음성 안내가 활성화되었습니다."
	msgVoiceOff       = "음성 안내가 비활성화되었습니다."
	msgRestoreNotice  = "이전 게임이 복원되었습니다. 같은 단계부터 이어집니다."
	msgPersistFailure = "게임 상태를 저장하지 못했습니다. 곧 다시 시도합니다."
	// Iteration 5 — host pause/resume narration.
	msgGamePaused  = "잠시 진행을 멈춥니다. 모두 자리를 지키시오."
	msgGameResumed = "다시 시간이 흐르기 시작합니다. 진행을 이어가시오."
)

// roleKr maps a Role to its Korean display name (BR-U2-CAT-6).
func roleKr(r game.Role) string {
	switch r {
	case game.RoleMafia:
		return "마피아"
	case game.RoleDoctor:
		return "의사"
	case game.RolePolice:
		return "경찰"
	case game.RoleCitizen:
		return "시민"
	default:
		return string(r)
	}
}

// SystemRestore returns the post-recovery toast (BR-U2-RESTORE-5).
func SystemRestore() Announcement {
	return Announcement{
		Subtitle:      msgRestoreNotice,
		AudioID:       cueSystemRestore,
		Severity:      SeverityInfo,
		ForPublicOnly: true,
	}
}

// SystemPersistFailure returns the toast emitted when SaveSnapshot fails
// (P-U2-3 — surface to operator without halting the game).
func SystemPersistFailure() Announcement {
	return Announcement{
		Subtitle:      msgPersistFailure,
		AudioID:       cueSystemPersistFailure,
		Severity:      SeverityWarn,
		ForPublicOnly: true,
	}
}

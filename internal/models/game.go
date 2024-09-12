package models

import (
	"time"

	"github.com/uptrace/bun"
)

type GameName string

const (
	GameNameCatia GameName = "catia"
)

type QuestionSetup struct {
	Difficulty QuestionDifficulty `json:"difficulty"`
	Score      int                `json:"score"`
	Extra      bool               `json:"extra"`
}

type ExtraSetupType int

const (
	ExtraSetupTypeUnknown ExtraSetupType = iota
	ExtraSetupTypeDouble
	ExtraSetupTypeHalf
	ExtraSetupTypeMinus5
	ExtraSetupTypeMinus15
	ExtraSetupTypeMinus50
	ExtraSetupTypePlus5
	ExtraSetupTypePlus15
	ExtraSetupTypePlus10
	ExtraSetupTypePlus30
	ExtraSetupTypePlus50
	ExtraSetupTypePlus100
	ExtraSetupTypeTo0
	ExtraSetupTypeUpTo40
	ExtraSetupTypeUpTo110
	ExtraSetupTypeTo360
	ExtraSetupTypeAnother
	ExtraSetupTypeNothing

	ExtraSetupType1Gem
	ExtraSetupType3Gem
	ExtraSetupType5Gem
	ExtraSetupType10Gem
	ExtraSetupType1Lifeline
	ExtraSetupType2Lifeline
	ExtraSetupType1Star
	ExtraSetupType2Star

	ExtraSetupTypePlus2
	ExtraSetupTypeMinus2
)

func (v ExtraSetupType) Valid() bool {
	switch v {
	case ExtraSetupTypeUnknown, ExtraSetupTypeDouble, ExtraSetupTypeHalf, ExtraSetupTypeMinus50,
		ExtraSetupTypePlus50, ExtraSetupTypeTo0, ExtraSetupTypeTo360,
		ExtraSetupTypeAnother, ExtraSetupTypeNothing, ExtraSetupTypePlus100, ExtraSetupTypePlus15, ExtraSetupTypePlus30,
		ExtraSetupTypeMinus15, ExtraSetupTypeUpTo110, ExtraSetupTypePlus5, ExtraSetupTypePlus10, ExtraSetupTypeMinus5,
		ExtraSetupTypeUpTo40, ExtraSetupType1Gem, ExtraSetupType3Gem, ExtraSetupType5Gem, ExtraSetupType10Gem,
		ExtraSetupType1Lifeline, ExtraSetupType2Lifeline, ExtraSetupType1Star, ExtraSetupType2Star, ExtraSetupTypePlus2, ExtraSetupTypeMinus2:
		return true
	default:
		return false
	}
}

func ToExtraSetupType(s string) ExtraSetupType {
	switch s {
	case "double":
		return ExtraSetupTypeDouble
	case "half":
		return ExtraSetupTypeHalf
	case "to_0":
		return ExtraSetupTypeTo0
	case "to_360":
		return ExtraSetupTypeTo360
	case "another":
		return ExtraSetupTypeAnother
	case "nothing":
		return ExtraSetupTypeNothing
	case "to_110":
		return ExtraSetupTypeUpTo110
	case "plus_2":
		return ExtraSetupTypePlus2
	case "plus_5":
		return ExtraSetupTypePlus5
	case "plus_10":
		return ExtraSetupTypePlus10
	case "plus_15":
		return ExtraSetupTypePlus15
	case "plus_30":
		return ExtraSetupTypePlus30
	case "plus_50":
		return ExtraSetupTypePlus50
	case "plus_100":
		return ExtraSetupTypePlus100
	case "minus_2":
		return ExtraSetupTypeMinus2
	case "minus_5":
		return ExtraSetupTypeMinus5
	case "minus_15":
		return ExtraSetupTypeMinus15
	case "minus_50":
		return ExtraSetupTypeMinus50
	case "to_40":
		return ExtraSetupTypeUpTo40
	case "1_gem":
		return ExtraSetupType1Gem
	case "3_gem":
		return ExtraSetupType3Gem
	case "5_gem":
		return ExtraSetupType5Gem
	case "10_gem":
		return ExtraSetupType10Gem
	case "1_lifeline":
		return ExtraSetupType1Lifeline
	case "2_lifeline":
		return ExtraSetupType2Lifeline
	case "1_star":
		return ExtraSetupType1Star
	case "2_star":
		return ExtraSetupType2Star
	default:
		return ExtraSetupTypeUnknown
	}
}

// convert extra setup type to string
func (v ExtraSetupType) String() string {
	// convert extra setup type to string
	switch v {
	case ExtraSetupTypeDouble:
		return "double"
	case ExtraSetupTypeHalf:
		return "half"
	case ExtraSetupTypeMinus2:
		return "minus_2"
	case ExtraSetupTypeMinus5:
		return "minus_5"
	case ExtraSetupTypeMinus15:
		return "minus_15"
	case ExtraSetupTypeMinus50:
		return "minus_50"
	case ExtraSetupTypePlus2:
		return "plus_2"
	case ExtraSetupTypePlus5:
		return "plus_5"
	case ExtraSetupTypePlus10:
		return "plus_10"
	case ExtraSetupTypePlus15:
		return "plus_15"
	case ExtraSetupTypePlus30:
		return "plus_30"
	case ExtraSetupTypePlus50:
		return "plus_50"
	case ExtraSetupTypePlus100:
		return "plus_100"
	case ExtraSetupTypeTo0:
		return "to_0"
	case ExtraSetupTypeTo360:
		return "to_360"
	case ExtraSetupTypeAnother:
		return "another"
	case ExtraSetupTypeNothing:
		return "nothing"
	case ExtraSetupTypeUpTo110:
		return "to_110"
	case ExtraSetupTypeUpTo40:
		return "to_40"
	case ExtraSetupType1Gem:
		return "1_gem"
	case ExtraSetupType3Gem:
		return "3_gem"
	case ExtraSetupType5Gem:
		return "5_gem"
	case ExtraSetupType10Gem:
		return "10_gem"
	case ExtraSetupType1Lifeline:
		return "1_lifeline"
	case ExtraSetupType2Lifeline:
		return "2_lifeline"
	case ExtraSetupType1Star:
		return "1_star"
	case ExtraSetupType2Star:
		return "2_star"
	default:
		return "unknown"
	}
}

func (v ExtraSetupType) ToScore(currentScore int) (int, bool) {
	switch v {
	case ExtraSetupTypeDouble:
		return currentScore * 2, false
	case ExtraSetupTypeHalf:
		return currentScore / 2, false
	case ExtraSetupTypeMinus2:
		score := currentScore - 2
		if score < 0 {
			score = 0
		}
		return score, false
	case ExtraSetupTypeMinus5:
		score := currentScore - 5
		if score < 0 {
			score = 0
		}
		return score, false
	case ExtraSetupTypeMinus15:
		score := currentScore - 15
		if score < 0 {
			score = 0
		}
		return score, false
	case ExtraSetupTypeMinus50:
		score := currentScore - 50
		if score < 0 {
			score = 0
		}
		return score, false
	case ExtraSetupTypeTo0:
		return 0, false
	case ExtraSetupTypeTo360:
		return 360, false
	case ExtraSetupTypePlus2:
		return currentScore + 2, false
	case ExtraSetupTypePlus5:
		return currentScore + 5, false
	case ExtraSetupTypePlus10:
		return currentScore + 10, false
	case ExtraSetupTypePlus15:
		return currentScore + 15, false
	case ExtraSetupTypePlus30:
		return currentScore + 30, false
	case ExtraSetupTypePlus50:
		return currentScore + 50, false
	case ExtraSetupTypePlus100:
		return currentScore + 100, false
	case ExtraSetupTypeUpTo110:
		return 110, false
	case ExtraSetupTypeUpTo40:
		return 40, false
	default:
		return currentScore, v == ExtraSetupTypeAnother
	}
}

type ExtraSetup struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Chance      int    `json:"chance"`
}

// db
type Game struct {
	bun.BaseModel  `bun:"table:game"`
	ID             int             `bun:"id,pk,autoincrement" json:"id"`
	Name           string          `bun:"name" json:"name"`
	Slug           string          `bun:"slug" json:"slug"`
	Description    string          `bun:"description" json:"description"`
	Questions      []QuestionSetup `bun:"questions,type:jsonb" json:"questions"`
	Checkpoints    map[int]bool    `bun:"checkpoints,type:jsonb" json:"checkpoints"`
	Logo           string          `bun:"logo" json:"logo"`
	StartTime      *time.Time      `bun:"start_time" json:"start_time"`
	EndTime        *time.Time      `bun:"end_time" json:"end_time"`
	Enabled        bool            `bun:"enabled" json:"-"`
	Config         []GameConfig    `bun:"config,type:jsonb" json:"config"`
	ExtraSetups    []ExtraSetup    `bun:"extra_setup,type:jsonb" json:"extra_setup"`
	Difficulty     *string         `bun:"difficulty" json:"difficulty"`
	SocialLinks    []Link          `bun:"social_links,type:jsonb" json:"social_links"`
	Priority       int             `bun:"priority" json:"priority"`
	BonusPrivilege string          `bun:"bonus_privilege" json:"bonus_privilege"`
	IsPublic       bool            `bun:"is_public" json:"is_public"` //if is public = false -> it's in arena
}

type GameConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type GameAnswer struct {
	Answer        int `json:"answer"`
	QuestionIndex int `json:"question"`
}

var GameDefault = Game{
	Questions: []QuestionSetup{
		{QuestionEasy, 1, false},
		{QuestionEasy, 2, false},
		{QuestionEasy, 4, false},
		{QuestionEasy, 8, false},
		{QuestionMedium, 16, false},
		{QuestionMedium, 32, false},
		{QuestionMedium, 64, false},
		{QuestionHard, 128, false},
		{QuestionHard, 256, false},
		{"", 0, true},
	},
	Checkpoints: map[int]bool{0: true, 4: true, 7: true},
}

type AssistanceType string

const (
	AssistanceTypeFiftyFifty     AssistanceType = "fifty_fifty"
	AssistanceTypeChangeQuestion AssistanceType = "change_question"
)

func (a AssistanceType) Valid() bool {
	// disable change question
	return a == AssistanceTypeFiftyFifty
}

type TotalScoreCheckpoint struct {
	RequireTotalInvite int
	Checkpoint         int
}

var CheckpointTotalScoreConfig = []TotalScoreCheckpoint{
	{RequireTotalInvite: 1, Checkpoint: 1000},
	{RequireTotalInvite: 4, Checkpoint: 6000},
}

type LongestStreak struct {
	Username    string `json:"username"`
	StreakPoint int    `json:"streak_point"`
}

type MostSessions struct {
	Username      string `json:"username"`
	TotalSessions int    `json:"total_sessions"`
}

type MostBonusPoint struct {
	Username        string `json:"username"`
	BonusPointsTime int    `json:"bonus_points_time"`
}

type MostMinusPoint struct {
	Username        string `json:"username"`
	MinusPointsTime int    `json:"minus_points_time"`
}

type Countdown struct {
	UserId string `json:"user_id"`
}

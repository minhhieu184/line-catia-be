package models

import (
	"time"
)

type Moon struct {
	NumberOfTaps     int       `json:"number_of_taps"`
	CurrentFullMoon  time.Time `json:"active_at"`
	ExpiredAt        time.Time `json:"expired_at"`
	NextFullMoon     time.Time `json:"next_active_at"`
	CurrentTimeFrame time.Time `json:"-"`
}

type UserMoon struct {
	Moon
	Claimed bool `json:"claimed"`
}

type GiftType string

const (
	GiftTypeNothing  GiftType = "nothing"
	GiftTypeGem      GiftType = "gem"
	GiftTypeLifeline GiftType = "lifeline"
	GiftTypeStar     GiftType = "star"
)

type Gift struct {
	Type  GiftType `json:"type"`
	Amout int      `json:"amount"`
}

func (v ExtraSetupType) ToGift() *Gift {
	switch v {
	case ExtraSetupType1Gem:
		return &Gift{Type: "gem", Amout: 1}
	case ExtraSetupType3Gem:
		return &Gift{Type: "gem", Amout: 3}
	case ExtraSetupType5Gem:
		return &Gift{Type: "gem", Amout: 5}
	case ExtraSetupType10Gem:
		return &Gift{Type: "gem", Amout: 10}
	case ExtraSetupType1Lifeline:
		return &Gift{Type: "lifeline", Amout: 1}
	case ExtraSetupType2Lifeline:
		return &Gift{Type: "lifeline", Amout: 2}
	case ExtraSetupType1Star:
		return &Gift{Type: "star", Amout: 1}
	case ExtraSetupType2Star:
		return &Gift{Type: "star", Amout: 2}
	case ExtraSetupTypeNothing:
		return &Gift{Type: "nothing", Amout: 0}
	}

	return nil
}

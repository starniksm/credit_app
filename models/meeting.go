package models

import (
	"time"
)

type Meeting struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	ClientID    string    `json:"clientId"`
	ClientName  string    `json:"clientName"`
	ClientPhone string    `json:"clientPhone"`
	ClientEmail string    `json:"clientEmail"`
	Location    string    `json:"location"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Status      string    `json:"status" gorm:"default:'scheduled'"` // scheduled, completed, cancelled, no_show
	Result      string    `json:"result"`                            // card_issued, client_thinking, refused, needs_docs, repeat
	CardType    string    `json:"cardType"`                          // classic, gold, platinum, premium
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (Meeting) TableName() string {
	return "meetings"
}

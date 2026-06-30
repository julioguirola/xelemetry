package internal

import (
	"time"
)

type Check struct {
	ID         int
	Time       *time.Time `gorm:"default:current_timestamp"`
	LocationID string
}

type Uptime struct {
	ID         int
	Duration   int
	StartTime  time.Time `gorm:"default:current_timestamp"`
	LocationID string
}

type Location struct {
	ID      string
	Nombre  string `gorm:"unique"`
	Checks  []Check
	Uptimes []Uptime
}

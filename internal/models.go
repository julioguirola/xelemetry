package internal

import "time"

type Check struct {
	ID   int
	Time *time.Time `gorm:"default:current_timestamp"`
}

package internal

import (
	"time"
)

type Uptime struct {
	ID         int
	Duration   *int
	StartTime  time.Time `gorm:"default:current_timestamp"`
	LocationID string
}

type Location struct {
	ID      string
	Nombre  string `gorm:"unique"`
	Uptimes []Uptime
	UserID  string
}

type User struct {
	ID        string
	UserName  string
	PassWord  string
	Locations []Location
}

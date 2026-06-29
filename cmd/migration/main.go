package main

import (
	"os"
	"xelemetry/internal"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type GetCheckQuery struct {
	Limit *int `form:"limit"`
}

func main() {
	db, err := gorm.Open(sqlite.Open("checks.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if os.Getenv("PORT") == "" {
		panic("PORT environment variable is not set")
	}

	err = db.Migrator().DropTable(&internal.Check{})
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(&internal.Check{})
	if err != nil {
		panic(err)
	}
}

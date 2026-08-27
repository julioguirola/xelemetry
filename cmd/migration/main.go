package main

import (
	"log"
	"os"
	"time"
	"xelemetry/internal"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GetCheckQuery struct {
	Limit *int `form:"limit"`
}

func main() {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  false,
		},
	)

	// Globally mode
	db, err := gorm.Open(sqlite.Open("checks.db"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic(err)
	}

	err = db.Migrator().DropTable(
		&internal.Uptime{},
		&internal.Location{},
		&internal.User{},
	)
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(
		&internal.Uptime{},
		&internal.Location{},
		&internal.User{},
	)
	if err != nil {
		panic(err)
	}
}

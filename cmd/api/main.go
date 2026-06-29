package main

import (
	"fmt"
	"net/http"
	"os"
	"xelemetry/internal"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.ErrBadRequest.Wrap(err)
	}
	return nil
}

type GetCheckQuery struct {
	Limit *int    `query:"limit" validate:"omitempty,gte=1,lte=100"`
	From  *string `query:"from"`
	To    *string `query:"to"`
}

func main() {
	db, err := gorm.Open(sqlite.Open("checks.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if os.Getenv("PORT") == "" {
		panic("PORT environment variable is not set")
	}

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.POST("/check", func(c *echo.Context) error {

		err := gorm.G[internal.Check](db).Create(c.Request().Context(), &internal.Check{})
		if err != nil {
			e.Logger.Error("failed to create check", "error", err)
			return err
		}
		return nil
	})

	e.GET("/check", func(c *echo.Context) error {
		var query GetCheckQuery
		if err := c.Bind(&query); err != nil {
			e.Logger.Error("failed to bind query", "error", err)
			return err
		}
		if err := c.Validate(&query); err != nil {
			e.Logger.Error("failed to validate query", "error", err)
			err = c.JSON(http.StatusBadRequest, err.Error())
			if err != nil {
				e.Logger.Error("failed to serialize error", "error", err)
			}
			return err
		}
		limit := 40
		if query.Limit != nil {
			limit = *query.Limit
		}
		sql := gorm.G[internal.Check](db).Where("id > ?", 0)
		if query.From != nil {
			sql = sql.Where("time >= ?", *query.From)
		}
		if query.To != nil {
			sql = sql.Where("time <= ?", *query.To)
		}
		checks, err := sql.Limit(limit).Find(c.Request().Context())
		if err != nil {
			e.Logger.Error("failed to get checks", "error", err)
			return nil
		}
		err = c.JSON(http.StatusOK, checks)
		if err != nil {
			e.Logger.Error("failed to serialize checks", "error", err)
			return err
		}
		return nil
	})

	if err := e.Start(fmt.Sprintf(":%s", os.Getenv("PORT"))); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

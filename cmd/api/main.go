package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
	"xelemetry/internal"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type CustomValidator struct {
	validator *validator.Validate
}

var (
	upgrader = websocket.Upgrader{}
)

func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.ErrBadRequest.Wrap(err)
	}
	return nil
}

type GetCheckQuery struct {
	Limit      *int    `query:"limit" validate:"omitempty,gte=1,lte=100"`
	From       *string `query:"from"`
	To         *string `query:"to"`
	LocationID *string `query:"location_id" validate:"omitempty,uuid"`
}

type GetUptimeQuery struct {
	Limit      *int    `query:"limit" validate:"omitempty,gte=1,lte=100"`
	From       *string `query:"from"`
	To         *string `query:"to"`
	LocationID *string `query:"location_id" validate:"omitempty,uuid"`
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	db, err := gorm.Open(sqlite.Open("checks.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if os.Getenv("PORT") == "" {
		panic("PORT environment variable is not set")
	}
	e := echo.New()
	e.Logger = slog.New(zerolog.NewSlogHandler(log.Output(zerolog.ConsoleWriter{Out: os.Stderr})))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			e.Logger.Info(fmt.Sprintf("%s %s %d", v.Method, v.URI, v.Status))
			return nil
		},
	}))
	e.Validator = &CustomValidator{validator: validator.New()}

	e.Use(middleware.Recover())

	e.POST("/check", func(c *echo.Context) error {
		var req struct {
			LocationID string `json:"location_id" validate:"required,uuid"`
		}
		if err := c.Bind(&req); err != nil {
			e.Logger.Error("failed to bind check", "error", err)
			return err
		}
		if err := c.Validate(&req); err != nil {
			e.Logger.Error("failed to validate check", "error", err)
			return err
		}

		check := internal.Check{
			LocationID: req.LocationID,
		}
		err := gorm.G[internal.Check](db).Create(c.Request().Context(), &check)
		if err != nil {
			e.Logger.Error("failed to create check", "error", err)
			return err
		}
		return c.JSON(http.StatusCreated, check)
	})

	e.GET("/ws", func(c *echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		locationID := c.QueryParam("location_id")
		if locationID == "" {
			return c.JSON(http.StatusBadRequest, "location_id is required")
		}

		startTime := time.Now()
		c.Logger().Info(fmt.Sprintf("Location %s conectada.", locationID))

		defer func() {
			duration := int(time.Since(startTime).Seconds())
			uptimeRecord := internal.Uptime{
				LocationID: locationID,
				Duration:   duration,
			}
			if err := gorm.G[internal.Uptime](db).Create(c.Request().Context(), &uptimeRecord); err != nil {
				c.Logger().Error(fmt.Sprintf("failed to save uptime for %s: %v", locationID, err))
			}
			err = ws.Close()
			if err != nil {
				return
			}
		}()

		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				c.Logger().Info(fmt.Sprintf("Location %s desconectada.", locationID))
				break
			}
		}
		return nil
	})

	e.POST("/location", func(c *echo.Context) error {
		var req struct {
			Nombre string `json:"nombre" validate:"required"`
		}
		if err := c.Bind(&req); err != nil {
			e.Logger.Error("failed to bind location", "error", err)
			return err
		}
		if err := c.Validate(&req); err != nil {
			e.Logger.Error("failed to validate location", "error", err)
			return err
		}
		location := internal.Location{
			ID:     uuid.New().String(),
			Nombre: req.Nombre,
		}
		if err := gorm.G[internal.Location](db).Create(c.Request().Context(), &location); err != nil {
			e.Logger.Error("failed to create location", "error", err)
			return err
		}
		return c.JSON(http.StatusCreated, location)
	})

	e.GET("/location", func(c *echo.Context) error {
		locations, err := gorm.G[internal.Location](db).Find(c.Request().Context())
		if err != nil {
			e.Logger.Error("failed to get locations", "error", err)
			return err
		}
		return c.JSON(http.StatusOK, locations)
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
		if query.LocationID != nil {
			sql = sql.Where("location_id = ?", *query.LocationID)
		}
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

	e.GET("/uptime", func(c *echo.Context) error {
		var query GetUptimeQuery
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
		sql := gorm.G[internal.Uptime](db).Where("id > ?", 0)
		if query.LocationID != nil {
			sql = sql.Where("location_id = ?", *query.LocationID)
		}
		if query.From != nil {
			sql = sql.Where("start_time >= ?", *query.From)
		}
		if query.To != nil {
			sql = sql.Where("start_time <= ?", *query.To)
		}
		uptimes, err := sql.Limit(limit).Find(c.Request().Context())
		if err != nil {
			e.Logger.Error("failed to get uptimes", "error", err)
			return nil
		}
		err = c.JSON(http.StatusOK, uptimes)
		if err != nil {
			e.Logger.Error("failed to serialize uptimes", "error", err)
			return err
		}
		return nil
	})

	if err := e.Start(fmt.Sprintf(":%s", os.Getenv("PORT"))); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

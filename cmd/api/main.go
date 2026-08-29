package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
	"xelemetry/internal"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/matthewhartstonge/argon2"
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

type telegramMessage struct {
	ChatID          string `json:"chat_id"`
	MessageThreadID int    `json:"message_thread_id,omitempty"`
	Text            string `json:"text"`
}

func sendTelegramNotification(text string) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	threadID := os.Getenv("TELEGRAM_MESSAGE_THREAD_ID")

	if botToken == "" || chatID == "" {
		return
	}

	msg := telegramMessage{
		ChatID: chatID,
		Text:   text,
	}
	if threadID != "" {
		fmt.Sscanf(threadID, "%d", &msg.MessageThreadID)
	}

	body, _ := json.Marshal(msg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+botToken+"/sendMessage", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}

func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.ErrBadRequest.Wrap(err)
	}
	return nil
}

func userIDFromToken(c *echo.Context) (string, error) {
	auth := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", echo.ErrUnauthorized
	}
	token, err := jwt.Parse(strings.TrimPrefix(auth, "Bearer "), func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return "", echo.ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", echo.ErrUnauthorized
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", echo.ErrUnauthorized
	}
	return sub, nil
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
	LocationID string  `query:"location_id" validate:"required,uuid"`
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	db, err := gorm.Open(sqlite.Open("checks.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if os.Getenv("JWT_SECRET") == "" {
		panic("JWT_SECRET environment variable is not set")
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

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"*"},
		AllowHeaders: []string{"*"},
	}))

	e.GET("/", func(c *echo.Context) error {
		return c.JSON(200, "ok")
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
		uptimeRecord := internal.Uptime{
			LocationID: locationID,
		}
		if err := gorm.G[internal.Uptime](db).Create(c.Request().Context(), &uptimeRecord); err != nil {
			c.Logger().Error(fmt.Sprintf("failed to create uptime for %s: %v", locationID, err))
			return err
		}

		c.Logger().Info(fmt.Sprintf("Location %s conectada.", locationID))
		sendTelegramNotification("Hay corriente y conexión ⚡")

		defer func() {
			duration := int(time.Since(startTime).Seconds())
			if _, err := gorm.G[internal.Uptime](db).Where("id = ?", uptimeRecord.ID).Update(c.Request().Context(), "duration", duration); err != nil {
				c.Logger().Error(fmt.Sprintf("failed to update uptime for %s: %v", locationID, err))
			}
			sendTelegramNotification("No hay corriente o conexión 🔌")
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
		userID, err := userIDFromToken(c)
		if err != nil {
			return err
		}

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
			UserID: userID,
		}
		if err := gorm.G[internal.Location](db).Create(c.Request().Context(), &location); err != nil {
			e.Logger.Error("failed to create location", "error", err)
			return err
		}
		return c.JSON(http.StatusCreated, location)
	})

	e.GET("/location", func(c *echo.Context) error {
		userID, err := userIDFromToken(c)
		if err != nil {
			return err
		}

		locations, err := gorm.G[internal.Location](db).Where("user_id = ?", userID).Find(c.Request().Context())
		if err != nil {
			e.Logger.Error("failed to get locations", "error", err)
			return err
		}
		return c.JSON(http.StatusOK, locations)
	})

	e.GET("/uptime", func(c *echo.Context) error {
		userID, err := userIDFromToken(c)
		if err != nil {
			return err
		}

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

		if _, err := gorm.G[internal.Location](db).Where("id = ? AND user_id = ?", query.LocationID, userID).First(c.Request().Context()); err != nil {
			return c.JSON(http.StatusNotFound, "location not found")
		}

		limit := 40
		if query.Limit != nil {
			limit = *query.Limit
		}
		sql := gorm.G[internal.Uptime](db).Where("location_id = ?", query.LocationID)
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

	e.POST("/user", func(c *echo.Context) error {
		var req struct {
			UserName string `json:"user_name" validate:"required"`
			PassWord string `json:"pass_word" validate:"required"`
		}
		if err := c.Bind(&req); err != nil {
			e.Logger.Error("failed to bind user", "error", err)
			return err
		}
		if err := c.Validate(&req); err != nil {
			e.Logger.Error("failed to validate user", "error", err)
			return err
		}

		cfg := argon2.DefaultConfig()
		raw, err := cfg.Hash([]byte(req.PassWord), nil)
		if err != nil {
			e.Logger.Error("failed to hash password", "error", err)
			return err
		}

		user := internal.User{
			ID:       uuid.New().String(),
			UserName: req.UserName,
			PassWord: string(raw.Encode()),
		}
		if err := gorm.G[internal.User](db).Create(c.Request().Context(), &user); err != nil {
			e.Logger.Error("failed to create user", "error", err)
			return err
		}
		return c.JSON(http.StatusCreated, user)
	})

	e.POST("/login", func(c *echo.Context) error {
		var req struct {
			UserName string `json:"user_name" validate:"required"`
			PassWord string `json:"pass_word" validate:"required"`
		}
		if err := c.Bind(&req); err != nil {
			e.Logger.Error("failed to bind login", "error", err)
			return err
		}
		if err := c.Validate(&req); err != nil {
			e.Logger.Error("failed to validate login", "error", err)
			return err
		}

		user, err := gorm.G[internal.User](db).Where("user_name = ?", req.UserName).First(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusUnauthorized, "invalid credentials")
		}

		ok, err := argon2.VerifyEncoded([]byte(req.PassWord), []byte(user.PassWord))
		if err != nil || !ok {
			return c.JSON(http.StatusUnauthorized, "invalid credentials")
		}

		claims := jwt.MapClaims{
			"sub":  user.ID,
			"name": user.UserName,
			"iat":  time.Now().Unix(),
			"exp":  time.Now().Add(72 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			e.Logger.Error("failed to sign token", "error", err)
			return err
		}

		return c.JSON(http.StatusOK, map[string]string{"token": signed})
	})

	if err := e.Start(fmt.Sprintf(":%d", 1323)); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

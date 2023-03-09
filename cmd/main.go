package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/joebjoe/go-mocker/internal/api"
	"github.com/joebjoe/go-mocker/internal/mocker"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"k8s.io/utils/env"
)

type Port int

func (p Port) String() string { return fmt.Sprintf(":%d", p) }

var port = Port(mustInt(env.GetInt("PORT", 80)))

func main() {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Logger.SetLevel(log.INFO)
	e.Use(
		middleware.RequestIDWithConfig(middleware.RequestIDConfig{
			Generator: uuid.NewString,
		}),
		middleware.Logger(),
		middleware.BodyDump(func(c echo.Context, req, resp []byte) {
			stat := c.Response().Status
			logFn := c.Logger().Infoj

			if stat >= http.StatusBadRequest {
				logFn = c.Logger().Debugj
			}
			if stat >= http.StatusInternalServerError {
				logFn = c.Logger().Errorj
			}

			var (
				reqCT    = c.Request().Header.Get(echo.HeaderContentType)
				respCT   = c.Response().Header().Get(echo.HeaderContentType)
				reqBody  = prepBody(req, reqCT)
				respBody = prepBody(resp, respCT)
			)

			logFn(log.JSON{
				"id": c.Response().Header().Get(echo.HeaderXRequestID),
				"request": log.JSON{
					"body":         reqBody,
					"content-type": reqCT,
				},
				"response": log.JSON{
					"body":         respBody,
					"content-type": respCT,
				},
			})
		}),
	)

	srv := api.NewServer(mocker.New())

	e.GET("/generate/:from/:to", srv.HandleGET, api.NotFoundHandler)

	go e.Start(port.String())

	e.Logger.Print("waiting for quit")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	e.Logger.Print("quit received")
}

func mustInt(n int, err error) int {
	if err != nil {
		panic(err)
	}
	return n
}

func prepBody(b []byte, ct string) (v any) {
	if strings.Contains(ct, "text/") {
		return string(b)
	}

	if !strings.Contains(ct, "json") {
		return "not logged due to content type"
	}

	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Sprintf("failed to encode body: %v", err)
	}

	return v
}

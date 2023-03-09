package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

var validRelationships = map[FromType]map[ToType]struct{}{
	FromTypeStruct: {
		ToTypeInterface: struct{}{},
		ToTypeMock:      struct{}{},
	},
	FromTypeInterface: {
		ToTypeMock: struct{}{},
	},
}

type Mocker interface {
	Generate(RequestGET) (io.Reader, error)
}

type Server interface {
	HandleGET(c echo.Context) error
}

type server struct {
	mocker Mocker
	now    func() time.Time
}

func NewServer(m Mocker) Server {
	return &server{
		mocker: m,
		now:    time.Now,
	}
}

func (s *server) HandleGET(c echo.Context) (err error) {
	var (
		req       = new(RequestGET)
		requestID = c.Response().Header().Get(echo.HeaderXRequestID)
	)

	if err = c.Bind(req); err != nil {
		c.Logger().Errorj(map[string]any{
			"id":      requestID,
			"message": "failed to bind request",
			"error":   err.Error(),
		})

		return err
	}

	if err = validateRequestGET(*req); err != nil {
		c.Logger().Errorj(map[string]any{
			"id":      requestID,
			"message": "failed to validate request",
			"error":   err.Error(),
		})

		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("failed to validate request: %v", err))
	}

	r, err := s.mocker.Generate(*req)
	if err != nil {
		c.Logger().Errorj(map[string]any{
			"id":      requestID,
			"message": "failed to generate file",
			"error":   err.Error(),
		})

		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to generate file: %v", err))
	}

	return c.Stream(http.StatusOK, echo.MIMETextPlain, r)
}

func isValidRelationship(from FromType, to ToType) bool {
	if rels, ok := validRelationships[FromType(strings.ToLower(string(from)))]; ok {
		if _, ok = rels[ToType(strings.ToLower(string(to)))]; ok {
			return true
		}
	}
	return false
}

func NotFoundHandler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if isValidRelationship(FromType(c.Param("from")), ToType(c.Param("to"))) {
			return next(c)
		}
		return echo.NewHTTPError(http.StatusNotFound)
	}
}

func validateRequestGET(req RequestGET) (err error) {
	if req.Module == "" {
		err = errors.Join(err, newMissingRequiredParameterError("module"))
	}

	if req.Package == "" {
		err = errors.Join(err, newMissingRequiredParameterError("package"))
	}

	if req.Type == "" {
		err = errors.Join(err, newMissingRequiredParameterError("type"))
	}

	if !isValidRelationship(req.From, req.To) {
		err = errors.Join(err, fmt.Errorf("invalid mapping '%s/%s'; must be one of 'struct/interface', "+
			"'struct/mock', or 'interface/mock'", req.From, req.To))
	}

	return err
}

func newMissingRequiredParameterError(param string) error {
	return fmt.Errorf("'%s' cannot be empty", param)
}

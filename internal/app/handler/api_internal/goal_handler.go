package api_internal

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// GoalHandler は目標APIを扱います。
type GoalHandler struct {
	goalUsecase usecase.GoalUsecase
}

// NewGoalHandler は GoalHandler を生成します。
func NewGoalHandler(goalUsecase usecase.GoalUsecase) *GoalHandler {
	return &GoalHandler{goalUsecase: goalUsecase}
}

func (h *GoalHandler) List(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	goals, err := h.goalUsecase.List(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	items := make([]*internaldto.GoalResponse, 0, len(goals))
	for _, g := range goals {
		items = append(items, toGoalResponse(g))
	}
	return c.JSON(http.StatusOK, &internaldto.GoalsResponse{Goals: items})
}

func (h *GoalHandler) Create(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	var req internaldto.GoalRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	in, err := toGoalInput(&req)
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	goal, err := h.goalUsecase.Create(c.Request().Context(), user.ID, in)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusCreated, toGoalResponse(goal))
}

func (h *GoalHandler) Update(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	id64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	var req internaldto.GoalRequest
	if err := bindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	in, err := toGoalInput(&req)
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	goal, err := h.goalUsecase.Update(c.Request().Context(), user.ID, uint32(id64), in)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toGoalResponse(goal))
}

func (h *GoalHandler) Delete(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	id64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := h.goalUsecase.Delete(c.Request().Context(), user.ID, uint32(id64)); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func bindStrictJSON(c echo.Context, out any) error {
	return decodeStrictJSON(c.Request().Body, c.Request().Header, out)
}

func decodeStrictJSON(body io.Reader, header http.Header, out any) error {
	ct := header.Get(echo.HeaderContentType)
	if ct == "" {
		return errors.New("content-type header is missing")
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || !strings.EqualFold(mediaType, echo.MIMEApplicationJSON) {
		return errors.New("content-type must be application/json")
	}

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func toGoalInput(req *internaldto.GoalRequest) (*usecase.GoalInput, error) {
	params, err := json.Marshal(req.AchievementParams)
	if err != nil {
		return nil, err
	}
	attrs := []byte("{}")
	if req.Attributes != nil {
		attrs, err = json.Marshal(req.Attributes)
		if err != nil {
			return nil, err
		}
	}
	return &usecase.GoalInput{Title: req.Title, AchievementType: req.AchievementType, AchievementParams: params, Attributes: attrs, Invert: req.Invert}, nil
}

func getUser(c echo.Context) (*entity.User, error) {
	user, ok := c.Get("userEntity").(*entity.User)
	if !ok || user == nil {
		return nil, apierror.ErrUnauthorized
	}
	return user, nil
}

func toGoalResponse(goal *usecase.GoalOutput) *internaldto.GoalResponse {
	return &internaldto.GoalResponse{
		ID:                goal.ID,
		Title:             goal.Title,
		AchievementType:   goal.AchievementType,
		AchievementParams: goal.AchievementParams,
		Attributes:        goal.Attributes,
		Invert:            goal.Invert,
		CreatedAt:         goal.CreatedAt,
	}
}

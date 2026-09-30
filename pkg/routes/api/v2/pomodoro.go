// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package apiv2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MBeggiato/fenster/pkg/db"
	"github.com/MBeggiato/fenster/pkg/models"
	"github.com/MBeggiato/fenster/pkg/web"
	"github.com/MBeggiato/fenster/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"xorm.io/xorm"
)

// pomodoroCurrentBody wraps a nullable session: no timer running is a 200 with
// a null body, not a 404, so the frontend polls one shape.
type pomodoroCurrentBody struct {
	Body *models.PomodoroSession
}

type pomodoroSessionListBody struct {
	Body Paginated[*models.PomodoroSession]
}

type pomodoroStatsBody struct {
	Body *models.PomodoroStats
}

// RegisterPomodoroRoutes wires the personal focus timer onto the Huma API.
//
// Every session belongs to the authenticated user; the user is taken from auth
// and never from a request body. Link shares have no user to own a timer and
// are refused throughout. Pomodoro itself is not license-gated — only the
// optional time-entry logging of a finished focus phase is.
func RegisterPomodoroRoutes(api huma.API) {
	tags := []string{"pomodoro"}

	Register(api, huma.Operation{
		OperationID: "pomodoro-current",
		Summary:     "Get your running pomodoro session",
		Description: "Returns the authenticated user's active session, or null when no timer is running. A session whose phase ran out while the client was away is closed first, so it comes back as null rather than as a session with a negative remainder. Count down locally from remaining_seconds.",
		Method:      http.MethodGet,
		Path:        "/pomodoro/current",
		Tags:        tags,
	}, pomodoroCurrent)

	Register(api, huma.Operation{
		OperationID: "pomodoro-start",
		Summary:     "Start a pomodoro phase",
		Description: "Starts a focus or break phase for the authenticated user. A session that was still running is closed as interrupted, so there is never more than one active timer. task_id is optional; when set you need read access to the task. planned_seconds must be between 60 and 14400.",
		Method:      http.MethodPost,
		Path:        "/pomodoro/sessions",
		Tags:        tags,
	}, pomodoroStart)

	Register(api, huma.Operation{
		OperationID: "pomodoro-pause",
		Summary:     "Pause your running pomodoro session",
		Description: "Pauses the authenticated user's session. Idempotent: pausing an already paused session succeeds and changes nothing. A paused session never expires on its own. Returns 404 when no session is active.",
		Method:      http.MethodPost,
		Path:        "/pomodoro/current/pause",
		// Override the wrapper's POST→201: this changes an existing session.
		DefaultStatus: http.StatusOK,
		Tags:          tags,
	}, pomodoroPause)

	Register(api, huma.Operation{
		OperationID:   "pomodoro-resume",
		Summary:       "Resume your paused pomodoro session",
		Description:   "Resumes the authenticated user's session, so the time it spent paused does not count against the phase. Idempotent: resuming a running session succeeds and changes nothing. Returns 404 when no session is active.",
		Method:        http.MethodPost,
		Path:          "/pomodoro/current/resume",
		DefaultStatus: http.StatusOK,
		Tags:          tags,
	}, pomodoroResume)

	Register(api, huma.Operation{
		OperationID:   "pomodoro-stop",
		Summary:       "Stop your pomodoro session early",
		Description:   "Ends the authenticated user's session now and stores it as interrupted with its real duration: the focus time counts towards the statistics, the pomodoro itself does not count as completed and nothing is logged to time tracking. Returns 404 when no session is active.",
		Method:        http.MethodPost,
		Path:          "/pomodoro/current/stop",
		DefaultStatus: http.StatusOK,
		Tags:          tags,
	}, pomodoroStop)

	Register(api, huma.Operation{
		OperationID: "pomodoro-sessions-list",
		Summary:     "List your pomodoro sessions",
		Description: "Returns the authenticated user's own sessions, newest first, paginated. Other users' sessions are never listed, not even for tasks you share. Optionally narrowed to a time range over started_at or to one task.",
		Method:      http.MethodGet,
		Path:        "/pomodoro/sessions",
		Tags:        tags,
	}, pomodoroSessionsList)

	Register(api, huma.Operation{
		OperationID: "pomodoro-stats",
		Summary:     "Get your pomodoro statistics",
		Description: "Aggregates the authenticated user's finished focus phases in a range: per day, per task and per project, plus the totals. Breaks are left out. Days are bucketed in the requested timezone, so they match the user's own calendar across DST. The range must not be longer than 366 days. Sessions whose task was deleted or is no longer readable are grouped as \"Other\".",
		Method:      http.MethodGet,
		Path:        "/pomodoro/stats",
		Tags:        tags,
	}, pomodoroStats)

	Register(api, huma.Operation{
		OperationID: "task-pomodoro-estimate-update",
		Summary:     "Estimate a task in pomodoros",
		Description: "Sets the authenticated user's personal estimate of how many focus phases a task will take, creating it if there was none. Idempotent. Requires read access to the task; the task itself and other users' estimates are unchanged.",
		Method:      http.MethodPut,
		Path:        "/tasks/{task}/pomodoro-estimate",
		Tags:        tags,
	}, taskPomodoroEstimateUpdate)

	Register(api, huma.Operation{
		OperationID: "task-pomodoro-estimate-delete",
		Summary:     "Clear your pomodoro estimate of a task",
		Description: "Removes the authenticated user's estimate. Idempotent: clearing a task that has no estimate succeeds.",
		Method:      http.MethodDelete,
		Path:        "/tasks/{task}/pomodoro-estimate",
		Tags:        tags,
	}, taskPomodoroEstimateDelete)
}

func init() { AddRouteRegistrar(RegisterPomodoroRoutes) }

// pomodoroCurrent is scoped to the caller's own timer, so it owns its session
// and needs no resource permission beyond authentication. It writes, because
// reading is what closes an overdue session.
func pomodoroCurrent(ctx context.Context, _ *struct{}) (*pomodoroCurrentBody, error) {
	return inPomodoroTx(ctx, models.GetCurrentPomodoroSession)
}

func pomodoroPause(ctx context.Context, _ *struct{}) (*pomodoroCurrentBody, error) {
	return inPomodoroTx(ctx, models.PausePomodoroSession)
}

func pomodoroResume(ctx context.Context, _ *struct{}) (*pomodoroCurrentBody, error) {
	return inPomodoroTx(ctx, models.ResumePomodoroSession)
}

func pomodoroStop(ctx context.Context, _ *struct{}) (*pomodoroCurrentBody, error) {
	return inPomodoroTx(ctx, models.StopPomodoroSession)
}

// inPomodoroTx is the shared shape of the non-CRUD timer actions: they act on
// the caller's own session, so they own the transaction and the permission
// check is the model's link-share refusal.
func inPomodoroTx(ctx context.Context, act func(*xorm.Session, web.Auth) (*models.PomodoroSession, error)) (*pomodoroCurrentBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	session, err := act(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &pomodoroCurrentBody{Body: session}, nil
}

func pomodoroStart(ctx context.Context, in *struct {
	Body models.PomodoroSession
}) (*singleBody[models.PomodoroSession], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.PomodoroSession]{Body: &in.Body}, nil
}

func pomodoroSessionsList(ctx context.Context, in *struct {
	ListParams
	From   time.Time `query:"from" doc:"Only return sessions that started at or after this time (RFC 3339)."`
	To     time.Time `query:"to" doc:"Only return sessions that started at or before this time (RFC 3339)."`
	TaskID int64     `query:"task_id" doc:"Only return sessions about this task. 0 or absent means no restriction."`
}) (*pomodoroSessionListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	m := &models.PomodoroSession{TaskID: in.TaskID}
	if !in.From.IsZero() {
		m.From = &in.From
	}
	if !in.To.IsZero() {
		m.To = &in.To
	}

	result, _, total, err := handler.DoReadAll(ctx, m, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.PomodoroSession)
	if !ok {
		return nil, fmt.Errorf("pomodoroSessions.ReadAll returned unexpected type %T (expected []*models.PomodoroSession)", result)
	}
	return &pomodoroSessionListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

func pomodoroStats(ctx context.Context, in *struct {
	From time.Time `query:"from" required:"true" doc:"Start of the range, inclusive (RFC 3339)."`
	To   time.Time `query:"to" required:"true" doc:"End of the range, inclusive (RFC 3339). At most 366 days after from."`
	TZ   string    `query:"tz" doc:"IANA timezone name the days are bucketed in, e.g. Europe/Berlin. Defaults to UTC."`
}) (*pomodoroStatsBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	stats, err := models.GetPomodoroStats(s, a, in.From, in.To, in.TZ)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	// Committed because reading the stats also closes an overdue session.
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &pomodoroStatsBody{Body: stats}, nil
}

func taskPomodoroEstimateUpdate(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Body   models.TaskPomodoroEstimate
}) (*singleBody[models.TaskPomodoroEstimate], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	e := &in.Body
	e.TaskID = in.TaskID // URL wins over body
	if err := handler.DoUpdate(ctx, e, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.TaskPomodoroEstimate]{Body: e}, nil
}

func taskPomodoroEstimateDelete(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.TaskPomodoroEstimate{TaskID: in.TaskID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

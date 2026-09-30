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

package webtests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MBeggiato/fenster/pkg/db"
	"github.com/MBeggiato/fenster/pkg/models"
	"github.com/MBeggiato/fenster/pkg/modules/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pomodoroLinkShareToken issues a share token on project 1, which holds task 1.
func pomodoroLinkShareToken(t *testing.T) string {
	t.Helper()
	token, err := auth.NewLinkShareJWTAuthtoken(&models.LinkSharing{
		ID:          1,
		Hash:        "test",
		ProjectID:   1,
		Permission:  models.PermissionRead,
		SharingType: models.SharingTypeWithoutPassword,
		SharedByID:  1,
	})
	require.NoError(t, err)
	return token
}

// Fixtures (pkg/db/fixtures/pomodoro_sessions.yml): user 1 has three finished
// focus phases on task 1 plus a break, user 2 has one on task 32, user 6 has an
// overdue running session, user 7 a session paused since 2018 and user 8 an
// overdue running session that asked for time-entry logging.
func TestHumaPomodoro(t *testing.T) {
	h := webHandlerTestV2{user: &testuser1, t: t}
	require.NoError(t, h.ensureEnv())
	tok1 := humaTokenFor(t, &testuser1)
	tok2 := humaTokenFor(t, &testuser2)

	req := func(method, path, body, token string) *httptest.ResponseRecorder {
		return humaRequest(t, h.e, method, path, body, token, "")
	}

	t.Run("Current", func(t *testing.T) {
		t.Run("null when no timer runs", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/pomodoro/current", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.JSONEq(t, "null", rec.Body.String(), "no timer is a 200 with a null body, not a 404")
		})
		t.Run("an expired session is closed and reported as none", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/pomodoro/current", "", humaTokenFor(t, &testuser6))
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.JSONEq(t, "null", rec.Body.String())
			db.AssertExists(t, "pomodoro_sessions", map[string]interface{}{
				"id": 6, "interrupted": false,
			}, false)
		})
		t.Run("link share", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/pomodoro/current", "", pomodoroLinkShareToken(t))
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("unauthenticated", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/pomodoro/current", "", "")
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
		})
	})

	t.Run("Start", func(t *testing.T) {
		t.Run("without a task", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPost, "/api/v2/pomodoro/sessions", `{"phase":"focus","planned_seconds":1500}`, tok1)
			require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"status":"running"`)
			assert.Contains(t, rec.Body.String(), `"phase":"focus"`)
		})
		t.Run("with a readable task", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPost, "/api/v2/pomodoro/sessions", `{"phase":"focus","planned_seconds":1500,"task_id":1}`, tok1)
			require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"task_id":1`)
			assert.Contains(t, rec.Body.String(), `"task":{`, "the session carries its task")
		})
		t.Run("closes the previous session as interrupted", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			first := req(http.MethodPost, "/api/v2/pomodoro/sessions", `{"phase":"focus","planned_seconds":1500}`, tok1)
			require.Equal(t, http.StatusCreated, first.Code, "body: %s", first.Body.String())

			var created struct {
				ID int64 `json:"id"`
			}
			require.NoError(t, json.Unmarshal(first.Body.Bytes(), &created))

			second := req(http.MethodPost, "/api/v2/pomodoro/sessions", `{"phase":"short_break","planned_seconds":300}`, tok1)
			require.Equal(t, http.StatusCreated, second.Code, "body: %s", second.Body.String())
			db.AssertExists(t, "pomodoro_sessions", map[string]interface{}{
				"id": created.ID, "interrupted": true,
			}, false)
		})
		t.Run("unknown phase", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPost, "/api/v2/pomodoro/sessions", `{"phase":"siesta","planned_seconds":1500}`, tok1)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("planned_seconds out of bounds", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPost, "/api/v2/pomodoro/sessions", `{"phase":"focus","planned_seconds":5}`, tok1)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("task the user cannot read", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPost, "/api/v2/pomodoro/sessions", `{"phase":"focus","planned_seconds":1500,"task_id":1}`, tok2)
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("link share", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPost, "/api/v2/pomodoro/sessions", `{"phase":"focus","planned_seconds":1500}`, pomodoroLinkShareToken(t))
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		})
	})

	t.Run("PauseResumeStop", func(t *testing.T) {
		start := func(t *testing.T) {
			t.Helper()
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPost, "/api/v2/pomodoro/sessions", `{"phase":"focus","planned_seconds":1500}`, tok1)
			require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		}

		t.Run("pause is a 200 and idempotent", func(t *testing.T) {
			start(t)
			rec := req(http.MethodPost, "/api/v2/pomodoro/current/pause", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"status":"paused"`)

			again := req(http.MethodPost, "/api/v2/pomodoro/current/pause", "", tok1)
			require.Equal(t, http.StatusOK, again.Code, "body: %s", again.Body.String())
			assert.Contains(t, again.Body.String(), `"status":"paused"`)
		})
		t.Run("resume is a 200 and idempotent", func(t *testing.T) {
			start(t)
			require.Equal(t, http.StatusOK, req(http.MethodPost, "/api/v2/pomodoro/current/pause", "", tok1).Code)

			rec := req(http.MethodPost, "/api/v2/pomodoro/current/resume", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"status":"running"`)

			again := req(http.MethodPost, "/api/v2/pomodoro/current/resume", "", tok1)
			require.Equal(t, http.StatusOK, again.Code, "body: %s", again.Body.String())
		})
		t.Run("stop marks the session interrupted", func(t *testing.T) {
			start(t)
			rec := req(http.MethodPost, "/api/v2/pomodoro/current/stop", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"status":"finished"`)
			assert.Contains(t, rec.Body.String(), `"interrupted":true`)

			current := req(http.MethodGet, "/api/v2/pomodoro/current", "", tok1)
			assert.JSONEq(t, "null", current.Body.String())
		})
		t.Run("404 when nothing is active", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			for _, action := range []string{"pause", "resume", "stop"} {
				rec := req(http.MethodPost, "/api/v2/pomodoro/current/"+action, "", tok1)
				assert.Equal(t, http.StatusNotFound, rec.Code, "%s body: %s", action, rec.Body.String())
			}
		})
		t.Run("link share", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			share := pomodoroLinkShareToken(t)
			for _, action := range []string{"pause", "resume", "stop"} {
				rec := req(http.MethodPost, "/api/v2/pomodoro/current/"+action, "", share)
				assert.Equal(t, http.StatusForbidden, rec.Code, "%s body: %s", action, rec.Body.String())
			}
		})
	})

	t.Run("SessionsList", func(t *testing.T) {
		t.Run("only the caller's own sessions", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/pomodoro/sessions", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

			var body struct {
				Items []struct {
					ID int64 `json:"id"`
				} `json:"items"`
				Total int64 `json:"total"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, int64(4), body.Total, "user 1 has four sessions, user 2's must not show up")
			for _, item := range body.Items {
				assert.NotEqual(t, int64(5), item.ID, "user 2's session must not leak")
			}
		})
		t.Run("filtered by task", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/pomodoro/sessions?task_id=1", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			var body struct {
				Total int64 `json:"total"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, int64(3), body.Total)
		})
		t.Run("filtered by range", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/pomodoro/sessions?from=2018-12-01T11:30:00Z&to=2018-12-01T23:00:00Z", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			var body struct {
				Total int64 `json:"total"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, int64(1), body.Total)
		})
		t.Run("link share", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/pomodoro/sessions", "", pomodoroLinkShareToken(t))
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		})
	})

	t.Run("Stats", func(t *testing.T) {
		const rangeQuery = "?from=2018-12-01T00:00:00Z&to=2018-12-02T00:00:00Z"

		t.Run("totals, days and groups", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/pomodoro/stats"+rangeQuery+"&tz=UTC", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

			var stats models.PomodoroStats
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stats))
			assert.Equal(t, int64(2), stats.Completed)
			assert.Equal(t, int64(1), stats.Interrupted)
			assert.Equal(t, int64((25+10+25)*60), stats.FocusSeconds, "breaks must not count")
			require.Len(t, stats.Days, 1)
			require.Len(t, stats.Tasks, 1)
			assert.Equal(t, int64(1), stats.Tasks[0].ID)
			assert.Equal(t, 5, stats.Tasks[0].Estimate)
		})
		t.Run("from and to are required", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/pomodoro/stats", "", tok1)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("an oversized range is refused", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/pomodoro/stats?from=2018-01-01T00:00:00Z&to=2024-01-01T00:00:00Z", "", tok1)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("an unknown timezone is refused", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/pomodoro/stats"+rangeQuery+"&tz=Mars/Olympus_Mons", "", tok1)
			assert.GreaterOrEqual(t, rec.Code, http.StatusBadRequest, "body: %s", rec.Body.String())
			assert.Less(t, rec.Code, http.StatusInternalServerError, "body: %s", rec.Body.String())
		})
		t.Run("link share", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/pomodoro/stats"+rangeQuery, "", pomodoroLinkShareToken(t))
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		})
	})

	t.Run("Estimate", func(t *testing.T) {
		t.Run("set with read access only", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPut, "/api/v2/tasks/32/pomodoro-estimate", `{"estimate":4}`, tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"estimated":true`)
			db.AssertExists(t, "task_pomodoro_estimates", map[string]interface{}{
				"task_id": 32, "user_id": 1, "estimate": 4,
			}, false)
			db.AssertExists(t, "task_pomodoro_estimates", map[string]interface{}{
				"task_id": 32, "user_id": 2, "estimate": 3,
			}, false)
		})
		t.Run("overwrites the existing estimate in place", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPut, "/api/v2/tasks/1/pomodoro-estimate", `{"estimate":8}`, tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			db.AssertExists(t, "task_pomodoro_estimates", map[string]interface{}{
				"id": 1, "task_id": 1, "user_id": 1, "estimate": 8,
			}, false)
		})
		t.Run("out of bounds", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			for _, body := range []string{`{"estimate":0}`, `{"estimate":100}`} {
				rec := req(http.MethodPut, "/api/v2/tasks/1/pomodoro-estimate", body, tok1)
				assert.GreaterOrEqual(t, rec.Code, http.StatusBadRequest, "%s body: %s", body, rec.Body.String())
				assert.Less(t, rec.Code, http.StatusInternalServerError, "%s body: %s", body, rec.Body.String())
			}
		})
		t.Run("forbidden", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPut, "/api/v2/tasks/1/pomodoro-estimate", `{"estimate":4}`, tok2)
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
			db.AssertMissing(t, "task_pomodoro_estimates", map[string]interface{}{
				"task_id": 1, "user_id": 2,
			})
		})
		t.Run("nonexistent task", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPut, "/api/v2/tasks/99999/pomodoro-estimate", `{"estimate":4}`, tok1)
			assert.Equal(t, http.StatusNotFound, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("link share", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPut, "/api/v2/tasks/1/pomodoro-estimate", `{"estimate":4}`, pomodoroLinkShareToken(t))
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("delete", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodDelete, "/api/v2/tasks/1/pomodoro-estimate", "", tok1)
			require.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body.String())
			db.AssertMissing(t, "task_pomodoro_estimates", map[string]interface{}{
				"task_id": 1, "user_id": 1,
			})

			// Deleting again is a no-op, not a 404.
			again := req(http.MethodDelete, "/api/v2/tasks/1/pomodoro-estimate", "", tok1)
			assert.Equal(t, http.StatusNoContent, again.Code, "body: %s", again.Body.String())
		})
	})

	t.Run("TaskExpand", func(t *testing.T) {
		t.Run("summarises the caller's own history", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/tasks/1?expand=pomodoro", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

			var task struct {
				Pomodoro *models.TaskPomodoroSummary `json:"pomodoro"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &task))
			require.NotNil(t, task.Pomodoro)
			assert.Equal(t, int64(2), task.Pomodoro.Completed)
			assert.Equal(t, int64(1), task.Pomodoro.Interrupted)
			assert.Equal(t, int64((25+10+25)*60), task.Pomodoro.FocusSeconds)
			assert.Equal(t, 5, task.Pomodoro.Estimate)
		})
		t.Run("absent without the expand", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/tasks/1", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.NotContains(t, rec.Body.String(), `"pomodoro"`)
		})
		t.Run("an unknown expand is still refused", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodGet, "/api/v2/tasks/1?expand=nonsense", "", tok1)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		})
	})
}

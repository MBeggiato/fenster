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

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodePaginatedTaskIDs returns the ids of a Paginated[*Task] response.
func decodePaginatedTaskIDs(t *testing.T, rec *httptest.ResponseRecorder) []int64 {
	t.Helper()
	var body struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	ids := make([]int64, 0, len(body.Items))
	for _, item := range body.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

// Fixtures: user 1 classified task 1 (do), 3 (schedule), 4 (delegate),
// 5 (eliminate) and the done task 2 (do). User 2 classified task 32 (do),
// which user 1 can read but left unclassified.
func TestHumaTaskEisenhower(t *testing.T) {
	h := webHandlerTestV2{user: &testuser1, t: t}
	require.NoError(t, h.ensureEnv())
	tok1 := humaTokenFor(t, &testuser1)
	tok2 := humaTokenFor(t, &testuser2)

	req := func(method, path, body, token string) *httptest.ResponseRecorder {
		return humaRequest(t, h.e, method, path, body, token, "")
	}

	t.Run("Read", func(t *testing.T) {
		t.Run("classified", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/tasks/3/eisenhower", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"urgent":false`)
			assert.Contains(t, rec.Body.String(), `"important":true`)
			assert.Contains(t, rec.Body.String(), `"classified":true`)
		})
		t.Run("unclassified", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/tasks/32/eisenhower", "", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"classified":false`)
			assert.Contains(t, rec.Body.String(), `"urgent":false`, "user 2's classification must not leak")
		})
		t.Run("forbidden", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/tasks/1/eisenhower", "", tok2)
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("nonexistent task", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/tasks/99999/eisenhower", "", tok1)
			assert.Equal(t, http.StatusNotFound, rec.Code, "body: %s", rec.Body.String())
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("classify with read access only", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPut, "/api/v2/tasks/32/eisenhower", `{"urgent":false,"important":true}`, tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"classified":true`)
			db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
				"task_id": 32, "user_id": 1, "urgent": false, "important": true,
			}, false)
			db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
				"task_id": 32, "user_id": 2, "urgent": true, "important": true,
			}, false)
		})
		t.Run("move to another quadrant", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPut, "/api/v2/tasks/1/eisenhower", `{"urgent":false,"important":false}`, tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
				"id": 1, "task_id": 1, "user_id": 1, "urgent": false, "important": false,
			}, false)
		})
		t.Run("patch one flag", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := humaRequest(t, h.e, http.MethodPatch, "/api/v2/tasks/1/eisenhower", `{"important":false}`, tok1, "application/merge-patch+json")
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
				"task_id": 1, "user_id": 1, "urgent": true, "important": false,
			}, false)
		})
		t.Run("forbidden", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodPut, "/api/v2/tasks/1/eisenhower", `{"urgent":true,"important":true}`, tok2)
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
			db.AssertMissing(t, "task_eisenhower_classifications", map[string]interface{}{
				"task_id": 1, "user_id": 2,
			})
		})
		t.Run("link share", func(t *testing.T) {
			token, err := auth.NewLinkShareJWTAuthtoken(&models.LinkSharing{
				ID:          1,
				Hash:        "test",
				ProjectID:   1,
				Permission:  models.PermissionRead,
				SharingType: models.SharingTypeWithoutPassword,
				SharedByID:  1,
			})
			require.NoError(t, err)
			rec := req(http.MethodPut, "/api/v2/tasks/1/eisenhower", `{"urgent":true,"important":true}`, token)
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("reset", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodDelete, "/api/v2/tasks/1/eisenhower", "", tok1)
			require.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body.String())
			db.AssertMissing(t, "task_eisenhower_classifications", map[string]interface{}{
				"task_id": 1, "user_id": 1,
			})
		})
		t.Run("unclassified is a no-op", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodDelete, "/api/v2/tasks/32/eisenhower", "", tok1)
			require.Equal(t, http.StatusNoContent, rec.Code, "body: %s", rec.Body.String())
			db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
				"task_id": 32, "user_id": 2,
			}, false)
		})
		t.Run("forbidden", func(t *testing.T) {
			require.NoError(t, db.LoadFixtures())
			rec := req(http.MethodDelete, "/api/v2/tasks/1/eisenhower", "", tok2)
			assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
			db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
				"task_id": 1, "user_id": 1,
			}, false)
		})
	})

	t.Run("List", func(t *testing.T) {
		require.NoError(t, db.LoadFixtures())
		list := func(query string, token string) *httptest.ResponseRecorder {
			return req(http.MethodGet, "/api/v2/eisenhower/tasks?"+query, "", token)
		}

		t.Run("quadrants leave out done tasks by default", func(t *testing.T) {
			for quadrant, want := range map[string][]int64{
				"do":        {1},
				"schedule":  {3},
				"delegate":  {4},
				"eliminate": {5},
			} {
				rec := list("quadrant="+quadrant, tok1)
				require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
				assert.ElementsMatch(t, want, decodePaginatedTaskIDs(t, rec), quadrant)
			}
		})
		t.Run("include done", func(t *testing.T) {
			rec := list("quadrant=do&include_done=true", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.ElementsMatch(t, []int64{1, 2}, decodePaginatedTaskIDs(t, rec))
		})
		t.Run("unclassified is per user", func(t *testing.T) {
			rec := list("quadrant=unclassified&per_page=500", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			ids := decodePaginatedTaskIDs(t, rec)
			assert.Contains(t, ids, int64(32))
			assert.Contains(t, ids, int64(6))
			assert.NotContains(t, ids, int64(1))
			assert.NotContains(t, ids, int64(2), "done tasks are hidden")

			rec = list("quadrant=do", tok2)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Equal(t, []int64{32}, decodePaginatedTaskIDs(t, rec))
		})
		t.Run("combines with filter, search and pagination", func(t *testing.T) {
			rec := list("quadrant=unclassified&filter=project%20in%201&per_page=2&page=1", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Len(t, decodePaginatedTaskIDs(t, rec), 2)
			assert.Contains(t, rec.Body.String(), `"per_page":2`)

			// Only task #6 has the word "unique" in its description.
			rec = list("quadrant=unclassified&q=unique", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Equal(t, []int64{6}, decodePaginatedTaskIDs(t, rec))

			rec = list("quadrant=do&filter=project%20in%203", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Empty(t, decodePaginatedTaskIDs(t, rec))
		})
		t.Run("expand eisenhower", func(t *testing.T) {
			rec := list("quadrant=schedule&expand=eisenhower", tok1)
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"eisenhower":{"task_id":3,"urgent":false,"important":true,"classified":true`)
		})
		t.Run("subtasks expansion is rejected", func(t *testing.T) {
			rec := list("quadrant=do&expand=subtasks", tok1)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
		})
		t.Run("quadrant is required and validated", func(t *testing.T) {
			rec := req(http.MethodGet, "/api/v2/eisenhower/tasks", "", tok1)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
			rec = list("quadrant=urgent", tok1)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		})
	})

	t.Run("expand on task read", func(t *testing.T) {
		require.NoError(t, db.LoadFixtures())
		rec := req(http.MethodGet, "/api/v2/tasks/4?expand=eisenhower", "", tok1)
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), `"eisenhower":{"task_id":4,"urgent":true,"important":false`)

		rec = req(http.MethodGet, "/api/v2/tasks/6?expand=eisenhower", "", tok1)
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.NotContains(t, rec.Body.String(), `"eisenhower"`)
	})
}

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

package models

import (
	"testing"

	"github.com/MBeggiato/fenster/pkg/db"
	"github.com/MBeggiato/fenster/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func TestTaskEisenhowerClassification_CanRead(t *testing.T) {
	t.Run("own classification", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		c := &TaskEisenhowerClassification{TaskID: 3}
		can, _, err := c.CanRead(s, &user.User{ID: 1})
		require.NoError(t, err)
		assert.True(t, can)
		assert.True(t, c.Classified)
		assert.False(t, c.Urgent)
		assert.True(t, c.Important)
	})
	t.Run("unclassified task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		c := &TaskEisenhowerClassification{TaskID: 6}
		can, _, err := c.CanRead(s, &user.User{ID: 1})
		require.NoError(t, err)
		assert.True(t, can)
		assert.False(t, c.Classified)
		assert.False(t, c.Urgent)
		assert.False(t, c.Important)
	})
	t.Run("other user's classification stays hidden", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user 2 classified task 32 as urgent+important, user 1 did not
		c := &TaskEisenhowerClassification{TaskID: 32}
		can, _, err := c.CanRead(s, &user.User{ID: 1})
		require.NoError(t, err)
		assert.True(t, can)
		assert.False(t, c.Classified)
	})
	t.Run("no access to task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		c := &TaskEisenhowerClassification{TaskID: 1}
		can, _, _ := c.CanRead(s, &user.User{ID: 2})
		assert.False(t, can)
		assert.False(t, c.Classified)
	})
	t.Run("link share", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		c := &TaskEisenhowerClassification{TaskID: 1}
		can, _, err := c.CanRead(s, &LinkSharing{ID: 1, ProjectID: 1})
		require.ErrorIs(t, err, ErrGenericForbidden{})
		assert.False(t, can)
	})
}

func TestTaskEisenhowerClassification_Update(t *testing.T) {
	update := func(t *testing.T, a *user.User, taskID int64, urgent, important bool) *TaskEisenhowerClassification {
		t.Helper()
		s := db.NewSession()
		defer s.Close()

		c := &TaskEisenhowerClassification{TaskID: taskID, Urgent: urgent, Important: important}
		can, err := c.CanUpdate(s, a)
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, c.Update(s, a))
		require.NoError(t, s.Commit())
		return c
	}

	t.Run("classify an unclassified task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		c := update(t, &user.User{ID: 1}, 6, true, false)
		assert.True(t, c.Classified)

		db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
			"task_id":   6,
			"user_id":   1,
			"urgent":    true,
			"important": false,
		}, false)
	})
	t.Run("change both flags of an existing classification", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		c := update(t, &user.User{ID: 1}, 1, false, false)
		assert.Equal(t, int64(1), c.ID)

		db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
			"id":        1,
			"task_id":   1,
			"user_id":   1,
			"urgent":    false,
			"important": false,
		}, false)
		db.AssertCount(t, "task_eisenhower_classifications", builder.Eq{"task_id": 1}, 1)
	})
	t.Run("read access is enough and leaves other users alone", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)

		// user 1 only has read access to project 3
		update(t, &user.User{ID: 1}, 32, false, true)

		db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
			"task_id":   32,
			"user_id":   1,
			"urgent":    false,
			"important": true,
		}, false)
		db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
			"id":        6,
			"task_id":   32,
			"user_id":   2,
			"urgent":    true,
			"important": true,
		}, false)
	})
	t.Run("no access to task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		c := &TaskEisenhowerClassification{TaskID: 1, Urgent: true}
		can, _ := c.CanUpdate(s, &user.User{ID: 2})
		assert.False(t, can)
	})
	t.Run("nonexistent task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		c := &TaskEisenhowerClassification{TaskID: 99999}
		can, err := c.CanUpdate(s, &user.User{ID: 1})
		require.Error(t, err)
		assert.True(t, IsErrTaskDoesNotExist(err))
		assert.False(t, can)
	})
	t.Run("link share", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		c := &TaskEisenhowerClassification{TaskID: 1}
		can, err := c.CanUpdate(s, &LinkSharing{ID: 1, ProjectID: 1})
		require.ErrorIs(t, err, ErrGenericForbidden{})
		assert.False(t, can)
	})
}

func TestTaskEisenhowerClassification_Delete(t *testing.T) {
	t.Run("reset own classification only", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user 1 classifies task 32 as well, then resets it
		_, err := s.Insert(&TaskEisenhowerClassification{TaskID: 32, UserID: 1, Urgent: true})
		require.NoError(t, err)

		u := &user.User{ID: 1}
		c := &TaskEisenhowerClassification{TaskID: 32}
		can, err := c.CanDelete(s, u)
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, c.Delete(s, u))
		require.NoError(t, s.Commit())

		db.AssertMissing(t, "task_eisenhower_classifications", map[string]interface{}{
			"task_id": 32,
			"user_id": 1,
		})
		db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
			"task_id": 32,
			"user_id": 2,
		}, false)
	})
	t.Run("unclassified task is a no-op", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u := &user.User{ID: 1}
		c := &TaskEisenhowerClassification{TaskID: 6}
		can, err := c.CanDelete(s, u)
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, c.Delete(s, u))
		require.NoError(t, s.Commit())
	})
	t.Run("no access to task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		c := &TaskEisenhowerClassification{TaskID: 1}
		can, _ := c.CanDelete(s, &user.User{ID: 2})
		assert.False(t, can)
	})
	t.Run("soft delete keeps classifications for a restore", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		task := &Task{ID: 1}
		require.NoError(t, task.Delete(s, &user.User{ID: 1}))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "task_eisenhower_classifications", map[string]interface{}{
			"task_id": 1,
		}, false)
	})
	t.Run("hard delete removes every user's classification", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		require.NoError(t, hardDeleteTask(s, &Task{ID: 32}))
		require.NoError(t, s.Commit())

		db.AssertMissing(t, "task_eisenhower_classifications", map[string]interface{}{
			"task_id": 32,
		})
	})
}

func TestTaskCollection_EisenhowerQuadrant(t *testing.T) {
	readQuadrant := func(t *testing.T, a *user.User, q EisenhowerQuadrant, filter string) map[int64]bool {
		t.Helper()
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		tc := &TaskCollection{Filter: filter}
		tc.SetForceFlatTasks()
		tc.SetEisenhowerQuadrant(q)
		result, _, _, err := tc.ReadAll(s, a, "", 1, 500)
		require.NoError(t, err)
		tasks, ok := result.([]*Task)
		require.True(t, ok)
		return taskIDsOf(tasks)
	}

	u1 := &user.User{ID: 1}

	t.Run("do", func(t *testing.T) {
		ids := readQuadrant(t, u1, EisenhowerQuadrantDo, "")
		assert.Equal(t, map[int64]bool{1: true, 2: true}, ids)
	})
	t.Run("do without done tasks", func(t *testing.T) {
		ids := readQuadrant(t, u1, EisenhowerQuadrantDo, "done = false")
		assert.Equal(t, map[int64]bool{1: true}, ids)
	})
	t.Run("schedule", func(t *testing.T) {
		ids := readQuadrant(t, u1, EisenhowerQuadrantSchedule, "")
		assert.Equal(t, map[int64]bool{3: true}, ids)
	})
	t.Run("delegate", func(t *testing.T) {
		ids := readQuadrant(t, u1, EisenhowerQuadrantDelegate, "")
		assert.Equal(t, map[int64]bool{4: true}, ids)
	})
	t.Run("eliminate", func(t *testing.T) {
		ids := readQuadrant(t, u1, EisenhowerQuadrantEliminate, "")
		assert.Equal(t, map[int64]bool{5: true}, ids)
	})
	t.Run("unclassified ignores other users' classifications", func(t *testing.T) {
		ids := readQuadrant(t, u1, EisenhowerQuadrantUnclassified, "")
		assert.True(t, ids[32], "task 32 is only classified by user 2")
		assert.True(t, ids[6])
		for _, classified := range []int64{1, 2, 3, 4, 5} {
			assert.False(t, ids[classified], "task %d is classified by user 1", classified)
		}
	})
	t.Run("soft-deleted tasks are left out", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		require.NoError(t, (&Task{ID: 1}).Delete(s, u1))
		require.NoError(t, s.Commit())
		s.Close()

		s = db.NewSession()
		defer s.Close()
		tc := &TaskCollection{}
		tc.SetForceFlatTasks()
		tc.SetEisenhowerQuadrant(EisenhowerQuadrantDo)
		result, _, _, err := tc.ReadAll(s, u1, "", 1, 500)
		require.NoError(t, err)
		tasks, ok := result.([]*Task)
		require.True(t, ok)
		assert.Equal(t, map[int64]bool{2: true}, taskIDsOf(tasks))
	})
	t.Run("other user's matrix", func(t *testing.T) {
		ids := readQuadrant(t, &user.User{ID: 2}, EisenhowerQuadrantDo, "")
		assert.Equal(t, map[int64]bool{32: true}, ids)
	})
	t.Run("invalid quadrant", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		tc := &TaskCollection{}
		tc.SetEisenhowerQuadrant("urgent")
		_, _, _, err := tc.ReadAll(s, u1, "", 1, 50)
		require.Error(t, err)
		assert.True(t, IsErrInvalidEisenhowerQuadrant(err))
	})
	t.Run("link share is refused", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		tc := &TaskCollection{}
		tc.SetEisenhowerQuadrant(EisenhowerQuadrantDo)
		_, _, _, err := tc.ReadAll(s, &LinkSharing{ID: 1, ProjectID: 1}, "", 1, 50)
		require.ErrorIs(t, err, ErrGenericForbidden{})
	})
}

func TestAddEisenhowerToTasks(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	taskMap := map[int64]*Task{1: {ID: 1}, 6: {ID: 6}, 32: {ID: 32}}
	require.NoError(t, addEisenhowerToTasks(s, []int64{1, 6, 32}, taskMap, &user.User{ID: 1}))

	require.NotNil(t, taskMap[1].Eisenhower)
	assert.True(t, taskMap[1].Eisenhower.Urgent)
	assert.True(t, taskMap[1].Eisenhower.Important)
	assert.True(t, taskMap[1].Eisenhower.Classified)
	assert.Nil(t, taskMap[6].Eisenhower)
	assert.Nil(t, taskMap[32].Eisenhower, "user 2's classification must not leak to user 1")
}

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
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/license"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

func TestPomodoroSession_Create(t *testing.T) {
	t.Run("without a task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		p := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: 1500}
		require.NoError(t, p.Create(s, &user.User{ID: 1}))
		assert.NotZero(t, p.ID)
		assert.Equal(t, PomodoroStatusRunning, p.Status)
		assert.Positive(t, p.RemainingSeconds)
		assert.Nil(t, p.Task)
	})
	t.Run("with a readable task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		p := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: 1500, TaskID: 1}
		can, err := p.CanCreate(s, &user.User{ID: 1})
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, p.Create(s, &user.User{ID: 1}))
		require.NotNil(t, p.Task)
		assert.Equal(t, int64(1), p.Task.ID)
	})
	t.Run("interrupts the session that was still running", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		first := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: 1500}
		require.NoError(t, first.Create(s, &user.User{ID: 1}))

		second := &PomodoroSession{Phase: PomodoroPhaseShortBreak, PlannedSeconds: 300}
		require.NoError(t, second.Create(s, &user.User{ID: 1}))

		closed := &PomodoroSession{}
		has, err := s.ID(first.ID).Get(closed)
		require.NoError(t, err)
		require.True(t, has)
		assert.NotNil(t, closed.EndedAt, "the previous session must be closed")
		assert.True(t, closed.Interrupted, "being replaced counts as interrupted")

		// Exactly one session is left active.
		active, err := getActivePomodoroSession(s, 1)
		require.NoError(t, err)
		require.NotNil(t, active)
		assert.Equal(t, second.ID, active.ID)
	})
	t.Run("rejects an unknown phase", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		p := &PomodoroSession{Phase: "siesta", PlannedSeconds: 1500}
		require.Error(t, p.Create(s, &user.User{ID: 1}))
	})
	t.Run("rejects a phase outside the bounds", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		for _, seconds := range []int64{0, 59, pomodoroMaxSeconds + 1} {
			p := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: seconds}
			require.Error(t, p.Create(s, &user.User{ID: 1}), "planned_seconds %d must be rejected", seconds)
		}
	})
	t.Run("link share is refused", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		p := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: 1500}
		_, err := p.CanCreate(s, &LinkSharing{ID: 1})
		require.Error(t, err)
		require.Error(t, p.Create(s, &LinkSharing{ID: 1}))
	})
	t.Run("task the user cannot read", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// task 1 lives in project 1, owned by user 1 and not shared with user 13
		p := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: 1500, TaskID: 1}
		can, _ := p.CanCreate(s, &user.User{ID: 13})
		assert.False(t, can)
	})
}

func TestPomodoroSession_PauseResumeStop(t *testing.T) {
	t.Run("pause then resume keeps the paused time out of the elapsed time", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u := &user.User{ID: 1}
		p := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: 1500}
		require.NoError(t, p.Create(s, u))

		paused, err := PausePomodoroSession(s, u)
		require.NoError(t, err)
		assert.Equal(t, PomodoroStatusPaused, paused.Status)
		require.NotNil(t, paused.PausedAt)

		// A paused session has no due time, so it never auto-completes.
		_, ok := paused.dueAt()
		assert.False(t, ok)

		// Pausing again is a no-op rather than an error.
		pausedAt := *paused.PausedAt
		again, err := PausePomodoroSession(s, u)
		require.NoError(t, err)
		assert.Equal(t, pausedAt.Unix(), again.PausedAt.Unix())

		resumed, err := ResumePomodoroSession(s, u)
		require.NoError(t, err)
		assert.Equal(t, PomodoroStatusRunning, resumed.Status)
		assert.Nil(t, resumed.PausedAt)

		// Resuming a running session is a no-op too.
		_, err = ResumePomodoroSession(s, u)
		require.NoError(t, err)
	})
	t.Run("stop stores the session as interrupted", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u := &user.User{ID: 1}
		p := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: 1500}
		require.NoError(t, p.Create(s, u))

		stopped, err := StopPomodoroSession(s, u)
		require.NoError(t, err)
		assert.Equal(t, PomodoroStatusFinished, stopped.Status)
		assert.True(t, stopped.Interrupted)
		assert.Zero(t, stopped.RemainingSeconds)
		assert.Nil(t, stopped.PausedAt, "stop must close an open pause")
	})
	t.Run("stop closes an open pause", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u := &user.User{ID: 1}
		p := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: 1500}
		require.NoError(t, p.Create(s, u))
		_, err := PausePomodoroSession(s, u)
		require.NoError(t, err)

		stopped, err := StopPomodoroSession(s, u)
		require.NoError(t, err)
		assert.Nil(t, stopped.PausedAt)
		assert.NotNil(t, stopped.EndedAt)
	})
	t.Run("nothing running", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user 12 has no sessions at all
		u := &user.User{ID: 12}
		_, err := PausePomodoroSession(s, u)
		assert.True(t, IsErrNoActivePomodoroSession(err))
		_, err = ResumePomodoroSession(s, u)
		assert.True(t, IsErrNoActivePomodoroSession(err))
		_, err = StopPomodoroSession(s, u)
		assert.True(t, IsErrNoActivePomodoroSession(err))
	})
	t.Run("link share is refused", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		_, err := PausePomodoroSession(s, &LinkSharing{ID: 1})
		require.Error(t, err)
		_, err = StopPomodoroSession(s, &LinkSharing{ID: 1})
		require.Error(t, err)
		_, err = GetCurrentPomodoroSession(s, &LinkSharing{ID: 1})
		require.Error(t, err)
	})
}

func TestPomodoroSession_elapsed(t *testing.T) {
	start := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

	t.Run("closed pauses are subtracted", func(t *testing.T) {
		p := &PomodoroSession{StartedAt: start, PausedSeconds: 300}
		assert.Equal(t, 25*time.Minute, p.elapsed(start.Add(30*time.Minute)))
	})
	t.Run("an open pause is subtracted too", func(t *testing.T) {
		pausedAt := start.Add(10 * time.Minute)
		p := &PomodoroSession{StartedAt: start, PausedAt: &pausedAt}
		// 30 wall minutes, paused for the last 20 of them
		assert.Equal(t, 10*time.Minute, p.elapsed(start.Add(30*time.Minute)))
	})
	t.Run("never negative", func(t *testing.T) {
		p := &PomodoroSession{StartedAt: start}
		assert.Zero(t, p.elapsed(start.Add(-time.Hour)))
	})
	t.Run("the due time includes the pauses", func(t *testing.T) {
		p := &PomodoroSession{StartedAt: start, PlannedSeconds: 1500, PausedSeconds: 300}
		due, ok := p.dueAt()
		require.True(t, ok)
		assert.Equal(t, start.Add(30*time.Minute), due)
	})
}

func TestPomodoroSession_lazyFinalize(t *testing.T) {
	t.Run("an overdue session is completed at its due time", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// Session 6 started 2018-12-01 13:00 with a 1500s phase.
		current, err := GetCurrentPomodoroSession(s, &user.User{ID: 6})
		require.NoError(t, err)
		assert.Nil(t, current, "an expired session is not current any more")

		closed := &PomodoroSession{}
		has, err := s.ID(6).Get(closed)
		require.NoError(t, err)
		require.True(t, has)
		require.NotNil(t, closed.EndedAt)
		assert.False(t, closed.Interrupted, "running out of time is not an interruption")
		assert.Equal(t,
			time.Date(2018, 12, 1, 13, 25, 0, 0, time.UTC).Unix(),
			closed.EndedAt.UTC().Unix(),
			"it must be stamped at the due time, not at now",
		)
	})
	t.Run("a paused session is left alone", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		current, err := GetCurrentPomodoroSession(s, &user.User{ID: 7})
		require.NoError(t, err)
		require.NotNil(t, current, "a paused session never expires on its own")
		assert.Equal(t, PomodoroStatusPaused, current.Status)
		assert.Nil(t, current.EndedAt)
	})
}

func TestPomodoroSession_logTimeEntry(t *testing.T) {
	countEntries := func(t *testing.T, s *xorm.Session, userID int64) int64 {
		t.Helper()
		count, err := s.Where("user_id = ?", userID).Count(&TimeEntry{})
		require.NoError(t, err)
		return count
	}

	t.Run("a finished focus phase is logged when licensed", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		license.SetForTests([]license.Feature{license.FeatureTimeTracking})
		defer license.ResetForTests()

		before := countEntries(t, s, 8)
		// Session 8 has log_time_entry and is overdue, so reading it finalizes it.
		_, err := GetCurrentPomodoroSession(s, &user.User{ID: 8})
		require.NoError(t, err)
		assert.Equal(t, before+1, countEntries(t, s, 8))

		entry := &TimeEntry{}
		has, err := s.Where("user_id = ?", 8).Desc("id").Get(entry)
		require.NoError(t, err)
		require.True(t, has)
		assert.Equal(t, int64(1), entry.TaskID)
		require.NotNil(t, entry.EndTime)
		assert.Equal(t, 25*time.Minute, entry.EndTime.Sub(entry.StartTime))
	})
	t.Run("nothing is logged when the feature is unlicensed", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		license.ResetForTests()

		before := countEntries(t, s, 8)
		_, err := GetCurrentPomodoroSession(s, &user.User{ID: 8})
		require.NoError(t, err)
		assert.Equal(t, before, countEntries(t, s, 8))
	})
	t.Run("a session without a task logs nothing", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		license.SetForTests([]license.Feature{license.FeatureTimeTracking})
		defer license.ResetForTests()

		before := countEntries(t, s, 6)
		// Session 6 is overdue but carries no task.
		_, err := GetCurrentPomodoroSession(s, &user.User{ID: 6})
		require.NoError(t, err)
		assert.Equal(t, before, countEntries(t, s, 6))
	})
	t.Run("a break logs nothing", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		license.SetForTests([]license.Feature{license.FeatureTimeTracking})
		defer license.ResetForTests()

		u := &user.User{ID: 1}
		before := countEntries(t, s, 1)
		p := &PomodoroSession{Phase: PomodoroPhaseShortBreak, PlannedSeconds: 300, TaskID: 1, LogTimeEntry: true}
		require.NoError(t, p.Create(s, u))
		require.NoError(t, p.stop(s, false))
		assert.Equal(t, before, countEntries(t, s, 1))
	})
	t.Run("an interrupted phase logs nothing", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		license.SetForTests([]license.Feature{license.FeatureTimeTracking})
		defer license.ResetForTests()

		u := &user.User{ID: 1}
		before := countEntries(t, s, 1)
		p := &PomodoroSession{Phase: PomodoroPhaseFocus, PlannedSeconds: 1500, TaskID: 1, LogTimeEntry: true}
		require.NoError(t, p.Create(s, u))
		_, err := StopPomodoroSession(s, u)
		require.NoError(t, err)
		assert.Equal(t, before, countEntries(t, s, 1))
	})
}

func TestPomodoroSession_ReadAll(t *testing.T) {
	t.Run("only the caller's own sessions", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		result, _, _, err := (&PomodoroSession{}).ReadAll(s, &user.User{ID: 1}, "", 1, 50)
		require.NoError(t, err)
		sessions, ok := result.([]*PomodoroSession)
		require.True(t, ok)
		require.NotEmpty(t, sessions)
		for _, session := range sessions {
			assert.Equal(t, int64(1), session.UserID)
		}
	})
	t.Run("filtered by task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		result, _, _, err := (&PomodoroSession{TaskID: 1}).ReadAll(s, &user.User{ID: 1}, "", 1, 50)
		require.NoError(t, err)
		sessions := result.([]*PomodoroSession)
		require.Len(t, sessions, 3)
		for _, session := range sessions {
			assert.Equal(t, int64(1), session.TaskID)
		}
	})
	t.Run("filtered by range", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		from := time.Date(2018, 12, 1, 11, 30, 0, 0, time.UTC)
		to := time.Date(2018, 12, 1, 23, 0, 0, 0, time.UTC)
		result, _, _, err := (&PomodoroSession{From: &from, To: &to}).ReadAll(s, &user.User{ID: 1}, "", 1, 50)
		require.NoError(t, err)
		sessions := result.([]*PomodoroSession)
		require.Len(t, sessions, 1)
		assert.Equal(t, int64(4), sessions[0].ID)
	})
	t.Run("link share is refused", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		_, _, _, err := (&PomodoroSession{}).ReadAll(s, &LinkSharing{ID: 1}, "", 1, 50)
		require.Error(t, err)
	})
}

func TestTaskPomodoroEstimate(t *testing.T) {
	t.Run("read an existing estimate", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		e := &TaskPomodoroEstimate{TaskID: 1}
		can, _, err := e.CanRead(s, &user.User{ID: 1})
		require.NoError(t, err)
		assert.True(t, can)
		assert.True(t, e.Estimated)
		assert.Equal(t, 5, e.Estimate)
	})
	t.Run("an unestimated task is not a 404", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		e := &TaskPomodoroEstimate{TaskID: 6}
		can, _, err := e.CanRead(s, &user.User{ID: 1})
		require.NoError(t, err)
		assert.True(t, can)
		assert.False(t, e.Estimated)
		assert.Zero(t, e.Estimate)
	})
	t.Run("another user's estimate stays hidden", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user 2 estimated task 32; user 1 shares the task but has no estimate
		e := &TaskPomodoroEstimate{TaskID: 32}
		can, _, err := e.CanRead(s, &user.User{ID: 1})
		require.NoError(t, err)
		assert.True(t, can)
		assert.False(t, e.Estimated)
	})
	t.Run("no access to the task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		e := &TaskPomodoroEstimate{TaskID: 1}
		can, _, _ := e.CanRead(s, &user.User{ID: 13})
		assert.False(t, can)
	})
	t.Run("link share", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		e := &TaskPomodoroEstimate{TaskID: 1}
		_, _, err := e.CanRead(s, &LinkSharing{ID: 1})
		require.Error(t, err)
	})
	t.Run("upsert", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u := &user.User{ID: 1}

		// insert
		created := &TaskPomodoroEstimate{TaskID: 6, Estimate: 2}
		can, err := created.CanUpdate(s, u)
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, created.Update(s, u))
		assert.True(t, created.Estimated)

		// update, same row
		updated := &TaskPomodoroEstimate{TaskID: 6, Estimate: 4}
		can, err = updated.CanUpdate(s, u)
		require.NoError(t, err)
		require.True(t, can)
		assert.Equal(t, created.ID, updated.ID, "the second write must reuse the row")
		require.NoError(t, updated.Update(s, u))

		count, err := s.Where("task_id = ? AND user_id = ?", 6, 1).Count(&TaskPomodoroEstimate{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
	t.Run("bounds", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		for _, estimate := range []int{0, -1, 100} {
			e := &TaskPomodoroEstimate{TaskID: 1, Estimate: estimate}
			_, err := e.CanUpdate(s, &user.User{ID: 1})
			require.Error(t, err, "estimate %d must be rejected", estimate)
		}
	})
	t.Run("delete", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u := &user.User{ID: 1}
		e := &TaskPomodoroEstimate{TaskID: 1}
		can, err := e.CanDelete(s, u)
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, e.Delete(s, u))

		count, err := s.Where("task_id = ? AND user_id = ?", 1, 1).Count(&TaskPomodoroEstimate{})
		require.NoError(t, err)
		assert.Zero(t, count)

		// deleting again is a no-op
		require.NoError(t, e.Delete(s, u))
	})
}

func TestAddPomodoroToTasks(t *testing.T) {
	t.Run("summarises the caller's own focus phases", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		task := &Task{ID: 1}
		taskMap := map[int64]*Task{1: task}
		require.NoError(t, addPomodoroToTasks(s, &user.User{ID: 1}, []int64{1}, taskMap))

		require.NotNil(t, task.Pomodoro)
		// sessions 1 (25m, done), 2 (10m, interrupted), 4 (30m wall - 5m paused, done)
		assert.Equal(t, int64(2), task.Pomodoro.Completed)
		assert.Equal(t, int64(1), task.Pomodoro.Interrupted)
		assert.Equal(t, int64((25+10+25)*60), task.Pomodoro.FocusSeconds)
		assert.Equal(t, 5, task.Pomodoro.Estimate)
	})
	t.Run("another user sees their own numbers", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		task := &Task{ID: 1}
		require.NoError(t, addPomodoroToTasks(s, &user.User{ID: 2}, []int64{1}, map[int64]*Task{1: task}))
		require.NotNil(t, task.Pomodoro)
		assert.Zero(t, task.Pomodoro.Completed)
		assert.Zero(t, task.Pomodoro.FocusSeconds)
		assert.Zero(t, task.Pomodoro.Estimate)
	})
	t.Run("link shares get nothing", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		task := &Task{ID: 1}
		require.NoError(t, addPomodoroToTasks(s, &LinkSharing{ID: 1}, []int64{1}, map[int64]*Task{1: task}))
		assert.Nil(t, task.Pomodoro)
	})
}

func TestGetPomodoroStats(t *testing.T) {
	from := time.Date(2018, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC)

	t.Run("totals and per-day buckets", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		stats, err := GetPomodoroStats(s, &user.User{ID: 1}, from, to, "UTC")
		require.NoError(t, err)
		assert.Equal(t, int64(2), stats.Completed)
		assert.Equal(t, int64(1), stats.Interrupted)
		assert.Equal(t, int64((25+10+25)*60), stats.FocusSeconds)
		require.Len(t, stats.Days, 1)
		assert.Equal(t, "2018-12-01", stats.Days[0].Date)
		assert.Equal(t, int64((25+10+25)*60), stats.Days[0].FocusSeconds)
	})
	t.Run("breaks do not count", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		stats, err := GetPomodoroStats(s, &user.User{ID: 1}, from, to, "UTC")
		require.NoError(t, err)
		// Session 3 is a 5 minute short break and must be absent from the total.
		assert.Equal(t, int64((25+10+25)*60), stats.FocusSeconds)
	})
	t.Run("grouped by task and project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		stats, err := GetPomodoroStats(s, &user.User{ID: 1}, from, to, "UTC")
		require.NoError(t, err)
		require.Len(t, stats.Tasks, 1)
		assert.Equal(t, int64(1), stats.Tasks[0].ID)
		assert.Equal(t, 5, stats.Tasks[0].Estimate)
		require.Len(t, stats.Projects, 1)
		assert.Equal(t, int64(1), stats.Projects[0].ID)
	})
	t.Run("an unreadable task is grouped as Other", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user 2's focus phase is on task 32, which they can read; point it at a
		// task in a project they have no access to instead.
		_, err := s.ID(5).Cols("task_id").Update(&PomodoroSession{TaskID: 1})
		require.NoError(t, err)

		stats, err := GetPomodoroStats(s, &user.User{ID: 2}, from, to, "UTC")
		require.NoError(t, err)
		require.Len(t, stats.Tasks, 1)
		assert.Equal(t, "Other", stats.Tasks[0].Title)
		assert.Equal(t, int64(25*60), stats.Tasks[0].FocusSeconds)
	})
	t.Run("buckets by the user's own timezone", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// The three focus phases are at 10:00, 11:00 and 12:00 UTC, so in UTC
		// they are one day...
		stats, err := GetPomodoroStats(s, &user.User{ID: 1}, from, to, "UTC")
		require.NoError(t, err)
		require.Len(t, stats.Days, 1)

		// ...but at UTC+13 the last two have already crossed midnight.
		stats, err = GetPomodoroStats(s, &user.User{ID: 1}, from, to, "Pacific/Auckland")
		require.NoError(t, err)
		require.Len(t, stats.Days, 2)
		assert.Equal(t, "2018-12-01", stats.Days[0].Date)
		assert.Equal(t, int64(25*60), stats.Days[0].FocusSeconds)
		assert.Equal(t, "2018-12-02", stats.Days[1].Date)
		assert.Equal(t, int64((10+25)*60), stats.Days[1].FocusSeconds)
	})
	t.Run("survives a DST boundary", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// Europe/Berlin falls back at 03:00 on 2026-10-25, making that local day
		// 25 hours long. Two sessions 24 hours apart in UTC therefore land on the
		// same local day, which is exactly what a SQL date function would get
		// wrong.
		berlin, err := time.LoadLocation("Europe/Berlin")
		require.NoError(t, err)

		for i, start := range []time.Time{
			time.Date(2026, 10, 25, 0, 30, 0, 0, time.UTC),  // 02:30 CEST
			time.Date(2026, 10, 25, 22, 30, 0, 0, time.UTC), // 23:30 CET, same local day
		} {
			end := start.Add(25 * time.Minute)
			_, err = s.Insert(&PomodoroSession{
				UserID:         1,
				Phase:          PomodoroPhaseFocus,
				PlannedSeconds: 1500,
				StartedAt:      start,
				EndedAt:        &end,
			})
			require.NoError(t, err, "session %d", i)
		}

		stats, err := GetPomodoroStats(s,
			&user.User{ID: 1},
			time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC),
			"Europe/Berlin",
		)
		require.NoError(t, err)
		require.Len(t, stats.Days, 1, "both sessions fall on the same 25-hour local day")
		assert.Equal(t, "2026-10-25", stats.Days[0].Date)
		assert.Equal(t, int64(2), stats.Days[0].Completed)

		// Sanity check that the boundary really is where we think it is.
		assert.NotEqual(t,
			time.Date(2026, 10, 25, 0, 30, 0, 0, time.UTC).In(berlin).Format(time.RFC3339),
			time.Date(2026, 10, 25, 22, 30, 0, 0, time.UTC).In(berlin).Format(time.RFC3339),
		)
	})
	t.Run("rejects a reversed or oversized range", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u := &user.User{ID: 1}
		_, err := GetPomodoroStats(s, u, to, from, "UTC")
		require.Error(t, err)
		_, err = GetPomodoroStats(s, u, from, from.AddDate(2, 0, 0), "UTC")
		require.Error(t, err)
	})
	t.Run("rejects an unknown timezone", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		_, err := GetPomodoroStats(s, &user.User{ID: 1}, from, to, "Mars/Olympus_Mons")
		require.Error(t, err)
	})
	t.Run("link share is refused", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		_, err := GetPomodoroStats(s, &LinkSharing{ID: 1}, from, to, "UTC")
		require.Error(t, err)
	})
}

func TestPomodoroCleanup(t *testing.T) {
	t.Run("deleting a task keeps the sessions but detaches them", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		require.NoError(t, hardDeleteTask(s, &Task{ID: 1}))

		count, err := s.Where("task_id = ?", 1).Count(&PomodoroSession{})
		require.NoError(t, err)
		assert.Zero(t, count, "sessions must be detached from the deleted task")

		// The focus time itself survives: sessions 1, 2 and 4 are still there.
		count, err = s.Where("user_id = ? AND task_id = 0 AND phase = ?", 1, PomodoroPhaseFocus).Count(&PomodoroSession{})
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)

		count, err = s.Where("task_id = ?", 1).Count(&TaskPomodoroEstimate{})
		require.NoError(t, err)
		assert.Zero(t, count, "estimates of a deleted task are gone")
	})
}

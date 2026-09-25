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
	"cmp"
	"slices"
	"time"

	"code.vikunja.io/api/pkg/license"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// PomodoroPhase is one leg of the pomodoro cycle.
type PomodoroPhase string

const (
	PomodoroPhaseFocus      PomodoroPhase = "focus"
	PomodoroPhaseShortBreak PomodoroPhase = "short_break"
	PomodoroPhaseLongBreak  PomodoroPhase = "long_break"
)

// Pomodoro session bounds. A phase shorter than a minute is a mistap, one
// longer than four hours is not a pomodoro.
const (
	pomodoroMinSeconds = 60
	pomodoroMaxSeconds = 4 * 60 * 60

	pomodoroMinEstimate = 1
	pomodoroMaxEstimate = 99

	// pomodoroMaxStatsDays caps a stats range so one request cannot ask the
	// server to bucket an unbounded number of sessions.
	pomodoroMaxStatsDays = 366
)

// Computed status values of a session, derived from its timestamps.
const (
	PomodoroStatusRunning  = "running"
	PomodoroStatusPaused   = "paused"
	PomodoroStatusFinished = "finished"
)

// PomodoroSession is one focus or break phase of a user's pomodoro cycle. It is
// private to that user.
//
// The lifecycle lives in the timestamps rather than in a status column, so the
// two can never disagree: EndedAt null + PausedAt null is running, PausedAt set
// is paused, EndedAt set is finished. Only "finished how" needs storing, which
// Interrupted covers.
type PomodoroSession struct {
	ID     int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this session."`
	UserID int64 `xorm:"bigint not null index" json:"-"`

	// TaskID is 0 for a session that is not tied to a task. It is also reset to
	// 0 when the task is deleted, so the focus time survives in the statistics.
	TaskID int64 `xorm:"bigint not null default 0 index" json:"task_id" doc:"The task this session is about, or 0 for a session without a task."`

	Phase          PomodoroPhase `xorm:"varchar(20) not null" json:"phase" enum:"focus,short_break,long_break" doc:"Which leg of the cycle this session is: focus, short_break or long_break."`
	PlannedSeconds int64         `xorm:"bigint not null" json:"planned_seconds" minimum:"60" maximum:"14400" doc:"How long the phase was planned to run, in seconds. Between 60 and 14400."`

	StartedAt time.Time  `xorm:"not null index" json:"started_at" readOnly:"true" doc:"When the phase started."`
	EndedAt   *time.Time `xorm:"null" json:"ended_at" readOnly:"true" doc:"When the phase ended. Null while it is still running or paused."`
	PausedAt  *time.Time `xorm:"null" json:"paused_at" readOnly:"true" doc:"When the running pause started. Null unless the session is paused."`

	// PausedSeconds is the total of all *closed* pauses. An open pause is added
	// on top when the elapsed time is computed, and folded in on resume.
	PausedSeconds int64 `xorm:"bigint not null default 0" json:"paused_seconds" readOnly:"true" doc:"Total seconds this session spent paused, excluding a pause that is still running."`

	Interrupted bool `xorm:"not null default false" json:"interrupted" readOnly:"true" doc:"True when the session was stopped early or replaced by a new one. An interrupted focus phase counts as focus time but not as a completed pomodoro."`

	// LogTimeEntry is frozen at start time so a session finalized much later
	// still logs what the user asked for back then.
	LogTimeEntry bool `xorm:"not null default false" json:"log_time_entry" doc:"Whether a finished focus phase should also be logged as a time entry. Only honoured with a task and a licensed time_tracking feature."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when this session was created."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"A timestamp when this session was last updated."`

	// Status is the lifecycle derived from the timestamps, so the frontend does
	// not have to re-derive it.
	Status string `xorm:"-" json:"status" readOnly:"true" enum:"running,paused,finished" doc:"The derived lifecycle of this session: running, paused or finished."`
	// RemainingSeconds is how much of the phase is left at the moment of the
	// response; the client counts down locally from there. 0 for a finished session.
	RemainingSeconds int64 `xorm:"-" json:"remaining_seconds" readOnly:"true" doc:"Seconds left in the phase when the response was built. Count down locally from here."`
	// Task is the slim task this session is about, when it has one and the
	// caller can still read it.
	Task *Task `xorm:"-" json:"task,omitempty" readOnly:"true" doc:"The task this session is about, when it has one and you can read it."`

	// Filter-only fields (not persisted): set by the list route, read by ReadAll.
	From *time.Time `xorm:"-" json:"-"`
	To   *time.Time `xorm:"-" json:"-"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*PomodoroSession) TableName() string {
	return "pomodoro_sessions"
}

// --- time arithmetic ---

// elapsed is the effective time the phase has actually been running at the
// given moment: wall time minus every pause, including one still open. Floored
// at zero so a clock that jumped backwards cannot produce a negative duration.
func (p *PomodoroSession) elapsed(at time.Time) time.Duration {
	elapsed := at.Sub(p.StartedAt) - time.Duration(p.PausedSeconds)*time.Second
	if p.PausedAt != nil {
		elapsed -= at.Sub(*p.PausedAt)
	}
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

// dueAt is the wall-clock moment a running session's phase runs out. A paused
// session has no due time — it never expires on its own — so ok is false.
func (p *PomodoroSession) dueAt() (due time.Time, ok bool) {
	if p.PausedAt != nil {
		return time.Time{}, false
	}
	return p.StartedAt.
		Add(time.Duration(p.PlannedSeconds) * time.Second).
		Add(time.Duration(p.PausedSeconds) * time.Second), true
}

// computeDerived fills Status and RemainingSeconds for the response.
func (p *PomodoroSession) computeDerived(now time.Time) {
	switch {
	case p.EndedAt != nil:
		p.Status = PomodoroStatusFinished
		p.RemainingSeconds = 0
		return
	case p.PausedAt != nil:
		p.Status = PomodoroStatusPaused
	default:
		p.Status = PomodoroStatusRunning
	}

	remaining := time.Duration(p.PlannedSeconds)*time.Second - p.elapsed(now)
	if remaining < 0 {
		remaining = 0
	}
	// Round up: a client showing 0:00 while the phase is technically still
	// running would look stuck.
	p.RemainingSeconds = int64((remaining + time.Second - 1) / time.Second)
}

// --- validation ---

func (p *PomodoroSession) validate() error {
	switch p.Phase {
	case PomodoroPhaseFocus, PomodoroPhaseShortBreak, PomodoroPhaseLongBreak:
	default:
		return InvalidFieldErrorWithMessage([]string{"phase"}, "Phase must be one of the following values: focus, short_break, long_break")
	}

	if p.PlannedSeconds < pomodoroMinSeconds || p.PlannedSeconds > pomodoroMaxSeconds {
		return InvalidFieldErrorWithMessage([]string{"planned_seconds"}, "planned_seconds must be between 60 and 14400.")
	}

	return nil
}

// --- lazy finalization ---

// getActivePomodoroSession returns the user's unfinished session, or nil.
//
// At most one session per user is active at a time. That invariant is kept in
// this model the way stopRunningTimerForUser does it, because a partial unique
// index is not portable across MySQL, Postgres and SQLite.
func getActivePomodoroSession(s *xorm.Session, userID int64) (*PomodoroSession, error) {
	session := &PomodoroSession{}
	has, err := s.Where("user_id = ? AND ended_at IS NULL", userID).
		OrderBy("started_at DESC").
		Get(session)
	if err != nil || !has {
		return nil, err
	}
	return session, nil
}

// finalizeOverduePomodoro closes the user's active session when its time ran
// out while nothing was asking. It runs first in every pomodoro operation, so
// there is no cron job and no background worker: a session whose phase expired
// while the browser was shut is completed the next time anything looks.
//
// The session is stamped at its *due* time, not at now, so a browser that was
// closed for a week does not turn a 25 minute focus phase into a week of focus.
func finalizeOverduePomodoro(s *xorm.Session, userID int64) error {
	active, err := getActivePomodoroSession(s, userID)
	if err != nil || active == nil {
		return err
	}

	due, ok := active.dueAt()
	if !ok || due.After(time.Now()) {
		return nil
	}

	active.EndedAt = &due
	if _, err := s.ID(active.ID).Cols("ended_at", "updated").Update(active); err != nil {
		return err
	}

	return active.logTimeEntry(s)
}

// logTimeEntry writes the finished focus phase into time tracking, when the
// user asked for it at start time, the phase is focus, it has a task and the
// feature is licensed. A break or a session without a task logs nothing.
//
// The insert is deliberately not TimeEntry.Create: that helper auto-stops the
// user's running manual timer and dispatches events, neither of which a
// pomodoro finishing in the background should do.
//
// The task's access is not re-checked here. readableTimeEntriesCond already
// hides entries whose task the user can no longer read, so a lost share cannot
// leak through the logged entry.
func (p *PomodoroSession) logTimeEntry(s *xorm.Session) error {
	if !p.LogTimeEntry || p.Phase != PomodoroPhaseFocus || p.TaskID == 0 || p.EndedAt == nil {
		return nil
	}
	if !license.IsFeatureEnabled(license.FeatureTimeTracking) {
		return nil
	}

	end := *p.EndedAt
	_, err := s.Insert(&TimeEntry{
		UserID:    p.UserID,
		TaskID:    p.TaskID,
		StartTime: end.Add(-p.elapsed(end)),
		EndTime:   &end,
	})
	return err
}

// --- operations ---

// pomodoroUserID resolves the acting user, refusing link shares. The user
// always comes from auth, never from the request body: a share's id is a share
// id, not a user id, so without this a share whose id collides with a user's
// would drive that user's timer.
func pomodoroUserID(a web.Auth) (int64, error) {
	if _, isShare := a.(*LinkSharing); isShare {
		return 0, ErrGenericForbidden{}
	}
	return a.GetID(), nil
}

// Create starts a new phase. Any session still active is marked interrupted
// first, so a user never has two running timers.
func (p *PomodoroSession) Create(s *xorm.Session, a web.Auth) (err error) {
	userID, err := pomodoroUserID(a)
	if err != nil {
		return err
	}

	if err = p.validate(); err != nil {
		return err
	}

	if err = finalizeOverduePomodoro(s, userID); err != nil {
		return err
	}

	active, err := getActivePomodoroSession(s, userID)
	if err != nil {
		return err
	}
	if active != nil {
		if err = active.stop(s, true); err != nil {
			return err
		}
	}

	// Reset everything the client does not get to choose.
	p.ID = 0
	p.UserID = userID
	p.StartedAt = time.Now()
	p.EndedAt = nil
	p.PausedAt = nil
	p.PausedSeconds = 0
	p.Interrupted = false

	if _, err = s.Insert(p); err != nil {
		return err
	}

	if err = p.loadTask(s, a); err != nil {
		return err
	}
	p.computeDerived(time.Now())
	return nil
}

// CanCreate allows any authenticated user to start a session on a task they can
// read. The session is personal and changes nothing about the task.
func (p *PomodoroSession) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	if _, isShare := a.(*LinkSharing); isShare {
		return false, ErrGenericForbidden{}
	}
	if p.TaskID == 0 {
		return true, nil
	}
	can, _, err := (&Task{ID: p.TaskID}).CanRead(s, a)
	return can, err
}

// ReadAll returns the caller's own sessions, newest first, optionally narrowed
// to a time range or a single task.
func (p *PomodoroSession) ReadAll(s *xorm.Session, a web.Auth, _ string, page int, perPage int) (result any, resultCount int, numberOfTotalItems int64, err error) {
	// DoReadAll skips the permission check, so the link-share refusal has to
	// live here too.
	if _, isShare := a.(*LinkSharing); isShare {
		return nil, 0, 0, ErrGenericForbidden{}
	}

	if err = finalizeOverduePomodoro(s, a.GetID()); err != nil {
		return nil, 0, 0, err
	}

	cond := builder.NewCond().And(builder.Eq{"user_id": a.GetID()})
	if p.TaskID > 0 {
		cond = cond.And(builder.Eq{"task_id": p.TaskID})
	}
	if p.From != nil {
		cond = cond.And(builder.Gte{"started_at": *p.From})
	}
	if p.To != nil {
		cond = cond.And(builder.Lte{"started_at": *p.To})
	}

	total, err := s.Where(cond).Count(&PomodoroSession{})
	if err != nil {
		return nil, 0, 0, err
	}

	sessions := []*PomodoroSession{}
	err = s.Where(cond).
		OrderBy("started_at DESC").
		Limit(getLimitFromPageIndex(page, perPage)).
		Find(&sessions)
	if err != nil {
		return nil, 0, 0, err
	}

	now := time.Now()
	for _, session := range sessions {
		session.computeDerived(now)
	}
	return sessions, len(sessions), total, nil
}

// GetCurrentPomodoroSession returns the caller's active session, or nil when
// nothing is running. Overdue sessions are finalized first, so a phase that
// expired while the client was away comes back as nil rather than as a running
// session with a negative remainder.
func GetCurrentPomodoroSession(s *xorm.Session, a web.Auth) (*PomodoroSession, error) {
	userID, err := pomodoroUserID(a)
	if err != nil {
		return nil, err
	}

	if err = finalizeOverduePomodoro(s, userID); err != nil {
		return nil, err
	}

	active, err := getActivePomodoroSession(s, userID)
	if err != nil || active == nil {
		return nil, err
	}

	if err = active.loadTask(s, a); err != nil {
		return nil, err
	}
	active.computeDerived(time.Now())
	return active, nil
}

// PausePomodoroSession pauses the caller's running session. Pausing an already
// paused session is a no-op, so there is no "already paused" error to handle.
func PausePomodoroSession(s *xorm.Session, a web.Auth) (*PomodoroSession, error) {
	return actOnCurrentPomodoro(s, a, func(active *PomodoroSession) error {
		if active.PausedAt != nil {
			return nil
		}
		now := time.Now()
		active.PausedAt = &now
		_, err := s.ID(active.ID).Cols("paused_at", "updated").Update(active)
		return err
	})
}

// ResumePomodoroSession resumes the caller's paused session, folding the pause
// that just ended into paused_seconds. Resuming a running session is a no-op.
func ResumePomodoroSession(s *xorm.Session, a web.Auth) (*PomodoroSession, error) {
	return actOnCurrentPomodoro(s, a, func(active *PomodoroSession) error {
		if active.PausedAt == nil {
			return nil
		}
		active.PausedSeconds += int64(time.Since(*active.PausedAt).Seconds())
		active.PausedAt = nil
		_, err := s.ID(active.ID).Cols("paused_at", "paused_seconds", "updated").Update(active)
		return err
	})
}

// StopPomodoroSession ends the caller's session early. It is stored as
// interrupted with its real duration: the focus time counts, the pomodoro does
// not.
func StopPomodoroSession(s *xorm.Session, a web.Auth) (*PomodoroSession, error) {
	return actOnCurrentPomodoro(s, a, func(active *PomodoroSession) error {
		return active.stop(s, true)
	})
}

// stop ends a session now. An open pause is closed first, so a finished row
// always has paused_at IS NULL and its elapsed time stays computable from
// paused_seconds alone.
func (p *PomodoroSession) stop(s *xorm.Session, interrupted bool) error {
	now := time.Now()
	if p.PausedAt != nil {
		p.PausedSeconds += int64(now.Sub(*p.PausedAt).Seconds())
		p.PausedAt = nil
	}
	p.EndedAt = &now
	p.Interrupted = interrupted

	if _, err := s.ID(p.ID).
		Cols("ended_at", "paused_at", "paused_seconds", "interrupted", "updated").
		Update(p); err != nil {
		return err
	}

	if interrupted {
		// An interrupted phase logs nothing: the user cut it short, and a
		// partial pomodoro is focus time in the stats, not a time entry.
		return nil
	}
	return p.logTimeEntry(s)
}

// actOnCurrentPomodoro is the shared shape of pause / resume / stop: finalize
// anything overdue, load the active session, apply the action, then return it
// with its task and derived fields.
func actOnCurrentPomodoro(s *xorm.Session, a web.Auth, act func(*PomodoroSession) error) (*PomodoroSession, error) {
	userID, err := pomodoroUserID(a)
	if err != nil {
		return nil, err
	}

	if err = finalizeOverduePomodoro(s, userID); err != nil {
		return nil, err
	}

	active, err := getActivePomodoroSession(s, userID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, ErrNoActivePomodoroSession{UserID: userID}
	}

	if err = act(active); err != nil {
		return nil, err
	}

	if err = active.loadTask(s, a); err != nil {
		return nil, err
	}
	active.computeDerived(time.Now())
	return active, nil
}

// loadTask attaches the session's task when it has one and the caller can still
// read it. A task the user lost access to simply stays absent; the session and
// its time survive.
func (p *PomodoroSession) loadTask(s *xorm.Session, a web.Auth) error {
	if p.TaskID == 0 {
		return nil
	}

	task, err := GetTaskByIDSimple(s, p.TaskID)
	if err != nil {
		if IsErrTaskDoesNotExist(err) {
			return nil
		}
		return err
	}

	can, _, err := task.CanRead(s, a)
	if err != nil {
		return err
	}
	if !can {
		// Losing access to the task does not take the timer away: the session
		// just comes back without it.
		return nil
	}

	p.Task = &task
	return nil
}

// ============================
// Per-user pomodoro estimates
// ============================

// TaskPomodoroEstimate is one user's personal estimate of how many focus phases
// a task will take. Like the Eisenhower classification it is private: other
// members of the project neither see nor change it.
type TaskPomodoroEstimate struct {
	ID     int64 `xorm:"bigint autoincr not null unique pk" json:"-"`
	TaskID int64 `xorm:"bigint not null unique(task_user)" json:"task_id" readOnly:"true" doc:"The id of the estimated task."`
	UserID int64 `xorm:"bigint not null unique(task_user) index" json:"-"`

	Estimate int `xorm:"int not null default 0" json:"estimate" minimum:"1" maximum:"99" doc:"How many focus phases the requesting user expects this task to take, 1 to 99."`

	// Estimated is false when the requesting user has not estimated the task
	// yet. Estimate is then 0.
	Estimated bool `xorm:"-" json:"estimated" readOnly:"true" doc:"Whether the requesting user has estimated this task. False means no estimate; estimate is then 0."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When the task was first estimated."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When the estimate last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*TaskPomodoroEstimate) TableName() string {
	return "task_pomodoro_estimates"
}

// canAccessTask allows any user who can read the task. The estimate is personal
// and doesn't change the task, so read access is enough. Link shares have no
// user to own an estimate.
func (e *TaskPomodoroEstimate) canAccessTask(s *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, ErrGenericForbidden{}
	}

	can, _, err := (&Task{ID: e.TaskID}).CanRead(s, a)
	return can, err
}

// loadOwn fills in the caller's stored estimate, if any, without touching the
// value bound from a request body.
func (e *TaskPomodoroEstimate) loadOwn(s *xorm.Session, a web.Auth) (existing *TaskPomodoroEstimate, err error) {
	e.UserID = a.GetID()
	existing = &TaskPomodoroEstimate{}
	has, err := s.Where("task_id = ? AND user_id = ?", e.TaskID, e.UserID).Get(existing)
	if err != nil || !has {
		return nil, err
	}
	return existing, nil
}

// CanRead checks whether the user can see their estimate of the task.
func (e *TaskPomodoroEstimate) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	can, err := e.canAccessTask(s, a)
	if err != nil || !can {
		return false, 0, err
	}

	existing, err := e.loadOwn(s, a)
	if err != nil {
		return false, 0, err
	}
	if existing != nil {
		*e = *existing
		e.Estimated = true
	}

	return true, int(PermissionRead), nil
}

// CanUpdate checks whether the user can estimate the task.
func (e *TaskPomodoroEstimate) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	can, err := e.canAccessTask(s, a)
	if err != nil || !can {
		return false, err
	}

	if e.Estimate < pomodoroMinEstimate || e.Estimate > pomodoroMaxEstimate {
		return false, InvalidFieldErrorWithMessage([]string{"estimate"}, "estimate must be between 1 and 99.")
	}

	existing, err := e.loadOwn(s, a)
	if err != nil {
		return false, err
	}
	if existing != nil {
		e.ID = existing.ID
		e.Created = existing.Created
	}

	return true, nil
}

// CanDelete checks whether the user can clear their estimate of the task.
func (e *TaskPomodoroEstimate) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return e.canAccessTask(s, a)
}

// ReadOne returns the caller's estimate. CanRead already loaded it; a task
// without one comes back with estimated=false instead of a 404.
func (e *TaskPomodoroEstimate) ReadOne(_ *xorm.Session, _ web.Auth) error {
	return nil
}

// Update stores the caller's estimate, creating the row when the task had none.
func (e *TaskPomodoroEstimate) Update(s *xorm.Session, a web.Auth) (err error) {
	e.UserID = a.GetID()

	if e.ID == 0 {
		_, err = s.Insert(e)
	} else {
		_, err = s.ID(e.ID).Cols("estimate", "updated").Update(e)
	}
	if err != nil {
		return err
	}

	e.Estimated = true
	return nil
}

// Delete clears the caller's estimate. Deleting a missing estimate is a no-op.
func (e *TaskPomodoroEstimate) Delete(s *xorm.Session, a web.Auth) error {
	_, err := s.Where("task_id = ? AND user_id = ?", e.TaskID, a.GetID()).
		Delete(&TaskPomodoroEstimate{})
	return err
}

// ==================
// Task expand
// ==================

// TaskPomodoroSummary is the caller's own pomodoro history for one task, as
// embedded by the `pomodoro` task expand.
type TaskPomodoroSummary struct {
	Completed    int64 `json:"completed" readOnly:"true" doc:"How many focus phases the requesting user completed on this task."`
	Interrupted  int64 `json:"interrupted" readOnly:"true" doc:"How many focus phases the requesting user stopped early on this task."`
	FocusSeconds int64 `json:"focus_seconds" readOnly:"true" doc:"Total focus seconds the requesting user spent on this task, interrupted phases included."`
	Estimate     int   `json:"estimate" readOnly:"true" doc:"The requesting user's estimate in focus phases, or 0 when they have not estimated it."`
}

// addPomodoroToTasks attaches each task's pomodoro summary for the `pomodoro`
// expand. Only the requesting user's own sessions count — the data is personal,
// so a link share gets nothing. Unlike the time-entry count this is not
// license-gated: pomodoro is a free feature.
func addPomodoroToTasks(s *xorm.Session, a web.Auth, taskIDs []int64, taskMap map[int64]*Task) error {
	if _, isShare := a.(*LinkSharing); isShare {
		return nil
	}
	if len(taskIDs) == 0 {
		return nil
	}

	for _, taskID := range taskIDs {
		if task, ok := taskMap[taskID]; ok {
			task.Pomodoro = &TaskPomodoroSummary{}
		}
	}

	sessions := []*PomodoroSession{}
	err := s.In("task_id", taskIDs).
		Where("user_id = ? AND phase = ? AND ended_at IS NOT NULL", a.GetID(), PomodoroPhaseFocus).
		Find(&sessions)
	if err != nil {
		return err
	}

	// Summed in Go rather than in SQL: the effective duration subtracts the
	// pauses, which no portable SQL expression covers across all three
	// databases.
	for _, session := range sessions {
		task, ok := taskMap[session.TaskID]
		if !ok {
			continue
		}
		task.Pomodoro.FocusSeconds += int64(session.elapsed(*session.EndedAt).Seconds())
		if session.Interrupted {
			task.Pomodoro.Interrupted++
		} else {
			task.Pomodoro.Completed++
		}
	}

	estimates := []*TaskPomodoroEstimate{}
	err = s.In("task_id", taskIDs).
		Where("user_id = ?", a.GetID()).
		Find(&estimates)
	if err != nil {
		return err
	}

	for _, estimate := range estimates {
		if task, ok := taskMap[estimate.TaskID]; ok {
			task.Pomodoro.Estimate = estimate.Estimate
		}
	}

	return nil
}

// ==================
// Statistics
// ==================

// PomodoroStatsDay is one calendar day in the user's timezone.
type PomodoroStatsDay struct {
	Date         string `json:"date" doc:"The day, as YYYY-MM-DD in the requested timezone."`
	FocusSeconds int64  `json:"focus_seconds" doc:"Focus seconds on that day, interrupted phases included."`
	Completed    int64  `json:"completed" doc:"Focus phases completed that day."`
	Interrupted  int64  `json:"interrupted" doc:"Focus phases stopped early that day."`
}

// PomodoroStatsGroup is the focus time of one task or one project.
type PomodoroStatsGroup struct {
	ID           int64  `json:"id" doc:"The id of the task or project, or 0 for the catch-all group."`
	Title        string `json:"title" doc:"The title of the task or project. \"Other\" collects sessions whose task was deleted or is no longer readable."`
	FocusSeconds int64  `json:"focus_seconds" doc:"Focus seconds in this group."`
	Completed    int64  `json:"completed" doc:"Focus phases completed in this group."`
	Interrupted  int64  `json:"interrupted" doc:"Focus phases stopped early in this group."`
	Estimate     int    `json:"estimate" doc:"The user's estimate in focus phases for this task, 0 for projects and unestimated tasks."`
}

// PomodoroStats is the response of the stats endpoint.
type PomodoroStats struct {
	From         time.Time             `json:"from" doc:"Start of the range these numbers cover."`
	To           time.Time             `json:"to" doc:"End of the range these numbers cover."`
	FocusSeconds int64                 `json:"focus_seconds" doc:"Total focus seconds in the range, interrupted phases included."`
	Completed    int64                 `json:"completed" doc:"Focus phases completed in the range."`
	Interrupted  int64                 `json:"interrupted" doc:"Focus phases stopped early in the range."`
	Days         []*PomodoroStatsDay   `json:"days" doc:"One entry per day in the range that has sessions, oldest first."`
	Tasks        []*PomodoroStatsGroup `json:"tasks" doc:"Tasks by focus time, most first."`
	Projects     []*PomodoroStatsGroup `json:"projects" doc:"Projects by focus time, most first."`
}

// GetPomodoroStats buckets the caller's finished focus phases in a range.
//
// The bucketing happens in Go using the caller's timezone rather than with SQL
// date functions: that behaves identically on MySQL, Postgres and SQLite, and
// it survives DST, where a local day is 23 or 25 hours long.
func GetPomodoroStats(s *xorm.Session, a web.Auth, from, to time.Time, timezone string) (*PomodoroStats, error) {
	userID, err := pomodoroUserID(a)
	if err != nil {
		return nil, err
	}

	if to.Before(from) {
		return nil, InvalidFieldErrorWithMessage([]string{"to"}, "to must not be before from.")
	}
	if to.Sub(from) > pomodoroMaxStatsDays*24*time.Hour {
		return nil, InvalidFieldErrorWithMessage([]string{"from", "to"}, "The range must not be longer than 366 days.")
	}

	loc := time.UTC
	if timezone != "" {
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return nil, ErrInvalidTimezone{Name: timezone, LoadError: err}
		}
	}

	if err = finalizeOverduePomodoro(s, userID); err != nil {
		return nil, err
	}

	sessions := []*PomodoroSession{}
	err = s.Where("user_id = ? AND phase = ? AND ended_at IS NOT NULL AND started_at >= ? AND started_at <= ?",
		userID, PomodoroPhaseFocus, from, to).
		OrderBy("started_at ASC").
		Find(&sessions)
	if err != nil {
		return nil, err
	}

	stats := &PomodoroStats{
		From:     from,
		To:       to,
		Days:     []*PomodoroStatsDay{},
		Tasks:    []*PomodoroStatsGroup{},
		Projects: []*PomodoroStatsGroup{},
	}

	days := map[string]*PomodoroStatsDay{}
	dayOrder := []string{}
	byTask := map[int64]*PomodoroStatsGroup{}

	for _, session := range sessions {
		focus := int64(session.elapsed(*session.EndedAt).Seconds())

		stats.FocusSeconds += focus
		if session.Interrupted {
			stats.Interrupted++
		} else {
			stats.Completed++
		}

		date := session.StartedAt.In(loc).Format(time.DateOnly)
		day, ok := days[date]
		if !ok {
			day = &PomodoroStatsDay{Date: date}
			days[date] = day
			dayOrder = append(dayOrder, date)
		}
		day.FocusSeconds += focus
		if session.Interrupted {
			day.Interrupted++
		} else {
			day.Completed++
		}

		group, ok := byTask[session.TaskID]
		if !ok {
			group = &PomodoroStatsGroup{ID: session.TaskID}
			byTask[session.TaskID] = group
		}
		group.FocusSeconds += focus
		if session.Interrupted {
			group.Interrupted++
		} else {
			group.Completed++
		}
	}

	// dayOrder follows started_at ASC, so the days come out oldest first
	// without a second sort.
	for _, date := range dayOrder {
		stats.Days = append(stats.Days, days[date])
	}

	stats.Tasks, stats.Projects, err = groupPomodoroStatsByTask(s, a, byTask)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// groupPomodoroStatsByTask turns the per-task-id buckets into titled task and
// project groups. A session whose task was deleted or whose project the user
// can no longer read is folded into a single "Other" group, so the totals stay
// honest without leaking a title.
func groupPomodoroStatsByTask(s *xorm.Session, a web.Auth, byTask map[int64]*PomodoroStatsGroup) (tasks, projects []*PomodoroStatsGroup, err error) {
	taskIDs := make([]int64, 0, len(byTask))
	for taskID := range byTask {
		if taskID != 0 {
			taskIDs = append(taskIDs, taskID)
		}
	}

	readable := map[int64]*Task{}
	if len(taskIDs) > 0 {
		accessible, err := accessibleProjectIDsCond(s, a, "project_id")
		if err != nil {
			return nil, nil, err
		}
		found := []*Task{}
		err = s.Where(builder.And(builder.In("id", taskIDs), accessible)).Find(&found)
		if err != nil {
			return nil, nil, err
		}
		for _, task := range found {
			readable[task.ID] = task
		}
	}

	estimates := []*TaskPomodoroEstimate{}
	if len(taskIDs) > 0 {
		err = s.In("task_id", taskIDs).Where("user_id = ?", a.GetID()).Find(&estimates)
		if err != nil {
			return nil, nil, err
		}
	}
	estimateByTask := map[int64]int{}
	for _, estimate := range estimates {
		estimateByTask[estimate.TaskID] = estimate.Estimate
	}

	projectTitles, err := pomodoroProjectTitles(s, readable)
	if err != nil {
		return nil, nil, err
	}

	other := &PomodoroStatsGroup{Title: "Other"}
	byProject := map[int64]*PomodoroStatsGroup{}
	tasks = []*PomodoroStatsGroup{}

	for taskID, group := range byTask {
		task, ok := readable[taskID]
		if !ok {
			other.FocusSeconds += group.FocusSeconds
			other.Completed += group.Completed
			other.Interrupted += group.Interrupted
			continue
		}

		group.Title = task.Title
		group.Estimate = estimateByTask[taskID]
		tasks = append(tasks, group)

		project, ok := byProject[task.ProjectID]
		if !ok {
			project = &PomodoroStatsGroup{ID: task.ProjectID, Title: projectTitles[task.ProjectID]}
			byProject[task.ProjectID] = project
		}
		project.FocusSeconds += group.FocusSeconds
		project.Completed += group.Completed
		project.Interrupted += group.Interrupted
	}

	projects = make([]*PomodoroStatsGroup, 0, len(byProject)+1)
	for _, project := range byProject {
		projects = append(projects, project)
	}

	if other.FocusSeconds > 0 || other.Completed > 0 || other.Interrupted > 0 {
		tasks = append(tasks, other)
		projects = append(projects, other)
	}

	sortPomodoroGroups(tasks)
	sortPomodoroGroups(projects)
	return tasks, projects, nil
}

// pomodoroProjectTitles loads the titles of the projects the given tasks live in.
func pomodoroProjectTitles(s *xorm.Session, tasks map[int64]*Task) (map[int64]string, error) {
	if len(tasks) == 0 {
		return map[int64]string{}, nil
	}

	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ProjectID)
	}

	found := []*Project{}
	if err := s.In("id", ids).Cols("id", "title").Find(&found); err != nil {
		return nil, err
	}

	titles := make(map[int64]string, len(found))
	for _, project := range found {
		titles[project.ID] = project.Title
	}
	return titles, nil
}

// sortPomodoroGroups orders groups by focus time, most first, with the id as
// the tie-breaker so the output is stable across requests (Go's map iteration
// is not).
func sortPomodoroGroups(groups []*PomodoroStatsGroup) {
	slices.SortFunc(groups, func(a, b *PomodoroStatsGroup) int {
		if c := cmp.Compare(b.FocusSeconds, a.FocusSeconds); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

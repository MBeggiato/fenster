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

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// PomodoroSession20260925150000 is one focus or break phase of a user's
// pomodoro cycle. The lifecycle lives in the timestamps, not in a status
// column: ended_at null + paused_at null is running, paused_at set is paused,
// ended_at set is finished.
type PomodoroSession20260925150000 struct {
	ID             int64      `xorm:"bigint autoincr not null unique pk"`
	UserID         int64      `xorm:"bigint not null index"`
	TaskID         int64      `xorm:"bigint not null default 0 index"`
	Phase          string     `xorm:"varchar(20) not null"`
	PlannedSeconds int64      `xorm:"bigint not null"`
	StartedAt      time.Time  `xorm:"not null index"`
	EndedAt        *time.Time `xorm:"null"`
	PausedAt       *time.Time `xorm:"null"`
	PausedSeconds  int64      `xorm:"bigint not null default 0"`
	Interrupted    bool       `xorm:"not null default false"`
	LogTimeEntry   bool       `xorm:"not null default false"`
	Created        time.Time  `xorm:"created not null"`
	Updated        time.Time  `xorm:"updated not null"`
}

func (PomodoroSession20260925150000) TableName() string {
	return "pomodoro_sessions"
}

// TaskPomodoroEstimate20260925150000 is one user's personal estimate of how
// many focus phases a task will take.
type TaskPomodoroEstimate20260925150000 struct {
	ID       int64     `xorm:"bigint autoincr not null unique pk"`
	TaskID   int64     `xorm:"bigint not null unique(task_user)"`
	UserID   int64     `xorm:"bigint not null unique(task_user) index"`
	Estimate int       `xorm:"int not null default 0"`
	Created  time.Time `xorm:"created not null"`
	Updated  time.Time `xorm:"updated not null"`
}

func (TaskPomodoroEstimate20260925150000) TableName() string {
	return "task_pomodoro_estimates"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260925150000",
		Description: "Add pomodoro sessions and per-user task pomodoro estimates",
		Migrate: func(tx *xorm.Engine) error {
			//nolint:forbidigo // brand-new tables, nothing to drop
			return tx.Sync(
				PomodoroSession20260925150000{},
				TaskPomodoroEstimate20260925150000{},
			)
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(
				PomodoroSession20260925150000{},
				TaskPomodoroEstimate20260925150000{},
			)
		},
	})
}

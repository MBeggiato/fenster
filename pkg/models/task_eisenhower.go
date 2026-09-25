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
	"time"

	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// TaskEisenhowerClassification is one user's personal placement of a task in
// the Eisenhower matrix. It is private to that user: other members of the
// project neither see nor change it. A task without a row is unclassified; a row
// with both flags false is the "neither urgent nor important" quadrant.
type TaskEisenhowerClassification struct {
	ID        int64 `xorm:"bigint autoincr not null unique pk" json:"-"`
	TaskID    int64 `xorm:"bigint not null unique(task_user)" json:"task_id" readOnly:"true" doc:"The id of the classified task."`
	UserID    int64 `xorm:"bigint not null unique(task_user) index" json:"-"`
	Urgent    bool  `xorm:"not null default false" json:"urgent" doc:"Whether the requesting user considers the task urgent."`
	Important bool  `xorm:"not null default false" json:"important" doc:"Whether the requesting user considers the task important."`

	// Classified is false when the requesting user has not placed the task in
	// the matrix yet. Urgent and important are then both false.
	Classified bool `xorm:"-" json:"classified" readOnly:"true" doc:"Whether the requesting user has classified this task. False means the task is unclassified; urgent and important are then false."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When the task was first classified."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When the classification last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func (*TaskEisenhowerClassification) TableName() string {
	return "task_eisenhower_classifications"
}

// EisenhowerQuadrant selects one area of the personal Eisenhower matrix.
type EisenhowerQuadrant string

const (
	EisenhowerQuadrantDo           EisenhowerQuadrant = "do"
	EisenhowerQuadrantSchedule     EisenhowerQuadrant = "schedule"
	EisenhowerQuadrantDelegate     EisenhowerQuadrant = "delegate"
	EisenhowerQuadrantEliminate    EisenhowerQuadrant = "eliminate"
	EisenhowerQuadrantUnclassified EisenhowerQuadrant = "unclassified"
)

// flags returns the urgent/important pair a quadrant stands for. ok is false
// for the unclassified area and unknown values.
func (q EisenhowerQuadrant) flags() (urgent, important, ok bool) {
	switch q {
	case EisenhowerQuadrantDo:
		return true, true, true
	case EisenhowerQuadrantSchedule:
		return false, true, true
	case EisenhowerQuadrantDelegate:
		return true, false, true
	case EisenhowerQuadrantEliminate:
		return false, false, true
	case EisenhowerQuadrantUnclassified:
		return false, false, false
	}
	return false, false, false
}

// eisenhowerQuadrantCond restricts a task query to one area of the caller's
// matrix. Other users' classifications never influence the result. The matrix
// is personal, so a link share has no matrix to read.
func eisenhowerQuadrantCond(q EisenhowerQuadrant, a web.Auth, taskAlias string) (builder.Cond, error) {
	if _, is := a.(*LinkSharing); is {
		return nil, ErrGenericForbidden{}
	}
	userID := a.GetID()

	own := builder.Select("1").
		From("task_eisenhower_classifications").
		Where(builder.And(
			builder.Expr("task_eisenhower_classifications.task_id = "+taskAlias+".id"),
			builder.Eq{"task_eisenhower_classifications.user_id": userID},
		))

	if q == EisenhowerQuadrantUnclassified {
		return builder.NotExists(own), nil
	}

	urgent, important, ok := q.flags()
	if !ok {
		return nil, ErrInvalidEisenhowerQuadrant{Quadrant: string(q)}
	}

	return builder.Exists(own.And(builder.Eq{
		"task_eisenhower_classifications.urgent":    urgent,
		"task_eisenhower_classifications.important": important,
	})), nil
}

// canAccessTask allows any user who can read the task. The classification is
// personal and doesn't change the task, so read access is enough. Link shares
// have no user to own a classification.
func (c *TaskEisenhowerClassification) canAccessTask(s *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, ErrGenericForbidden{}
	}

	can, _, err := (&Task{ID: c.TaskID}).CanRead(s, a)
	return can, err
}

// loadOwn fills in the caller's stored classification, if any, without
// touching the urgent/important values bound from a request body.
func (c *TaskEisenhowerClassification) loadOwn(s *xorm.Session, a web.Auth) (existing *TaskEisenhowerClassification, err error) {
	c.UserID = a.GetID()
	existing = &TaskEisenhowerClassification{}
	has, err := s.Where("task_id = ? AND user_id = ?", c.TaskID, c.UserID).Get(existing)
	if err != nil || !has {
		return nil, err
	}
	return existing, nil
}

// CanRead checks whether the user can see their classification of the task.
func (c *TaskEisenhowerClassification) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	can, err := c.canAccessTask(s, a)
	if err != nil || !can {
		return false, 0, err
	}

	existing, err := c.loadOwn(s, a)
	if err != nil {
		return false, 0, err
	}
	if existing != nil {
		*c = *existing
		c.Classified = true
	}

	return true, int(PermissionRead), nil
}

// CanUpdate checks whether the user can classify the task.
func (c *TaskEisenhowerClassification) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	can, err := c.canAccessTask(s, a)
	if err != nil || !can {
		return false, err
	}

	existing, err := c.loadOwn(s, a)
	if err != nil {
		return false, err
	}
	if existing != nil {
		c.ID = existing.ID
		c.Created = existing.Created
	}

	return true, nil
}

// CanDelete checks whether the user can reset their classification of the task.
func (c *TaskEisenhowerClassification) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return c.canAccessTask(s, a)
}

// ReadOne returns the caller's classification. CanRead already loaded it; an
// unclassified task comes back with classified=false.
func (c *TaskEisenhowerClassification) ReadOne(_ *xorm.Session, _ web.Auth) error {
	return nil
}

// Update stores both flags of the caller's classification at once, creating the
// row when the task was unclassified.
func (c *TaskEisenhowerClassification) Update(s *xorm.Session, a web.Auth) (err error) {
	c.UserID = a.GetID()

	if c.ID == 0 {
		_, err = s.Insert(c)
	} else {
		_, err = s.ID(c.ID).Cols("urgent", "important", "updated").Update(c)
	}
	if err != nil {
		return err
	}

	c.Classified = true
	return nil
}

// Delete resets the caller's classification, making the task unclassified
// again. Deleting an unclassified task is a no-op.
func (c *TaskEisenhowerClassification) Delete(s *xorm.Session, a web.Auth) error {
	_, err := s.Where("task_id = ? AND user_id = ?", c.TaskID, a.GetID()).
		Delete(&TaskEisenhowerClassification{})
	return err
}

// addEisenhowerToTasks attaches the caller's classification to each task that
// has one. Unclassified tasks keep a nil Eisenhower field.
func addEisenhowerToTasks(s *xorm.Session, taskIDs []int64, taskMap map[int64]*Task, a web.Auth) error {
	if len(taskIDs) == 0 {
		return nil
	}
	if _, is := a.(*LinkSharing); is {
		return nil
	}

	classifications := []*TaskEisenhowerClassification{}
	err := s.In("task_id", taskIDs).
		Where("user_id = ?", a.GetID()).
		Find(&classifications)
	if err != nil {
		return err
	}

	for _, c := range classifications {
		if task, exists := taskMap[c.TaskID]; exists {
			c.Classified = true
			task.Eisenhower = c
		}
	}

	return nil
}

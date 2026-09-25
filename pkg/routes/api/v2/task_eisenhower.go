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
	"net/http"
	"slices"

	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
)

type eisenhowerTaskPath struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
}

// eisenhowerListInput keeps quadrant and include_done as direct fields next to
// the shared task-list params; the embed must stay exported (see
// TaskListQueryParams).
type eisenhowerListInput struct {
	Quadrant    string `query:"quadrant" required:"true" enum:"do,schedule,delegate,eliminate,unclassified" doc:"The matrix area to list: do (urgent and important), schedule (important, not urgent), delegate (urgent, not important), eliminate (neither), or unclassified (tasks the user has not placed yet)."`
	IncludeDone bool   `query:"include_done" doc:"If true, also return done tasks. By default only open tasks are listed."`
	TaskListQueryParams
}

// RegisterTaskEisenhowerRoutes wires the personal Eisenhower matrix onto the
// Huma API: reading and changing one task's classification, and listing the
// tasks of one matrix area.
//
// The classification is private to the requesting user. Reading the task is
// enough to classify it, since nothing about the task itself changes. Link
// shares have no user and are refused.
func RegisterTaskEisenhowerRoutes(api huma.API) {
	tags := []string{"tasks"}

	Register(api, huma.Operation{
		OperationID: "task-eisenhower-read",
		Summary:     "Get your Eisenhower classification of a task",
		Description: "Returns the authenticated user's personal classification of a task. An unclassified task returns classified=false instead of 404. Requires read access to the task; other users' classifications are never visible.",
		Method:      http.MethodGet,
		Path:        "/tasks/{task}/eisenhower",
		Tags:        tags,
	}, taskEisenhowerRead)

	Register(api, huma.Operation{
		OperationID: "task-eisenhower-update",
		Summary:     "Classify a task in your Eisenhower matrix",
		Description: "Sets both urgent and important of the authenticated user's classification at once, creating it if the task was unclassified. Idempotent. Requires read access to the task; the task itself and other users' classifications are unchanged.",
		Method:      http.MethodPut,
		Path:        "/tasks/{task}/eisenhower",
		Tags:        tags,
	}, taskEisenhowerUpdate)

	Register(api, huma.Operation{
		OperationID: "task-eisenhower-delete",
		Summary:     "Reset your Eisenhower classification of a task",
		Description: "Removes the authenticated user's classification so the task is unclassified again. Idempotent: resetting an unclassified task succeeds.",
		Method:      http.MethodDelete,
		Path:        "/tasks/{task}/eisenhower",
		Tags:        tags,
	}, taskEisenhowerDelete)

	Register(api, huma.Operation{
		OperationID: "eisenhower-tasks-list",
		Summary:     "List the tasks in one area of your Eisenhower matrix",
		Description: "Returns the tasks across every project the authenticated user can read that fall into the requested area of their personal matrix, paginated and flat. Done tasks are left out unless include_done is set; their classification is kept. Each task carries its classification when expand=eisenhower is passed. The subtasks expansion is not supported here. " + taskListFilterDoc,
		Method:      http.MethodGet,
		Path:        "/eisenhower/tasks",
		Tags:        tags,
	}, eisenhowerTasksList)
}

func init() { AddRouteRegistrar(RegisterTaskEisenhowerRoutes) }

func taskEisenhowerRead(ctx context.Context, in *eisenhowerTaskPath) (*singleBody[models.TaskEisenhowerClassification], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	c := &models.TaskEisenhowerClassification{TaskID: in.TaskID}
	if _, err := handler.DoReadOne(ctx, c, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.TaskEisenhowerClassification]{Body: c}, nil
}

func taskEisenhowerUpdate(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Body   models.TaskEisenhowerClassification
}) (*singleBody[models.TaskEisenhowerClassification], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	c := &in.Body
	c.TaskID = in.TaskID // URL wins over body
	if err := handler.DoUpdate(ctx, c, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.TaskEisenhowerClassification]{Body: c}, nil
}

func taskEisenhowerDelete(ctx context.Context, in *eisenhowerTaskPath) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.TaskEisenhowerClassification{TaskID: in.TaskID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

func eisenhowerTasksList(ctx context.Context, in *eisenhowerListInput) (*taskListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if slices.Contains(in.Expand, string(models.TaskCollectionExpandSubtasks)) {
		return nil, huma.Error400BadRequest("the subtasks expansion is not supported for the eisenhower matrix")
	}

	f := taskListFilters{in.Q, in.Filter, in.FilterTimezone, in.FilterIncludeNulls, in.SortBy, in.OrderBy, in.Expand}
	if !in.IncludeDone {
		if f.Filter == "" {
			f.Filter = "done = false"
		} else {
			f.Filter = "done = false && (" + f.Filter + ")"
		}
	}

	tc, err := f.collection(0, 0, true)
	if err != nil {
		return nil, err
	}
	tc.SetEisenhowerQuadrant(models.EisenhowerQuadrant(in.Quadrant))

	return readTaskCollection(ctx, a, tc, f.Q, in.Page, in.PerPage)
}

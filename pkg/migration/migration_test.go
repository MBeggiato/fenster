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
	"testing"

	"github.com/stretchr/testify/assert"
	"src.techknowlogick.com/xormigrate"
)

func TestUnknownMigrationIDs(t *testing.T) {
	known := []*xormigrate.Migration{{ID: "20260101000000"}, {ID: "20260201000000"}}
	applied := []*xormigrate.Migration{{ID: "SCHEMA_INIT"}, {ID: "20260101000000"}, {ID: "20260928140648"}}

	assert.Equal(t, []string{"20260928140648"}, unknownMigrationIDs(applied, known))
	assert.Empty(t, unknownMigrationIDs(known, known))
}

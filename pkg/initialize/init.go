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

package initialize

import (
	"context"
	"time"

	"github.com/MBeggiato/fenster/pkg/audit"
	"github.com/MBeggiato/fenster/pkg/config"
	"github.com/MBeggiato/fenster/pkg/cron"
	"github.com/MBeggiato/fenster/pkg/db"
	"github.com/MBeggiato/fenster/pkg/events"
	"github.com/MBeggiato/fenster/pkg/files"
	"github.com/MBeggiato/fenster/pkg/i18n"
	"github.com/MBeggiato/fenster/pkg/log"
	"github.com/MBeggiato/fenster/pkg/mail"
	"github.com/MBeggiato/fenster/pkg/migration"
	"github.com/MBeggiato/fenster/pkg/models"
	"github.com/MBeggiato/fenster/pkg/modules/auth/ldap"
	"github.com/MBeggiato/fenster/pkg/modules/auth/openid"
	"github.com/MBeggiato/fenster/pkg/modules/keyvalue"
	migrationmodule "github.com/MBeggiato/fenster/pkg/modules/migration"
	migrationHandler "github.com/MBeggiato/fenster/pkg/modules/migration/handler"
	"github.com/MBeggiato/fenster/pkg/plugins"
	_ "github.com/MBeggiato/fenster/pkg/plugins/yaegi" // register yaegi plugin loader
	"github.com/MBeggiato/fenster/pkg/red"
	"github.com/MBeggiato/fenster/pkg/user"
	ws "github.com/MBeggiato/fenster/pkg/websocket"
)

// LightInit will only init config, redis, logger but no db connection.
func LightInit() {
	// Set logger
	log.InitLogger()

	// Init the config
	config.InitConfig()

	// Check if the configured time zone is valid
	if _, err := time.LoadLocation(config.ServiceTimeZone.GetString()); err != nil {
		log.Criticalf("Error parsing default time zone: %s", err)
	}

	// Init redis
	red.InitRedis()

	// Init keyvalue store
	keyvalue.InitStorage()
}

// InitEngines intializes all db connections
func InitEngines() {
	err := models.SetEngine()
	if err != nil {
		log.Fatal(err.Error())
	}
	err = files.SetEngine()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = db.CreateParadeDBIndexes()
	if err != nil {
		log.Fatal(err.Error())
	}
}

// FullInitWithoutAsync does a full init without any async handlers (cron or events)
func FullInitWithoutAsync() {
	LightInit()

	// Initialize the files handler
	err := files.InitFileHandler(context.Background())
	if err != nil {
		log.Fatalf("Could not init file handler: %s", err)
	}

	// Run the migrations
	migration.Migrate(nil)

	// Set Engine
	InitEngines()

	if config.AuditEnabled.GetBool() {
		if err := audit.Init(); err != nil {
			log.Fatalf("Could not initialize audit logging: %s", err)
		}
	}

	// Start the mail daemon
	mail.StartMailDaemon()

	// Connect to ldap if enabled
	ldap.InitializeLDAPConnection()

	// Check all OpenID Connect providers at startup
	_, err = openid.GetAllProviders()
	if err != nil {
		if openid.IsErrDuplicateOIDCIssuer(err) {
			log.Fatalf("OpenID Connect configuration error: %s", err)
		}
		log.Errorf("Error initializing OpenID Connect providers: %s", err)
	}

	// Load translations
	i18n.Init()

	// Initialize plugins
	plugins.Initialize()
}

// FullInit initializes all kinds of things in the right order
func FullInit() {

	FullInitWithoutAsync()

	// Start the cron
	cron.Init()
	models.RegisterReminderCron()
	models.RegisterOverdueReminderCron()
	models.RegisterUserDeletionCron()
	models.RegisterTaskCleanupCron()
	models.RegisterOldExportCleanupCron()
	migrationmodule.RegisterImportUploadCleanupCron()
	models.RegisterAddTaskToFilterViewCron()
	user.RegisterTokenCleanupCron()
	models.RegisterSessionCleanupCron()
	user.RegisterDeletionNotificationCron()
	openid.CleanupSavedOpenIDProviders()
	openid.RegisterEmptyOpenIDTeamCleanupCron()
	openid.RegisterProviderAvailabilityCron()
	models.RegisterAPITokenExpiryCheckCron()

	// Initialize WebSocket hub
	ws.InitHub()

	// Start processing events
	go func() {
		models.RegisterListeners()
		migrationHandler.RegisterListeners()
		ws.RegisterListeners()
		err := events.InitEvents()
		if err != nil {
			log.Fatal(err.Error())
		}

		err = events.Dispatch(&BootedEvent{
			BootedAt: time.Now(),
		})
		if err != nil {
			log.Fatal(err)
		}
	}()
}

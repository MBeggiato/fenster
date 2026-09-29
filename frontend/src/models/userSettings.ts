import AbstractModel from './abstractModel'

import type {IFrontendSettings, IUserSettings} from '@/modelTypes/IUserSettings'
import {getBrowserLanguage} from '@/i18n'
import {PrefixMode} from '@/modules/quickAddMagic'
import {DEFAULT_PROJECT_VIEW_SETTINGS} from '@/constants/projectView'
import {PRIORITIES} from '@/constants/priorities'
import {DATE_DISPLAY} from '@/constants/dateDisplay'
import {TIME_FORMAT} from '@/constants/timeFormat'
import {RELATION_KIND} from '@/types/IRelationKind'

// Also merged in by the auth store: users who never saved these have no keys in the api.
export const POMODORO_DEFAULTS = {
	pomodoroFocusMinutes: 25,
	pomodoroShortBreakMinutes: 5,
	pomodoroLongBreakMinutes: 15,
	pomodoroLongBreakEvery: 4,
	pomodoroAutoStartBreaks: false,
	pomodoroAutoStartFocus: false,
	pomodoroSound: true,
	pomodoroNotifications: false,
	pomodoroLogTimeEntries: false,
}

export default class UserSettingsModel extends AbstractModel<IUserSettings> implements IUserSettings {
	name = ''
	emailRemindersEnabled = true
	discoverableByName = false
	discoverableByEmail = false
	overdueTasksRemindersEnabled = true
	overdueTasksRemindersTime = undefined
	defaultProjectId = undefined
	weekStart = 0 as IUserSettings['weekStart']
	timezone = ''
	language = getBrowserLanguage() 
	frontendSettings: IFrontendSettings = {
		playSoundWhenDone: true,
		quickAddMagicMode: PrefixMode.Default,
		colorSchema: 'auto',
		allowIconChanges: true,
		filterIdUsedOnOverview: null,
		defaultView: DEFAULT_PROJECT_VIEW_SETTINGS.FIRST,
		minimumPriority: PRIORITIES.MEDIUM,
		dateDisplay: DATE_DISPLAY.RELATIVE,
		timeFormat: TIME_FORMAT.HOURS_24,
		defaultTaskRelationType: RELATION_KIND.RELATED,
		backgroundBrightness: null,
		alwaysShowBucketTaskCount: false,
		showLastViewed: true,
		sidebarWidth: null,
		commentSortOrder: 'asc',
		desktopQuickEntryShortcut: 'CmdOrCtrl+Shift+A',
		quickAddDefaultReminders: [],
		defaultDueTime: undefined,
		...POMODORO_DEFAULTS,
	}
	extraSettingsLinks = {}

	constructor(data: Partial<IUserSettings> = {}) {
		super()
		this.assignData(data)

		// The api returns an empty string when no language was ever set, and assignData
		// only falls back to defaults for null/undefined.
		if (!this.language) {
			this.language = getBrowserLanguage()
		}
	}
}

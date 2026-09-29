import {computed, ref, watch} from 'vue'
import {createGlobalState, useIntervalFn} from '@vueuse/core'
import {useQuery, useQueryClient} from '@tanstack/vue-query'

import {
	currentPomodoroQuery,
	pomodoroKeys,
	pomodoroSessionsQuery,
	usePausePomodoroMutation,
	useResumePomodoroMutation,
	useStartPomodoroMutation,
	useStopPomodoroMutation,
	type PomodoroPhase,
	type PomodoroSessionResponse,
} from '@/client/queries/pomodoro'
import {useAuthStore} from '@/stores/auth'
import popSoundFile from '@/assets/audio/pop.mp3'
import {i18n} from '@/i18n'

export const POMODORO_PHASES = ['focus', 'short_break', 'long_break'] as const satisfies readonly PomodoroPhase[]

// The countdown ticks every second, but the value is recomputed from an absolute
// deadline each time rather than decremented: a background tab throttles timers,
// so a decrementing counter would drift behind the wall clock.
const TICK_INTERVAL = 1000

function startOfToday(): Date {
	const date = new Date()
	date.setHours(0, 0, 0, 0)
	return date
}

// The pure arithmetic lives out here rather than in the singleton's closure, so
// it can be read — and tested — without standing up a query client.

// The effective focus time of a finished session: wall time minus every pause.
// Mirrors the server's own arithmetic so the strip and the server agree.
export function elapsedSecondsOf(session: PomodoroSessionResponse): number {
	if (!session.ended_at) return 0
	const started = new Date(session.started_at).getTime()
	const ended = new Date(session.ended_at).getTime()
	if (Number.isNaN(started) || Number.isNaN(ended)) return 0
	return Math.max(0, Math.round((ended - started) / 1000) - session.paused_seconds)
}

export function isFinishedFocus(session: PomodoroSessionResponse): boolean {
	return session.phase === 'focus' && session.status === 'finished'
}

// Total focus time of a day, interrupted phases included: they were still spent
// focusing, they just did not earn a tomato.
export function focusSecondsOf(sessions: readonly PomodoroSessionResponse[]): number {
	return sessions.filter(isFinishedFocus).reduce((total, session) => total + elapsedSecondsOf(session), 0)
}

// The phase that follows the one that just ended: a focus phase earns a break,
// long after every `longBreakEvery` completed phases; a break is followed by focus.
export function nextPhaseFor(
	phase: PomodoroPhase | null,
	completedToday: number,
	longBreakEvery: number,
): PomodoroPhase {
	if (phase !== 'focus') return 'focus'
	const every = Math.max(1, longBreakEvery)
	return completedToday > 0 && completedToday % every === 0 ? 'long_break' : 'short_break'
}

// How many more completed focus phases until the long break. The count restarts
// after every long break the user actually took, so a break taken early does not
// leave the cycle permanently off by one. Sessions come oldest first.
export function focusSessionsUntilLongBreak(
	sessions: readonly PomodoroSessionResponse[],
	longBreakEvery: number,
): number {
	const every = Math.max(1, longBreakEvery)
	const sinceLastLongBreak = sessions.reduce((count, session) => {
		if (session.phase === 'long_break' && session.status === 'finished') return 0
		if (isFinishedFocus(session) && !session.interrupted) return count + 1
		return count
	}, 0)
	return Math.max(0, every - (sinceLastLongBreak % every))
}

// The remainder of a phase at a given moment. A paused phase has no deadline: it
// holds whatever the server last reported.
export function remainingSecondsAt(deadline: number | null, now: number, pausedRemainder: number): number {
	if (deadline === null) return pausedRemainder
	return Math.max(0, Math.ceil((deadline - now) / 1000))
}

// Records that a session has been rung for, returning false when it already was.
// A refetch that still reports the expired session must not ring a second time.
export function markRung(rung: Set<number>, id: number): boolean {
	if (rung.has(id)) return false
	rung.add(id)
	return true
}

// Clamped to the bounds the API enforces, so a stale or hand-edited setting
// cannot make every start fail with a validation error.
export function plannedSecondsForMinutes(minutes: number): number {
	return Math.min(4 * 60 * 60, Math.max(60, Math.round(minutes * 60)))
}

export function formatCountdown(totalSeconds: number): string {
	const seconds = Math.max(0, Math.round(totalSeconds))
	const pad = (value: number) => value.toString().padStart(2, '0')
	const hours = Math.floor(seconds / 3600)
	const mmss = `${pad(Math.floor((seconds % 3600) / 60))}:${pad(seconds % 60)}`
	return hours >= 1 ? `${hours}:${mmss}` : mmss
}

// Our own prefix is stripped before it is reapplied, so the countdown composes
// with whatever else on the page owns the title.
const TITLE_PREFIX = /^\d+:\d{2}(:\d{2})? · [^·]+ · /

function setCountdownTitle(prefix: string | null) {
	const base = document.title.replace(TITLE_PREFIX, '')
	document.title = prefix === null ? base : `${prefix}${base}`
}

// One shared timer for the whole app: the header badge and the focus page read
// the same countdown and the ring fires once, not once per mounted component.
export const usePomodoro = createGlobalState(() => {
	const auth = useAuthStore()
	const client = useQueryClient()

	const enabled = computed(() => !!auth.info?.id && !auth.isLinkShareAuth)
	const query = useQuery(computed(() => ({
		...currentPomodoroQuery(),
		enabled: enabled.value,
	})))

	const session = computed<PomodoroSessionResponse | null>(() => query.data.value ?? null)
	const isRunning = computed(() => session.value?.status === 'running')
	const isPaused = computed(() => session.value?.status === 'paused')
	const isActive = computed(() => session.value !== null)
	const phase = computed<PomodoroPhase | null>(() => session.value?.phase ?? null)

	const settings = computed(() => auth.settings.frontendSettings)

	// The deadline is captured whenever the server tells us how much is left, so
	// the countdown survives a re-render and a throttled tab.
	const deadline = ref<number | null>(null)
	const now = ref(Date.now())

	watch(session, current => {
		if (current === null || current.status !== 'running') {
			deadline.value = null
			return
		}
		deadline.value = Date.now() + current.remaining_seconds * 1000
		now.value = Date.now()
	}, {immediate: true})

	const remaining = computed(() => session.value === null
		? 0
		: remainingSecondsAt(deadline.value, now.value, session.value.remaining_seconds),
	)

	const formattedRemaining = computed(() => formatCountdown(remaining.value))

	const progress = computed(() => {
		const planned = session.value?.planned_seconds ?? 0
		if (planned <= 0) return 0
		return Math.min(1, Math.max(0, 1 - remaining.value / planned))
	})

	// Today's finished focus phases, for the tomato strip and for deciding when
	// the next break is a long one.
	const todayQuery = useQuery(computed(() => ({
		...pomodoroSessionsQuery(startOfToday().toISOString(), '', 0, 1, 100),
		enabled: enabled.value,
	})))

	// The list comes back newest first; the strip and the cycle read oldest first.
	const todaySessions = computed(() => (todayQuery.data.value?.items ?? []).slice().reverse())

	const todayFocusSessions = computed(() => todaySessions.value.filter(isFinishedFocus))
	const completedToday = computed(() => todayFocusSessions.value.filter(item => !item.interrupted).length)
	const todayFocusSeconds = computed(() => focusSecondsOf(todaySessions.value))
	const untilLongBreak = computed(() => focusSessionsUntilLongBreak(
		todaySessions.value,
		settings.value.pomodoroLongBreakEvery,
	))

	function plannedSecondsFor(target: PomodoroPhase): number {
		const minutes = target === 'focus'
			? settings.value.pomodoroFocusMinutes
			: target === 'short_break'
				? settings.value.pomodoroShortBreakMinutes
				: settings.value.pomodoroLongBreakMinutes
		return plannedSecondsForMinutes(minutes)
	}

	const nextPhase = computed(() => nextPhaseFor(
		phase.value,
		completedToday.value,
		settings.value.pomodoroLongBreakEvery,
	))

	const startMutation = useStartPomodoroMutation()
	const pauseMutation = usePausePomodoroMutation()
	const resumeMutation = useResumePomodoroMutation()
	const stopMutation = useStopPomodoroMutation()

	async function start(target: PomodoroPhase, taskId = 0) {
		await startMutation.mutateAsync({
			phase: target,
			plannedSeconds: plannedSecondsFor(target),
			taskId,
			// Only a focus phase on a task can become a time entry; sending the
			// flag for anything else would just be noise the server ignores.
			logTimeEntry: target === 'focus' && taskId > 0 && settings.value.pomodoroLogTimeEntries,
		})
	}

	async function pause() {
		await pauseMutation.mutateAsync(undefined)
	}

	async function resume() {
		await resumeMutation.mutateAsync(undefined)
	}

	async function stop() {
		await stopMutation.mutateAsync(undefined)
	}

	async function toggle() {
		if (isPaused.value) return resume()
		if (isRunning.value) return pause()
		return start('focus')
	}

	// The phase that was ringing, so the offer to start the next one survives the
	// refetch that clears the session.
	const finishedPhase = ref<PomodoroPhase | null>(null)
	const finishedTaskId = ref(0)

	function dismissFinished() {
		finishedPhase.value = null
		finishedTaskId.value = 0
	}

	function ringSound() {
		if (!settings.value.pomodoroSound) return
		try {
			void new Audio(popSoundFile).play()
		} catch (e) {
			console.error('Could not play the pomodoro sound:', e)
		}
	}

	function ringNotification(ended: PomodoroSessionResponse, upcoming: PomodoroPhase) {
		if (!settings.value.pomodoroNotifications) return
		if (typeof Notification === 'undefined' || Notification.permission !== 'granted') return

		const minutes = Math.round(plannedSecondsFor(upcoming) / 60)
		try {
			new Notification(i18n.global.t(`pomodoro.done.${ended.phase}`), {
				body: i18n.global.t(`pomodoro.done.next.${upcoming}`, {minutes}),
				// One notification per session, so a second tab replaces it
				// instead of stacking a duplicate.
				tag: `pomodoro-${ended.id}`,
				data: {taskId: ended.task_id},
				...(ended.task_id > 0 ? {actions: [{action: 'show-task', title: i18n.global.t('task.detail.title')}]} : {}),
			} as NotificationOptions)
		} catch (e) {
			console.error('Could not show the pomodoro notification:', e)
		}
	}

	// Several open tabs share one countdown but not one JS context, so the lock
	// keeps them from ringing in chorus. Without the Web Locks API every tab
	// rings — annoying, but never silent.
	async function ringOnce(ended: PomodoroSessionResponse, upcoming: PomodoroPhase) {
		const ring = () => {
			ringSound()
			ringNotification(ended, upcoming)
		}
		if (!navigator.locks) {
			ring()
			return
		}
		await navigator.locks.request(`pomodoro-ring-${ended.id}`, {ifAvailable: true}, async lock => {
			if (lock === null) return
			ring()
			// Held briefly so a tab that wakes from throttling a moment later
			// finds the lock taken instead of ringing again.
			await new Promise(resolve => setTimeout(resolve, 2000))
		})
	}

	// Sessions already rung for, so a refetch that still reports the expired
	// session cannot ring twice.
	const rung = new Set<number>()

	async function onReachedZero() {
		const expired = session.value
		if (expired === null || expired.status !== 'running') return
		if (!markRung(rung, expired.id)) return

		const upcoming = nextPhase.value
		finishedPhase.value = expired.phase
		finishedTaskId.value = expired.task_id

		await ringOnce(expired, upcoming)
		// The server closes the expired session lazily, so ask it now: that is
		// what turns the countdown into "finished" for every open tab.
		await client.invalidateQueries({queryKey: pomodoroKeys.all})

		const autoStart = expired.phase === 'focus'
			? settings.value.pomodoroAutoStartBreaks
			: settings.value.pomodoroAutoStartFocus
		if (autoStart) {
			await start(upcoming, upcoming === 'focus' ? expired.task_id : 0)
			dismissFinished()
		}
	}

	useIntervalFn(() => {
		now.value = Date.now()
		if (isRunning.value && remaining.value <= 0) void onReachedZero()
		setCountdownTitle(isRunning.value || isPaused.value
			? `${formattedRemaining.value} · ${i18n.global.t(`pomodoro.phase.${phase.value}`)} · `
			: null,
		)
	}, TICK_INTERVAL)

	async function requestNotificationPermission(): Promise<boolean> {
		if (typeof Notification === 'undefined') return false
		if (Notification.permission === 'granted') return true
		if (Notification.permission === 'denied') return false
		return (await Notification.requestPermission()) === 'granted'
	}

	return {
		session,
		phase,
		isActive,
		isRunning,
		isPaused,
		remaining,
		formattedRemaining,
		progress,
		nextPhase,
		untilLongBreak,
		completedToday,
		todayFocusSessions,
		todayFocusSeconds,
		todayLoading: todayQuery.isPending,
		finishedPhase,
		finishedTaskId,
		dismissFinished,
		plannedSecondsFor,
		start,
		pause,
		resume,
		stop,
		toggle,
		requestNotificationPermission,
		isPending: computed(() => startMutation.isPending.value
			|| pauseMutation.isPending.value
			|| resumeMutation.isPending.value
			|| stopMutation.isPending.value),
	}
})

import {queryOptions, useMutation, type QueryClient} from '@tanstack/vue-query'
import {
	pomodoroCurrent,
	pomodoroPause,
	pomodoroResume,
	pomodoroSessionsList,
	pomodoroStart,
	pomodoroStats,
	pomodoroStop,
	taskPomodoroEstimateDelete,
	taskPomodoroEstimateUpdate,
} from '@/client/generated'
import type {
	PomodoroSession,
	PomodoroStats,
	TaskPomodoroSummary,
} from '@/client/generated'
import {contextMutationOptions} from './contextMutation'
import {pageSizeFor, type Paginated} from './pagination'
import {mapTaskEverywhere} from './taskCache'

export type PomodoroPhase = 'focus' | 'short_break' | 'long_break'
export type PomodoroStatus = 'running' | 'paused' | 'finished'

// Guaranteed fields are the backend fields without omitempty; task stays
// optional because it is only present when the session has a readable task.
export type PomodoroSessionResponse = Omit<PomodoroSession,
	'id' | 'task_id' | 'phase' | 'planned_seconds' | 'started_at' | 'status' | 'remaining_seconds' | 'paused_seconds' | 'interrupted' | 'log_time_entry'
> & Required<Pick<PomodoroSession,
	'id' | 'task_id' | 'phase' | 'planned_seconds' | 'started_at' | 'status' | 'remaining_seconds' | 'paused_seconds' | 'interrupted' | 'log_time_entry'
>>

export type PomodoroSessionPage = Paginated<PomodoroSessionResponse>

export function normalizePomodoroSession(session: PomodoroSession): PomodoroSessionResponse {
	return {
		...session,
		id: session.id ?? 0,
		task_id: session.task_id ?? 0,
		phase: session.phase ?? 'focus',
		planned_seconds: session.planned_seconds ?? 0,
		started_at: session.started_at ?? '',
		status: session.status ?? 'finished',
		remaining_seconds: session.remaining_seconds ?? 0,
		paused_seconds: session.paused_seconds ?? 0,
		interrupted: session.interrupted ?? false,
		log_time_entry: session.log_time_entry ?? false,
	}
}

export const pomodoroKeys = {
	all: ['pomodoro'] as const,
	current: ['pomodoro', 'current'] as const,
	sessions: ['pomodoro', 'sessions'] as const,
	sessionList: (from: string, to: string, taskId: number, page: number, perPage: number) =>
		['pomodoro', 'sessions', from, to, taskId, page, perPage] as const,
	stats: ['pomodoro', 'stats'] as const,
	stat: (from: string, to: string, tz: string) => ['pomodoro', 'stats', from, to, tz] as const,
}

// The server is the source of truth, so the running session is re-read on focus
// and once a minute. That is enough for cross-device sync: between refetches the
// frontend counts down locally from remaining_seconds.
export const POMODORO_POLL_INTERVAL = 60 * 1000

export function currentPomodoroQuery() {
	return queryOptions({
		queryKey: pomodoroKeys.current,
		queryFn: async ({signal}) => {
			const {data} = await pomodoroCurrent({signal})
			return data ? normalizePomodoroSession(data) : null
		},
		refetchOnWindowFocus: true,
		refetchInterval: POMODORO_POLL_INTERVAL,
	})
}

export function pomodoroSessionsQuery(from: string, to: string, taskId = 0, page = 1, perPage = 50) {
	const clampedPerPage = pageSizeFor(perPage)
	return queryOptions({
		queryKey: pomodoroKeys.sessionList(from, to, taskId, page, clampedPerPage),
		queryFn: async ({signal}): Promise<PomodoroSessionPage> => {
			const {data} = await pomodoroSessionsList({
				query: {
					...(from ? {from} : {}),
					...(to ? {to} : {}),
					...(taskId > 0 ? {task_id: taskId} : {}),
					page,
					per_page: clampedPerPage,
				},
				signal,
			})
			return {
				...data,
				items: (data.items ?? []).map(normalizePomodoroSession),
				page: data.page ?? page,
				per_page: data.per_page ?? clampedPerPage,
				total: data.total ?? 0,
				total_pages: data.total_pages ?? 0,
			}
		},
	})
}

export function pomodoroStatsQuery(from: string, to: string, tz: string) {
	return queryOptions({
		queryKey: pomodoroKeys.stat(from, to, tz),
		queryFn: async ({signal}): Promise<PomodoroStats> =>
			(await pomodoroStats({query: {from, to, tz}, signal})).data,
	})
}

// Writing the response into the current-session cache is what keeps the badge
// and the focus page in step without either of them refetching.
function patchCurrent(client: QueryClient, session: PomodoroSession | null) {
	const normalized = session ? normalizePomodoroSession(session) : null
	client.setQueryData<PomodoroSessionResponse | null>(
		pomodoroKeys.current,
		// A finished session is no longer current.
		normalized?.status === 'finished' ? null : normalized,
	)
}

function settle(client: QueryClient) {
	return client.invalidateQueries({queryKey: pomodoroKeys.all})
}

export type StartPomodoroInput = {
	phase: PomodoroPhase
	plannedSeconds: number
	taskId?: number
	logTimeEntry?: boolean
}

export function startPomodoroMutationOptions() {
	return contextMutationOptions({
		mutationFn: async ({phase, plannedSeconds, taskId = 0, logTimeEntry = false}: StartPomodoroInput) =>
			(await pomodoroStart({
				body: {
					phase,
					planned_seconds: plannedSeconds,
					task_id: taskId,
					log_time_entry: logTimeEntry,
				},
			})).data,
		onSuccess: (session, _input, client) => patchCurrent(client, session),
		onSettled: (_input, client) => settle(client),
	})
}

export function pausePomodoroMutationOptions() {
	return contextMutationOptions({
		mutationFn: async () => (await pomodoroPause()).data,
		onSuccess: (session, _input, client) => patchCurrent(client, session),
		onSettled: (_input, client) => settle(client),
	})
}

export function resumePomodoroMutationOptions() {
	return contextMutationOptions({
		mutationFn: async () => (await pomodoroResume()).data,
		onSuccess: (session, _input, client) => patchCurrent(client, session),
		onSettled: (_input, client) => settle(client),
	})
}

export function stopPomodoroMutationOptions() {
	return contextMutationOptions({
		mutationFn: async () => (await pomodoroStop()).data,
		onSuccess: (session, _input, client) => patchCurrent(client, session),
		onSettled: (_input, client) => settle(client),
	})
}

export type EstimatePomodoroInput = {
	taskId: number
	// null clears the estimate.
	estimate: number | null
}

// The estimate lives inside the task's pomodoro expand, so it is patched there
// rather than in a query of its own.
function patchEstimate(client: QueryClient, taskId: number, estimate: number) {
	mapTaskEverywhere(client, taskId, task => task.pomodoro === undefined
		? task
		: {...task, pomodoro: {...task.pomodoro, estimate} satisfies TaskPomodoroSummary},
	)
}

export function estimatePomodorosMutationOptions() {
	return contextMutationOptions({
		mutationFn: async ({taskId, estimate}: EstimatePomodoroInput) => {
			if (estimate === null) {
				await taskPomodoroEstimateDelete({path: {task: taskId}})
				return 0
			}
			return (await taskPomodoroEstimateUpdate({
				path: {task: taskId},
				body: {estimate},
			})).data.estimate ?? estimate
		},
		optimistic: {
			queryKeys: () => [],
			update: ({taskId, estimate}, client) => patchEstimate(client, taskId, estimate ?? 0),
		},
		onSuccess: (stored, {taskId}, client) => patchEstimate(client, taskId, stored),
	})
}

export function useStartPomodoroMutation() {
	return useMutation(startPomodoroMutationOptions())
}

export function usePausePomodoroMutation() {
	return useMutation(pausePomodoroMutationOptions())
}

export function useResumePomodoroMutation() {
	return useMutation(resumePomodoroMutationOptions())
}

export function useStopPomodoroMutation() {
	return useMutation(stopPomodoroMutationOptions())
}

export function useEstimatePomodorosMutation() {
	return useMutation(estimatePomodorosMutationOptions())
}

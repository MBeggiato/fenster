import {beforeEach, describe, expect, it, vi} from 'vitest'
import {QueryClient} from '@tanstack/vue-query'

import {normalizeTask, taskKeys, type TaskResponse} from './tasks'
import {
	estimatePomodorosMutationOptions,
	normalizePomodoroSession,
	pomodoroKeys,
	startPomodoroMutationOptions,
	stopPomodoroMutationOptions,
	type PomodoroSessionResponse,
	type StartPomodoroInput,
} from './pomodoro'

const sdk = vi.hoisted(() => ({
	pomodoroStart: vi.fn(),
	pomodoroStop: vi.fn(),
	taskPomodoroEstimateUpdate: vi.fn(),
	taskPomodoroEstimateDelete: vi.fn(),
}))
vi.mock('@/client/generated', () => sdk)
vi.mock('@/message', () => ({
	error: vi.fn(),
	success: vi.fn(),
	translatedError: (message: string) => new Error(message),
}))
vi.mock('@/client/requestContext', () => ({
	captureClientRequestContext: () => 1,
	assertClientRequestContext: vi.fn(),
	isClientRequestContextCurrent: vi.fn(() => true),
}))

function running(overrides: Partial<PomodoroSessionResponse> = {}): PomodoroSessionResponse {
	return normalizePomodoroSession({
		id: 1,
		task_id: 0,
		phase: 'focus',
		planned_seconds: 1500,
		started_at: '2026-09-25T10:00:00Z',
		status: 'running',
		remaining_seconds: 1500,
		...overrides,
	})
}

function current(client: QueryClient) {
	return client.getQueryData<PomodoroSessionResponse | null>(pomodoroKeys.current)
}

describe('pomodoro cache', () => {
	beforeEach(() => vi.resetAllMocks())

	it('fills in the fields the wire leaves out', () => {
		const session = normalizePomodoroSession({})
		expect(session).toMatchObject({
			id: 0,
			task_id: 0,
			phase: 'focus',
			planned_seconds: 0,
			status: 'finished',
			remaining_seconds: 0,
			paused_seconds: 0,
			interrupted: false,
			log_time_entry: false,
		})
	})

	it('writes the started session into the current-session cache', async () => {
		const client = new QueryClient()
		client.setQueryData(pomodoroKeys.current, null)
		sdk.pomodoroStart.mockResolvedValue({data: running({id: 7, task_id: 3})})

		await client.getMutationCache()
			.build(client, startPomodoroMutationOptions())
			.execute({phase: 'focus', plannedSeconds: 1500, taskId: 3} satisfies StartPomodoroInput)

		expect(current(client)).toMatchObject({id: 7, task_id: 3, status: 'running'})
		expect(sdk.pomodoroStart).toHaveBeenCalledWith({
			body: {phase: 'focus', planned_seconds: 1500, task_id: 3, log_time_entry: false},
		})
	})

	it('clears the current session when the response says it finished', async () => {
		const client = new QueryClient()
		client.setQueryData(pomodoroKeys.current, running())
		sdk.pomodoroStop.mockResolvedValue({data: running({status: 'finished', interrupted: true})})

		await client.getMutationCache()
			.build(client, stopPomodoroMutationOptions())
			.execute(undefined)

		expect(current(client)).toBeNull()
	})

	it('patches the estimate into the task the expand lives on', async () => {
		const client = new QueryClient()
		const task: TaskResponse = normalizeTask({
			id: 4,
			title: 'task 4',
			project_id: 1,
			pomodoro: {completed: 2, interrupted: 1, focus_seconds: 600, estimate: 3},
		})
		client.setQueryData(taskKeys.detail(4), task)
		sdk.taskPomodoroEstimateUpdate.mockResolvedValue({data: {task_id: 4, estimate: 6, estimated: true}})

		await client.getMutationCache()
			.build(client, estimatePomodorosMutationOptions())
			.execute({taskId: 4, estimate: 6})

		expect(client.getQueryData<TaskResponse>(taskKeys.detail(4))?.pomodoro).toMatchObject({
			completed: 2,
			interrupted: 1,
			focus_seconds: 600,
			estimate: 6,
		})
	})

	it('clears the estimate without touching the rest of the summary', async () => {
		const client = new QueryClient()
		client.setQueryData(taskKeys.detail(4), normalizeTask({
			id: 4,
			title: 'task 4',
			project_id: 1,
			pomodoro: {completed: 2, interrupted: 0, focus_seconds: 600, estimate: 3},
		}))
		sdk.taskPomodoroEstimateDelete.mockResolvedValue({})

		await client.getMutationCache()
			.build(client, estimatePomodorosMutationOptions())
			.execute({taskId: 4, estimate: null})

		expect(client.getQueryData<TaskResponse>(taskKeys.detail(4))?.pomodoro).toMatchObject({
			completed: 2,
			focus_seconds: 600,
			estimate: 0,
		})
		expect(sdk.taskPomodoroEstimateDelete).toHaveBeenCalledWith({path: {task: 4}})
	})

	it('leaves a task without the expand alone', async () => {
		const client = new QueryClient()
		client.setQueryData(taskKeys.detail(4), normalizeTask({id: 4, title: 'task 4', project_id: 1}))
		sdk.taskPomodoroEstimateUpdate.mockResolvedValue({data: {task_id: 4, estimate: 2, estimated: true}})

		await client.getMutationCache()
			.build(client, estimatePomodorosMutationOptions())
			.execute({taskId: 4, estimate: 2})

		// Materialising a summary from a single field would report 0 completed
		// pomodoros for a task that has some.
		expect(client.getQueryData<TaskResponse>(taskKeys.detail(4))?.pomodoro).toBeUndefined()
	})
})

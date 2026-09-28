<template>
	<div class="pomodoro-view">
		<div class="pomodoro-view__tabs">
			<XButton
				variant="secondary"
				:class="{'is-active': tab === 'timer'}"
				@click="tab = 'timer'"
			>
				{{ $t('pomodoro.tabs.timer') }}
			</XButton>
			<XButton
				variant="secondary"
				:class="{'is-active': tab === 'stats'}"
				@click="tab = 'stats'"
			>
				{{ $t('pomodoro.tabs.stats') }}
			</XButton>
		</div>

		<PomodoroStats v-if="tab === 'stats'" />

		<div
			v-else
			class="pomodoro-view__timer"
		>
			<!-- The phase is only choosable while nothing runs; during a session the ring shows it. -->
			<div
				v-if="!pomodoro.isActive.value"
				class="pomodoro-view__phases"
			>
				<XButton
					v-for="option in POMODORO_PHASES"
					:key="option"
					variant="secondary"
					:class="{'is-active': plannedPhase === option}"
					@click="plannedPhase = option"
				>
					{{ $t(`pomodoro.phase.${option}`) }}
				</XButton>
			</div>

			<PomodoroRing
				:phase="pomodoro.phase.value ?? plannedPhase"
				:time="pomodoro.isActive.value ? pomodoro.formattedRemaining.value : idleTime"
				:progress="pomodoro.isActive.value ? pomodoro.progress.value : 0"
			/>

			<div class="pomodoro-view__controls">
				<XButton
					v-if="!pomodoro.isActive.value"
					v-cy="'pomodoroStartPhase'"
					icon="play"
					:loading="pomodoro.isPending.value"
					@click="pomodoro.start(plannedPhase, selectedTask?.id ?? 0)"
				>
					{{ $t('pomodoro.start') }}
				</XButton>
				<template v-else>
					<XButton
						v-cy="'pomodoroTogglePhase'"
						:icon="pomodoro.isPaused.value ? 'play' : 'pause'"
						:loading="pomodoro.isPending.value"
						@click="pomodoro.isPaused.value ? pomodoro.resume() : pomodoro.pause()"
					>
						{{ pomodoro.isPaused.value ? $t('pomodoro.resume') : $t('pomodoro.pause') }}
					</XButton>
					<XButton
						v-cy="'pomodoroStopPhase'"
						variant="secondary"
						icon="stop"
						:loading="pomodoro.isPending.value"
						@click="confirmStop"
					>
						{{ $t('pomodoro.stop') }}
					</XButton>
				</template>
			</div>

			<!-- The phase that just ended offers the next one; auto-start skips this. -->
			<Message
				v-if="pomodoro.finishedPhase.value !== null && !pomodoro.isActive.value"
				class="pomodoro-view__finished"
			>
				<span>{{ $t(`pomodoro.done.${pomodoro.finishedPhase.value}`) }}</span>
				<div class="pomodoro-view__finished-actions">
					<XButton @click="startNext">
						{{ $t('pomodoro.next.start', {phase: $t(`pomodoro.phase.${pomodoro.nextPhase.value}`)}) }}
					</XButton>
					<XButton
						variant="tertiary"
						@click="pomodoro.dismissFinished()"
					>
						{{ $t('pomodoro.next.skip') }}
					</XButton>
				</div>
			</Message>

			<div class="pomodoro-view__task">
				<label
					class="label"
					for="pomodoro-task"
				>{{ $t('pomodoro.task.working') }}</label>
				<template v-if="pomodoro.isActive.value">
					<RouterLink
						v-if="pomodoro.session.value?.task"
						:to="{name: 'task.detail', params: {id: pomodoro.session.value.task.id}}"
					>
						{{ pomodoro.session.value.task.title }}
					</RouterLink>
					<span v-else>{{ $t('pomodoro.task.none') }}</span>
				</template>
				<Multiselect
					v-else
					id="pomodoro-task"
					v-model="selectedTask"
					:placeholder="$t('pomodoro.task.search')"
					:loading="taskQuery.isFetching.value"
					:search-results="taskQuery.tasks.value"
					label="title"
					@search="query => taskSearch = query"
				>
					<template #searchResult="{option}">
						{{ option.title }}
					</template>
				</Multiselect>
			</div>

			<div class="pomodoro-view__today">
				<span class="pomodoro-view__today-label">{{ $t('pomodoro.today.title') }}</span>
				<ul
					v-if="pomodoro.todayFocusSessions.value.length"
					class="pomodoro-view__tomatoes"
				>
					<li
						v-for="item in pomodoro.todayFocusSessions.value"
						:key="item.id"
						v-tooltip="tomatoTitle(item)"
						class="pomodoro-view__tomato"
						:class="{'pomodoro-view__tomato--interrupted': item.interrupted}"
					/>
				</ul>
				<span v-else>{{ $t('pomodoro.today.empty') }}</span>
				<span class="pomodoro-view__today-sep">·</span>
				<span>{{ $t('pomodoro.today.focusTime', {duration: formatDuration(pomodoro.todayFocusSeconds.value)}) }}</span>
				<span class="pomodoro-view__today-sep">·</span>
				<span>{{ longBreakHint }}</span>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {onKeyStroke} from '@vueuse/core'
import {useI18n} from 'vue-i18n'

import XButton from '@/components/input/Button.vue'
import Message from '@/components/misc/Message.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import PomodoroRing from '@/components/pomodoro/PomodoroRing.vue'
import PomodoroStats from '@/components/pomodoro/PomodoroStats.vue'

import {POMODORO_PHASES, formatCountdown, usePomodoro} from '@/composables/usePomodoro'
import {useTasks} from '@/composables/useTasks'
import {useTitle} from '@/composables/useTitle'
import {formatDuration} from '@/helpers/time/formatDuration'
import {formatDateShort} from '@/helpers/time/formatDate'
import {SHORTCUTS} from '@/constants/shortcuts'
import type {PomodoroPhase, PomodoroSessionResponse} from '@/client/queries/pomodoro'
import type {TaskResponse} from '@/client/queries/tasks'

const {t} = useI18n()
const pomodoro = usePomodoro()

useTitle(() => t('pomodoro.title'))

const tab = ref<'timer' | 'stats'>('timer')
const plannedPhase = ref<PomodoroPhase>('focus')

const idleTime = computed(() => formatCountdown(pomodoro.plannedSecondsFor(plannedPhase.value)))

const selectedTask = ref<TaskResponse | null>(null)
const taskSearch = ref('')
const taskQuery = useTasks(
	() => ({params: {q: taskSearch.value, sort_by: ['done']}}),
	{enabled: () => taskSearch.value !== ''},
)

const longBreakHint = computed(() => pomodoro.untilLongBreak.value <= 1
	? t('pomodoro.today.untilLongBreakNow')
	: t('pomodoro.today.untilLongBreak', {count: pomodoro.untilLongBreak.value - 1}),
)

function tomatoTitle(session: PomodoroSessionResponse): string {
	const time = session.ended_at ? formatDateShort(session.ended_at) : ''
	return session.interrupted
		? t('pomodoro.today.interruptedAt', {time})
		: t('pomodoro.today.completedAt', {time})
}

function confirmStop() {
	// Stopping records the session as interrupted, which cannot be undone.
	if (!window.confirm(t('pomodoro.stopConfirm'))) return
	void pomodoro.stop()
}

async function startNext() {
	const upcoming = pomodoro.nextPhase.value
	const taskId = upcoming === 'focus' ? pomodoro.finishedTaskId.value : 0
	pomodoro.dismissFinished()
	await pomodoro.start(upcoming, taskId)
}

// Space and Escape are page-level, so they are listened for rather than bound to
// a button with v-shortcut. A keystroke inside a field stays with the field.
function isTyping(event: KeyboardEvent): boolean {
	const target = event.target as HTMLElement | null
	if (target === null) return false
	return target.isContentEditable
		|| ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)
}

onKeyStroke(SHORTCUTS.pomodoro.toggle, event => {
	if (isTyping(event)) return
	// Space would otherwise scroll the page.
	event.preventDefault()
	void pomodoro.toggle()
})

onKeyStroke(SHORTCUTS.pomodoro.stop, event => {
	if (isTyping(event) || !pomodoro.isActive.value) return
	confirmStop()
})
</script>

<style lang="scss" scoped>
.pomodoro-view__tabs {
	display: flex;
	gap: var(--space-2);
	margin-block-end: var(--space-6);
}

.pomodoro-view__timer {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: var(--space-6);
}

.pomodoro-view__phases {
	display: flex;
	flex-wrap: wrap;
	justify-content: center;
	gap: var(--space-2);
}

.pomodoro-view__controls {
	display: flex;
	flex-wrap: wrap;
	justify-content: center;
	gap: var(--space-2);
}

.pomodoro-view__finished {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: var(--space-3);
}

.pomodoro-view__finished-actions {
	display: flex;
	gap: var(--space-2);
}

.pomodoro-view__task {
	inline-size: 22rem;
	max-inline-size: 100%;
	text-align: center;
}

.pomodoro-view__today {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: center;
	gap: var(--space-2);
	color: var(--grey-500);
	font-size: var(--font-size-sm);
}

.pomodoro-view__today-label {
	font-weight: 600;
	color: var(--text);
}

.pomodoro-view__today-sep {
	color: var(--grey-300);
}

.pomodoro-view__tomatoes {
	display: flex;
	gap: var(--space-1);
	margin: 0;
	padding: 0;
	list-style: none;
}

.pomodoro-view__tomato {
	inline-size: .75rem;
	block-size: .75rem;
	border-radius: 100%;
	background: var(--danger);
}

// An interrupted phase leaves a hollow tomato: the time counted, the pomodoro did not.
.pomodoro-view__tomato--interrupted {
	background: transparent;
	border: 2px solid var(--danger);
}
</style>

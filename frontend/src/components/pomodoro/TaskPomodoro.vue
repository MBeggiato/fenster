<template>
	<div class="task-pomodoro">
		<span class="action-heading">
			{{ $t('pomodoro.title') }}
			<RouterLink
				v-tooltip="$t('pomodoro.openFocusPage')"
				:to="{name: 'pomodoro'}"
				class="task-pomodoro__link"
				:aria-label="$t('pomodoro.openFocusPage')"
			>
				<Icon icon="play" />
			</RouterLink>
		</span>

		<XButton
			v-cy="'taskStartFocus'"
			variant="secondary"
			icon="play"
			:loading="pomodoro.isPending.value"
			@click="pomodoro.start('focus', task.id)"
		>
			{{ otherSessionRunning ? $t('pomodoro.switchHere') : $t('pomodoro.start') }}
		</XButton>

		<p
			v-if="summary.completed || summary.interrupted"
			class="task-pomodoro__summary"
		>
			<span class="task-pomodoro__tomatoes">
				<span
					v-for="index in summary.completed"
					:key="`done-${index}`"
					class="task-pomodoro__tomato"
				/>
				<span
					v-for="index in summary.interrupted"
					:key="`stopped-${index}`"
					class="task-pomodoro__tomato task-pomodoro__tomato--interrupted"
				/>
			</span>
			<span v-if="summary.estimate">
				{{ $t('pomodoro.task.estimateShort', {completed: summary.completed, estimate: summary.estimate}) }}
			</span>
			<span>·</span>
			<span>{{ formatDuration(summary.focus_seconds) }}</span>
		</p>
		<p
			v-else
			class="task-pomodoro__summary"
		>
			{{ $t('pomodoro.task.noneYet') }}
		</p>

		<label
			class="task-pomodoro__estimate-label"
			:for="`task-pomodoro-estimate-${task.id}`"
		>{{ $t('pomodoro.task.estimate') }}</label>
		<input
			:id="`task-pomodoro-estimate-${task.id}`"
			v-cy="'taskPomodoroEstimate'"
			class="input task-pomodoro__estimate"
			type="number"
			min="1"
			max="99"
			:value="estimateInput"
			@change="onEstimateChange"
		>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import Icon from '@/components/misc/Icon'
import XButton from '@/components/input/Button.vue'

import {useEstimatePomodorosMutation} from '@/client/queries/pomodoro'
import {usePomodoro} from '@/composables/usePomodoro'
import {formatDuration} from '@/helpers/time/formatDuration'
import type {TaskResponse} from '@/client/queries/tasks'

const props = defineProps<{
	task: TaskResponse
}>()

const pomodoro = usePomodoro()
const estimate = useEstimatePomodorosMutation()

// The expand is absent until the task has been read with expand=pomodoro, and
// every field of it is optional on the wire.
const summary = computed(() => ({
	completed: props.task.pomodoro?.completed ?? 0,
	interrupted: props.task.pomodoro?.interrupted ?? 0,
	focus_seconds: props.task.pomodoro?.focus_seconds ?? 0,
	estimate: props.task.pomodoro?.estimate ?? 0,
}))

// Starting here while another session runs interrupts it, so the button says so.
const otherSessionRunning = computed(() => pomodoro.isActive.value
	&& pomodoro.session.value?.task_id !== props.task.id)

const estimateInput = computed(() => summary.value.estimate || '')

// Emptying the field clears the estimate; a value outside 1–99 is ignored rather
// than sent, since the API would refuse it anyway.
function onEstimateChange(event: Event) {
	const raw = (event.target as HTMLInputElement).value.trim()
	if (raw === '') {
		estimate.mutate({taskId: props.task.id, estimate: null})
		return
	}
	const parsed = Number(raw)
	if (!Number.isInteger(parsed) || parsed < 1 || parsed > 99) return
	estimate.mutate({taskId: props.task.id, estimate: parsed})
}
</script>

<style lang="scss" scoped>
// Matches the task detail's own action column, whose scoped styles don't reach in here.
.task-pomodoro {
	display: flex;
	flex-direction: column;

	.button {
		inline-size: 100%;
		margin-block-end: .5rem;
		justify-content: left;
	}
}

.action-heading {
	text-transform: uppercase;
	color: var(--grey-700);
	font-size: .75rem;
	font-weight: 700;
	margin: .5rem 0;
	display: inline-block;
}

.task-pomodoro__link {
	margin-inline-start: .25rem;
	color: var(--grey-500);
}

.task-pomodoro__summary {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .375rem;
	margin-block: .25rem .5rem;
	color: var(--grey-500);
	font-size: .85rem;
}

.task-pomodoro__tomatoes {
	display: inline-flex;
	gap: .1875rem;
}

.task-pomodoro__tomato {
	inline-size: .625rem;
	block-size: .625rem;
	border-radius: 100%;
	background: var(--danger);
}

.task-pomodoro__tomato--interrupted {
	background: transparent;
	border: 1.5px solid var(--danger);
}

.task-pomodoro__estimate-label {
	color: var(--grey-500);
	font-size: .85rem;
	margin-block-end: .25rem;
}

.task-pomodoro__estimate {
	margin-block-end: .75rem;
}
</style>

<template>
	<div
		v-if="!authStore.isLinkShareAuth"
		v-cy="'pomodoroBadge'"
		class="pomodoro-badge"
	>
		<template v-if="pomodoro.isActive.value">
			<RouterLink
				v-tooltip="taskTitle || $t('pomodoro.title')"
				:to="{name: 'pomodoro'}"
				class="pomodoro-badge__time"
				:class="{'pomodoro-badge__time--paused': pomodoro.isPaused.value}"
			>
				<span
					class="pomodoro-badge__dot"
					:class="`pomodoro-badge__dot--${pomodoro.phase.value}`"
					aria-hidden="true"
				/>
				{{ pomodoro.formattedRemaining.value }}
				<span class="is-sr-only">{{ $t(`pomodoro.phase.${pomodoro.phase.value}`) }}</span>
			</RouterLink>
			<BaseButton
				v-tooltip="pomodoro.isPaused.value ? $t('pomodoro.resume') : $t('pomodoro.pause')"
				v-cy="'pomodoroToggle'"
				class="pomodoro-badge__action"
				:aria-label="pomodoro.isPaused.value ? $t('pomodoro.resume') : $t('pomodoro.pause')"
				:disabled="pomodoro.isPending.value"
				@click="pomodoro.isPaused.value ? pomodoro.resume() : pomodoro.pause()"
			>
				<Icon :icon="pomodoro.isPaused.value ? 'play' : 'pause'" />
			</BaseButton>
			<BaseButton
				v-tooltip="$t('pomodoro.stop')"
				v-cy="'pomodoroStop'"
				class="pomodoro-badge__action pomodoro-badge__action--stop"
				:aria-label="$t('pomodoro.stop')"
				:disabled="pomodoro.isPending.value"
				@click="pomodoro.stop()"
			>
				<Icon icon="stop" />
			</BaseButton>
		</template>

		<Popup
			v-else
			v-model:open="isOpen"
			:anchor="trigger"
			placement="bottom-end"
		>
			<template #trigger="{toggle}">
				<BaseButton
					ref="triggerButton"
					v-tooltip="$t('pomodoro.title')"
					v-cy="'pomodoroStart'"
					class="pomodoro-badge__action"
					:aria-label="$t('pomodoro.title')"
					:aria-expanded="isOpen"
					@click="toggle()"
				>
					<Icon :icon="['far', 'clock']" />
				</BaseButton>
			</template>
			<template #content="{close}">
				<Card
					class="pomodoro-badge__popup"
					:title="$t('pomodoro.title')"
				>
					<div class="field">
						<label
							class="label"
							for="pomodoro-badge-task"
						>{{ $t('pomodoro.task.working') }}</label>
						<Multiselect
							id="pomodoro-badge-task"
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

					<XButton
						v-cy="'pomodoroStartFocus'"
						:loading="pomodoro.isPending.value"
						@click="startFocus(close)"
					>
						{{ $t('pomodoro.startWithDuration', {minutes: focusMinutes}) }}
					</XButton>

					<BaseButton
						:to="{name: 'pomodoro'}"
						class="pomodoro-badge__link"
						@click="close"
					>
						{{ $t('pomodoro.openFocusPage') }}
					</BaseButton>
				</Card>
			</template>
		</Popup>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, useTemplateRef} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import XButton from '@/components/input/Button.vue'
import Card from '@/components/misc/Card.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import Popup from '@/components/misc/Popup.vue'

import {usePomodoro} from '@/composables/usePomodoro'
import {useTasks} from '@/composables/useTasks'
import {useAuthStore} from '@/stores/auth'
import type {TaskResponse} from '@/client/queries/tasks'

const authStore = useAuthStore()
const pomodoro = usePomodoro()

const isOpen = ref(false)
const triggerButton = useTemplateRef<{$el: HTMLElement} | null>('triggerButton')
const trigger = computed(() => triggerButton.value?.$el ?? null)

const taskTitle = computed(() => pomodoro.session.value?.task?.title ?? '')
const focusMinutes = computed(() => Math.round(pomodoro.plannedSecondsFor('focus') / 60))

const selectedTask = ref<TaskResponse | null>(null)
const taskSearch = ref('')
const taskQuery = useTasks(
	() => ({params: {q: taskSearch.value, sort_by: ['done']}}),
	{enabled: () => taskSearch.value !== ''},
)

async function startFocus(close: () => void) {
	try {
		await pomodoro.start('focus', selectedTask.value?.id ?? 0)
	} catch {
		return
	}
	selectedTask.value = null
	taskSearch.value = ''
	close()
}
</script>

<style lang="scss" scoped>
.pomodoro-badge {
	display: inline-flex;
	align-items: center;
	gap: var(--space-1);
	white-space: nowrap;
}

.pomodoro-badge__time {
	display: inline-flex;
	align-items: center;
	gap: .375rem;
	padding-inline: var(--space-3) var(--space-1);
	color: var(--text);
	font-variant-numeric: tabular-nums;
	font-weight: 600;
}

// A paused timer dims and breathes, so a forgotten pause is visible at a glance.
.pomodoro-badge__time--paused {
	color: var(--grey-400);
	animation: pomodoro-badge-blink 2s ease-in-out infinite;
}

@keyframes pomodoro-badge-blink {
	50% { opacity: .45; }
}

@media (prefers-reduced-motion: reduce) {
	.pomodoro-badge__time--paused {
		animation: none;
		opacity: .6;
	}
}

.pomodoro-badge__dot {
	inline-size: .5rem;
	block-size: .5rem;
	border-radius: 100%;
	background: var(--primary);
}

.pomodoro-badge__dot--focus { background: var(--danger); }
.pomodoro-badge__dot--short_break { background: var(--success); }
.pomodoro-badge__dot--long_break { background: var(--info); }

.pomodoro-badge__action {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	padding-inline: var(--space-2);
	color: var(--grey-400);
	transition: color $transition;

	&:hover {
		color: var(--primary);
	}
}

.pomodoro-badge__action--stop:hover {
	color: var(--danger);
}

.pomodoro-badge__popup {
	inline-size: 20rem;
	max-inline-size: calc(100vw - 1rem);
}

.pomodoro-badge__link {
	display: block;
	margin-block-start: var(--space-3);
	color: var(--primary);
	font-size: var(--font-size-sm);
}
</style>

<template>
	<div class="eisenhower-toggles">
		<span class="action-heading">
			{{ $t('task.detail.eisenhower.heading') }}
			<RouterLink
				v-tooltip="$t('task.detail.eisenhower.openMatrix')"
				:to="{name: 'tasks.eisenhower'}"
				class="matrix-link"
				:aria-label="$t('task.detail.eisenhower.openMatrix')"
			>
				<Icon icon="table-cells-large" />
			</RouterLink>
		</span>
		<XButton
			variant="secondary"
			:icon="current?.urgent ? 'bolt' : ['far', 'clock']"
			:aria-pressed="current?.urgent ?? false"
			:class="{'is-active': current?.urgent}"
			class="eisenhower-urgent"
			@click="toggle('urgent')"
		>
			{{ current?.urgent ? $t('task.eisenhower.urgent') : $t('task.eisenhower.notUrgent') }}
		</XButton>
		<XButton
			variant="secondary"
			:icon="current?.important ? 'star' : ['far', 'star']"
			:aria-pressed="current?.important ?? false"
			:class="{'is-active': current?.important}"
			class="eisenhower-important"
			@click="toggle('important')"
		>
			{{ current?.important ? $t('task.eisenhower.important') : $t('task.eisenhower.notImportant') }}
		</XButton>
		<p class="eisenhower-state">
			{{ current ? $t(`task.eisenhower.quadrants.${quadrantFor(current)}.title`) : $t('task.detail.eisenhower.unclassified') }}
			<BaseButton
				v-if="current"
				class="eisenhower-reset"
				@click="reset"
			>
				{{ $t('task.detail.eisenhower.reset') }}
			</BaseButton>
		</p>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import Icon from '@/components/misc/Icon'
import {classificationOf, quadrantFor, useClassifyTaskMutation} from '@/client/queries/eisenhower'
import type {TaskResponse} from '@/client/queries/tasks'

const props = defineProps<{
	task: TaskResponse
}>()

const classify = useClassifyTaskMutation()
const current = computed(() => classificationOf(props.task))

function toggle(flag: 'urgent' | 'important') {
	// Classifying an unclassified task counts the other flag as false.
	const base = current.value ?? {urgent: false, important: false}
	classify.mutate({task: props.task, classification: {...base, [flag]: !base[flag]}})
}

function reset() {
	classify.mutate({task: props.task, classification: null})
}
</script>

<style lang="scss" scoped>
// Matches the task detail's own action column, whose scoped styles don't reach in here.
.eisenhower-toggles {
	display: flex;
	flex-direction: column;

	.button {
		inline-size: 100%;
		margin-block-end: var(--space-2);
		justify-content: left;
	}
}

.action-heading {
	text-transform: uppercase;
	color: var(--grey-700);
	font-size: var(--font-size-xs);
	font-weight: 700;
	margin: var(--space-2) 0;
	display: inline-block;
}

.matrix-link {
	margin-inline-start: var(--space-1);
	color: var(--grey-500);
}

.is-active {
	color: var(--primary);
}

.eisenhower-state {
	margin-block: var(--space-1) var(--space-3);
	color: var(--grey-500);
	font-size: .85rem;
}

.eisenhower-reset {
	margin-inline-start: var(--space-2);
	color: var(--primary);
	text-decoration: underline;
}
</style>

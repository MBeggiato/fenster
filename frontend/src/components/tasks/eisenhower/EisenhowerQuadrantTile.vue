<template>
	<button
		type="button"
		class="eisenhower-tile"
		:class="`is-${quadrant}`"
		@click="$emit('open')"
	>
		<span class="eisenhower-tile__header">
			<span class="eisenhower-tile__title">{{ title }}</span>
			<span
				v-if="total > 0"
				class="eisenhower-tile__count"
			>{{ total }}</span>
		</span>
		<ul
			v-if="previewTasks.length > 0"
			class="eisenhower-tile__tasks"
		>
			<li
				v-for="task in previewTasks"
				:key="task.id"
				class="eisenhower-tile__task"
			>
				{{ task.title }}
			</li>
		</ul>
		<p
			v-else-if="!isLoading"
			class="eisenhower-tile__empty"
		>
			{{ $t('task.eisenhower.empty') }}
		</p>
	</button>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import type {EisenhowerParams, EisenhowerQuadrant} from '@/client/queries/eisenhower'
import {useEisenhowerQuadrant} from '@/composables/useEisenhowerQuadrant'

const props = defineProps<{
	quadrant: EisenhowerQuadrant
	title: string
	params: EisenhowerParams
}>()

defineEmits<{
	(e: 'open'): void
}>()

// The full list (with the same query key) is fetched again by EisenhowerQuadrant
// once this tile is opened, so this stays cheap thanks to query cache reuse.
const {tasks, total, isLoading} = useEisenhowerQuadrant(
	() => props.quadrant,
	() => props.params,
)

const visibleTasks = computed(() => props.params.include_done
	? tasks.value
	: tasks.value.filter(task => !task.done))

const previewTasks = computed(() => visibleTasks.value.slice(0, 2))
</script>

<style lang="scss" scoped>
.eisenhower-tile {
	display: flex;
	flex-direction: column;
	gap: var(--space-2);
	min-block-size: 44px;
	padding: var(--space-3);
	border: none;
	border-block-start: 4px solid var(--quadrant-accent, var(--grey-300));
	border-radius: $radius;
	background: var(--white);
	box-shadow: var(--shadow-sm);
	text-align: start;
	cursor: pointer;

	&.is-do {
		--quadrant-accent: var(--danger);
	}

	&.is-schedule {
		--quadrant-accent: var(--primary);
	}

	&.is-delegate {
		--quadrant-accent: var(--warning);
	}

	&.is-eliminate {
		--quadrant-accent: var(--grey-400);
	}
}

.eisenhower-tile__header {
	display: flex;
	align-items: center;
	gap: var(--space-2);
}

.eisenhower-tile__title {
	font-size: var(--font-size-sm);
	font-weight: 700;
}

.eisenhower-tile__count {
	padding-inline: var(--space-2);
	border-radius: 1rem;
	background: var(--grey-100);
	color: var(--grey-600);
	font-size: var(--font-size-xs);
}

.eisenhower-tile__tasks {
	margin: 0;
	padding: 0;
	list-style: none;
}

.eisenhower-tile__task {
	overflow: hidden;
	color: var(--grey-600);
	font-size: var(--font-size-xs);
	text-overflow: ellipsis;
	white-space: nowrap;
}

.eisenhower-tile__empty {
	margin: 0;
	color: var(--grey-500);
	font-size: var(--font-size-xs);
}
</style>

<template>
	<Modal
		:enabled="enabled"
		variant="sheet"
		:title="task ? $t('mobile.eisenhower.moveSheet.title', {task: task.title}) : ''"
		@close="$emit('close')"
	>
		<ul class="quadrant-move-list">
			<li
				v-for="target in targets"
				:key="target.quadrant"
			>
				<button
					type="button"
					class="quadrant-move-option"
					:disabled="target.quadrant === currentQuadrant || classify.isPending.value"
					@click="move(target.quadrant)"
				>
					{{ target.title }}
					<span
						v-if="target.quadrant === currentQuadrant"
						class="quadrant-move-current"
					>{{ $t('mobile.eisenhower.moveSheet.current') }}</span>
				</button>
			</li>
		</ul>
	</Modal>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'

import Modal from '@/components/misc/Modal.vue'
import {
	EISENHOWER_QUADRANTS,
	classificationOf,
	flagsFor,
	quadrantFor,
	useClassifyTaskMutation,
	type EisenhowerQuadrant,
} from '@/client/queries/eisenhower'
import type {TaskResponse} from '@/client/queries/tasks'

const props = defineProps<{
	enabled: boolean
	task: TaskResponse | null
}>()

const emit = defineEmits<{
	(e: 'close'): void
}>()

const {t} = useI18n({useScope: 'global'})

const classify = useClassifyTaskMutation()

const currentQuadrant = computed(() => props.task
	? quadrantFor(classificationOf(props.task))
	: null)

const targets = computed<{quadrant: EisenhowerQuadrant, title: string}[]>(() => [
	...EISENHOWER_QUADRANTS.map(({quadrant}) => ({
		quadrant,
		title: t(`task.eisenhower.quadrants.${quadrant}.title`),
	})),
	{quadrant: 'unclassified', title: t('task.eisenhower.quadrants.unclassified.title')},
])

async function move(target: EisenhowerQuadrant) {
	if (!props.task || target === currentQuadrant.value) return
	await classify.mutateAsync({task: props.task, classification: flagsFor(target)})
	emit('close')
}
</script>

<style lang="scss" scoped>
.quadrant-move-list {
	margin: 0;
	padding: 0;
	list-style: none;
}

.quadrant-move-option {
	display: flex;
	align-items: center;
	justify-content: space-between;
	inline-size: 100%;
	min-block-size: 44px;
	padding: var(--space-2) var(--space-3);
	border: none;
	border-radius: $radius;
	background: transparent;
	color: var(--text);
	font-size: var(--font-size-md);
	text-align: start;
	cursor: pointer;

	&:hover:not(:disabled),
	&:focus-visible {
		background: var(--grey-100);
	}

	&:disabled {
		color: var(--grey-500);
		cursor: default;
	}
}

.quadrant-move-current {
	color: var(--grey-500);
	font-size: var(--font-size-xs);
}
</style>

<template>
	<div class="empty-state">
		<Icon
			:icon="icon"
			class="empty-state__icon"
			aria-hidden="true"
		/>
		<p class="empty-state__title">
			{{ title }}
		</p>
		<p
			v-if="text"
			class="empty-state__text"
		>
			{{ text }}
		</p>
		<div
			v-if="$slots.default"
			class="empty-state__action"
		>
			<slot />
		</div>
	</div>
</template>

<script setup lang="ts">
import type {IconProp} from '@fortawesome/fontawesome-svg-core'

import Icon from '@/components/misc/Icon'

defineProps<{
	icon: IconProp
	title: string
	text?: string
}>()
</script>

<style lang="scss" scoped>
.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	text-align: center;
	padding: 2rem var(--space-4);
	color: var(--text-muted);
}

.empty-state__icon {
	font-size: var(--font-size-2xl);
	color: var(--grey-400);
	margin-block-end: var(--space-3);
}

.empty-state__title {
	font-size: var(--font-size-md);
	font-weight: var(--font-weight-semibold);
	color: var(--text);
}

.empty-state__text {
	font-size: var(--font-size-sm);
	margin-block-start: var(--space-1);
}

.empty-state__action {
	margin-block-start: var(--space-4);
}
</style>

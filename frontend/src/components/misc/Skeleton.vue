<template>
	<span
		class="skeleton"
		:class="`skeleton--${shape}`"
	>
		<span
			v-for="(style, n) in barStyles"
			:key="n"
			class="skeleton__bar"
			aria-hidden="true"
			:style="style"
		/>
		<span class="is-sr-only">{{ $t('misc.loading') }}</span>
	</span>
</template>

<script setup lang="ts">
import {computed} from 'vue'

const props = withDefaults(defineProps<{
	shape?: 'text' | 'block' | 'circle'
	lines?: number
	width?: string
	height?: string
}>(), {
	shape: 'text',
	lines: 1,
	width: undefined,
	height: undefined,
})

defineOptions({name: 'SkeletonLoader'})

// A lone trailing line at full width reads as a hard-edged block rather than
// text, so it's shortened unless the caller pinned an explicit width.
const barStyles = computed(() => {
	const count = props.shape === 'text' ? Math.max(1, props.lines) : 1
	return Array.from({length: count}, (_, i) => ({
		inlineSize: props.width ?? (i === count - 1 && count > 1 ? '60%' : undefined),
		blockSize: props.height,
	}))
})
</script>

<style lang="scss" scoped>
.skeleton {
	display: flex;
	flex-direction: column;
	gap: .4em;
	inline-size: 100%;
}

.skeleton--block,
.skeleton--circle {
	display: inline-flex;
}

.skeleton__bar {
	display: block;
	inline-size: 100%;
	block-size: 1em;
	border-radius: var(--radius-sm);
	background: linear-gradient(90deg, var(--grey-200) 25%, var(--grey-100) 50%, var(--grey-200) 75%);
	background-size: 200% 100%;
	animation: skeleton-shimmer 1.4s ease-in-out infinite;
}

.skeleton--block .skeleton__bar {
	block-size: 4rem;
}

.skeleton--circle .skeleton__bar {
	inline-size: 2.5rem;
	block-size: 2.5rem;
	border-radius: 50%;
}

@keyframes skeleton-shimmer {
	0% { background-position: 200% 0; }
	100% { background-position: -200% 0; }
}

@media (prefers-reduced-motion: reduce) {
	.skeleton__bar {
		animation: none;
		background: var(--grey-200);
	}
}
</style>

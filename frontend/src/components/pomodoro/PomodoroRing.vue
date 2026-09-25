<template>
	<div
		class="pomodoro-ring"
		:class="`pomodoro-ring--${phase}`"
		role="timer"
		:aria-label="label"
	>
		<svg
			class="pomodoro-ring__svg"
			viewBox="0 0 100 100"
			aria-hidden="true"
		>
			<circle
				class="pomodoro-ring__track"
				cx="50"
				cy="50"
				:r="RADIUS"
			/>
			<circle
				class="pomodoro-ring__progress"
				cx="50"
				cy="50"
				:r="RADIUS"
				:stroke-dasharray="CIRCUMFERENCE"
				:stroke-dashoffset="dashOffset"
			/>
		</svg>
		<div class="pomodoro-ring__label">
			<span class="pomodoro-ring__time">{{ time }}</span>
			<span class="pomodoro-ring__phase">{{ $t(`pomodoro.phase.${phase}`) }}</span>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'

import type {PomodoroPhase} from '@/client/queries/pomodoro'

const props = defineProps<{
	phase: PomodoroPhase
	time: string
	// 0 = the phase just started, 1 = it is over.
	progress: number
}>()

const RADIUS = 45
const CIRCUMFERENCE = 2 * Math.PI * RADIUS

// The ring drains as time passes, so the remaining arc is what stays drawn.
const dashOffset = computed(() => CIRCUMFERENCE * Math.min(1, Math.max(0, props.progress)))

const {t} = useI18n()
const label = computed(() => `${props.time} ${t(`pomodoro.phase.${props.phase}`)}`)
</script>

<style lang="scss" scoped>
.pomodoro-ring {
	position: relative;
	inline-size: 14rem;
	block-size: 14rem;
	max-inline-size: 100%;

	--pomodoro-ring-color: var(--primary);
}

.pomodoro-ring--focus { --pomodoro-ring-color: var(--danger); }
.pomodoro-ring--short_break { --pomodoro-ring-color: var(--success); }
.pomodoro-ring--long_break { --pomodoro-ring-color: var(--info); }

.pomodoro-ring__svg {
	inline-size: 100%;
	block-size: 100%;
	// The arc starts at twelve o'clock instead of three.
	transform: rotate(-90deg);
}

.pomodoro-ring__track,
.pomodoro-ring__progress {
	fill: none;
	stroke-width: 6;
}

.pomodoro-ring__track {
	stroke: var(--grey-200);
}

.pomodoro-ring__progress {
	stroke: var(--pomodoro-ring-color);
	stroke-linecap: round;
	transition: stroke-dashoffset $transition;
}

@media (prefers-reduced-motion: reduce) {
	.pomodoro-ring__progress {
		transition: none;
	}
}

.pomodoro-ring__label {
	position: absolute;
	inset: 0;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: .25rem;
}

.pomodoro-ring__time {
	font-size: 2.5rem;
	font-weight: 700;
	font-variant-numeric: tabular-nums;
	line-height: 1;
	color: var(--text);
}

.pomodoro-ring__phase {
	font-size: .875rem;
	color: var(--grey-500);
}
</style>

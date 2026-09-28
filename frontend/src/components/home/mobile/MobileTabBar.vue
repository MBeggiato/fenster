<template>
	<nav
		class="mobile-tab-bar d-print-none"
		:aria-label="$t('mobile.tabBar.label')"
	>
		<RouterLink
			:to="{name: 'home'}"
			class="mobile-tab-bar__item"
			:class="{'is-active': activeRouteName === 'home'}"
		>
			<span class="mobile-tab-bar__icon">
				<Icon icon="calendar" />
			</span>
			<span>{{ $t('mobile.tabBar.home') }}</span>
		</RouterLink>

		<RouterLink
			:to="{name: 'tasks.range'}"
			class="mobile-tab-bar__item"
			:class="{'is-active': activeRouteName === 'tasks.range'}"
		>
			<span class="mobile-tab-bar__icon">
				<Icon :icon="['far', 'calendar-alt']" />
			</span>
			<span>{{ $t('navigation.upcoming') }}</span>
		</RouterLink>

		<button
			type="button"
			class="mobile-tab-bar__capture"
			:aria-label="$t('mobile.tabBar.capture')"
			@click="$emit('capture')"
		>
			<Icon icon="plus" />
		</button>

		<RouterLink
			:to="{name: 'projects.index'}"
			class="mobile-tab-bar__item"
			:class="{'is-active': activeRouteName === 'projects.index'}"
		>
			<span class="mobile-tab-bar__icon">
				<Icon icon="layer-group" />
			</span>
			<span>{{ $t('project.projects') }}</span>
		</RouterLink>

		<RouterLink
			:to="{name: 'pomodoro'}"
			class="mobile-tab-bar__item"
			:class="{'is-active': activeRouteName === 'pomodoro'}"
		>
			<span class="mobile-tab-bar__icon">
				<Icon icon="play" />
				<span
					v-if="isPomodoroActive"
					class="mobile-tab-bar__live-indicator"
					:aria-label="$t('mobile.tabBar.focusActive')"
				/>
			</span>
			<span>{{ $t('pomodoro.title') }}</span>
		</RouterLink>
	</nav>
</template>

<script lang="ts" setup>
import {computed} from 'vue'
import type {RouteRecordNameGeneric} from 'vue-router'

import {usePomodoro} from '@/composables/usePomodoro'

defineProps<{
	activeRouteName: RouteRecordNameGeneric,
}>()

defineEmits<{
	(e: 'capture'): void,
}>()

const {isActive} = usePomodoro()
const isPomodoroActive = computed(() => isActive.value)
</script>

<style lang="scss" scoped>
.mobile-tab-bar {
	position: fixed;
	inset-inline-start: 0;
	inset-inline-end: 0;
	inset-block-end: 0;
	z-index: 30;

	display: flex;
	align-items: stretch;
	justify-content: space-around;
	min-block-size: var(--mobile-tabbar-height);
	padding-block-end: env(safe-area-inset-bottom);

	background: var(--glass-chrome-bg);
	backdrop-filter: var(--glass-filter);
	box-shadow: var(--glass-specular);
}

.mobile-tab-bar__item {
	flex: 1 1 0;
	min-inline-size: 44px;
	min-block-size: 44px;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: var(--space-1);
	font-size: var(--font-size-xs);
	color: var(--grey-400);

	&.is-active {
		color: var(--primary);
	}
}

.mobile-tab-bar__icon {
	position: relative;
	display: inline-flex;
	// FontAwesome icons size themselves in `em`s off this font-size, inherited
	// through the child <Icon> component regardless of its own scoped styles.
	font-size: 1.25rem;
}

.mobile-tab-bar__live-indicator {
	position: absolute;
	inset-block-start: -2px;
	inset-inline-end: -4px;
	inline-size: 8px;
	block-size: 8px;
	border-radius: 100%;
	background: var(--success);
	box-shadow: 0 0 0 2px var(--glass-chrome-bg);
}

.mobile-tab-bar__capture {
	flex: 0 0 auto;
	align-self: center;
	inline-size: 52px;
	block-size: 52px;
	margin-block-start: -20px;
	border: none;
	border-radius: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.375rem;
	color: var(--white);
	background: var(--primary);
	box-shadow: var(--glass-specular), var(--shadow-md);
}
</style>

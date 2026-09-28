<template>
	<header
		class="mobile-header d-print-none"
		:class="{'is-hidden': isHidden}"
	>
		<div class="mobile-header__title">
			<template v-if="currentProject?.id">
				<ColorBubble
					v-if="projectColor"
					:color="projectColor"
					class="mobile-header__color"
				/>
				<span class="mobile-header__title-text">
					{{ currentProject.title === '' ? $t('misc.loading') : getProjectTitle(currentProject) }}
				</span>
			</template>
			<span
				v-else
				class="mobile-header__title-text"
			>{{ pageTitle }}</span>
		</div>

		<div class="mobile-header__actions">
			<OpenQuickActions />
			<Notifications />
			<BaseButton
				class="mobile-header__avatar"
				:aria-label="$t('mobile.header.openMore')"
				@click="$emit('openMore')"
			>
				<UserAvatar
					:user="authStore.info"
					:size="32"
				/>
			</BaseButton>
		</div>
	</header>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useRoute} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {useScroll, usePreferredReducedMotion} from '@vueuse/core'

import BaseButton from '@/components/base/BaseButton.vue'
import OpenQuickActions from '@/components/misc/OpenQuickActions.vue'
import Notifications from '@/components/notifications/Notifications.vue'
import UserAvatar from '@/components/misc/UserAvatar.vue'
import ColorBubble from '@/components/misc/ColorBubble.vue'

import {getProjectTitle} from '@/helpers/getProjectTitle'
import {getHexColor} from '@/helpers/task'
import {useAuthStore} from '@/stores/auth'
import {useCurrentProject} from '@/composables/useCurrentProject'

defineEmits<{
	(e: 'openMore'): void,
}>()

const authStore = useAuthStore()
const {currentProject} = useCurrentProject()
const projectColor = computed(() => currentProject.value && getHexColor(currentProject.value.hex_color))

// Standalone pages (no project) surface their route's title, same as AppHeader.
const route = useRoute()
const {t} = useI18n()
const pageTitle = computed(() => {
	const title = route.meta.title as string | undefined
	return title ? t(title) : ''
})

// Hide on scroll down, reveal on scroll up; always visible when the user prefers reduced motion.
const {y} = useScroll(window, {throttle: 100})
const reducedMotion = usePreferredReducedMotion()
const isHidden = ref(false)
let lastY = 0
watch(y, newY => {
	const delta = newY - lastY
	lastY = newY

	if (reducedMotion.value === 'reduce' || newY <= 8) {
		isHidden.value = false
		return
	}
	if (delta > 4) {
		isHidden.value = true
	} else if (delta < -4) {
		isHidden.value = false
	}
})
</script>

<style lang="scss" scoped>
.mobile-header {
	position: fixed;
	inset-block-start: 0;
	inset-inline-start: 0;
	inset-inline-end: 0;
	z-index: 30;

	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: var(--space-2);
	min-block-size: var(--mobile-header-height);
	padding-block-start: env(safe-area-inset-top);
	padding-inline: var(--space-4);

	background: var(--glass-chrome-bg);
	backdrop-filter: var(--glass-filter);
	box-shadow: var(--glass-specular);

	transition: transform var(--duration-glass) var(--ease-glass);

	&.is-hidden {
		transform: translateY(-100%);
	}

	@media (prefers-reduced-motion: reduce) {
		transition: none;
		transform: none !important;
	}
}

.mobile-header__title {
	display: flex;
	align-items: center;
	gap: var(--space-2);
	min-inline-size: 0;
}

.mobile-header__title-text {
	font-family: $vikunja-font;
	font-weight: 700;
	font-size: var(--font-size-md);
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.mobile-header__actions {
	flex: 0 0 auto;
	display: flex;
	align-items: center;
	gap: var(--space-1);

	:deep(.trigger-button),
	.mobile-header__avatar {
		min-inline-size: 44px;
		min-block-size: 44px;
		display: flex;
		align-items: center;
		justify-content: center;
	}
}

.mobile-header__avatar :deep(img) {
	border-radius: 100%;
}
</style>

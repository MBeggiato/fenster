<template>
	<div class="content-auth">
		<template v-if="!isMobile">
			<BaseButton
				v-show="menuActive"
				:aria-label="$t('navigation.closeSidebar')"
				class="menu-hide-button d-print-none"
				@click="baseStore.setMenuActive(false)"
			>
				<Icon icon="times" />
			</BaseButton>
			<AppHeader />
		</template>

		<div
			class="app-container"
			:class="{'has-background': background || blurHash, 'is-mobile': isMobile}"
			:style="{'background-image': blurHash && `url(${blurHash})`}"
		>
			<div
				:class="{'is-visible': background}"
				class="app-container-background background-fade-in d-print-none"
				:style="{
					'background-image': background && `url(${background})`,
					'filter': backgroundBrightness && `brightness(${backgroundBrightness}%)`
				}"
			/>
			<Navigation
				v-if="!isMobile"
				class="d-print-none"
			/>
			<MobileHeader
				v-else
				@openMore="moreSheetOpen = true"
			/>
			<main
				id="main-content"
				tabindex="-1"
				class="app-content"
				:class="[
					{ 'is-menu-enabled': menuActive },
					$route.name,
				]"
				:style="{'--sidebar-width': sidebarWidth}"
			>
				<BaseButton
					v-if="!isMobile"
					v-show="menuActive"
					:aria-label="$t('navigation.closeSidebar')"
					class="mobile-overlay d-print-none"
					@click="baseStore.setMenuActive(false)"
				/>

				<QuickActions />

				<RouterView
					v-slot="{ Component }"
					:route="routeWithModal"
				>
					<keep-alive :include="['project.view']">
						<component :is="Component" />
					</keep-alive>
				</RouterView>

				<Modal
					:enabled="typeof currentModal !== 'undefined'"
					variant="scrolling"
					class="task-detail-view-modal"
					:aria-label="$t('task.detail.title')"
					@close="closeModal()"
				>
					<component
						:is="currentModal"
						@close="closeModal()"
					/>
				</Modal>

				<BaseButton
					v-if="!isMobile"
					v-shortcut="SHORTCUTS.showKeyboardShortcuts"
					class="keyboard-shortcuts-button d-print-none"
					@click="showKeyboardShortcuts()"
				>
					<span class="is-sr-only">{{ $t('keyboardShortcuts.title') }}</span>
					<Icon icon="keyboard" />
				</BaseButton>
			</main>

			<MobileTabBar
				v-if="isMobile"
				:active-route-name="routeWithModal.name"
				@capture="captureSheet.open()"
			/>
		</div>

		<MoreSheet
			v-if="isMobile"
			:enabled="moreSheetOpen"
			@close="moreSheetOpen = false"
		/>

		<CaptureSheet v-if="isMobile" />
	</div>
</template>

<script lang="ts" setup>
import {watch, computed, onBeforeUnmount, ref} from 'vue'
import {useRoute, useRouter} from 'vue-router'

import {SHORTCUTS} from '@/constants/shortcuts'
import AppHeader from '@/components/home/AppHeader.vue'
import Navigation from '@/components/home/Navigation.vue'
import MobileHeader from '@/components/home/mobile/MobileHeader.vue'
import MobileTabBar from '@/components/home/mobile/MobileTabBar.vue'
import MoreSheet from '@/components/home/mobile/MoreSheet.vue'
import CaptureSheet from '@/components/home/mobile/CaptureSheet.vue'
import QuickActions from '@/components/quick-actions/QuickActions.vue'
import BaseButton from '@/components/base/BaseButton.vue'

import {useBaseStore} from '@/stores/base'

import {useIsMobile} from '@/composables/useIsMobile'
import {useCaptureSheet} from '@/composables/useCaptureSheet'
import {useRouteWithModal} from '@/composables/useRouteWithModal'
import {useRenewTokenOnFocus} from '@/composables/useRenewTokenOnFocus'
import {useSidebarResize} from '@/composables/useSidebarResize'
import {useWebSocket} from '@/composables/useWebSocket'
import {useServerCacheEvents} from '@/composables/useServerCacheEvents'
import {useAuthStore} from '@/stores/auth'

const authStore = useAuthStore()
const backgroundBrightness = computed(() =>
	authStore.settings?.frontendSettings?.backgroundBrightness,
)

const {sidebarWidth} = useSidebarResize()

const {routeWithModal, currentModal, closeModal} = useRouteWithModal()

const isMobile = useIsMobile()
const moreSheetOpen = ref(false)
const captureSheet = useCaptureSheet()

const baseStore = useBaseStore()
const background = computed(() => baseStore.background)
const blurHash = computed(() => baseStore.blurHash)
const menuActive = computed(() => baseStore.menuActive)

function showKeyboardShortcuts() {
	baseStore.setKeyboardShortcutsActive(true)
}

const route = useRoute()
const router = useRouter()

// FIXME: this is really error prone
// Reset the current project highlight in menu if the current route is not project related.
watch(() => route.name as string, (routeName) => {
	if (
		routeName &&
		(
			[
				'home',
				'teams.index',
				'teams.edit',
				'tasks.range',
				'labels.index',
				'migrate.start',
				'migrate.wunderlist',
				'projects.index',
			].includes(routeName) ||
			routeName.startsWith('user.settings')
		)
	) {
		baseStore.setCurrentProject(null)
	}
})

// The mobile "New task" PWA shortcut and any other deep link land on `?capture=1`;
// open the capture sheet once and strip the query so it doesn't reopen on refresh/back.
watch(() => route.query.capture, (capture) => {
	if (capture !== '1') return
	captureSheet.open()
	const query = {...route.query}
	delete query.capture
	router.replace({query})
}, {immediate: true})

// TODO: Reset the title if the page component does not set one itself

useRenewTokenOnFocus()

useServerCacheEvents()
const {connect} = useWebSocket()
connect()

// Listen for task creation from the quick-entry window
const taskUpdateChannel = new BroadcastChannel('vikunja-task-updates')
taskUpdateChannel.onmessage = (event) => {
	if (event.data?.type === 'task-created-open' && event.data?.taskId) {
		router.push({name: 'task.detail', params: {id: event.data.taskId}})
	}
}

onBeforeUnmount(() => {
	taskUpdateChannel.close()
})
</script>

<style lang="scss" scoped>
.menu-hide-button {
	position: fixed;
	inset-block-start: var(--space-2);
	inset-inline-end: var(--space-2);
	z-index: 31;
	inline-size: 3rem;
	block-size: 3rem;
	display: flex;
	justify-content: center;
	align-items: center;
	font-size: 2rem;
	color: var(--grey-400);
	line-height: 1;
	transition: all $transition;

	@media screen and (min-width: $tablet) {
		display: none;
	}

	&:hover,
	&:focus {
		color: var(--grey-600);
	}
}

.app-container {
	min-block-size: calc(100vh - 65px);

	@media screen and (max-width: $tablet) {
		padding-block-start: $navbar-height;
	}

	&.is-mobile {
		min-block-size: 100dvh;
		padding-block-start: calc(var(--mobile-header-height) + env(safe-area-inset-top));
		padding-block-end: calc(var(--mobile-tabbar-height) + env(safe-area-inset-bottom));
	}
}

.app-content {
	--sidebar-width: #{$navbar-width};

	display: flow-root;
	z-index: 10;
	position: relative;
	padding: var(--space-6) var(--space-2) 0;
	// TODO refactor: DRY `transition-timing-function` with `./Navigation.vue`.
	transition: margin-inline-start $transition-duration;

	@media screen and (max-width: $tablet) {
		margin-inline-start: 0;
		margin-inline-end: 0;
		min-block-size: calc(100vh - 4rem);
	}

	@media screen and (min-width: $tablet) {
		padding: $navbar-height + 1.5rem 1.5rem 0 1.5rem;
	}

	.is-mobile & {
		min-block-size: 0;
		padding: 0 var(--space-2);
	}

	&.is-menu-enabled {
		@media screen and (min-width: $tablet) {
			margin-inline-start: var(--sidebar-width);
		}
	}

	// Used to make sure the spinner is always in the middle while loading
	> .loader-container {
		min-block-size: calc(100vh - #{$navbar-height + 1.5rem + 1rem});
	}

	// FIXME: This should be somehow defined inside Card.vue
	.card {
		background: var(--white);
	}
}

.mobile-overlay {
	display: none;
	position: fixed;
	inset-block-start: 0;
	inset-block-end: 0;
	inset-inline-start: 0;
	inset-inline-end: 0;
	block-size: 100vh;
	inline-size: 100vw;
	background: hsla(var(--grey-100-hsl), 0.8);
	z-index: 5;
	opacity: 0;
	transition: all $transition;

	@media screen and (max-width: $tablet) {
		display: block;
		opacity: 1;
	}
}

.keyboard-shortcuts-button {
	position: fixed;
	inset-block-end: calc(1rem - 4px);
	inset-inline-end: var(--space-4);
	z-index: 4500; // The modal has a z-index of 4000
	color: var(--grey-500);
	transition: color $transition;

	@media screen and (max-width: $tablet) {
		display: none;
	}
}

.content-auth {
	position: relative;
	z-index: 1;
}
</style>

<template>
	<div class="content-widescreen">
		<div class="side-nav-shell">
			<div class="navigation-select select is-fullwidth">
				<select
					:value="activeValue"
					:aria-label="$t('navigation.section')"
					@change="onSelect"
				>
					<option
						v-if="activeValue === ''"
						value=""
						disabled
					>
						{{ $t('navigation.section') }}
					</option>
					<option
						v-for="item in navigationItems"
						:key="item.routeName"
						:value="item.routeName"
					>
						{{ item.title }}
					</option>
					<option
						v-for="({text}, index) in extraLinks"
						:key="`extra-${index}`"
						:value="`extra-${index}`"
					>
						{{ text }} ↗
					</option>
				</select>
			</div>
			<nav class="navigation">
				<ul>
					<li
						v-for="(item, index) in navigationItems"
						:key="`nav-${index}`"
					>
						<RouterLink
							v-slot="{href, navigate, isActive, isExactActive}"
							:to="{name: item.routeName}"
							custom
						>
							<a
								:href="href"
								class="navigation-link"
								:class="{'is-active': (exact ? isExactActive : isActive) || isAliasActive(item)}"
								@click="navigate"
							>
								{{ item.title }}
							</a>
						</RouterLink>
					</li>
					<li
						v-for="({url, text}, index) in extraLinks"
						:key="`extra-${index}`"
					>
						<BaseButton
							class="navigation-link is-flex is-align-items-center"
							:href="url"
						>
							<span>
								{{ text }}
							</span>
							<span class="ml-1 has-text-grey-light is-size-7">
								<Icon
									icon="arrow-up-right-from-square"
								/>
							</span>
						</BaseButton>
					</li>
				</ul>
			</nav>
			<section class="view">
				<RouterView />
			</section>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useRoute, useRouter} from 'vue-router'

import BaseButton from '@/components/base/BaseButton.vue'

export interface SideNavItem {
	title: string
	routeName: string
	activeRouteNames?: string[]
}

export interface SideNavExtraLink {
	url: string
	text: string
}

const props = withDefaults(defineProps<{
	navigationItems: SideNavItem[]
	extraLinks?: SideNavExtraLink[]
	exact?: boolean
}>(), {
	extraLinks: () => [],
	exact: false,
})

const route = useRoute()

const router = useRouter()

function isAliasActive(item: SideNavItem) {
	return item.activeRouteNames?.includes(route.name as string) ?? false
}

// Same rule as the links: exact route, or the item's route anywhere in the matched chain (child pages).
const activeValue = computed(() => props.navigationItems.find(item => isAliasActive(item) || (props.exact
	? route.name === item.routeName
	: route.matched.some(record => record.name === item.routeName)))?.routeName ?? '')

function onSelect(event: Event) {
	const select = event.target as HTMLSelectElement
	const extra = select.value.startsWith('extra-') ? props.extraLinks[Number(select.value.slice(6))] : undefined
	if (extra) {
		window.open(extra.url, '_blank', 'noopener,noreferrer')
		// External links leave the current page open, so the picker keeps showing it.
		select.value = activeValue.value
		return
	}
	router.push({name: select.value})
}
</script>

<style lang="scss" scoped>
.side-nav-shell {
	display: flex;

	@media screen and (max-width: $tablet) {
		flex-direction: column;
	}
}

.navigation {
	inline-size: 25%;
	padding-inline-end: var(--space-4);

	@media screen and (max-width: $tablet) {
		display: none;
	}
}

// Phones get the native picker instead of a long stacked list.
.navigation-select {
	display: none;

	@media screen and (max-width: $tablet) {
		display: block;
	}

	select {
		min-block-size: 44px;
	}
}

.navigation-link {
	display: block;
	padding: var(--space-2);
	color: var(--text);
	inline-size: 100%;
	border-inline-start: 3px solid transparent;

	&:hover,
	&.is-active {
		background: var(--white);
		border-color: var(--primary);
	}
}

.view {
	inline-size: 75%;

	@media screen and (max-width: $tablet) {
		inline-size: 100%;
		padding-inline-start: 0;
		padding-block-start: var(--space-4);
	}
}
</style>

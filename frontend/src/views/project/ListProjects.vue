<template>
	<div
		v-cy="'projects-list'"
		class="content loader-container"
		:class="{'is-loading': loading}"
	>
		<header class="project-header">
			<FancyCheckbox
				v-model="showArchived"
				v-cy="'show-archived-check'"
			>
				{{ $t('project.showArchived') }}
			</FancyCheckbox>

			<div class="action-buttons">
				<XButton
					:to="{name: 'filters.create'}"
					icon="filter"
				>
					{{ $t('filters.create.title') }}
				</XButton>
				<XButton
					v-cy="'new-project'"
					:to="{name: 'project.create'}"
					icon="plus"
				>
					{{ $t('project.create.header') }}
				</XButton>
			</div>
		</header>

		<ProjectCardGrid
			v-if="!isMobile"
			:projects="projects"
			:show-archived="showArchived"
		/>
		<ul
			v-else
			class="project-list-mobile"
		>
			<li
				v-for="project in mobileProjects"
				:key="project.id"
			>
				<RouterLink
					:to="{name: 'project.index', params: {projectId: project.id}}"
					class="project-list-mobile__row"
				>
					<span
						class="project-list-mobile__dot"
						:style="{backgroundColor: project.hex_color || undefined}"
					/>
					<span class="project-list-mobile__title">{{ getProjectTitle(project) }}</span>
					<Icon
						v-if="project.is_favorite"
						icon="star"
						aria-hidden="true"
						class="project-list-mobile__favorite"
					/>
					<span
						v-if="project.is_archived"
						class="is-archived"
					>{{ $t('project.archived') }}</span>
				</RouterLink>
			</li>
		</ul>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'

import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import Icon from '@/components/misc/Icon'
import ProjectCardGrid from '@/components/project/partials/ProjectCardGrid.vue'

import {useTitle} from '@/composables/useTitle'
import {useStorage} from '@vueuse/core'

import {useProjects} from '@/composables/useProjects'
import {useIsMobile} from '@/composables/useIsMobile'
import {getProjectTitle} from '@/helpers/getProjectTitle'

const {t} = useI18n()
const projectList = useProjects()
const isMobile = useIsMobile()

useTitle(() => t('project.title'))
const showArchived = useStorage('showArchived', false)

const loading = computed(() => projectList.isLoading)
const projects = computed(() => {
	return showArchived.value
		? projectList.projectsArray
		: projectList.projectsArray.filter(({is_archived}) => !is_archived)
})

// Favourites (incl. the pseudo "Favorites" filter) first, then the rest in their existing order.
const mobileProjects = computed(() => {
	const favourites = projectList.favoriteProjects.filter(({is_archived}) => showArchived.value || !is_archived)
	const favouriteIds = new Set(favourites.map(({id}) => id))
	return [...favourites, ...projects.value.filter(({id}) => !favouriteIds.has(id))]
})
</script>

<style lang="scss" scoped>
.project-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: var(--space-4);
	margin-block-end: var(--space-4);

	@media screen and (max-width: $tablet) {
		flex-direction: column;
	}
}

.action-buttons {
	display: flex;
	justify-content: space-between;
	gap: var(--space-4);

	@media screen and (max-width: $tablet) {
		inline-size: 100%;
		flex-direction: column;
		align-items: stretch;
	}
}

.project:not(:first-child) {
	margin-block-start: var(--space-4);
}

.project-title {
	display: flex;
	align-items: center;
}

.is-archived {
	font-size: var(--font-size-xs);
	border: 1px solid var(--grey-500);
	color: $grey !important;
	padding: 2px 4px;
	border-radius: 3px;
	font-family: $vikunja-font;
	background: var(--white-translucent);
	margin-inline-start: var(--space-2);
}

.project-list-mobile {
	list-style: none;
	margin: 0;
	padding: 0;
}

.project-list-mobile__row {
	display: flex;
	align-items: center;
	gap: var(--space-3);
	min-block-size: 44px;
	padding: var(--space-2) 0;
	color: var(--text);
	border-block-end: 1px solid var(--card-border-color);
}

.project-list-mobile__dot {
	flex: 0 0 auto;
	inline-size: 0.75rem;
	block-size: 0.75rem;
	border-radius: 100%;
	background: var(--grey-300);
}

.project-list-mobile__title {
	flex: 1 1 auto;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.project-list-mobile__favorite {
	flex: 0 0 auto;
	color: var(--warning);
}
</style>

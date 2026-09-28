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
			:projects="projects"
			:show-archived="showArchived"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'

import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import ProjectCardGrid from '@/components/project/partials/ProjectCardGrid.vue'

import {useTitle} from '@/composables/useTitle'
import {useStorage} from '@vueuse/core'

import {useProjects} from '@/composables/useProjects'

const {t} = useI18n()
const projectList = useProjects()

useTitle(() => t('project.title'))
const showArchived = useStorage('showArchived', false)

const loading = computed(() => projectList.isLoading)
const projects = computed(() => {
	return showArchived.value
		? projectList.projectsArray
		: projectList.projectsArray.filter(({is_archived}) => !is_archived)
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
</style>

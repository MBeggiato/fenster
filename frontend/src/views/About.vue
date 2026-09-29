<template>
	<Modal
		variant="hint-modal"
		@close="$router.back()"
	>
		<Card
			class="has-no-shadow"
			:title="$t('about.title')"
			:padding="false"
			:show-close="true"
			@close="$router.back()"
		>
			<div class="p-4">
				<p v-if="versionsEqual">
					{{ $t('about.version', {version: apiVersion}) }}
				</p>
				<template v-else>
					<p>{{ $t('about.frontendVersion', {version: frontendVersion}) }}</p>
					<p>{{ $t('about.apiVersion', {version: apiVersion}) }}</p>
				</template>
				<p v-if="proActive">
					{{ $t('about.proActive') }}
				</p>
				<p>
					<a
						:href="LICENSE_URL"
						target="_blank"
						rel="noopener noreferrer"
					>{{ $t('about.license') }}</a>
				</p>
				<p>
					<a
						:href="SOURCE_CODE"
						target="_blank"
						rel="noopener noreferrer"
					>{{ $t('about.source') }}</a>
					·
					<a
						:href="UPSTREAM"
						target="_blank"
						rel="noopener noreferrer"
					>{{ $t('about.basedOn') }}</a>
				</p>
				<p>{{ $t('about.notAffiliated') }}</p>
			</div>
			<template #footer>
				<XButton
					variant="secondary"
					@click.prevent.stop="$router.back()"
				>
					{{ $t('misc.close') }}
				</XButton>
			</template>
		</Card>
	</Modal>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import {VERSION as frontendVersion} from '@/version.json'

import {useConfigStore} from '@/stores/config'
import {SOURCE_CODE, UPSTREAM, LICENSE_URL} from '@/urls'

const configStore = useConfigStore()
const apiVersion = computed(() => configStore.version)
const versionsEqual = computed(() => apiVersion.value === frontendVersion)
const proActive = computed(() => configStore.enabledProFeatures.length > 0)
</script>

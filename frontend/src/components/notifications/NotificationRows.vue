<template>
	<div>
		<div
			v-for="(n, index) in notifications"
			:key="n.id"
			class="single-notification"
			:class="{'is-clickable': notificationHasRoute(n), 'is-sheet': isSheet}"
			@click="() => notificationHasRoute(n) && $emit('open', n, index)"
		>
			<div
				class="read-indicator"
				:class="{'read': n.readAt !== null}"
			/>
			<User
				v-if="n.notification.doer"
				:user="n.notification.doer"
				:show-username="false"
				:avatar-size="16"
			/>
			<div class="detail">
				<div>
					<span
						v-if="n.notification.doer"
						class="has-text-weight-bold mie-1"
					>
						{{ getDisplayName(n.notification.doer) }}
					</span>
					{{ n.toText(userInfo) }}
				</div>
				<span
					v-tooltip="formatDateLong(n.created)"
					class="created"
				>
					{{ formatDisplayDate(n.created) }}
				</span>
			</div>
		</div>
		<XButton
			v-if="notifications.length > 0 && unreadNotifications > 0"
			variant="tertiary"
			class="mbs-2 is-fullwidth"
			@click="$emit('markAllRead')"
		>
			{{ $t('notification.markAllRead') }}
		</XButton>
		<EmptyState
			v-if="notifications.length === 0"
			:icon="['far', 'bell-slash']"
			:title="$t('notification.none')"
			:text="$t('notification.explainer')"
		/>
	</div>
</template>

<script lang="ts" setup>
import User from '@/components/misc/User.vue'
import EmptyState from '@/components/misc/EmptyState.vue'
import {NOTIFICATION_NAMES as names, type INotification} from '@/modelTypes/INotification'
import {formatDateLong, formatDisplayDate} from '@/helpers/time/formatDate'
import {getDisplayName} from '@/models/user'
import XButton from '@/components/input/Button.vue'
import type NotificationModel from '@/models/notification'
import type {IUser} from '@/modelTypes/IUser'

interface Props {
	notifications: NotificationModel[]
	unreadNotifications: number
	userInfo: Pick<IUser, 'id'> | null
	isSheet?: boolean
}

defineProps<Props>()

defineEmits<{
	open: [notification: INotification, index: number]
	markAllRead: []
}>()

function getNotificationRoute(n: INotification): import('vue-router').RouteLocationRaw | null {
	switch (n.name) {
		case names.TASK_COMMENT:
		case names.TASK_ASSIGNED:
		case names.TASK_REMINDER:
		case names.TASK_MENTIONED:
		case names.TASK_CREATED:
			return {name: 'task.detail', params: {id: (n.notification as {task: {id: number}}).task.id}}
		case names.PROJECT_CREATED:
			return {name: 'task.index', params: {projectId: (n.notification as {project: {id: number}}).project.id}}
		case names.TEAM_MEMBER_ADDED:
			return {name: 'teams.edit', params: {id: (n.notification as {team: {id: number}}).team.id}}
		default:
			return null
	}
}

function notificationHasRoute(n: INotification): boolean {
	return getNotificationRoute(n) !== null
}
</script>

<style lang="scss" scoped>
.single-notification {
	display: flex;
	align-items: center;
	padding: var(--space-1) 0;

	transition: background-color $transition;

	&.is-clickable {
		cursor: pointer;
	}

	&:hover {
		background: var(--grey-100);
		border-radius: $radius;
	}

	.read-indicator {
		inline-size: .35rem;
		block-size: .35rem;
		background: var(--primary);
		border-radius: 100%;
		margin: 0 var(--space-2);
		flex-shrink: 0;

		&.read {
			background: transparent;
		}
	}

	.user {
		display: inline-flex;
		align-items: center;
		inline-size: auto;
		margin: 0 var(--space-2);

		span {
			font-family: $family-sans-serif;
		}

		.avatar {
			block-size: 16px;
		}

		img {
			margin-inline-end: 0;
		}
	}

	.detail {
		flex: 1;

		.created {
			color: var(--grey-400);
		}
	}

	&:last-child {
		margin-block-end: var(--space-1);
	}

	a {
		color: var(--grey-800);
	}

	// Sheet variant: Modal owns the glass panel + scroll, this just needs 44px
	// tap targets for the list rows (same tokens Dropdown.vue's sheet uses).
	&.is-sheet {
		min-block-size: 44px;
	}
}
</style>

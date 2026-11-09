<template>
  <ContextMenu>
    <ContextMenuTrigger asChild>
      <router-link
        :to="conversationRoute"
        class="conversation-card group relative block min-w-0 transition-colors duration-150 ease-in-out cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
        data-conversation-card
        :class="[
          {
            'bg-primary/15 dark:bg-primary/20 before:absolute before:inset-y-0 before:left-0 before:w-0.5 before:bg-primary':
              isCurrent,
            'bg-primary/5 hover:bg-primary/10': isItemSelected && !isCurrent,
            'hover:bg-accent/40': !isCurrent && !isItemSelected
          }
        ]"
      >
        <div class="card-body flex min-w-0 items-start gap-2 px-3 py-2.5">
          <!-- Avatar with channel indicator (checkbox overlays on hover / when selecting) -->
          <div class="card-avatar relative h-8 w-8 shrink-0">
            <div
              class="h-full w-full transition-opacity"
              :class="avatarOpacityClass"
              :aria-hidden="showCheckbox"
            >
              <Avatar class="h-full w-full rounded-full">
                <AvatarImage :src="conversation.contact.avatar_url || ''" class="object-cover" />
                <AvatarFallback>
                  {{ conversation.contact.first_name.substring(0, 2).toUpperCase() }}
                </AvatarFallback>
              </Avatar>
            </div>
            <div
              v-if="canBulkAct"
              class="absolute inset-0 items-center justify-center"
              :class="showCheckbox ? 'flex' : 'hidden can-hover:group-hover:flex'"
              @click.prevent.stop="handleCheckboxClick"
            >
              <Checkbox
                :checked="isItemSelected"
                :aria-label="t('conversation.bulkActions.selectConversation')"
                class="w-5 h-5"
              />
            </div>
          </div>

          <div class="card-content min-w-0 flex-1 space-y-2">
            <div class="card-heading-content grid min-w-0 items-start gap-x-2">
              <Tooltip v-if="showSubject && conversation.subject">
                <TooltipTrigger asChild>
                  <h3
                    class="card-title min-w-0 truncate text-[13px] leading-5 text-foreground"
                    :class="isUnread ? 'font-bold' : 'font-semibold'"
                    data-conversation-title
                  >
                    {{ conversation.subject }}
                  </h3>
                </TooltipTrigger>
                <TooltipContent>{{ conversation.subject }}</TooltipContent>
              </Tooltip>
              <h3
                v-else
                class="card-title min-w-0 truncate text-[13px] leading-5 text-foreground"
                :class="isUnread ? 'font-bold' : 'font-semibold'"
                :title="contactFullName"
                data-conversation-title
              >
                {{ contactFullName }}
              </h3>

              <p
                v-if="showSubject && conversation.subject"
                class="card-contact min-w-0 truncate text-xs font-normal leading-4 text-muted-foreground"
                :title="contactFullName"
              >
                {{ contactFullName }}
              </p>

              <div class="card-metadata flex min-w-0 items-center justify-end gap-1">
                <PriorityMarker :priority="conversation.priority" />
                <Tooltip>
                  <TooltipTrigger asChild>
                    <component
                      :is="conversation.inbox_channel === 'livechat' ? MessageSquare : Mail"
                      class="h-3 w-3 shrink-0 text-muted-foreground"
                      role="img"
                      :aria-label="conversation.inbox_name"
                    />
                  </TooltipTrigger>
                  <TooltipContent>{{ conversation.inbox_name }}</TooltipContent>
                </Tooltip>
                <div
                  v-if="isUnread"
                  data-conversation-unread
                  class="flex h-5 min-w-5 shrink-0 whitespace-nowrap px-1 tabular-nums items-center justify-center rounded-full bg-primary text-xs font-medium text-primary-foreground"
                >
                  {{ conversation.unread_message_count }}
                </div>
              </div>
            </div>

            <div class="flex min-w-0 items-center gap-2" data-conversation-preview>
              <Reply
                v-if="
                  !isTyping &&
                  !(hasDraftForConversation && !isCurrent) &&
                  conversation.last_message_sender === 'agent'
                "
                class="shrink-0 text-success"
                :size="14"
              />
              <p
                class="min-w-0 flex-1 truncate text-xs font-normal leading-4"
                :class="isUnread ? 'text-foreground' : 'text-muted-foreground'"
              >
                <span v-if="isTyping" class="italic text-foreground">{{
                  $t('globals.terms.typing')
                }}</span>
                <template v-else-if="hasDraftForConversation && !isCurrent">
                  <span class="font-medium text-foreground">{{ $t('globals.terms.draft') }}:</span>
                  {{ draftPreview }}
                </template>
                <template v-else>
                  <span
                    v-if="lastMessageTime"
                    data-conversation-time
                    class="text-muted-foreground/80 tabular-nums"
                    >{{ lastMessageTime }} · </span
                  ><span>{{ conversation.last_message }}</span>
                </template>
              </p>
            </div>

            <div class="card-footer flex min-w-0 items-center">
              <span
                class="card-status inline-flex h-6 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 text-[11px] font-medium leading-4 transition-colors"
                :class="statusClass"
                data-conversation-status
              >
                <span class="h-1.5 w-1.5 shrink-0 rounded-full bg-current" aria-hidden="true" />
                {{ conversation.status }}
              </span>
              <ConversationTagOverflow
                v-if="conversation.tags?.length"
                :tags="conversation.tags"
                separator
                class="flex-1"
              />
            </div>

            <div v-if="hasSlaDeadlines" class="flex min-w-0 flex-wrap items-center gap-1">
              <SlaBadge
                v-show="frdStatus === 'overdue' || frdStatus === 'remaining'"
                :dueAt="conversation.first_response_deadline_at"
                :actualAt="conversation.first_reply_at"
                :label="'FRD'"
                :showExtra="false"
                @status="frdStatus = $event"
              />
              <SlaBadge
                v-show="rdStatus === 'overdue' || rdStatus === 'remaining'"
                :dueAt="conversation.resolution_deadline_at"
                :actualAt="conversation.resolved_at"
                :label="'RD'"
                :showExtra="false"
                @status="rdStatus = $event"
              />
              <SlaBadge
                v-show="nrdStatus === 'overdue' || nrdStatus === 'remaining'"
                :dueAt="conversation.next_response_deadline_at"
                :actualAt="conversation.next_response_met_at"
                :label="'NRD'"
                :showExtra="false"
                @status="nrdStatus = $event"
              />
            </div>
          </div>
        </div>
      </router-link>
    </ContextMenuTrigger>
    <ContextMenuContent>
      <!-- Long press is the only way to reach the first checkbox on touch. -->
      <ContextMenuItem v-if="canBulkAct && !showCheckbox" @click="handleSelect">
        <SquareCheck class="w-4 h-4 mr-2" />
        {{ $t('conversation.bulkActions.selectConversation') }}
      </ContextMenuItem>
      <ContextMenuItem @click="handleMarkAsUnread">
        <MailOpen class="w-4 h-4 mr-2" />
        {{ $t('globals.messages.markAsUnread') }}
      </ContextMenuItem>
    </ContextMenuContent>
  </ContextMenu>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { formatFullTimestamp } from '@shared-ui/utils/datetime.js'
import { Mail, MessageSquare, Reply, MailOpen, SquareCheck } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger
} from '@shared-ui/components/ui/context-menu'
import SlaBadge from '@main/features/sla/SlaBadge.vue'
import PriorityMarker from '@main/features/conversation/PriorityMarker.vue'
import ConversationTagOverflow from './ConversationTagOverflow.vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { useConversationStore } from '@main/stores/conversation'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const conversationStore = useConversationStore()
const appSettingsStore = useAppSettingsStore()
const { canBulkAct } = useBulkActionPermissions()
const { t } = useI18n()
const frdStatus = ref('')
const rdStatus = ref('')
const nrdStatus = ref('')

const props = defineProps({
  conversation: Object,
  currentConversation: Object,
  contactFullName: String
})

const handleMarkAsUnread = () => {
  conversationStore.markAsUnread(props.conversation.uuid)
}

const conversationRoute = computed(() => {
  const baseRoute = route.params.inboxID
    ? 'personal-inbox-conversation'
    : route.params.teamID
      ? 'team-inbox-conversation'
      : route.params.viewID
        ? 'view-inbox-conversation'
        : 'inbox-conversation'
  return {
    name: baseRoute,
    params: {
      uuid: props.conversation.uuid,
      ...(baseRoute === 'personal-inbox-conversation' && { inboxID: route.params.inboxID }),
      ...(baseRoute === 'team-inbox-conversation' && { teamID: route.params.teamID }),
      ...(baseRoute === 'view-inbox-conversation' && { viewID: route.params.viewID })
    },
    query: {
      ...route.query,
      ...(props.conversation.mentioned_message_uuid && {
        scrollTo: props.conversation.mentioned_message_uuid
      })
    }
  }
})

const lastMessageTime = computed(() => formatFullTimestamp(props.conversation.last_message_at))

const hasSlaDeadlines = computed(() => {
  const c = props.conversation
  return c.first_response_deadline_at || c.resolution_deadline_at || c.next_response_deadline_at
})

const hasDraftForConversation = computed(() => {
  return conversationStore.conversationHasDraft(props.conversation.uuid)
})

const isTyping = computed(() => conversationStore.typingByUUID[props.conversation.uuid] === true)

const draftPreview = computed(() => {
  const draft = conversationStore.conversationDraftPreview(props.conversation.uuid)
  if (!draft?.content && !draft?.meta?.attachments?.length) return ''
  const text = (draft.content || '').replace(/<[^>]*>/g, '').trim()
  if (text) return text.length > 120 ? text.slice(0, 120) + '...' : text
  if (draft.meta?.attachments?.length)
    return conversationStore.getMediaPreview(draft.meta.attachments)
  if (/<img\b/i.test(draft.content || '')) return t('globals.terms.image', 1)
  return ''
})

const showSubject = computed(
  () => appSettingsStore.settings['app.show_conversation_subject'] !== false
)

const isUnread = computed(() => props.conversation.unread_message_count > 0)

const isCurrent = computed(() => props.conversation.uuid === props.currentConversation?.uuid)

const statusClass = computed(() => {
  switch (props.conversation.status) {
    case 'Open':
      return [
        'border-red-500/20',
        'bg-red-500/10',
        'text-red-700',
        'dark:border-red-400/20',
        'dark:bg-red-500/20',
        'dark:text-red-300'
      ]

    case 'Resolved':
      return [
        'border-green-500/20',
        'bg-green-500/10',
        'text-green-700',
        'dark:border-green-400/20',
        'dark:bg-green-500/20',
        'dark:text-green-300'
      ]

    case 'Snoozed':
      return [
        'border-blue-500/20',
        'bg-blue-500/10',
        'text-blue-700',
        'dark:border-blue-400/20',
        'dark:bg-blue-500/20',
        'dark:text-blue-300'
      ]

    case 'Closed':
      return [
        'border-violet-500/20',
        'bg-violet-500/10',
        'text-violet-700',
        'dark:border-violet-400/20',
        'dark:bg-violet-500/20',
        'dark:text-violet-300'
      ]

    default:
      return ['border-border', 'bg-muted', 'text-muted-foreground']
  }
})

const isItemSelected = computed(() => {
  return conversationStore.isSelected(props.conversation.uuid)
})

const showCheckbox = computed(() => {
  if (!canBulkAct.value) return false
  return isItemSelected.value || conversationStore.selectedCount > 0
})

const avatarOpacityClass = computed(() => {
  if (showCheckbox.value) return 'opacity-0'
  if (canBulkAct.value) return 'opacity-100 can-hover:group-hover:opacity-0'
  return 'opacity-100'
})

const handleCheckboxClick = (event) => {
  conversationStore.toggleSelect(props.conversation.uuid, event.shiftKey)
}

const handleSelect = () => {
  conversationStore.toggleSelect(props.conversation.uuid, false)
}
</script>

<style scoped>
.conversation-card {
  container: conversation-card / inline-size;
}

.card-heading-content {
  grid-template-columns: minmax(0, 1fr) auto;
}

.card-title {
  grid-column: 1;
  grid-row: 1;
}
.card-contact {
  grid-column: 1;
  grid-row: 2;
}
.card-metadata {
  grid-column: 2;
  grid-row: 1 / span 2;
}

@container conversation-card (max-width: 260px) {
  .card-heading-content {
    grid-template-columns: minmax(0, 1fr);
  }
  .card-metadata {
    grid-column: 1;
    grid-row: 3;
    justify-content: flex-end;
    flex-wrap: wrap;
    margin-top: 4px;
  }
}

@container conversation-card (max-width: 200px) {
  .card-body {
    padding-inline: 8px;
  }
  .card-avatar {
    width: 24px;
    height: 24px;
  }
}
@container conversation-card (max-width: 120px) {
  .card-body {
    padding-inline: 4px;
    gap: 4px;
  }
  .card-avatar {
    width: 12px;
    height: 12px;
  }
}
</style>

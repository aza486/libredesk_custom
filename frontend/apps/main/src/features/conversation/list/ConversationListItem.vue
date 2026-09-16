<template>
  <ContextMenu>
    <ContextMenuTrigger asChild>
      <router-link
        :to="conversationRoute"
        class="group relative block px-3 py-2.5 transition-colors duration-150 ease-in-out cursor-pointer"
        :class="[
          tagHighlightClass,
          {
            'bg-accent': isCurrent,
            'bg-primary/5 hover:bg-primary/10': isItemSelected && !isCurrent,
            'hover:bg-accent/40': !isCurrent && !isItemSelected
          }
        ]"
      >
        <div class="flex items-start gap-2">
          <!-- Avatar with channel indicator (checkbox overlays on hover / when selecting) -->
          <div class="relative flex-shrink-0 w-10 h-10">
            <div class="transition-opacity" :class="avatarOpacityClass" :aria-hidden="showCheckbox">
              <Avatar class="w-10 h-10 rounded-full">
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

          <!-- Content container -->
          <div class="flex-1 min-w-0 space-y-1.5">
            <!-- Name + Subject group -->
            <div class="space-y-0.5">
              <!-- Contact name + channel + time -->
              <div class="flex items-baseline justify-between gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <h3
                      class="text-sm truncate min-w-0 text-foreground"
                      :class="isUnread ? 'font-semibold' : 'font-medium'"
                    >
                      {{ contactFullName }}
                    </h3>
                  </TooltipTrigger>
                  <TooltipContent>{{ contactFullName }}</TooltipContent>
                </Tooltip>
                <div class="flex items-center gap-1 flex-shrink-0">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <component
                        :is="conversation.inbox_channel === 'livechat' ? MessageSquare : Mail"
                        class="w-3 h-3 text-muted-foreground"
                        role="img"
                        :aria-label="conversation.inbox_name"
                      />
                    </TooltipTrigger>
                    <TooltipContent>{{ conversation.inbox_name }}</TooltipContent>
                  </Tooltip>
                  <span
                    class="text-xs text-muted-foreground whitespace-nowrap tabular-nums"
                    v-if="conversation.last_message_at"
                  >
                    {{ relativeLastMessageTime }}
                  </span>
                </div>
              </div>

              <!-- Subject -->
              <p
                v-if="showSubject && conversation.subject"
                class="text-xs text-muted-foreground truncate"
              >
                {{ conversation.subject }}
              </p>
            </div>

            <!-- Message preview + unread count -->
            <div class="flex items-center justify-between gap-2">
              <p
                class="text-sm flex-1 min-w-0 truncate"
                :class="isUnread ? 'text-foreground font-medium' : 'text-muted-foreground'"
              >
                <template v-if="isTyping">
                  <span class="italic text-foreground">{{ $t('globals.terms.typing') }}</span>
                </template>
                <template v-else-if="hasDraftForConversation && !isCurrent">
                  <span class="font-medium text-foreground">{{ $t('globals.terms.draft') }}:</span>
                  {{ draftPreview }}
                </template>
                <template v-else>
                  <Reply
                    class="text-success inline-block align-text-bottom mr-0.5"
                    :size="14"
                    v-if="conversation.last_message_sender === 'agent'"
                  />{{ trimmedLastMessage }}
                </template>
              </p>
              <div
                v-if="isUnread"
                class="flex items-center justify-center w-5 h-5 bg-primary text-primary-foreground text-xs font-medium rounded-full flex-shrink-0"
              >
                {{
                  conversation.unread_message_count > 9 ? '9+' : conversation.unread_message_count
                }}
              </div>
            </div>

            <!-- SLA Badges -->
            <div
              v-if="conversation.tags?.length"
              class="flex min-w-0 items-center gap-1 overflow-hidden"
            >
              <span
                v-for="tag in visibleTags"
                :key="tag"
                class="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground"
              >
                {{ tag }}
              </span>
              <span v-if="hiddenTagCount" class="shrink-0 text-xs text-muted-foreground">…</span>
            </div>

            <div class="flex items-center">
              <span
                class="inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium leading-4 transition-colors"
                :class="statusClass"
              >
                {{ conversation.status }}
              </span>
            </div>

            <div v-if="hasSlaDeadlines" class="flex items-center gap-1">
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { Mail, MessageSquare, Reply, MailOpen, SquareCheck } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger
} from '@shared-ui/components/ui/context-menu'
import SlaBadge from '@main/features/sla/SlaBadge.vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { useConversationStore } from '@main/stores/conversation'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import { useI18n } from 'vue-i18n'

let timer = null
const now = ref(new Date())
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
  const baseRoute = route.params.teamID
    ? 'team-inbox-conversation'
    : route.params.viewID
      ? 'view-inbox-conversation'
      : 'inbox-conversation'
  return {
    name: baseRoute,
    params: {
      uuid: props.conversation.uuid,
      ...(baseRoute === 'team-inbox-conversation' && { teamID: route.params.teamID }),
      ...(baseRoute === 'view-inbox-conversation' && { viewID: route.params.viewID })
    },
    query: props.conversation.mentioned_message_uuid
      ? { scrollTo: props.conversation.mentioned_message_uuid }
      : {}
  }
})

onMounted(() => {
  timer = setInterval(() => {
    now.value = new Date()
  }, 60000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const trimmedLastMessage = computed(() => {
  const message = props.conversation.last_message || ''
  return message.length > 120 ? message.slice(0, 120) + '...' : message
})

const relativeLastMessageTime = computed(() => {
  return props.conversation.last_message_at
    ? getRelativeTime(props.conversation.last_message_at, now.value)
    : ''
})

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

const visibleTags = computed(() => (props.conversation.tags || []).slice(0, 3))
const hiddenTagCount = computed(() =>
  Math.max(0, (props.conversation.tags || []).length - visibleTags.value.length)
)
const tagHighlightClass = computed(() => {
  const tags = (props.conversation.tags || []).map((tag) =>
    String(tag).replace(/\s+/g, '').toLocaleLowerCase()
  )
  if (tags.includes('🛑verarbeitungsfehler')) return 'ticket-highlight-processing-error'
  if (tags.includes('😎menscherforderlich')) return 'ticket-highlight-human-required'
  return ''
})

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
      return [
        'border-border',
        'bg-muted',
        'text-muted-foreground'
      ]
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
.ticket-highlight-processing-error {
  @apply bg-red-100 text-red-950 hover:bg-red-200 dark:bg-red-950/70 dark:text-red-100 dark:hover:bg-red-950;
}

.ticket-highlight-processing-error :deep(.text-foreground) {
  @apply text-red-950 dark:text-red-100;
}

.ticket-highlight-processing-error :deep(.text-muted-foreground) {
  @apply text-red-800 dark:text-red-200;
}

.ticket-highlight-human-required {
  @apply bg-amber-100 text-amber-950 hover:bg-amber-200 dark:bg-amber-950/70 dark:text-amber-100 dark:hover:bg-amber-950;
}

.ticket-highlight-human-required :deep(.text-foreground) {
  @apply text-amber-950 dark:text-amber-100;
}

.ticket-highlight-human-required :deep(.text-muted-foreground) {
  @apply text-amber-800 dark:text-amber-200;
}
</style>

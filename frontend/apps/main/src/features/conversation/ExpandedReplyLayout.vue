<template>
  <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
    <header class="max-h-[30%] shrink-0 overflow-y-auto border-b">
      <div class="flex items-start gap-2 p-2 md:p-4">
        <div class="flex min-w-0 flex-1 flex-wrap items-center gap-x-4 gap-y-2 text-sm">
          <DropdownMenu v-if="conversation?.status">
            <DropdownMenuTrigger as-child>
              <Button class="max-w-full" size="sm">
                <span class="truncate">{{ conversationStatusLabel(conversation.status, t) }}</span>
                <ChevronDown class="ml-2 h-4 w-4 shrink-0" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem
                v-for="status in conversationStore.statusOptions"
                :key="status.value"
                @click="updateStatus(status.label)"
                >{{ status.displayLabel ?? status.label }}</DropdownMenuItem
              >
            </DropdownMenuContent>
          </DropdownMenu>
          <span v-if="conversation?.priority" class="flex min-w-0 items-center gap-2">
            <PriorityMarker :priority="conversation.priority" :size="16" />
            <span class="truncate">{{ conversationPriorityLabel(conversation.priority, t) }}</span>
          </span>
          <span
            class="flex min-w-0 items-center gap-2"
            :title="conversationStore.currentContactName"
          >
            <UserRound class="h-4 w-4 shrink-0" />
            <span class="truncate">{{ conversationStore.currentContactName }}</span>
          </span>
          <span
            v-if="conversation?.contact?.email"
            class="flex min-w-0 items-center gap-2"
            :title="conversation.contact.email"
          >
            <Mail class="h-4 w-4 shrink-0" />
            <span class="truncate">{{ conversation.contact.email }}</span>
          </span>
        </div>
        <Button
          variant="ghost"
          class="h-11 w-11 shrink-0 p-0 md:h-8 md:w-8"
          :aria-label="t('replyBox.restoreEditor')"
          @click="emit('restore')"
        >
          <Minimize2 class="h-4 w-4" />
        </Button>
      </div>
      <div class="w-full min-w-0 border-t px-2 py-2 md:px-4">
        <p class="truncate text-sm" :title="conversation?.subject">
          <span class="font-semibold">{{ t('globals.terms.subject') }}:</span>
          {{ conversation?.subject }}
        </p>
      </div>
    </header>

    <Tabs v-model="activePane" class="flex min-h-0 flex-1 flex-col">
      <TabsList
        class="mx-2 my-1 grid h-auto shrink-0 grid-cols-2 md:hidden"
        :aria-label="t('replyBox.expandedEditor')"
      >
        <TabsTrigger value="history" class="min-h-11">{{ t('replyBox.history') }}</TabsTrigger>
        <TabsTrigger value="reply" class="min-h-11">{{ t('globals.terms.reply') }}</TabsTrigger>
      </TabsList>
      <div class="grid min-h-0 min-w-0 flex-1 grid-cols-1 overflow-hidden md:grid-cols-2">
        <TabsContent
          value="history"
          force-mount
          :hidden="isMobile && activePane !== 'history'"
          v-show="!isMobile || activePane === 'history'"
          class="m-0 block min-h-0 min-w-0 overflow-hidden md:border-r"
        >
          <div
            v-show="!contextActive"
            :ref="setHistoryTarget"
            class="flex h-full min-h-0 min-w-0 flex-col overflow-hidden"
          />
          <div
            v-show="contextActive"
            :ref="setContextTarget"
            class="h-full min-h-0 min-w-0 overflow-hidden"
          />
        </TabsContent>
        <TabsContent
          value="reply"
          force-mount
          :hidden="isMobile && activePane !== 'reply'"
          v-show="!isMobile || activePane === 'reply'"
          class="m-0 flex min-h-0 min-w-0 flex-col overflow-hidden p-2 md:p-4"
          :class="{ 'bg-private': privateNote }"
        >
          <slot />
        </TabsContent>
      </div>
    </Tabs>
  </div>
</template>

<script setup>
import { conversationPriorityLabel } from '@/utils/conversation-priority'
import { conversationStatusLabel } from '@/utils/conversation-status'
import { computed, ref } from 'vue'
import { ChevronDown, Mail, Minimize2, UserRound } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useIsMobile } from '@shared-ui/composables'
import { Button } from '@shared-ui/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { CONVERSATION_DEFAULT_STATUSES } from '@main/constants/conversation'
import PriorityMarker from './PriorityMarker.vue'

defineProps({ privateNote: Boolean, contextActive: Boolean })
const emit = defineEmits(['restore', 'historyTarget', 'contextTarget'])
const { t } = useI18n()
const conversationStore = useConversationStore()
const conversation = computed(() => conversationStore.current)
const emitter = useEmitter()
const isMobile = useIsMobile()
const activePane = ref('reply')
defineExpose({
  showReply: () => {
    activePane.value = 'reply'
  },
  showHistory: () => {
    activePane.value = 'history'
  }
})
const setHistoryTarget = (element) => emit('historyTarget', element)
const setContextTarget = (element) => emit('contextTarget', element)
const updateStatus = (status) => {
  if (status === CONVERSATION_DEFAULT_STATUSES.SNOOZED) {
    emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, { command: 'snooze', open: true })
    return
  }
  conversationStore.updateStatus(status)
}
</script>

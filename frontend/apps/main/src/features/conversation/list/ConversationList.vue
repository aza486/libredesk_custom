<template>
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="flex items-center space-x-4 px-2 h-12 border-b shrink-0">
      <SidebarTrigger class="cursor-pointer" />
      <span class="text-xl font-semibold">{{ title }}</span>
    </div>

    <!-- Bulk Action Toolbar (when items selected) -->
    <ConversationBulkActionToolbar v-if="hasSelection && canBulkAct" />

    <!-- Filters (hidden when bulk selecting) -->
    <div v-else class="p-2 flex justify-between items-center">
      <!-- Status dropdown-menu, hidden when a view is selected as views are pre-filtered -->
      <DropdownMenu v-if="!route.params.viewID && !route.query.folder && !['customer-redistributed', 'customer-redistributed-unfiled'].includes(route.params.type)">
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" class="w-30">
            <div>
              <span class="mr-1">{{ conversationStore.conversations.total }}</span>
              <span>{{ conversationStatusLabel(conversationStore.getListStatus, t) }}</span>
            </div>
            <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem
            v-for="status in conversationStore.statusOptions"
            :key="status.value"
            @click="handleStatusChange(status)"
          >
            {{ status.displayLabel ?? status.label }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <div v-else>
        <Button variant="ghost" class="w-30">
          <span>{{ conversationStore.conversations.total }}</span>
        </Button>
      </div>

      <!-- Sort dropdown-menu -->
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" class="w-30">
            {{ conversationStore.getListSortField }}
            <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem @click="handleSortChange('oldest')">
            {{ $t('conversation.sort.oldestActivity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('newest')">
            {{ $t('conversation.sort.newestActivity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_first')">
            {{ $t('conversation.sort.startedFirst') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_last')">
            {{ $t('conversation.sort.startedLast') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('waiting_longest')">
            {{ $t('conversation.sort.waitingLongest') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('next_sla_target')">
            {{ $t('conversation.sort.nextSLATarget') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('priority_first')">
            {{ $t('conversation.sort.priorityFirst') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Content -->
    <div ref="listContainer" class="flex-grow min-h-0 overflow-y-auto overscroll-contain" style="overflow-anchor: none" @scroll.passive="saveListPosition">
      <EmptyList
        v-if="showEmpty"
        key="empty"
        class="px-4 py-8"
        :title="t('conversation.noConversationsFound')"
        :message="t('conversation.tryAdjustingFilters')"
        :icon="Inbox"
      />

      <EmptyList
        v-if="hasErrored"
        key="error"
        class="px-4 py-8"
        :title="t('conversation.couldNotFetch')"
        :message="conversationStore.conversations.errorMessage"
        :icon="MessageCircleWarning"
      />

      <div
        v-if="!hasErrored && !conversationStore.conversations.loading"
        key="list"
        class="space-y-1.5 p-1.5"
        :class="{ 'border-b border-border': hasConversations }"
      >
        <ConversationListItem
          v-for="conversation in conversationStore.conversationsList"
          :key="conversation.uuid"
          :conversation="conversation"
          :currentConversation="conversationStore.current"
          :contactFullName="conversationStore.getContactFullName(conversation.uuid)"
          class="transition-colors duration-200"
        />
      </div>

      <div v-if="conversationStore.conversations.loading" key="loading">
        <ConversationListItemSkeleton v-for="i in 12" :key="i" :index="i - 1" />
      </div>

      <!-- Load More -->
      <div
        v-if="!hasErrored && (conversationStore.conversations.hasMore || hasConversations)"
        class="flex justify-center items-center p-5"
      >
        <Button
          v-if="conversationStore.conversations.hasMore"
          variant="outline"
          @click="conversationStore.fetchNextConversations"
          :disabled="conversationStore.conversations.fetching"
          class="max-md:h-11 transition-all duration-200 ease-in-out transform hover:scale-105"
        >
          <Loader2 v-if="conversationStore.conversations.fetching" class="mr-2 h-4 w-4 animate-spin" />
          {{ conversationStore.conversations.fetching ? t('globals.terms.loading') : t('globals.terms.loadMore') }}
        </Button>
        <p
          class="text-sm text-muted-foreground"
          v-else-if="conversationStore.conversationsList.length > 10"
        >
          {{ $t('conversation.allLoaded') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { conversationStatusLabel } from '@/utils/conversation-status'
import { computed, ref } from 'vue'
import { useConversationListPosition } from '@/composables/useConversationListPosition'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Inbox, MessageCircleWarning, ChevronDown, Loader2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useConversationStore } from '@/stores/conversation'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import EmptyList from '@/features/conversation/list/ConversationEmptyList.vue'
import ConversationBulkActionToolbar from '@/features/conversation/list/ConversationBulkActionToolbar.vue'
import ConversationListItem from '@/features/conversation/list/ConversationListItem.vue'
import ConversationListItemSkeleton from '@/features/conversation/list/ConversationListItemSkeleton.vue'

const conversationStore = useConversationStore()
const listContainer = ref(null)
const { saveListPosition } = useConversationListPosition(listContainer, conversationStore)
const { canBulkAct } = useBulkActionPermissions()
const route = useRoute()
const { t } = useI18n()

const hasSelection = computed(() => conversationStore.selectedCount > 0)

const title = computed(() => {
  if (route.query.folder) return route.query.folderName || 'Ordner'
  const typeKey = route.meta?.typeKey?.(route)
  if (typeKey) {
    return t(typeKey)
  }
  const key = route.meta?.titleKey
  if (!key) return ''
  return t(key, route.meta?.titleCount || 1)
})

const handleStatusChange = (status) => {
  conversationStore.setListStatus(status.label)
}

const handleSortChange = (order) => {
  conversationStore.setListSortField(order)
}

const hasConversations = computed(() => conversationStore.conversationsList.length !== 0)
const hasErrored = computed(() => !!conversationStore.conversations.errorMessage)
const showEmpty = computed(
  () =>
    !hasConversations.value &&
    !hasErrored.value &&
    !conversationStore.conversations.loading &&
    conversationStore.conversations.initialized
)
</script>

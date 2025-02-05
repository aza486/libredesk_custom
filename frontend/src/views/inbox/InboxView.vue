<template>
  <ConversationPlaceholder v-if="!conversationStore.current" />
  <router-view />
</template>

<script setup>
import { computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useConversationStore } from '@/stores/conversation'
import { CONVERSATION_LIST_TYPE, CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import ConversationPlaceholder from '@/features/conversation/ConversationPlaceholder.vue'

const route = useRoute()
const type = computed(() => route.params.type)
const teamID = computed(() => route.params.teamID)
const viewID = computed(() => route.params.viewID)

const conversationStore = useConversationStore()

// Init conversations list based on route params
onMounted(() => {
  // Fetch list based on type
  if (type.value) {
    conversationStore.setListStatus(CONVERSATION_DEFAULT_STATUSES.OPEN, false)
    conversationStore.resetCurrentConversation()
    conversationStore.fetchConversationsList(true, type.value)
  }
  // Fetch team list.
  if (teamID.value) {
    conversationStore.resetCurrentConversation()
    conversationStore.fetchConversationsList(
      true,
      CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
      teamID.value
    )
  }
  // Fetch view list.
  if (viewID.value) {
    conversationStore.resetCurrentConversation()
    conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], viewID.value)
  }
})

// Refetch when route params change
watch(
  [type, teamID, viewID],
  ([newType, newTeamID, newViewID], [oldType, oldTeamID, oldViewID]) => {
    if (newType !== oldType && newType) {
      conversationStore.fetchConversationsList(true, newType)
    }
    if (newTeamID !== oldTeamID && newTeamID) {
      conversationStore.fetchConversationsList(
        true,
        CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
        newTeamID
      )
    }
    if (newViewID !== oldViewID && newViewID) {
      conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], newViewID)
    }
  }
)
</script>

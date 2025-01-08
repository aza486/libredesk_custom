<template>
  <ResizablePanelGroup direction="horizontal" auto-save-id="conversation.vue.resizable.panel">
    <ResizablePanel :min-size="23" :default-size="23" :max-size="40">
      <ConversationList />
    </ResizablePanel>
    <ResizableHandle />
    <ResizablePanel>
      <div class="border-r">
        <Conversation v-if="conversationStore.current"></Conversation>
        <ConversationPlaceholder v-else></ConversationPlaceholder>
      </div>
    </ResizablePanel>
  </ResizablePanelGroup>
</template>

<script setup>
import { watch, onUnmounted, onMounted } from 'vue'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import ConversationList from '@/components/conversation/list/ConversationList.vue'
import Conversation from '@/components/conversation/Conversation.vue'
import ConversationPlaceholder from '@/components/conversation/ConversationPlaceholder.vue'
import { useConversationStore } from '@/stores/conversation'
import { CONVERSATION_LIST_TYPE, CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import { unsetCurrentConversation, setCurrentConversation } from '@/websocket'

const props = defineProps({
  uuid: String,
  type: String,
  teamID: String,
  viewID: String
})
const conversationStore = useConversationStore()

onMounted(() => {
  if (props.uuid) {
    fetchConversation(props.uuid)
  }
  if (props.type) {
    conversationStore.setListStatus(CONVERSATION_DEFAULT_STATUSES.OPEN, false)
    conversationStore.fetchConversationsList(true, props.type)
  }
  if (props.teamID) {
    conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED, props.teamID)
  }
  if (props.viewID) {
    conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], props.viewID)
  }
})

onUnmounted(() => {
  unsetCurrentConversation()
  conversationStore.resetCurrentConversation()
  conversationStore.resetMessages()
})

// Fetch conversation for a specific UUID.
watch(
  () => props.uuid,
  (newUUID, oldUUID) => {
    if (newUUID !== oldUUID && newUUID) {
      unsetCurrentConversation()
      fetchConversation(newUUID)
    }
  }
)

// Fetch conversations for a specific type.
watch(
  () => props.type,
  (newType, oldType) => {
    if (newType !== oldType && newType) {
      conversationStore.fetchConversationsList(true, newType)
    }
  }
)

// Fetch conversations for a specific team that are not assigned to any user.
watch(
  () => props.teamID,
  (newTeamID, oldTeamID) => {
    if (newTeamID !== oldTeamID && newTeamID) {
      conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED, newTeamID)
    }
  }
)

watch(
  () => props.viewID,
  (newViewID, oldViewID) => {
    if (newViewID !== oldViewID && newViewID) {
      conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], newViewID)
    }
  }
)

const fetchConversation = async (uuid) => {
  setCurrentConversation(uuid)
  await conversationStore.fetchConversation(uuid)
  await conversationStore.fetchMessages(uuid)
  await conversationStore.fetchParticipants(uuid)
  await conversationStore.updateAssigneeLastSeen(uuid)
}
</script>

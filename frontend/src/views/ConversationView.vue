<template>
  <ResizablePanelGroup direction="horizontal" auto-save-id="conversation.vue.resizable.panel">
    <ResizablePanel :min-size="23" :default-size="23" :max-size="40" class="shadow-md shadow-gray-300">
      <ConversationList></ConversationList>
    </ResizablePanel>
    <ResizableHandle />
    <ResizablePanel>
      <Conversation v-if="conversationStore.current"></Conversation>
      <ConversationPlaceholder v-else></ConversationPlaceholder>
    </ResizablePanel>
    <ResizableHandle />
    <ResizablePanel :min-size="10" :default-size="16" :max-size="30" v-if="conversationStore.current"
      class="shadow shadow-gray-300">
      <ConversationSideBar></ConversationSideBar>
    </ResizablePanel>
  </ResizablePanelGroup>
</template>

<script setup>
import { onMounted, watch, onUnmounted } from 'vue'

import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import ConversationList from '@/components/conversation/list/ConversationList.vue'
import Conversation from '@/components/conversation/ConversationPage.vue'
import ConversationSideBar from '@/components/conversation/sidebar/ConversationSideBar.vue'
import ConversationPlaceholder from '@/components/conversation/ConversationPlaceholder.vue'
import { useConversationStore } from '@/stores/conversation'
import { unsetCurrentConversation, setCurrentConversation } from '@/websocket'

const props = defineProps({
  uuid: String
})
const conversationStore = useConversationStore()

onMounted(() => {
  fetchConversation(props.uuid)
})

onUnmounted(() => {
  unsetCurrentConversation()
  conversationStore.resetCurrentConversation()
  conversationStore.resetMessages()
})

watch(
  () => props.uuid,
  (newUUID, oldUUID) => {
    if (newUUID !== oldUUID) {
      unsetCurrentConversation()
      fetchConversation(newUUID)
    }
  }
)

const fetchConversation = (uuid) => {
  if (!uuid) return
  conversationStore.fetchParticipants(uuid)
  conversationStore.fetchConversation(uuid)
  setCurrentConversation(uuid)
  conversationStore.fetchMessages(uuid)
  conversationStore.updateAssigneeLastSeen(uuid)
}

</script>

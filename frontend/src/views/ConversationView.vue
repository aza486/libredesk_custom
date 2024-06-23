<template>
  <!-- Resizable panel last resize value is stored in the localstorage -->
  <ResizablePanelGroup direction="horizontal" auto-save-id="conversation.vue.resizable.panel">
    <ResizablePanel :min-size="18" :default-size="23" :max-size="23">
      <ConversationList></ConversationList>
    </ResizablePanel>
    <ResizableHandle />
    <ResizablePanel>
      <Conversation v-if="conversationStore.conversation.data"></Conversation>
      <ConversationPlaceholder v-else></ConversationPlaceholder>
    </ResizablePanel>
    <ResizableHandle />
    <ResizablePanel :min-size="10" :default-size="16" :max-size="30" v-if="conversationStore.conversation.data">
      <ConversationSideBar></ConversationSideBar>
    </ResizablePanel>
  </ResizablePanelGroup>
</template>

<script setup>
import { onMounted, watch } from "vue"
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from '@/components/ui/resizable'
import ConversationList from '@/components/conversationlist/ConversationList.vue'
import Conversation from '@/components/conversation/Conversation.vue'
import ConversationSideBar from '@/components/conversation/ConversationSideBar.vue'
import ConversationPlaceholder from "@/components/conversation/ConversationPlaceholder.vue"
import { useConversationStore } from '@/stores/conversation'

const props = defineProps({
  uuid: String
})
const conversationStore = useConversationStore();

onMounted(() => {
  if (props.uuid) {
    fetchConversation(props.uuid)
  }
});

watch(() => props.uuid, (uuid) => {
  if (uuid) {
    fetchConversation(uuid)
  }
});

const fetchConversation = (uuid) => {  
  conversationStore.fetchParticipants(uuid)
  conversationStore.fetchConversation(uuid)
  conversationStore.fetchMessages(uuid)
  conversationStore.updateAssigneeLastSeen(uuid)
}
</script>

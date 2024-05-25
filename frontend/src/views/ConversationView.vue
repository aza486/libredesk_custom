<template>
  <!-- Resizable panel last resize value is stored in the localstorage -->
  <ResizablePanelGroup direction="horizontal" auto-save-id="conversation.vue.resizable.panel">
    <ResizablePanel :min-size="15" :default-size="23" :max-size="23">
      <ConversationList></ConversationList>
    </ResizablePanel>
    <ResizableHandle with-handle />
    <ResizablePanel>
      <ConversationThread v-if="conversationStore.conversation.data"></ConversationThread>
      <ConversationEmpty></ConversationEmpty>
    </ResizablePanel>
    <ResizableHandle with-handle />
    <ResizablePanel :min-size="10" :default-size="17" :max-size="30" v-if="conversationStore.conversation.data">
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
import ConversationList from '@/components/ConversationList.vue'
import ConversationThread from '@/components/ConversationThread.vue'
import ConversationSideBar from '@/components/ConversationSideBar.vue'
import ConversationEmpty from "@/components/ConversationEmpty.vue"
import { useConversationStore } from '@/stores/conversation'

const props = defineProps({
  uuid: String
})
const conversationStore = useConversationStore();

onMounted(() => {
  if (props.uuid) {
    conversationStore.fetchConversation(props.uuid)
    conversationStore.fetchMessages(props.uuid)
  } else {
    conversationStore.$reset()
  }
});

watch(() => props.uuid, (uuid) => {
  if (uuid)
    conversationStore.fetchConversation(uuid)
    conversationStore.fetchMessages(props.uuid)
});
</script>

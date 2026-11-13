<template>
  <TooltipProvider>
    <button data-test="expand" @click="fullscreen = true">Expand</button>
    <output data-test="draft-state">{{ htmlContent }}</output>
    <output data-test="mode-state">{{ messageType }}</output>
    <div data-test="normal-history" />
    <!-- Match Conversation.vue: history stays mounted outside the composer's
         Vue subtree and moves into its DOM through Teleport. -->
    <Teleport :to="historyTarget || '[data-test=normal-history]'">
      <div data-test="history" class="h-full overflow-y-auto">
        <BubbleAttachmentPreview :attachments="images" />
        <div style="height: 1600px">Selected conversation history</div>
      </div>
    </Teleport>
    <Dialog :open="fullscreen" @update:open="fullscreen = false">
      <DialogContent
        class="lightbox-fullscreen-harness inset-4 top-4 left-4 translate-x-0 translate-y-0 max-w-none w-auto h-[calc(100vh-2rem)] p-0 flex flex-col overflow-hidden"
        :aria-describedby="undefined"
        hide-close-button
        @escape-key-down="fullscreen = false"
      >
        <DialogTitle class="sr-only">Expanded reply editor</DialogTitle>
        <ExpandedReplyLayout
          v-if="fullscreen"
          :private-note="messageType === 'private_note'"
          @restore="fullscreen = false"
          @history-target="historyTarget = $event"
        >
          <ReplyBoxContent
            is-fullscreen
            hide-fullscreen-toggle
            :ai-prompts="[]"
            :is-sending="false"
            :uploading-files="[]"
            v-model:html-content="htmlContent"
            v-model:text-content="textContent"
            v-model:message-type="messageType"
          />
        </ExpandedReplyLayout>
      </DialogContent>
    </Dialog>
  </TooltipProvider>
</template>

<script setup>
import { ref, shallowRef } from 'vue'
import { Dialog, DialogContent, DialogTitle } from '@shared-ui/components/ui/dialog'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import { useConversationStore } from '@main/stores/conversation'
import ExpandedReplyLayout from '@main/features/conversation/ExpandedReplyLayout.vue'
import ReplyBoxContent from '@main/features/conversation/ReplyBoxContent.vue'
import BubbleAttachmentPreview from '@main/features/conversation/message/attachment/BubbleAttachmentPreview.vue'

const fullscreen = ref(false)
const historyTarget = shallowRef(null)
const htmlContent = ref('<p>Prepared answer</p>')
const textContent = ref('Prepared answer')
const messageType = ref('reply')
const store = useConversationStore()
store.conversation.data = {
  uuid: 'lightbox-regression-conversation',
  subject: 'Selected conversation',
  inbox_channel: 'email',
  status: 'Open',
  priority: 'High',
  contact: { first_name: 'Test', last_name: 'Customer', email: 'test@example.test' }
}
const images = ['First image', 'Second image', 'Third image'].map((name, i) => ({
  uuid: `image-${i}`,
  name,
  content_type: 'image/svg+xml',
  size: 100,
  url: `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="160" height="120"><rect width="160" height="120" fill="${['red', 'green', 'blue'][i]}"/></svg>`)}`
}))
</script>

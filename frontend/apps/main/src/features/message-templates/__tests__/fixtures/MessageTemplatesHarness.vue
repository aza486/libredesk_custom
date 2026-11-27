<script setup>
import { ref } from 'vue'
import { Dialog, DialogContent, DialogTitle } from '@shared-ui/components/ui/dialog'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import ReplyBoxContent from '@main/features/conversation/ReplyBoxContent.vue'
const fullscreen = ref(false)
const html = ref('<p>Bestehender Entwurf</p>')
const text = ref('Bestehender Entwurf')
const mode = ref('reply')
const to = ref('customer@example.test')
const cc = ref('cc@example.test')
const bcc = ref('bcc@example.test')
const signature = ref(72)
const files = ref([])
const signatures = [{ id: 72, name: 'Bestehende Signatur' }]
const store = useConversationStore()
store.conversation.data = {
  uuid: 'template-test-conversation',
  inbox_id: 102,
  inbox_channel: 'email',
  subject: 'Unveränderter Betreff',
  status: 'Open',
  priority: 'High',
  contact: { email: to.value }
}
const user = useUserStore()
user.setCurrentUser({
  id: 101,
  roles: ['Kundensupport'],
  permissions: ['messages:write'],
  teams: []
})
</script>
<template>
  <TooltipProvider>
    <button
      data-test="switch-identity"
      @click="user.setCurrentUser({ id: 102, roles: [], permissions: [], teams: [] })"
    >
      Benutzer wechseln
    </button>
    <button data-test="ai-adopt" @click="html = '<p>Übernommener KI-Entwurf</p>'">
      KI-Entwurf übernehmen
    </button>
    <output data-test="draft">{{ html }}</output>
    <output data-test="mode">{{ mode }}</output>
    <output data-test="recipients">{{ [to, cc, bcc].join('|') }}</output>
    <output data-test="attachments">{{ files.map((file) => file.filename).join('|') }}</output>
    <output data-test="signature">{{ signature }}</output>
    <div
      v-if="!fullscreen"
      data-test="normal-composer"
      class="fixed bottom-4 left-4 w-[700px] h-[430px] bg-background overflow-hidden border rounded-lg p-3"
    >
      <ReplyBoxContent
        v-model:htmlContent="html"
        v-model:textContent="text"
        v-model:messageType="mode"
        v-model:to="to"
        v-model:cc="cc"
        v-model:bcc="bcc"
        v-model:selectedTemplateId="signature"
        :outgoing-templates="signatures"
        :ai-prompts="[]"
        :is-sending="false"
        :uploading-files="[]"
        :uploaded-files="files"
        @template-attachments="files = $event"
        @file-delete="files = files.filter((file) => file.uuid !== $event)"
        @toggle-fullscreen="fullscreen = true"
      />
    </div>
    <Dialog :open="fullscreen" @update:open="fullscreen = false">
      <DialogContent
        data-test="fullscreen-composer"
        class="template-fullscreen-composer inset-4 top-4 left-4 translate-x-0 translate-y-0 w-auto max-w-none h-[calc(100vh-2rem)] flex flex-col overflow-hidden"
        :aria-describedby="undefined"
        hide-close-button
      >
        <DialogTitle class="sr-only">Vollbildeditor</DialogTitle>
        <ReplyBoxContent
          is-fullscreen
          v-model:htmlContent="html"
          v-model:textContent="text"
          v-model:messageType="mode"
          v-model:to="to"
          v-model:cc="cc"
          v-model:bcc="bcc"
          v-model:selectedTemplateId="signature"
          :outgoing-templates="signatures"
          :ai-prompts="[]"
          :is-sending="false"
          :uploading-files="[]"
          :uploaded-files="files"
          @template-attachments="files = $event"
          @file-delete="files = files.filter((file) => file.uuid !== $event)"
          @toggle-fullscreen="fullscreen = false"
        />
      </DialogContent>
    </Dialog>
  </TooltipProvider>
</template>

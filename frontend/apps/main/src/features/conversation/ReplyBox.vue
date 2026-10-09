<template>
  <Dialog :open="aiModalOpen" @update:open="aiModalOpen = $event">
    <DialogContent @close-auto-focus="restoreAiFocus">
      <DialogHeader>
        <DialogTitle>{{
          aiAction?.mode === 'adopt' ? 'KI-Antwort übernehmen' : 'Eigene Antwort verfassen'
        }}</DialogTitle>
        <DialogDescription>
          {{
            aiAction?.mode === 'adopt'
              ? 'Der Vorschlag ersetzt den öffentlichen Antwortentwurf. Du kannst ihn vor dem Versand bearbeiten.'
              : 'Schreibe deine Antwort anschließend im vorhandenen Editor. Feedback wird erst nach bestätigtem Versand gespeichert.'
          }}
        </DialogDescription>
      </DialogHeader>
      <div
        class="max-h-56 overflow-auto whitespace-pre-wrap rounded-md border bg-muted p-3 text-sm"
        aria-label="Ursprünglicher KI-Antwortvorschlag"
      >
        {{ aiAction?.note.suggestion }}
      </div>
      <template v-if="aiAction?.mode === 'own'">
        <label for="ai-feedback-reason" class="text-sm font-medium"
          >Warum möchtest du anders antworten? (optional)</label
        >
        <textarea
          id="ai-feedback-reason"
          v-model="aiReason"
          rows="4"
          class="w-full rounded-md border bg-background p-2 text-sm"
          :aria-invalid="aiReasonBytes > 8000"
          aria-describedby="ai-feedback-limit"
        />
        <p
          id="ai-feedback-limit"
          class="text-xs"
          :class="aiReasonBytes > 8000 ? 'text-destructive' : 'text-muted-foreground'"
        >
          {{ aiReasonBytes }} / 8000 UTF-8-Bytes
        </p>
      </template>
      <p v-if="aiModalError" role="alert" class="text-sm text-destructive">{{ aiModalError }}</p>
      <DialogFooter>
        <Button variant="outline" :disabled="aiModalBusy" @click="aiModalOpen = false"
          >Abbrechen</Button
        >
        <Button
          :disabled="aiModalBusy || aiReasonBytes > 8000 || isSending"
          @click="continueAiReply"
          >{{ aiAction?.mode === 'adopt' ? 'KI-Antwort übernehmen' : 'Weiter zur Antwort' }}</Button
        >
      </DialogFooter>
    </DialogContent>
  </Dialog>
  <div
    v-if="feedbackRecord"
    class="mx-2 mt-2 rounded-md border p-3 text-sm"
    role="status"
    aria-live="polite"
  >
    <p>{{ feedbackStatus }}</p>
    <p v-if="feedbackRecord.phase === 'composing'" class="mt-1 text-muted-foreground">
      Die nächste öffentliche Antwort wird mit der ursprünglichen Kundenmail verknüpft. Private
      Notizen erzeugen kein Feedback.
    </p>
    <Button
      v-if="feedbackRecord.phase === 'composing'"
      size="sm"
      variant="ghost"
      @click="cancelFeedback"
      >Feedback abbrechen (Antwort behalten)</Button
    >
    <Button
      v-if="['error', 'uncertain', 'failed', 'waiting'].includes(feedbackRecord.phase)"
      size="sm"
      variant="outline"
      class="mt-2"
      @click="retryFeedback"
      >{{
        feedbackRecord.deliveryConfirmed && feedbackRecord.phase === 'error'
          ? 'Feedback erneut speichern'
          : 'Versandstatus prüfen'
      }}</Button
    >
  </div>

  <AlertDialog :open="showContactEmailWarning" @update:open="showContactEmailWarning = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('replyBox.contactEmailMissing') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{
            $t('replyBox.contactEmailMissingDescription', {
              email: conversationStore.current?.contact?.email
            })
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="processSend(true, true, deferredStatus)">{{
          $t('replyBox.sendAnyway')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <AlertDialog :open="showMissingTagsWarning" @update:open="showMissingTagsWarning = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('replyBox.missingTagsTitle') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('replyBox.missingTagsDescription') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="processSend(false, true, deferredStatus)">{{
          $t('replyBox.sendAnyway')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <div class="text-foreground bg-background">
    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = false">
      <DialogContent
        class="bg-card text-card-foreground p-4 flex flex-col overflow-hidden"
        :class="[
          isCramped
            ? 'top-0 left-0 translate-x-0 translate-y-0 w-full max-w-none h-[var(--visual-viewport-height,100dvh)] max-h-none rounded-none'
            : 'max-w-[60%] h-[70%] max-h-[75%] rounded-lg',
          { '!bg-private': messageType === 'private_note', 'ai-generating': isGenerating }
        ]"
        @escapeKeyDown="isEditorFullscreen = false"
        :hide-close-button="true"
      >
        <ReplyBoxContent
          v-if="isEditorFullscreen"
          ref="fullscreenContentRef"
          :isFullscreen="true"
          :aiPrompts="aiPrompts"
          :isSending="isSending"
          :isDraftLoading="isDraftLoading"
          :uploadingFiles="uploadingFiles"
          :uploadedFiles="mediaFiles"
          v-model:htmlContent="htmlContent"
          v-model:textContent="textContent"
          v-model:to="to"
          v-model:cc="cc"
          v-model:bcc="bcc"
          v-model:emailErrors="emailErrors"
          v-model:messageType="messageType"
          v-model:showBcc="showBcc"
          v-model:mentions="mentions"
          @toggleFullscreen="isEditorFullscreen = !isEditorFullscreen"
          @send="processSend"
          @sendAndSetStatus="processSendAndSetStatus"
          @fileUpload="handleFileUpload"
          @fileDelete="handleFileDelete"
          @filesDropped="uploadFiles"
          @aiPromptSelected="handleAiPromptSelected"
          :isGenerating="isGenerating"
          @generateReply="handleGenerateReply"
          class="h-full flex-grow"
        />
      </DialogContent>
    </Dialog>

    <div v-if="isCramped && !isEditorFullscreen" class="p-2">
      <Button
        type="button"
        variant="outline"
        class="w-full h-11 justify-start font-normal min-w-0"
        :class="{ '!bg-private': messageType === 'private_note', 'ai-generating': isGenerating }"
        @click="isEditorFullscreen = true"
      >
        <Pencil class="shrink-0 text-muted-foreground" />
        <span v-if="draftPreview" class="truncate">{{ draftPreview }}</span>
        <span v-else class="truncate text-muted-foreground">
          {{
            messageType === 'private_note'
              ? $t('globals.terms.privateNote')
              : $t('globals.terms.reply')
          }}
        </span>
        <span
          v-if="attachmentCount"
          class="ml-auto flex shrink-0 items-center gap-1 text-xs text-muted-foreground"
        >
          <Paperclip class="w-3.5 h-3.5" />
          {{ attachmentCount }}
        </span>
      </Button>
    </div>

    <!-- Main Editor non-fullscreen -->
    <div
      class="bg-background text-card-foreground box m-2 px-2 pt-2 flex flex-col relative"
      :class="{ '!bg-private': messageType === 'private_note', 'ai-generating': isGenerating }"
      v-if="!isCramped && !isEditorFullscreen"
    >
      <ReplyBoxContent
        ref="replyBoxContentRef"
        :isFullscreen="false"
        :aiPrompts="aiPrompts"
        :isSending="isSending"
        :isDraftLoading="isDraftLoading"
        :uploadingFiles="uploadingFiles"
        :uploadedFiles="mediaFiles"
        :outgoingTemplates="outgoingTemplates"
        v-model:selectedTemplateId="selectedTemplateId"
        v-model:htmlContent="htmlContent"
        v-model:textContent="textContent"
        v-model:to="to"
        v-model:cc="cc"
        v-model:bcc="bcc"
        v-model:emailErrors="emailErrors"
        v-model:messageType="messageType"
        v-model:showBcc="showBcc"
        v-model:mentions="mentions"
        @toggleFullscreen="isEditorFullscreen = !isEditorFullscreen"
        @send="processSend"
        @sendAndSetStatus="processSendAndSetStatus"
        @fileUpload="handleFileUpload"
        @fileDelete="handleFileDelete"
        @filesDropped="uploadFiles"
        @aiPromptSelected="handleAiPromptSelected"
        :isGenerating="isGenerating"
        @generateReply="handleGenerateReply"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed, toRaw, nextTick, onMounted, onUnmounted } from 'vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { useUserStore } from '@main/stores/user'
import { useDraftManager } from '@main/composables/useDraftManager'
import api from '@main/api'
import { useI18n } from 'vue-i18n'
import { useConversationStore } from '@main/stores/conversation'
import { useInboxStore } from '@main/stores/inbox'
import { useAiPromptStore } from '@main/stores/aiPrompt'
import { useNotificationStore } from '@main/stores/notification'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'
import { Pencil, Paperclip } from 'lucide-vue-next'
import { useVisualViewportHeight } from '@main/composables/useVisualViewportHeight'
import { useIsComposerCramped } from '@main/composables/useIsComposerCramped'
import { useEmitter } from '@main/composables/useEmitter'
import { useFileUpload } from '@main/composables/useFileUpload'
import { hasInlineImage, hasPendingInlineUpload } from '@main/composables/useInlineImageUpload'
import ReplyBoxContent from '@/features/conversation/ReplyBoxContent.vue'
import { createAIFeedback } from '@main/utils/ai-feedback'
import {
  feedbackReasonBytes,
  isOriginalCustomerMessage,
  suggestionToHTML
} from '@main/utils/ai-note'
import { UserTypeAgent } from '@/constants/user'

const { t } = useI18n()
const conversationStore = useConversationStore()
const notificationStore = useNotificationStore()
const inboxStore = useInboxStore()
const emitter = useEmitter()
const userStore = useUserStore()
const isCramped = useIsComposerCramped()
useVisualViewportHeight()

// Setup file upload composable
const {
  uploadingFiles,
  handleFileUpload,
  handleFileDelete,
  uploadFiles,
  mediaFiles,
  clearMediaFiles,
  setMediaFiles
} = useFileUpload({
  linkedModel: 'messages'
})

const messageType = ref('reply')
const currentConversationUUID = computed(() => conversationStore.current?.uuid || null)
watch(
  currentConversationUUID,
  async (uuid, prevUuid) => {
    if (prevUuid) conversationStore.setSelectedDraftType(prevUuid, messageType.value)
    if (!uuid) {
      messageType.value = 'reply'
      return
    }
    messageType.value = conversationStore.resolveDraftType(uuid)
    // Prefetch may still be in flight on first load; re-resolve once drafts land.
    await conversationStore.draftsReady
    if (uuid !== currentConversationUUID.value) return
    messageType.value = conversationStore.resolveDraftType(uuid)
  },
  { immediate: true }
)

// Setup draft management composable, keyed per conversation and message type.
const {
  htmlContent,
  textContent,
  isLoading: isDraftLoading,
  clearDraft,
  loadedAttachments,
  loadedMacroActions,
  loadedMacroID
} = useDraftManager(currentConversationUUID, messageType, mediaFiles)

// Rest of existing state
const isEditorFullscreen = ref(false)
const isSending = ref(false)
const isGenerating = ref(false)
const to = ref('')
const cc = ref('')
const bcc = ref('')
const showBcc = ref(false)
const emailErrors = ref([])

// Outgoing email templates available for this reply.
const outgoingTemplates = ref([])
const selectedTemplateId = ref(null)

const aiPromptStore = useAiPromptStore()
const aiPrompts = computed(() => aiPromptStore.prompts)
const replyBoxContentRef = ref(null)
const showContactEmailWarning = ref(false)
const showMissingTagsWarning = ref(false)
const deferredStatus = ref(null)
const mentions = ref([])

const feedback = createAIFeedback({
  api,
  drafts: conversationStore,
  userID: () => userStore.userID
})
const feedbackRecord = computed(() => feedback.records[currentConversationUUID.value])
const aiModalOpen = ref(false)
const aiAction = ref(null)
const aiReason = ref('')
const aiModalError = ref('')
const aiModalBusy = ref(false)
const aiReasonBytes = computed(() => feedbackReasonBytes(aiReason.value))
const fullscreenContentRef = ref(null)
let aiPreviousFocus = null
let focusAfterModal = false
let pollTimer = null
let pollGeneration = 0
const pollCount = ref(0)
const feedbackStatus = computed(
  () =>
    ({
      composing: 'Eigene Antwort: optionales Feedback vorgemerkt',
      waiting:
        pollCount.value >= 40
          ? 'Versand noch nicht bestätigt. Du kannst den Status erneut prüfen.'
          : 'Antwort wird versendet',
      uncertain:
        'Die Annahme der Antwort ist noch unklar. Bitte den Versandstatus prüfen; nicht erneut senden.',
      saving: 'Versand bestätigt, Feedback wird gespeichert',
      saved: 'Feedback gespeichert',
      error:
        'Feedback konnte nicht gespeichert werden. Die Begründung bleibt erhalten; die Antwort wird nicht erneut versendet.',
      failed: 'Nachrichtenversand fehlgeschlagen. Es wurde kein Feedback gespeichert.'
    })[feedbackRecord.value?.phase]
)
const focusReply = async (uuid) => {
  messageType.value = 'reply'
  isEditorFullscreen.value = isCramped.value
  await nextTick()
  for (let count = 0; isDraftLoading.value && count < 150; count++)
    await new Promise((resolve) => setTimeout(resolve, 20))
  if (uuid !== currentConversationUUID.value || isDraftLoading.value) return false
  await nextTick()
  ;(fullscreenContentRef.value || replyBoxContentRef.value)?.focus()
  return true
}
const restoreAiFocus = (event) => {
  event.preventDefault()
  if (!focusAfterModal) aiPreviousFocus?.focus?.()
  focusAfterModal = false
}
const openReply = async ({ aiNote, mode, content } = {}) => {
  const uuid = currentConversationUUID.value
  if (aiNote) {
    if (
      isSending.value ||
      isGenerating.value ||
      isDraftLoading.value ||
      !userStore.can('messages:write') ||
      conversationStore.current?.inbox_channel !== 'email'
    )
      return
    if (feedbackRecord.value && !['composing', 'saved'].includes(feedbackRecord.value.phase)) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: 'Bitte zuerst das ausstehende Feedback abschließen.'
      })
      return
    }
    try {
      const original = (await api.getConversationMessage(uuid, aiNote.originalUUID))?.data?.data
      if (uuid !== currentConversationUUID.value) return
      if (!isOriginalCustomerMessage(original, uuid, aiNote.originalUUID))
        throw new Error('Die ursprüngliche Kundenmail ist nicht eindeutig zugeordnet.')
      aiPreviousFocus = document.activeElement
      aiAction.value = { uuid, note: aiNote, mode }
      aiReason.value =
        feedbackRecord.value?.phase === 'composing' &&
        feedbackRecord.value.originalUUID === aiNote.originalUUID
          ? feedbackRecord.value.reason
          : ''
      aiModalError.value = ''
      aiModalOpen.value = true
    } catch {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: 'Die ursprüngliche Kundenmail konnte nicht sicher geprüft werden.'
      })
    }
    return
  }
  if ((await focusReply(uuid)) && typeof content === 'string') htmlContent.value = content
}
const continueAiReply = async () => {
  if (aiModalBusy.value || aiReasonBytes.value > 8000) return
  const action = aiAction.value
  if (!action || action.uuid !== currentConversationUUID.value) return
  aiModalBusy.value = true
  try {
    if (action.mode === 'own') await feedback.arm(action.uuid, action.note, aiReason.value)
    else await feedback.cancel(action.uuid)
    if (action.uuid !== currentConversationUUID.value) return
    focusAfterModal = true
    aiModalOpen.value = false
    if ((await focusReply(action.uuid)) && action.mode === 'adopt')
      htmlContent.value = suggestionToHTML(action.note.suggestion)
  } catch {
    aiModalError.value =
      'Die Feedback-Vormerkung konnte nicht sicher gespeichert werden. Bitte erneut versuchen.'
  } finally {
    aiModalBusy.value = false
  }
}
const cancelFeedback = async () => {
  try {
    await feedback.cancel(currentConversationUUID.value)
  } catch {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: 'Abbruch konnte nicht gespeichert werden.'
    })
  }
}
const retryFeedback = async () => {
  const uuid = currentConversationUUID.value
  await feedback.check(uuid, true)
  if (
    uuid === currentConversationUUID.value &&
    ['waiting', 'uncertain'].includes(feedback.records[uuid]?.phase)
  ) {
    stopPolling()
    pollCount.value = 0
    pollFeedback(uuid)
  }
}
const stopPolling = () => {
  pollGeneration++
  clearTimeout(pollTimer)
  pollTimer = null
}
const pollFeedback = async (uuid, generation = pollGeneration) => {
  if (
    generation !== pollGeneration ||
    uuid !== currentConversationUUID.value ||
    pollCount.value >= 40
  )
    return
  pollCount.value++
  await feedback.check(uuid)
  if (
    generation === pollGeneration &&
    uuid === currentConversationUUID.value &&
    ['waiting', 'uncertain'].includes(feedback.records[uuid]?.phase) &&
    pollCount.value < 40
  ) {
    pollTimer = setTimeout(() => pollFeedback(uuid, generation), 3000)
  }
}
watch(
  currentConversationUUID,
  async (uuid) => {
    stopPolling()
    aiModalOpen.value = false
    aiReason.value = ''
    aiAction.value = null
    pollCount.value = 0
    await conversationStore.draftsReady
    if (uuid && uuid === currentConversationUUID.value) {
      feedback.hydrate(uuid)
      pollFeedback(uuid)
    }
  },
  { immediate: true }
)
watch(
  () => conversationStore.messages.version,
  () => {
    if (currentConversationUUID.value) feedback.check(currentConversationUUID.value)
  }
)
onUnmounted(stopPolling)
watch(
  () => userStore.userID,
  () => {
    stopPolling()
    feedback.reset()
    aiModalOpen.value = false
    aiReason.value = ''
    aiAction.value = null
  }
)

defineExpose({ openReply })

aiPromptStore.fetchPrompts()

const runAiGeneration = async (requestFn) => {
  if (isGenerating.value) return
  const uuid = currentConversationUUID.value
  if (!uuid) return
  isGenerating.value = true
  try {
    const resp = await requestFn(uuid)
    if (uuid !== currentConversationUUID.value) return
    htmlContent.value = resp.data.data || ''
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isGenerating.value = false
  }
}

const handleAiPromptSelected = (key) =>
  runAiGeneration(() => api.aiCompletion({ prompt_key: key, content: htmlContent.value }))

const handleGenerateReply = () =>
  runAiGeneration((uuid) =>
    api.aiGenerateReply({ conversation_uuid: uuid, instruction: textContent.value })
  )

// Copilot's "Insert into reply" replaces the draft with its answer (already HTML from the panel),
// forcing reply mode so a private note in progress does not silently receive customer-facing text.
const handleCopilotInsertReply = (html) => {
  if (!html) return
  if (messageType.value === 'private_note') messageType.value = 'reply'
  htmlContent.value = html
}

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.COPILOT_INSERT_REPLY, handleCopilotInsertReply)
})

/**
 * Returns true if the editor has text content.
 */
const hasTextContent = computed(() => {
  return textContent.value.trim().length > 0
})

const draftPreview = computed(() => textContent.value.trim())

const attachmentCount = computed(() => mediaFiles.value.length + uploadingFiles.value.length)

const processSend = async (
  skipContactEmailCheck = false,
  skipMissingTagsCheck = false,
  statusToSet = null
) => {
  if (isSending.value) return
  let hasMessageSendingErrored = false
  isEditorFullscreen.value = false

  const html = htmlContent.value
  if (hasPendingInlineUpload(html)) return
  const hasContent = hasTextContent.value || hasInlineImage(html) || mediaFiles.value.length > 0
  const convUUID = conversationStore.current.uuid
  const sendingAgentID = userStore.userID
  const isPrivate = messageType.value === 'private_note'
  const macroForSend = JSON.parse(
    JSON.stringify(conversationStore.getMacro(MACRO_CONTEXT.REPLY) || {})
  )

  const currentInbox = inboxStore.inboxes.find((i) => i.id === conversationStore.current.inbox_id)
  if (
    !isPrivate &&
    !skipMissingTagsCheck &&
    currentInbox?.prompt_tags_on_reply &&
    !(conversationStore.current.tags?.length > 0)
  ) {
    deferredStatus.value = statusToSet
    showMissingTagsWarning.value = true
    return
  }

  if (!isPrivate && conversationStore.current.inbox_channel === 'email') {
    // Require at least one recipient in `to`.
    if (!to.value.trim()) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: t('replyBox.toRequired')
      })
      return
    }

    // Warn if the contact's email is not in any recipient field.
    if (!skipContactEmailCheck) {
      const contactEmail = conversationStore.current.contact?.email?.toLowerCase()
      if (contactEmail) {
        const allRecipients = [to.value, cc.value, bcc.value].join(',').toLowerCase()
        if (
          !allRecipients
            .split(',')
            .map((e) => e.trim())
            .includes(contactEmail)
        ) {
          deferredStatus.value = statusToSet
          showContactEmailWarning.value = true
          return
        }
      }
    }
  }
  let tempUUID = null

  // Add pending message to cache for instant display.
  if (hasContent) {
    const savedContent = htmlContent.value
    const savedAttachmentIDs = mediaFiles.value.map((file) => file.id)
    const savedMentions = [...mentions.value]
    const savedTemplateID = selectedTemplateId.value
    const author = {
      id: userStore.userID,
      first_name: userStore.firstName,
      last_name: userStore.lastName,
      avatar_url: userStore.avatar,
      type: 'agent'
    }
    const parsedTo =
      !isPrivate && to.value
        ? to.value
            .split(',')
            .map((e) => e.trim())
            .filter(Boolean)
        : []
    const parsedCC =
      !isPrivate && cc.value
        ? cc.value
            .split(',')
            .map((e) => e.trim())
            .filter(Boolean)
        : []
    const parsedBCC =
      !isPrivate && bcc.value
        ? bcc.value
            .split(',')
            .map((e) => e.trim())
            .filter(Boolean)
        : []
    const meta = {}
    if (parsedTo.length) meta.to = parsedTo
    if (parsedCC.length) meta.cc = parsedCC
    if (parsedBCC.length) meta.bcc = parsedBCC

    tempUUID = conversationStore.addPendingMessage(
      convUUID,
      savedContent,
      isPrivate,
      author,
      mediaFiles.value,
      textContent.value,
      meta
    )

    let feedbackOriginalUUID = null
    if (!isPrivate && feedbackRecord.value?.phase === 'composing') {
      isSending.value = true
      try {
        feedbackOriginalUUID = await feedback.prepare(convUUID, tempUUID)
      } catch {
        conversationStore.removePendingMessage(convUUID, tempUUID)
        isSending.value = false
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: 'Feedback konnte nicht vorgemerkt werden. Die Antwort wurde nicht versendet.'
        })
        return
      }
    }
    if (sendingAgentID !== userStore.userID) {
      conversationStore.removePendingMessage(convUUID, tempUUID)
      isSending.value = false
      return
    }
    // Clear editor immediately.
    if (convUUID === currentConversationUUID.value) htmlContent.value = ''

    try {
      isSending.value = true
      const response = await api.sendMessage(convUUID, {
        sender_type: UserTypeAgent,
        private: isPrivate,
        message: savedContent,
        attachments: savedAttachmentIDs,
        mentions: isPrivate ? savedMentions : [],
        cc: parsedCC,
        bcc: parsedBCC,
        to: parsedTo,
        echo_id: isPrivate ? '' : tempUUID,
        template_id: isPrivate ? null : savedTemplateID,
        ...(feedbackOriginalUUID ? { ai_feedback_original_message_uuid: feedbackOriginalUUID } : {})
      })

      if (sendingAgentID !== userStore.userID) {
        isSending.value = false
        return
      }
      if (feedbackOriginalUUID) {
        try {
          await feedback.queued(convUUID, response?.data?.data)
        } catch {
          // The message was already accepted. Never route feedback persistence
          // failures through the send error handler or restore the sent draft.
          if (feedback.records[convUUID]) feedback.records[convUUID].phase = 'uncertain'
        }
        pollCount.value = 0
        if (convUUID === currentConversationUUID.value) {
          stopPolling()
          pollFeedback(convUUID)
        }
      }

      // Private notes are sent immediately so replace immediately.
      if (isPrivate && response?.data?.data) {
        conversationStore.replacePendingMessage(convUUID, tempUUID, response.data.data)
      }

      await conversationStore.refreshCurrentConversation(convUUID).catch(() => {})
      notificationStore.markAssignmentAsReadForConversation(convUUID)
    } catch (error) {
      hasMessageSendingErrored = true
      // Remove pending message and restore editor content.
      conversationStore.removePendingMessage(convUUID, tempUUID)
      if (feedbackOriginalUUID) {
        await feedback.uncertain(convUUID).catch(() => {})
        // A lost HTTP response is not proof that the queue rejected the reply.
        // Keep the intent for status recovery and never restore a resendable draft.
      } else if (convUUID === currentConversationUUID.value) htmlContent.value = savedContent
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  // Apply macro actions if any.
  if (!hasMessageSendingErrored) {
    const macroID = macroForSend.id
    const macroActions = macroForSend.actions || []
    if (macroID > 0) {
      try {
        await api.applyMacro(convUUID, macroID, macroActions)
      } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }
  }

  // Clear state on success.
  if (!hasMessageSendingErrored) {
    clearDraft(convUUID, isPrivate ? 'private_note' : 'reply')
    if (convUUID === currentConversationUUID.value) {
      conversationStore.resetMacro(MACRO_CONTEXT.REPLY)
      clearMediaFiles()
      emailErrors.value = []
      mentions.value = []
      if (statusToSet) conversationStore.updateStatus(statusToSet)
    }
  }
  isSending.value = false
}

const loadOutgoingTemplates = async () => {
  try {
    const response = await api.getOutgoingTemplates()
    outgoingTemplates.value = response?.data?.data || []

    const defaultTemplate = outgoingTemplates.value.find((template) => template.is_default)

    selectedTemplateId.value = defaultTemplate?.id || outgoingTemplates.value[0]?.id || null
  } catch (error) {
    outgoingTemplates.value = []
    selectedTemplateId.value = null

    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.COPILOT_INSERT_REPLY, handleCopilotInsertReply)
  loadOutgoingTemplates()
})

const processSendAndSetStatus = (status) => processSend(false, false, status)

/**
 * Watches for changes in the conversation's macro id and update message content.
 */
watch(
  () => conversationStore.getMacro('reply').id,
  (newId) => {
    // No macro set.
    if (!newId) return

    // If macro has message content, set it in the editor.
    if (conversationStore.getMacro('reply').message_content) {
      htmlContent.value = conversationStore.getMacro('reply').message_content
    }
  },
  { deep: true }
)

// Reset first so a loaded draft never inherits the previous conversation's macro (drafts store no message_content).
watch(
  [loadedMacroID, loadedMacroActions],
  ([id, actions]) => {
    conversationStore.resetMacro(MACRO_CONTEXT.REPLY)
    if (id > 0)
      conversationStore.setMacro({ id, actions: [...toRaw(actions)] }, MACRO_CONTEXT.REPLY)
    else if (actions.length)
      conversationStore.setMacroActions([...toRaw(actions)], MACRO_CONTEXT.REPLY)
  },
  { deep: true }
)

/**
 * Watch for loaded attachments from draft and restore them to mediaFiles.
 */
watch(
  loadedAttachments,
  (attachments) => {
    setMediaFiles([...attachments])
  },
  { deep: true }
)

// Initialize to, cc, and bcc fields with the current conversation's values.
watch(
  () => conversationStore.currentCC,
  (newVal) => {
    cc.value = newVal?.join(', ') || ''
  },
  { deep: true, immediate: true }
)

watch(
  () => conversationStore.currentTo,
  (newVal) => {
    to.value = newVal?.join(', ') || ''
  },
  { immediate: true }
)

watch(
  () => conversationStore.currentBCC,
  (newVal) => {
    const newBcc = newVal?.join(', ') || ''
    bcc.value = newBcc
    // Only show BCC field if it has content
    if (newBcc.length > 0) {
      showBcc.value = true
    }
  },
  { deep: true, immediate: true }
)

// Media files and macro state are restored per draft by the draft manager; resetting here would race ahead of the save and drop them.
watch(
  () => conversationStore.current?.uuid,
  () => {
    setTimeout(() => {
      replyBoxContentRef.value?.focus()
    }, 100)
  }
)
</script>

<style scoped>
/* While the AI drafts a reply, a point of light orbits the reply box: a bright
   comet head that fades to a transparent tail, with its glow travelling along. */
@property --ai-angle {
  syntax: '<angle>';
  initial-value: 0deg;
  inherits: false;
}

.ai-generating {
  box-shadow: 0 6px 22px -10px hsl(var(--primary) / 0.28);
}

.ai-generating::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  padding: 1.5px;
  background: conic-gradient(
    from var(--ai-angle),
    hsl(var(--primary)) 0deg,
    hsl(var(--primary) / 0) 90deg,
    hsl(var(--primary) / 0) 180deg,
    hsl(var(--primary)) 180deg,
    hsl(var(--primary) / 0) 270deg,
    hsl(var(--primary) / 0) 360deg
  );
  filter: drop-shadow(0 0 5px hsl(var(--primary) / 0.5));
  -webkit-mask:
    linear-gradient(#000 0 0) content-box,
    linear-gradient(#000 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  animation: ai-border-spin 2.4s linear infinite;
  pointer-events: none;
  z-index: 20;
}

@keyframes ai-border-spin {
  to {
    --ai-angle: 360deg;
  }
}

@media (prefers-reduced-motion: reduce) {
  /* Steady even glow so the active state stays legible without motion. */
  .ai-generating {
    box-shadow: 0 0 0 1.5px hsl(var(--primary) / 0.4);
  }
  .ai-generating::after {
    display: none;
  }
}
</style>

<template>
  <div>
    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = $event">
      <DialogContent class="max-w-[70%] max-h-[70%] h-[70%] m-0 p-6">
        <div v-if="isEditorFullscreen">
          <div v-if="filteredCannedResponses.length > 0" class="w-full overflow-hidden p-2 border-t backdrop-blur">
            <ul ref="cannedResponsesRef" class="space-y-2 max-h-96 overflow-y-auto">
              <li v-for="(response, index) in filteredCannedResponses" :key="response.id" :class="[
                'cursor-pointer rounded p-1 hover:bg-secondary',
                { 'bg-secondary': index === selectedResponseIndex }
              ]" @click="selectCannedResponse(response.content)" @mouseenter="selectedResponseIndex = index">
                <span class="font-semibold">{{ response.title }}</span> - {{ getTextFromHTML(response.content).slice(0,
                  150)
                }}...
              </li>
            </ul>
          </div>
          <Editor v-model:selectedText="selectedText" v-model:isBold="isBold" v-model:isItalic="isItalic"
            v-model:htmlContent="htmlContent" v-model:textContent="textContent" :placeholder="editorPlaceholder"
            :aiPrompts="aiPrompts" @keydown="handleKeydown" @aiPromptSelected="handleAiPromptSelected"
            @editorReady="onEditorReady" @clearContent="clearContent" :contentToSet="contentToSet"
            v-model:cursorPosition="cursorPosition" />
        </div>
      </DialogContent>
    </Dialog>

    <!-- Canned responses on non-fullscreen editor -->
    <div v-if="filteredCannedResponses.length > 0 && !isEditorFullscreen"
      class="w-full overflow-hidden p-2 border-t backdrop-blur">
      <ul ref="cannedResponsesRef" class="space-y-2 max-h-96 overflow-y-auto">
        <li v-for="(response, index) in filteredCannedResponses" :key="response.id" :class="[
          'cursor-pointer rounded p-1 hover:bg-secondary',
          { 'bg-secondary': index === selectedResponseIndex }
        ]" @click="selectCannedResponse(response.content)" @mouseenter="selectedResponseIndex = index">
          <span class="font-semibold">{{ response.title }}</span> - {{ getTextFromHTML(response.content).slice(0, 150)
          }}...
        </li>
      </ul>
    </div>

    <!-- Main Editor non-fullscreen -->
    <div class="border-t" v-if="!isEditorFullscreen">
      <!-- Message type toggle -->
      <div class="flex justify-between px-2 border-b py-2">
        <Tabs v-model="messageType">
          <TabsList>
            <TabsTrigger value="reply"> Reply </TabsTrigger>
            <TabsTrigger value="private_note"> Private note </TabsTrigger>
          </TabsList>
        </Tabs>
        <div class="flex items-center mr-2 cursor-pointer" @click="isEditorFullscreen = !isEditorFullscreen">
          <Fullscreen size="20" />
        </div>
      </div>

      <!-- Main Editor -->
      <Editor v-model:selectedText="selectedText" v-model:isBold="isBold" v-model:isItalic="isItalic"
        v-model:htmlContent="htmlContent" v-model:textContent="textContent" :placeholder="editorPlaceholder"
        :aiPrompts="aiPrompts" @keydown="handleKeydown" @aiPromptSelected="handleAiPromptSelected"
        @editorReady="onEditorReady" @clearContent="clearContent" :contentToSet="contentToSet"
        v-model:cursorPosition="cursorPosition" />

      <!-- Attachments preview -->
      <AttachmentsPreview :attachments="attachments" :onDelete="handleOnFileDelete"></AttachmentsPreview>

      <!-- Bottom menu bar -->
      <ReplyBoxBottomMenuBar :handleFileUpload="handleFileUpload" :handleInlineImageUpload="handleInlineImageUpload"
        :isBold="isBold" :isItalic="isItalic" @toggleBold="toggleBold" @toggleItalic="toggleItalic" :hasText="hasText"
        :handleSend="handleSend">
      </ReplyBoxBottomMenuBar>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { transformImageSrcToCID } from '@/utils/strings'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { Fullscreen } from 'lucide-vue-next';
import api from '@/api'

import { getTextFromHTML } from '@/utils/strings'
import Editor from './ConversationTextEditor.vue'
import { useConversationStore } from '@/stores/conversation'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useEmitter } from '@/composables/useEmitter'
import AttachmentsPreview from '@/components/attachment/AttachmentsPreview.vue'
import ReplyBoxBottomMenuBar from '@/components/conversation/ReplyBoxMenuBar.vue'

const conversationStore = useConversationStore()

const emitter = useEmitter()

let editorInstance = ref(null)
const isEditorFullscreen = ref(false)
const cursorPosition = ref(0)
const selectedText = ref('')
const htmlContent = ref('')
const textContent = ref('')
const clearContent = ref(false)
const contentToSet = ref('')
const isBold = ref(false)
const isItalic = ref(false)

const uploadedFiles = ref([])
const messageType = ref('reply')

const filteredCannedResponses = ref([])
const selectedResponseIndex = ref(-1)
const cannedResponsesRef = ref(null)
const cannedResponses = ref([])

const aiPrompts = ref([])

onMounted(async () => {
  await Promise.all([fetchCannedResponses(), fetchAiPrompts()])
})

const fetchAiPrompts = async () => {
  try {
    const resp = await api.getAiPrompts()
    aiPrompts.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const fetchCannedResponses = async () => {
  try {
    const resp = await api.getCannedResponses()
    cannedResponses.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const handleAiPromptSelected = async (key) => {
  try {
    const resp = await api.aiCompletion({
      prompt_key: key,
      content: selectedText.value,
    })
    contentToSet.value = resp.data.data.replace(/\n/g, '<br>')
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const toggleBold = () => {
  isBold.value = !isBold.value
}

const toggleItalic = () => {
  isItalic.value = !isItalic.value
}

const editorPlaceholder = computed(() => {
  return "Press enter to add a new line; Press '/' to select a Canned Response."
})

const attachments = computed(() => {
  return uploadedFiles.value.filter(upload => upload.disposition === 'attachment')
})

// Watch for text content changes and filter canned responses
watch(textContent, (newVal) => {
  filterCannedResponses(newVal)
})

const filterCannedResponses = (input) => {
  // Extract the text after the last `/`
  const lastSlashIndex = input.lastIndexOf('/')
  if (lastSlashIndex !== -1) {
    const searchText = input.substring(lastSlashIndex + 1).trim()

    // Filter canned responses based on the search text
    filteredCannedResponses.value = cannedResponses.value.filter((response) =>
      response.title.toLowerCase().includes(searchText.toLowerCase())
    )

    // Reset the selected response index
    selectedResponseIndex.value = filteredCannedResponses.value.length > 0 ? 0 : -1
  } else {
    filteredCannedResponses.value = []
    selectedResponseIndex.value = -1
  }
}

const hasText = computed(() => {
  return textContent.value.trim().length > 0 ? true : false
})

const onEditorReady = (editor) => {
  editorInstance.value = editor
}

const handleFileUpload = (event) => {
  for (const file of event.target.files) {
    api
      .uploadMedia({
        files: file,
        inline: false,
        linked_model: "messages",
      })
      .then((resp) => {
        uploadedFiles.value.push(resp.data.data)
      })
      .catch((error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          title: 'Error',
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      })
  }
}

const handleInlineImageUpload = (event) => {
  for (const file of event.target.files) {
    api
      .uploadMedia({
        files: file,
        inline: true,
        linked_model: "messages",
      })
      .then((resp) => {
        editorInstance.value.commands.setImage({
          src: resp.data.data.url,
          alt: resp.data.data.filename,
          title: resp.data.data.uuid,
        })
        uploadedFiles.value.push(resp.data.data)
      })
      .catch((error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          title: 'Error',
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      })
  }
}

const handleSend = async () => {
  try {
    // Replace inline image url with cid.
    const message = transformImageSrcToCID(htmlContent.value)

    // Check which images are still in editor before sending.
    const parser = new DOMParser()
    const doc = parser.parseFromString(htmlContent.value, 'text/html')
    const inlineImageUUIDs = Array.from(doc.querySelectorAll('img.inline-image'))
      .map(img => img.getAttribute('title'))
      .filter(Boolean)

    uploadedFiles.value = uploadedFiles.value.filter(file =>
      // Keep if:
      // 1. Not an inline image OR
      // 2. Is an inline image that exists in editor
      file.disposition !== 'inline' || inlineImageUUIDs.includes(file.uuid)
    )

    await api.sendMessage(conversationStore.current.uuid, {
      private: messageType.value === 'private_note',
      message: message,
      attachments: uploadedFiles.value.map((file) => file.id)
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    clearContent.value = true
    uploadedFiles.value = []
  }
  api.updateAssigneeLastSeen(conversationStore.current.uuid)
}

const handleOnFileDelete = (uuid) => {
  uploadedFiles.value = uploadedFiles.value.filter((item) => item.uuid !== uuid)
}

const handleKeydown = (event) => {
  if (filteredCannedResponses.value.length > 0) {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      selectedResponseIndex.value =
        (selectedResponseIndex.value + 1) % filteredCannedResponses.value.length
      scrollToSelectedItem()
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      selectedResponseIndex.value =
        (selectedResponseIndex.value - 1 + filteredCannedResponses.value.length) %
        filteredCannedResponses.value.length
      scrollToSelectedItem()
    } else if (event.key === 'Enter') {
      event.preventDefault()
      selectCannedResponse(filteredCannedResponses.value[selectedResponseIndex.value].content)
    }
  }
}

const scrollToSelectedItem = () => {
  const list = cannedResponsesRef.value
  const selectedItem = list.children[selectedResponseIndex.value]
  if (selectedItem) {
    selectedItem.scrollIntoView({
      behavior: 'smooth',
      block: 'nearest'
    })
  }
}

const selectCannedResponse = (content) => {
  contentToSet.value = content
  filteredCannedResponses.value = []
  selectedResponseIndex.value = -1
}
</script>

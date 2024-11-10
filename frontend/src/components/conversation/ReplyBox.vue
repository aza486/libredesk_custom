<template>
  <div>

    <!-- Canned responses -->
    <div v-if="filteredCannedResponses.length > 0" class="w-full overflow-hidden p-2 border-t backdrop-blur">
      <ul ref="responsesList" class="space-y-2 max-h-96 overflow-y-auto">
        <li v-for="(response, index) in filteredCannedResponses" :key="response.id" :class="[
          'cursor-pointer rounded p-1 hover:bg-secondary',
          { 'bg-secondary': index === selectedResponseIndex }
        ]" @click="selectCannedResponse(response.content)" @mouseenter="selectedResponseIndex = index">
          <span class="font-semibold">{{ response.title }}</span> - {{ getTextFromHTML(response.content).slice(0, 150)
          }}...
        </li>
      </ul>
    </div>
    <div class="border-t">

      <!-- Message type toggle -->
      <div class="flex justify-between px-2 border-b py-2">
        <Tabs v-model:model-value="messageType">
          <TabsList>
            <TabsTrigger value="reply"> Reply </TabsTrigger>
            <TabsTrigger value="private_note"> Private note </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <!-- Main Editor -->
      <Editor @keydown="handleKeydown" @editorText="handleEditorText" :placeholder="editorPlaceholder" :isBold="isBold"
        :clearContent="clearContent" :isItalic="isItalic" @updateBold="updateBold" @updateItalic="updateItalic"
        @contentCleared="handleContentCleared" @contentSet="clearContentToSet" @editorReady="onEditorReady"
        :contentToSet="contentToSet" :cannedResponses="cannedResponses" />


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
import { ref, onMounted, computed } from 'vue'
import { transformImageSrcToCID } from '@/utils/strings'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

import { getTextFromHTML } from '@/utils/strings'
import Editor from './ConversationTextEditor.vue'
import { useConversationStore } from '@/stores/conversation'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useEmitter } from '@/composables/useEmitter'
import AttachmentsPreview from '@/components/attachment/AttachmentsPreview.vue'
import ReplyBoxBottomMenuBar from '@/components/conversation/ReplyBoxMenuBar.vue'

const emitter = useEmitter()
const clearContent = ref(false)
const isBold = ref(false)
const isItalic = ref(false)
const editorText = ref('')
const editorHTML = ref('')
const contentToSet = ref('')
const conversationStore = useConversationStore()
const filteredCannedResponses = ref([])
const uploadedFiles = ref([])
const messageType = ref('reply')
const selectedResponseIndex = ref(-1)
const responsesList = ref(null)
let editorInstance = ref(null)

onMounted(async () => {
  try {
    const resp = await api.getCannedResponses()
    cannedResponses.value = resp.data.data
  } catch (error) {
    console.error(error)
  }
})

const cannedResponses = ref([])

const updateBold = (newState) => {
  isBold.value = newState
}

const updateItalic = (newState) => {
  isItalic.value = newState
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

const handleEditorText = (text) => {
  editorText.value = text.text
  editorHTML.value = text.html
  filterCannedResponses(text.text)
}

const hasText = computed(() => {
  return editorText.value.length > 0 ? true : false
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
          title: 'Error uploading file',
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
          title: 'Error uploading file',
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      })
  }
}

const handleContentCleared = () => {
  clearContent.value = false
}

const handleSend = async () => {
  try {
    // Replace inline image url with cid.
    const message = transformImageSrcToCID(editorHTML.value)

    // Check which images are still in editor before sending.
    const parser = new DOMParser()
    const doc = parser.parseFromString(editorHTML.value, 'text/html')
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
      title: 'Error sending message',
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
  const list = responsesList.value
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

const clearContentToSet = () => {
  contentToSet.value = null
}
</script>

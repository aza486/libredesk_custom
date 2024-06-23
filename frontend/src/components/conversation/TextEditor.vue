<template>
    <div>
        <div class="flex justify-between bg-[#F5F5F4] px-1">
            <Tabs v-model:model-value="messageType">
                <TabsList>
                    <TabsTrigger value="reply">
                        Reply
                    </TabsTrigger>
                    <TabsTrigger value="private_note">
                        Private note
                    </TabsTrigger>
                </TabsList>
            </Tabs>
        </div>
        <EditorContent :editor="editor" class="max-h-[600px]" />
        <AttachmentsPreview :attachments="uploadedFiles" :onDelete="handleOnFileDelete" />
        <div class="flex justify-between items-center border-y h-14 p-1 px-2">
            <div class="flex justify-items-start gap-2">
                <input type="file" class="hidden" ref="attachmentInput" multiple @change="handleFileUpload">
                <Toggle class="px-2 py-2 border-0" variant="outline" @click="applyBold" :pressed="isBold">
                    <Bold class="h-4 w-4" />
                </Toggle>
                <Toggle class="px-2 py-2 border-0" variant="outline" @click="applyItalic" :pressed="isItalic">
                    <Italic class="h-4 w-4" />
                </Toggle>
                <Toggle class="px-2 py-2 border-0" variant="outline" @click="triggerFileUpload">
                    <Paperclip class="h-4 w-4" />
                </Toggle>
            </div>
            <Button class="h-8 w-6 px-8 " @click="handleSend" :disabled="!hasText">
                Send
            </Button>
        </div>
    </div>
</template>

<script setup>
import { ref, watchEffect, onUnmounted, onMounted, computed, watch, nextTick } from "vue"
import { Button } from '@/components/ui/button'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import Placeholder from "@tiptap/extension-placeholder"
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import ImageResize from 'tiptap-extension-resize-image';
import { Toggle } from '@/components/ui/toggle'
import { Paperclip, Bold, Italic } from "lucide-vue-next"
import AttachmentsPreview from "@/components/attachment/AttachmentsPreview.vue"
import {
    Tabs,
    TabsList,
    TabsTrigger,
} from '@/components/ui/tabs'
import api from '@/api';
import {
    useLocalStorage,
} from '@vueuse/core'

const emit = defineEmits(['send'])

const props = defineProps({
    conversationuuid: String,
    cannedResponses: Array
})


const getPlaceholder = () => {
    return `Shift + Enter to add a new line; Press '/' to select a Canned Response.`
}

const getDraftLocalStorageKey = () => {
    return `draft.${props.identifier}`
}

const editorContent = useLocalStorage(getDraftLocalStorageKey(), '')
const messageType = ref("reply")
const cannedResponseRefItems = ref([])
const inputText = ref('')
const isBold = ref(false)
const isItalic = ref(false)
const attachmentInput = ref(null)
const imageInput = ref(null)
const cannedResponseIndex = ref(0)
const uploadedFiles = ref([])


const editor = ref(useEditor({
    content: editorContent.value,
    extensions: [
        StarterKit,
        Placeholder.configure({
            placeholder: getPlaceholder(),
            keyboardShortcuts: {
                'Control-b': () => applyBold(),
                'Control-i': () => applyItalic(),
            }
        }),
        Image.configure({
            inline: false,
            HTMLAttributes: {
                class: 'tiptap-editor-image',
            },
        }),
        ImageResize,
    ],
    autofocus: true,
    editorProps: {
        attributes: {
            class: "outline-none",
        },
        handleKeyDown: (_, event) => {
            if (filteredCannedResponses.value.length > 0) {
                switch (event.key) {
                    case 'ArrowDown':
                        event.preventDefault()
                        cannedResponseIndex.value = (cannedResponseIndex.value + 1) % filteredCannedResponses.value.length
                        break
                    case 'ArrowUp':
                        event.preventDefault()
                        cannedResponseIndex.value = (cannedResponseIndex.value - 1 + filteredCannedResponses.value.length) % filteredCannedResponses.value.length
                        break
                    case 'Enter':
                        selectResponse(filteredCannedResponses.value[cannedResponseIndex.value].content)
                        break
                }
            } else if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault()
                handleSend()
                return true
            }
            return false
        }
    },
}))

const saveEditorContent = () => {
    if (editor.value && props.identifier) {
        if (editor.value.getText() === "/" || editor.value.getText() === "@") {
            return
        }
        const content = editor.value.getHTML()
        localStorage.setItem(getDraftLocalStorageKey(), content)
    }
}

const contentSaverInterval = setInterval(saveEditorContent, 200)

watchEffect(() => {
    if (editor.value) {
        isBold.value = editor.value.isActive('bold')
        isItalic.value = editor.value.isActive('italic')
    }
})

watchEffect(() => {
    if (editor.value) {
        inputText.value = editor.value.getText()
    }
})

watch(cannedResponseIndex, () => {
    nextTick(() => {
        if (cannedResponseRefItems.value[cannedResponseIndex.value]) {
            cannedResponseRefItems.value[cannedResponseIndex.value].scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        }
    })
})

// Moving cursor to the end.
onMounted(async () => {
    if (editor.value) {
        // hack.
        setTimeout(() => {
            editor.value.commands.focus()
            editor.value.commands.setTextSelection(editor.value.state.doc.content.size)
        })
    }
})

// Cleanup.
onUnmounted(() => {
    clearInterval(contentSaverInterval)
})

const filteredCannedResponses = computed(() => {
    if (inputText.value.endsWith('/')) {
        const searchQuery = inputText.value.slice(1).toLowerCase()
        return props.cannedResponses.filter(response => response.title.toLowerCase().includes(searchQuery))
    }
    return []
})

const hasText = computed(() => {
    if (editor.value) {
        return editor.value.getText().length === 0 ? false : true
    }
    return false
})

const selectResponse = (message) => {
    editor.value.chain().setContent(message).focus().run()
}

const handleSend = () => {
    const attachmentUUIDs = uploadedFiles.value.map((file) => file.uuid)
    emit('send', {
        html: editor.value.getHTML(),
        text: editor.value.getText(),
        private: messageType.value === "private_note",
        attachments: attachmentUUIDs,
    })
    editor.value.commands.clearContent()
    uploadedFiles.value = []
}

const applyBold = () => {
    editor.value.chain().focus().toggleBold().run()
}

const applyItalic = () => {
    editor.value.chain().focus().toggleItalic().run()
}

const triggerFileUpload = () => {
    attachmentInput.value.click()
};

const handleFileUpload = event => {
    for (const file of event.target.files) {
        api.uploadAttachment({
            files: file,
            disposition: "attachment",
        }).then((resp) => {
            uploadedFiles.value.push(resp.data.data)
        }).catch((err) => {
            console.error(err)
        })
    }
};

const handleOnFileDelete = uuid => {
    uploadedFiles.value = uploadedFiles.value.filter(item => item.uuid !== uuid);
};
</script>


<style lang="scss">
// Moving placeholder to the top
.tiptap p.is-editor-empty:first-child::before {
    content: attr(data-placeholder);
    float: left;
    color: #adb5bd;
    pointer-events: none;
    height: 0;
}

// Editor height
.ProseMirror {
    min-height: 150px;
    overflow: scroll;
    padding: 10px 10px;
}
</style>

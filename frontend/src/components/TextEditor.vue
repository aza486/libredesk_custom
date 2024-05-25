<template>
    <div v-if="showResponses && filteredCannedResponses.length > 0"
        class="w-full drop-shadow-sm overflow-hidden p-2 border-t">
        <ScrollArea>
            <ul class="space-y-2 max-h-96">
                <li v-for="(response, index) in filteredCannedResponses" :key="response.id"
                    @click="selectResponse(response.content)" class="cursor-pointer rounded p-1"
                    :class="{ 'bg-secondary': cannedResponseIndex === index }"
                    :ref="el => cannedResponseRefItems.push(el)">
                    <span class="font-semibold">{{ response.title }}</span> - {{ response.content }}
                </li>
            </ul>
        </ScrollArea>
    </div>
    <div class="relative w-auto rounded-none border-y mb-[49px] fullscreen">
        <div class="flex justify-between bg-[#F5F5F4] dark:bg-white">
            <Tabs default-value="account">
                <TabsList>
                    <TabsTrigger value="account">
                        Reply
                    </TabsTrigger>
                    <TabsTrigger value="password">
                        Internal note
                    </TabsTrigger>
                </TabsList>
            </Tabs>
            <!-- <Toggle class="px-2 py-2 bg-white" variant="outline" @click="toggleFullScreen">
                <Fullscreen class="w-full h-full" />
            </Toggle> -->
        </div>
        <EditorContent :editor="editor" @keyup="checkTrigger" @keydown="navigateResponses" />
        <div class="flex justify-between items-center border h-14 p-1 px-2">
            <div class="flex justify-items-start gap-2">
                <Toggle class="px-2 py-2 " variant="outline" @click="applyBold" :pressed="isBold">
                    <Bold class="w-full h-full" />
                </Toggle>
                <Toggle class="px-2 py-2 " variant="outline" @click="applyItalic" :pressed="isItalic">
                    <Italic class="w-full h-full" />
                </Toggle>
                <!-- File Upload -->
                <input type="files" class="hidden" ref="fileInput" multiple>
                <Toggle class="px-2 py-2" variant="outline" @click="handleFileUpload">
                    <Paperclip class="w-full h-full" />
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
import { Toggle } from '@/components/ui/toggle'
import { Paperclip, Bold, Italic } from "lucide-vue-next"
import { ScrollArea } from '@/components/ui/scroll-area'
import {
    Tabs,
    TabsList,
    TabsTrigger,
} from '@/components/ui/tabs'

const emit = defineEmits(['send'])

const props = defineProps({
    identifier: String,  // Unique identifier for the editor could be the uuid of conversation.
    cannedResponses: Array
})

const editor = ref(useEditor({
    content: '',
    extensions: [
        StarterKit,
        Placeholder.configure({
            placeholder: "Type a message...",
            keyboardShortcuts: {
                'Control-b': () => applyBold(),
                'Control-i': () => applyItalic(),
            }
        }),
    ],
    autofocus: true,
    editorProps: {
        attributes: {
            class: "outline-none",
        },
    },
}))

const saveEditorContent = () => {
    if (editor.value && props.identifier) {
        // Skip single `/`
        if (editor.value.getText() === "/") {
            return
        }
        const content = editor.value.getHTML()
        localStorage.setItem(getDraftLocalStorageKey(), content)
    }
}

const cannedResponseRefItems = ref([])
const inputText = ref('')
const isBold = ref(false)
const isItalic = ref(false)
const fileInput = ref(null)
const cannedResponseIndex = ref(0)
const contentSaverInterval = setInterval(saveEditorContent, 200)
const showResponses = ref(false)


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

onMounted(() => {
    if (editor.value) {
        const draftContent = localStorage.getItem(getDraftLocalStorageKey())
        editor.value.commands.setContent(draftContent)
    }
})

onUnmounted(() => {
    clearInterval(contentSaverInterval)
})

const filteredCannedResponses = computed(() => {
    if (inputText.value.startsWith('/')) {
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

const getDraftLocalStorageKey = () => {
    return `content.${props.identifier}`
}

const navigateResponses = (event) => {
    if (!showResponses.value) return

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
}

const checkTrigger = (e) => {
    if (e.key === "/")
        showResponses.value = true
}

const selectResponse = (message) => {
    editor.value.commands.setContent(message)
    showResponses.value = false
    editor.value.chain().focus()
}

const handleSend = () => {
    emit('send', editor.value.getHTML())
    editor.value.commands.clearContent()
}

const applyBold = () => {
    editor.value.chain().focus().toggleBold().run()
}

const applyItalic = () => {
    editor.value.chain().focus().toggleItalic().run()
}

const handleFileUpload = (event) => {
    const files = Array.from(event.target.files)
    console.log(files)
}

function toggleFullScreen () {
    const editorElement = editor.value?.editorView?.dom
    if (editorElement && editorElement.requestFullscreen) {
        editorElement.requestFullscreen().catch(err => {
            console.error(`Error attempting to enable full-screen mode: ${err.message} (${err.name})`)
        })
    } else {
        console.error("Fullscreen API is not available.")
    }
}
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

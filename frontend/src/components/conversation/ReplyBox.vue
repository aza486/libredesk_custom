<template>
    <div>
        <div v-if="filteredCannedResponses.length > 0" class="w-full drop-shadow-sm overflow-hidden p-2 border-t">
            <ul ref="responsesList" class="space-y-2 max-h-96 overflow-y-auto">
                <li v-for="(response, index) in filteredCannedResponses" :key="response.id"
                    :class="['cursor-pointer rounded p-1 hover:bg-secondary', { 'bg-secondary': index === selectedResponseIndex }]"
                    @click="selectResponse(response.content)" @mouseenter="selectedResponseIndex = index">
                    <span class="font-semibold">{{ response.title }}</span> - {{ response.content }}
                </li>
            </ul>
        </div>
        <div class="border-t ">
            <!-- Message type toggle -->
            <div class="flex justify-between px-2 border-b py-2">
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

            <!-- Editor -->
            <Editor @keydown="handleKeydown" @editorText="handleEditorText" :placeholder="editorPlaceholder"
                :isBold="isBold" :clearContent="clearContent" :isItalic="isItalic" @updateBold="updateBold"
                @updateItalic="updateItalic" @contentCleared="handleContentCleared" @contentSet="clearContentToSet"
                @editorReady="onEditorReady" :messageType="messageType" :contentToSet="contentToSet"
                :cannedResponses="cannedResponsesStore.responses" />

            <!-- Attachments preview -->
            <AttachmentsPreview :attachments="uploadedFiles" :onDelete="handleOnFileDelete"></AttachmentsPreview>

            <!-- Bottom menu bar -->
            <ReplyBoxBottomMenuBar :handleFileUpload="handleFileUpload"
                :handleInlineImageUpload="handleInlineImageUpload" :isBold="isBold" :isItalic="isItalic"
                @toggleBold="toggleBold" @toggleItalic="toggleItalic" :hasText="hasText" :handleSend="handleSend">
            </ReplyBoxBottomMenuBar>
        </div>
    </div>
</template>


<script setup>
import { ref, onMounted, computed } from "vue";
import api from '@/api';

import Editor from './Editor.vue'
import { useConversationStore } from '@/stores/conversation'
import { useCannedResponses } from '@/stores/canned_responses'
import {
    Tabs,
    TabsList,
    TabsTrigger,
} from '@/components/ui/tabs'
import AttachmentsPreview from "@/components/attachment/AttachmentsPreview.vue"
import ReplyBoxBottomMenuBar from "@/components/conversation/ReplyBoxBottomMenuBar.vue"

const clearContent = ref(false)
const isBold = ref(false)
const isItalic = ref(false)
const editorText = ref("")
const editorHTML = ref("")
const contentToSet = ref("")
const conversationStore = useConversationStore()
const cannedResponsesStore = useCannedResponses()
const filteredCannedResponses = ref([])
const uploadedFiles = ref([])
const messageType = ref("reply")
const selectedResponseIndex = ref(-1)
const responsesList = ref(null)
let editorInstance = null

onMounted(() => {
    cannedResponsesStore.fetchAll()
})

const updateBold = (newState) => {
    isBold.value = newState;
}

const updateItalic = (newState) => {
    isItalic.value = newState;
}

const toggleBold = () => {
    isBold.value = !isBold.value;
}

const toggleItalic = () => {
    isItalic.value = !isItalic.value;
}

const editorPlaceholder = computed(() => {
    return "Shift + Enter to add a new line; Press '/' to select a Canned Response."
})

const filterCannedResponses = (input) => {
    // Extract the text after the last `/`
    const lastSlashIndex = input.lastIndexOf('/');
    if (lastSlashIndex !== -1) {
        const searchText = input.substring(lastSlashIndex + 1).trim();

        // Filter canned responses based on the search text
        filteredCannedResponses.value = cannedResponsesStore.responses.filter(response =>
            response.title.toLowerCase().includes(searchText.toLowerCase())
        );

        // Reset the selected response index
        selectedResponseIndex.value = filteredCannedResponses.value.length > 0 ? 0 : -1;
    } else {
        filteredCannedResponses.value = [];
        selectedResponseIndex.value = -1;
    }
}

const handleEditorText = (text) => {
    editorText.value = text.text
    editorHTML.value = text.html
    filterCannedResponses(text.text)
}

const hasText = computed(() => {
    return editorText.value.length > 0 ? true : false
});

const onEditorReady = (editor) => {
    editorInstance = editor
};

const handleFileUpload = event => {
    for (const file of event.target.files) {
        api.uploadMedia({
            files: file,
        }).then((resp) => {
            uploadedFiles.value.push(resp.data.data)
        }).catch((err) => {
            console.error(err)
        })
    }
};

const handleInlineImageUpload = event => {
    for (const file of event.target.files) {
        api.uploadMedia({
            files: file,
        }).then((resp) => {
            editorInstance.commands.setImage({
                src: resp.data.data.url,
                alt: resp.data.data.filename,
                title: resp.data.data.filename,
            })
        }).catch((err) => {
            console.error(err)
        })
    }
};

const handleContentCleared = () => {
    clearContent.value = false
}

const handleSend = async () => {
    const attachmentIDs = uploadedFiles.value.map((file) => file.id)
    await api.sendMessage(conversationStore.conversation.data.uuid, {
        private: messageType.value === "private_note",
        message: editorHTML.value,
        attachments: attachmentIDs,
    })
    api.updateAssigneeLastSeen(conversationStore.conversation.data.uuid)
    clearContent.value = true
    uploadedFiles.value = []
}

const handleOnFileDelete = uuid => {
    uploadedFiles.value = uploadedFiles.value.filter(item => item.uuid !== uuid);
};

const handleKeydown = (event) => {
    if (filteredCannedResponses.value.length > 0) {
        if (event.key === 'ArrowDown') {
            event.preventDefault();
            selectedResponseIndex.value = (selectedResponseIndex.value + 1) % filteredCannedResponses.value.length;
            scrollToSelectedItem();
        } else if (event.key === 'ArrowUp') {
            event.preventDefault();
            selectedResponseIndex.value = (selectedResponseIndex.value - 1 + filteredCannedResponses.value.length) % filteredCannedResponses.value.length;
            scrollToSelectedItem();
        } else if (event.key === 'Enter') {
            event.preventDefault();
            selectResponse(filteredCannedResponses.value[selectedResponseIndex.value].content);
        }
    }
}

const scrollToSelectedItem = () => {
    const list = responsesList.value;
    const selectedItem = list.children[selectedResponseIndex.value];
    if (selectedItem) {
        selectedItem.scrollIntoView({
            behavior: 'smooth',
            block: 'nearest'
        });
    }
}

const selectResponse = (content) => {
    contentToSet.value = content
    filteredCannedResponses.value = [];
    selectedResponseIndex.value = -1;
}

const clearContentToSet = () => {
    contentToSet.value = null;
}

</script>

<template>
    <div class="max-h-[600px] overflow-y-auto">
        <EditorContent :editor="editor" />
    </div>
</template>

<script setup>
import { ref, watch, watchEffect, onUnmounted } from "vue"

import { useEditor, EditorContent } from '@tiptap/vue-3'
import Placeholder from "@tiptap/extension-placeholder"
import StarterKit from '@tiptap/starter-kit'

const emit = defineEmits(['send', 'input', 'editorText', 'updateBold', 'updateItalic', 'contentCleared', 'contentSet'])

const props = defineProps({
    placeholder: String,
    messageType: String,
    isBold: Boolean,
    isItalic: Boolean,
    clearContent: Boolean,
    contentToSet: String
})

const editor = ref(useEditor({
    content: '',
    extensions: [
        StarterKit,
        Placeholder.configure({
            placeholder: () => {
                return props.placeholder
            },
        })
    ],
    autofocus: true,
    editorProps: {
        attributes: {
            // No outline for the editor.
            class: "outline-none",
        },
        // Emit new input text.
        handleTextInput: (view, from, to, text) => {
            emit('input', text)
        }
    },
}))

watchEffect(() => {
    if (editor.value) {
        emit("editorText", {
            text: editor.value.getText(),
            html: editor.value.getHTML(),
        });

        // Emit bold and italic state changes
        emit('updateBold', editor.value.isActive('bold'));
        emit('updateItalic', editor.value.isActive('italic'));
    }
})

// Watcher for bold and italic changes
watchEffect(() => {
    if (props.isBold !== editor.value?.isActive('bold')) {
        if (props.isBold) {
            editor.value?.chain().focus().setBold().run();
        } else {
            editor.value?.chain().focus().unsetBold().run();
        }
    }
    if (props.isItalic !== editor.value?.isActive('italic')) {
        if (props.isItalic) {
            editor.value?.chain().focus().setItalic().run();
        } else {
            editor.value?.chain().focus().unsetItalic().run();
        }
    }
});

// Watcher for clearContent prop
watchEffect(() => {
    if (props.clearContent) {
        editor.value?.commands.clearContent()
        emit('contentCleared')
    }
});

watch(() => props.contentToSet, (newContent) => {
    if (newContent) {
        editor.value.commands.setContent(newContent);
        editor.value.commands.focus();
        emit('contentSet');
    }
});

onUnmounted(() => {
    editor.value.destroy()
})
</script>


<style lang="scss">
// Moving placeholder to the top.
.tiptap p.is-editor-empty:first-child::before {
    content: attr(data-placeholder);
    float: left;
    color: #adb5bd;
    pointer-events: none;
    height: 0;
}

// Editor height
.ProseMirror {
    min-height: 150px !important;
    max-height: 100% !important;
    overflow-y: scroll !important;
    padding: 10px 10px;
}
</style>

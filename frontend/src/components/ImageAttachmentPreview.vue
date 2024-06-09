<template>
    <div class="flex flex-wrap items-center group text-left">
        <div class="relative">
            <img :src=attachment.url class="w-36 h-28 flex items-center object-cover" />
            <div class="
                p-1
                absolute
                inset-0
                text-gray-50
                opacity-0
                group-hover:opacity-100
                overlay
                text-wrap
            ">
                <p class="font-bold text-xs">{{ getAttachmentName(attachment.filename) }}</p>
                <p class="text-xs">{{ formatBytes(attachment.size) }}</p>
            </div>
        </div>
    </div>
</template>

<script setup>
import { formatBytes } from '@/utils/file.js'
defineProps({
    attachment: {
        type: Object,
        required: true,
    },
})

const getAttachmentName = (name) => {
    return name.substring(0, 40)
}
</script>
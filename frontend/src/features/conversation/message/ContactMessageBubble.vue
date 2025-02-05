<template>
  <div class="flex flex-col items-start">
    <!-- Sender Name -->
    <div class="pl-[47px] mb-1">
      <p class="text-muted-foreground text-sm font-medium">
        {{ getFullName }}
      </p>
    </div>

    <!-- Message Bubble -->
    <div class="flex flex-row gap-2">
      <!-- Avatar -->
      <Avatar class="cursor-pointer w-8 h-8">
        <AvatarImage :src="getAvatar" />
        <AvatarFallback class="font-medium">
          {{ avatarFallback }}
        </AvatarFallback>
      </Avatar>

      <!-- Message Content -->
      <div
        class="flex flex-col justify-end message-bubble bg-white border border-border rounded-lg p-3 max-w-[80%]"
        :class="{
          '!rounded-tl-none': true,
          'show-quoted-text': showQuotedText,
          'hide-quoted-text': !showQuotedText
        }"
      >
        <!-- Message Text -->
        <Letter
          :html="sanitizedMessageContent"
          :allowedSchemas="['cid', 'https', 'http']"
          class="mb-1"
          :class="{ 'mb-3': message.attachments.length > 0 }"
        />

        <!-- Quoted Text Toggle -->
        <div
          v-if="hasQuotedContent"
          @click="toggleQuote"
          class="text-xs cursor-pointer text-muted-foreground px-2 py-1 w-max hover:bg-muted hover:text-primary rounded-md transition-all"
        >
          {{ showQuotedText ? 'Hide quoted text' : 'Show quoted text' }}
        </div>

        <!-- Attachments -->
        <MessageAttachmentPreview :attachments="nonInlineAttachments" />
      </div>
    </div>

    <!-- Timestamp -->
    <div class="pl-[47px]">
      <Tooltip>
        <TooltipTrigger>
          <span class="text-muted-foreground text-xs mt-1">
            {{ format(message.updated_at, 'h:mm a') }}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>
            {{ format(message.updated_at, "MMMM dd, yyyy 'at' HH:mm") }}
          </p>
        </TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { format } from 'date-fns'
import { useConversationStore } from '@/stores/conversation'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Letter } from 'vue-letter'
import MessageAttachmentPreview from '@/features/conversation/message/attachment/MessageAttachmentPreview.vue'

const props = defineProps({
  message: Object
})

const convStore = useConversationStore()
const showQuotedText = ref(false)

const getAvatar = computed(() => {
  return convStore.current?.avatar_url || ''
})

const sanitizedMessageContent = computed(() => {
  const content = props.message.content || ''
  return props.message.attachments.reduce(
    (acc, { content_id, url }) => acc.replace(new RegExp(`cid:${content_id}`, 'g'), url),
    content
  )
})

const hasQuotedContent = computed(() => sanitizedMessageContent.value.includes('<blockquote'))

const toggleQuote = () => {
  showQuotedText.value = !showQuotedText.value
}

const nonInlineAttachments = computed(() =>
  props.message.attachments.filter((attachment) => attachment.disposition !== 'inline')
)

const getFullName = computed(() => {
  const contact = convStore.current?.contact || {}
  return `${contact.first_name || ''} ${contact.last_name || ''}`.trim()
})

const avatarFallback = computed(() => {
  const contact = convStore.current?.contact || {}
  return (contact.first_name || '').toUpperCase().substring(0, 2)
})
</script>

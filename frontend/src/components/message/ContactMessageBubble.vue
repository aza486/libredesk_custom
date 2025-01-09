<template>
  <div class="flex flex-col items-start">
    <div class="pl-[47px] mb-1">
      <p class="text-muted-foreground text-sm">
        {{ getFullName }}
      </p>
    </div>
    <div class="flex flex-row gap-2">
      <Avatar class="cursor-pointer">
        <AvatarImage :src="getAvatar" />
        <AvatarFallback>
          {{ avatarFallback }}
        </AvatarFallback>
      </Avatar>
      <div class="flex flex-col justify-end message-bubble !rounded-tl-none" :class="{
        'show-quoted-text': showQuotedText,
        'hide-quoted-text': !showQuotedText
      }">
        <Letter :html="sanitizedMessageContent" :allowedSchemas="['cid', 'https', 'http']" class="mb-1"
          :class="{ 'mb-3': message.attachments.length > 0 }" />
        <div v-if="hasQuotedContent" @click="toggleQuote"
          class="text-xs cursor-pointer text-muted-foreground px-2 py-1 w-max hover:bg-muted hover:text-primary rounded-md transition-all">
          {{ showQuotedText ? 'Hide quoted text' : 'Show quoted text' }}
        </div>

        <MessageAttachmentPreview :attachments="nonInlineAttachments" />
      </div>
    </div>
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
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Letter } from 'vue-letter'
import MessageAttachmentPreview from '@/components/attachment/MessageAttachmentPreview.vue'

const props = defineProps({
  message: Object
})

const convStore = useConversationStore()
const showQuotedText = ref(false)

const getAvatar = computed(() => {
  return convStore.current.avatar_url || ''
})

const sanitizedMessageContent = computed(() => {
  const content = props.message.content || ''
  return props.message.attachments.reduce((acc, { content_id, url }) =>
    acc.replace(new RegExp(`cid:${content_id}`, 'g'), url),
    content
  )
})

const hasQuotedContent = computed(() => sanitizedMessageContent.value.includes('<blockquote'))

const toggleQuote = () => {
  showQuotedText.value = !showQuotedText.value
}

const nonInlineAttachments = computed(() =>
  props.message.attachments.filter(attachment => attachment.disposition !== 'inline')
)

const getFullName = computed(() => {
  const contact = convStore.current.contact || {}
  return `${contact.first_name || ''} ${contact.last_name || ''}`.trim()
})

const avatarFallback = computed(() => {
  const contact = convStore.current.contact || {}
  return (contact.first_name || '').toUpperCase().substring(0, 2)
})
</script>

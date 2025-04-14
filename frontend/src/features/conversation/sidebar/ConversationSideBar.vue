<template>
  <div>
    <ConversationSideBarContact class="p-4" />
    <Accordion type="multiple" collapsible v-model="accordionState">
      <AccordionItem value="actions" class="border-0 mb-2">
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded-lg mx-2">
          {{ $t('conversation.sidebar.action', 2) }}
        </AccordionTrigger>

        <!-- `Agent, team, and priority assignment -->
        <AccordionContent class="space-y-4 p-4">
          <!-- Agent assignment -->
          <ComboBox
            v-model="assignedUserID"
            :items="[{ value: 'none', label: 'None' }, ...usersStore.options]"
            :placeholder="t('form.field.selectAgent')"
            @select="selectAgent"
          >
            <template #item="{ item }">
              <div class="flex items-center gap-3 py-2">
                <Avatar class="w-8 h-8">
                  <AvatarImage
                    :src="item.value === 'none' ? '' : item.avatar_url || ''"
                    :alt="item.value === 'none' ? 'N' : item.label.slice(0, 2)"
                  />
                  <AvatarFallback>
                    {{ item.value === 'none' ? 'N' : item.label.slice(0, 2).toUpperCase() }}
                  </AvatarFallback>
                </Avatar>
                <span class="text-sm">{{ item.label }}</span>
              </div>
            </template>

            <template #selected="{ selected }">
              <div class="flex items-center gap-3">
                <Avatar class="w-7 h-7" v-if="selected">
                  <AvatarImage
                    :src="selected?.value === 'none' ? '' : selected?.avatar_url || ''"
                    :alt="selected?.value === 'none' ? 'N' : selected?.label?.slice(0, 2)"
                  />
                  <AvatarFallback>
                    {{
                      selected?.value === 'none' ? 'N' : selected?.label?.slice(0, 2)?.toUpperCase()
                    }}
                  </AvatarFallback>
                </Avatar>
                <span class="text-sm">{{ selected?.label || t('form.field.assignAgent') }}</span>
              </div>
            </template>
          </ComboBox>

          <!-- Team assignment -->
          <ComboBox
            v-model="assignedTeamID"
            :items="[{ value: 'none', label: 'None' }, ...teamsStore.options]"
            :placeholder="t('form.field.selectTeam')"
            @select="selectTeam"
          >
            <template #item="{ item }">
              <div class="flex items-center gap-3 py-2">
                <div class="w-7 h-7 flex items-center justify-center">
                  <span v-if="item.emoji">{{ item.emoji }}</span>
                  <div
                    v-else
                    class="text-primary bg-muted rounded-full w-7 h-7 flex items-center justify-center"
                  >
                    <Users size="14" />
                  </div>
                </div>
                <span class="text-sm">{{ item.label }}</span>
              </div>
            </template>

            <template #selected="{ selected }">
              <div class="flex items-center gap-3">
                <div class="w-7 h-7 flex items-center justify-center" v-if="selected">
                  {{ selected?.emoji }}
                </div>
                <span class="text-sm">{{ selected?.label || t('form.field.assignTeam') }}</span>
              </div>
            </template>
          </ComboBox>

          <!-- Priority assignment -->
          <ComboBox
            v-model="priorityID"
            :items="priorityOptions"
            :placeholder="t('form.field.selectPriority')"
            @select="selectPriority"
          >
            <template #item="{ item }">
              <div class="flex items-center gap-3 py-2">
                <div
                  class="w-7 h-7 flex items-center text-center justify-center bg-muted rounded-full"
                >
                  <component :is="getPriorityIcon(item.value)" size="14" />
                </div>
                <span class="text-sm">{{ item.label }}</span>
              </div>
            </template>

            <template #selected="{ selected }">
              <div class="flex items-center gap-3">
                <div
                  class="w-7 h-7 flex items-center text-center justify-center bg-muted rounded-full"
                  v-if="selected"
                >
                  <component :is="getPriorityIcon(selected?.value)" size="14" />
                </div>
                <span class="text-sm">{{ selected?.label || t('form.field.selectPriority') }}</span>
              </div>
            </template>
          </ComboBox>

          <!-- Tags assignment -->
          <SelectTag
            v-if="conversationStore.current"
            v-model="conversationStore.current.tags"
            :items="tags.map((tag) => ({ label: tag, value: tag }))"
            :placeholder="t('form.field.selectTag', 2)"
          />
        </AccordionContent>
      </AccordionItem>

      <!-- Information -->
      <AccordionItem value="information" class="border-0 mb-2">
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded-lg mx-2">
          {{ $t('conversation.sidebar.information') }}
        </AccordionTrigger>
        <AccordionContent class="p-4">
          <ConversationInfo />
        </AccordionContent>
      </AccordionItem>

      <!-- Contact attributes -->
      <AccordionItem
        value="contact_attributes"
        class="border-0 mb-2"
        v-if="customAttributeStore.contactAttributeOptions.length > 0"
      >
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded-lg mx-2">
          {{ $t('conversation.sidebar.contactAttributes') }}
        </AccordionTrigger>
        <AccordionContent class="p-4">
          <CustomAttributes
            :loading="conversationStore.current.loading"
            :attributes="customAttributeStore.contactAttributeOptions"
            :customAttributes="conversationStore.current?.contact?.custom_attributes || {}"
            @update:setattributes="updateContactCustomAttributes"
          />
        </AccordionContent>
      </AccordionItem>

      <!-- Previous conversations -->
      <AccordionItem value="previous_conversations" class="border-0 mb-2">
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded-lg mx-2">
          {{ $t('conversation.sidebar.previousConvo') }}
        </AccordionTrigger>
        <AccordionContent class="p-4">
          <PreviousConversations />
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@/components/ui/accordion'
import ConversationInfo from './ConversationInfo.vue'
import ConversationSideBarContact from '@/features/conversation/sidebar/ConversationSideBarContact.vue'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'
import { SelectTag } from '@/components/ui/select'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { CircleAlert, SignalLow, SignalMedium, SignalHigh, Users } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '@/stores/customAttributes'
import PreviousConversations from '@/features/conversation/sidebar/PreviousConversations.vue'
import api from '@/api'

const customAttributeStore = useCustomAttributeStore()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tags = ref([])
// Save the accordion state in local storage
const accordionState = useStorage('conversation-sidebar-accordion', [])
const { t } = useI18n()
let isConversationChange = false
customAttributeStore.fetchCustomAttributes()

// Watch for changes in the current conversation and set the flag
watch(
  () => conversationStore.current,
  (newConversation, oldConversation) => {
    // Set the flag when the conversation changes
    if (newConversation?.uuid !== oldConversation?.uuid) {
      isConversationChange = true
    }
  },
  { immediate: true }
)

onMounted(async () => {
  await fetchTags()
})

// Watch for changes in the tags and upsert the tags
watch(
  () => conversationStore.current?.tags,
  (newTags, oldTags) => {
    // Skip if the tags change is due to a conversation change.
    if (isConversationChange) {
      isConversationChange = false
      return
    }

    // Skip if the tags are the same (deep comparison)
    if (
      Array.isArray(newTags) &&
      Array.isArray(oldTags) &&
      newTags.length === oldTags.length &&
      newTags.every((item) => oldTags.includes(item))
    ) {
      return
    }

    conversationStore.upsertTags({
      tags: JSON.stringify(newTags)
    })
  },
  { immediate: false }
)

const assignedUserID = computed(() => String(conversationStore.current?.assigned_user_id))
const assignedTeamID = computed(() => String(conversationStore.current?.assigned_team_id))
const priorityID = computed(() => String(conversationStore.current?.priority_id))
const priorityOptions = computed(() => conversationStore.priorityOptions)

const fetchTags = async () => {
  try {
    const resp = await api.getTags()
    tags.value = resp.data.data.map((item) => item.name)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const handleAssignedUserChange = (id) => {
  conversationStore.updateAssignee('user', {
    assignee_id: id
  })
}

const handleAssignedTeamChange = (id) => {
  conversationStore.updateAssignee('team', {
    assignee_id: id
  })
}

const handleRemoveAssignee = (type) => {
  conversationStore.removeAssignee(type)
}

const handlePriorityChange = (priority) => {
  conversationStore.updatePriority(priority)
}

const selectAgent = (agent) => {
  if (agent.value === 'none') {
    handleRemoveAssignee('user')
    return
  }
  if (conversationStore.current.assigned_user_id == agent.value) return
  conversationStore.current.assigned_user_id = agent.value
  handleAssignedUserChange(agent.value)
}

const selectTeam = (team) => {
  if (team.value === 'none') {
    handleRemoveAssignee('team')
    return
  }
  if (conversationStore.current.assigned_team_id == team.value) return
  conversationStore.current.assigned_team_id = team.value
  handleAssignedTeamChange(team.value)
}

const selectPriority = (priority) => {
  if (conversationStore.current.priority_id == priority.value) return
  conversationStore.current.priority = priority.label
  conversationStore.current.priority_id = priority.value
  handlePriorityChange(priority.label)
}

const getPriorityIcon = (value) => {
  switch (value) {
    case '1':
      return SignalLow
    case '2':
      return SignalMedium
    case '3':
      return SignalHigh
    default:
      return CircleAlert
  }
}

const updateContactCustomAttributes = async (attributes) => {
  let previousAttributes = conversationStore.current.contact.custom_attributes
  try {
    conversationStore.current.contact.custom_attributes = attributes
    await api.updateContactCustomAttribute(conversationStore.current.uuid, attributes)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.attribute')
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
    conversationStore.current.contact.custom_attributes = previousAttributes
  }
}
</script>

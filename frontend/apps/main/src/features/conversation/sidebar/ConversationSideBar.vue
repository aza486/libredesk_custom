<template>
  <div>
    <ConversationSideBarContact class="p-4" />
    <Accordion type="multiple" collapsible v-model="accordionState">
      <AccordionItem value="actions" class="accordion-item">
        <AccordionTrigger class="accordion-trigger">
          {{ $t('globals.terms.action', 2) }}
        </AccordionTrigger>

        <!-- Agent, team, priority, and tags assignment -->
        <AccordionContent class="accordion-content--actions">
          <div>
            <SelectComboBox
              v-model="conversationStore.current.assigned_user_id"
              :items="[{ value: 'none', label: t('globals.terms.none') }, ...usersStore.options]"
              :placeholder="t('placeholders.selectAgent')"
              @select="selectAgent"
              type="user"
            />
          </div>

          <div>
            <SelectComboBox
              v-model="conversationStore.current.assigned_team_id"
              :items="[{ value: 'none', label: t('globals.terms.none') }, ...teamsStore.options]"
              :placeholder="t('placeholders.selectTeam')"
              @select="selectTeam"
              type="team"
            />
          </div>

          <div>
            <SelectComboBox
              v-model="conversationStore.current.priority_id"
              :items="priorityOptions"
              :placeholder="t('placeholders.selectPriority')"
              @select="selectPriority"
              type="priority"
            />
          </div>

          <div>
            <SelectTag
              v-if="conversationStore.current"
              :model-value="conversationStore.current.tags || []"
              @update:modelValue="onTagsChange"
              :items="tags.map((tag) => ({ label: tag, value: tag }))"
              :placeholder="t('placeholders.selectTags')"
            />
          </div>
        </AccordionContent>
      </AccordionItem>

      <AccordionItem
        value="visibility"
        class="accordion-item"
        v-if="conversationStore.current?.custom_attributes?.visible_users"
      >
        <AccordionTrigger class="accordion-trigger">
          Sichtbarkeit
          ({{ conversationStore.current.custom_attributes.visible_users.length }})
        </AccordionTrigger>

        <AccordionContent class="accordion-content">

          <div v-for="userId in sortedVisibleUsers" :key="userId" class="flex justify-between items-center py-1" :class="{'font-medium': userId === conversationStore.current.custom_attributes.creator_id }">
            <span>
              {{ getVisibleUserName(userId) }}

              <span
                v-if="userId === conversationStore.current.custom_attributes.creator_id" class="text-xs text-muted-foreground" >
                (Creator)
              </span>
            </span>

            <button
              v-if="userId !== conversationStore.current.custom_attributes.creator_id"
              @click="removeVisibleUser(userId)"
            >
              ✕
            </button>
          </div>

            <div class="mt-3 space-y-2">

              <UserMultiSelect
                v-model="selectedVisibleUsers"
                :items="availableUsers"
                placeholder="Benutzer auswählen"
              />

              <button
                v-if="selectedVisibleUsers.length"
                class="w-full px-3 py-2 rounded border text-sm"
                @click="addSelectedVisibleUsers"
              >
                {{ selectedVisibleUsers.length }}
                Benutzer hinzufügen
              </button>

            </div>

        </AccordionContent>
      </AccordionItem>

      <!-- Information -->
      <AccordionItem value="information" class="accordion-item">
        <AccordionTrigger class="accordion-trigger">
          {{ $t('conversation.sidebar.information') }}
        </AccordionTrigger>
        <AccordionContent class="accordion-content">
          <ConversationInfo />
        </AccordionContent>
      </AccordionItem>

      <!-- Contact attributes -->
      <AccordionItem
        value="contact_attributes"
        class="accordion-item"
        v-if="customAttributeStore.contactAttributeOptions.length > 0"
      >
        <AccordionTrigger class="accordion-trigger">
          {{ $t('conversation.sidebar.contactAttributes') }}
        </AccordionTrigger>
        <AccordionContent class="accordion-content">
          <CustomAttributes
            :loading="conversationStore.current.loading"
            :attributes="customAttributeStore.contactAttributeOptions"
            :customAttributes="conversationStore.current?.contact?.custom_attributes || {}"
            @update:setattributes="updateContactCustomAttributes"
          />
        </AccordionContent>
      </AccordionItem>

      <!-- Page visits (livechat only) -->
      <AccordionItem
        value="page_visits"
        class="accordion-item"
        v-if="conversationStore.current?.inbox_channel === 'livechat'"
      >
        <AccordionTrigger class="accordion-trigger">
          {{ $t('conversation.sidebar.lastVisitedPages') }}
        </AccordionTrigger>
        <AccordionContent class="accordion-content">
          <ConversationSideBarPageVisits />
        </AccordionContent>
      </AccordionItem>

      <!-- Contact notes -->
      <AccordionItem
        value="contact_notes"
        class="accordion-item"
        v-if="conversationStore.current?.contact?.id && userStore.can('contact_notes:read')"
      >
        <AccordionTrigger class="accordion-trigger">
          {{ $t('globals.terms.note', 2) }}
        </AccordionTrigger>
        <AccordionContent class="accordion-content">
          <ContactNotes :contact-id="conversationStore.current.contact.id" compact />
        </AccordionContent>
      </AccordionItem>

      <!-- Previous conversations -->
      <AccordionItem value="previous_conversations" class="accordion-item">
        <AccordionTrigger class="accordion-trigger">
          {{ $t('conversation.sidebar.previousConvo') }}
        </AccordionTrigger>
        <AccordionContent class="accordion-content">
          <PreviousConversations />
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useTagStore } from '@/stores/tag'
import { useUserStore } from '@/stores/user'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'
import ConversationInfo from './ConversationInfo.vue'
import ConversationSideBarContact from '@/features/conversation/sidebar/ConversationSideBarContact.vue'
import { SelectTag } from '@shared-ui/components/ui/select'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '../../../stores/customAttributes'
import ContactNotes from '@/features/contact/ContactNotes.vue'
import PreviousConversations from '@/features/conversation/sidebar/PreviousConversations.vue'
import ConversationSideBarPageVisits from '@/features/conversation/sidebar/ConversationSideBarPageVisits.vue'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import UserMultiSelect from '@main/components/combobox/UserMultiSelect.vue'
import { TAG_ACTION } from '@/constants/conversation'
import api from '../../../api'

const customAttributeStore = useCustomAttributeStore()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tagStore = useTagStore()
const userStore = useUserStore()
const tags = ref([])
const selectedVisibleUsers = ref([])
const accordionState = useStorage('conversation-sidebar-accordion', [])
const { t } = useI18n()
customAttributeStore.fetchCustomAttributes()

onMounted(async () => {
  await fetchTags()
})

const onTagsChange = (newTags) => {
  const conv = conversationStore.current
  if (!conv) return
  const current = conv.tags || []
  if (newTags.length === current.length && newTags.every((t) => current.includes(t))) return
  conversationStore.updateConversationTags(conv.uuid, TAG_ACTION.SET, newTags)
}

const priorityOptions = computed(() => conversationStore.priorityOptions)

const fetchTags = async () => {
  await tagStore.fetchTags()
  tags.value = tagStore.tags.map((item) => item.name)
}

const handleAssignedUserChange = (id) => {
  conversationStore.updateAssignee('user', {
    assignee_id: parseInt(id)
  })
}

const handleAssignedTeamChange = (id) => {
  conversationStore.updateAssignee('team', {
    assignee_id: parseInt(id)
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
  conversationStore.current.assigned_user_id = agent.value
  handleAssignedUserChange(agent.value)
}

const selectTeam = (team) => {
  if (team.value === 'none') {
    handleRemoveAssignee('team')
    return
  }
  handleAssignedTeamChange(team.value)
}

const selectPriority = (priority) => {
  conversationStore.current.priority = priority.label
  conversationStore.current.priority_id = priority.value
  handlePriorityChange(priority.label)
}

const updateContactCustomAttributes = async (attributes) => {
  let previousAttributes = conversationStore.current.contact.custom_attributes
  try {
    conversationStore.current.contact.custom_attributes = attributes
    await api.updateContactCustomAttribute(conversationStore.current.uuid, attributes)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
    conversationStore.current.contact.custom_attributes = previousAttributes
  }
}

const getVisibleUserName = (userId) => {

  const user = usersStore.options.find(
    u => Number(u.value) === Number(userId)
  )

  return user
    ? `${user.first_name} ${user.last_name}`
    : `User ${userId}`
}

const removeVisibleUser = async (userId) => {

  try {

    await api.removeVisibleUser(
      conversationStore.current.uuid,
      userId
    )

    conversationStore.current.custom_attributes.visible_users =
      conversationStore.current.custom_attributes.visible_users.filter(
        id => Number(id) !== Number(userId)
      )

  } catch (error) {

    emitter.emit(
      EMITTER_EVENTS.SHOW_TOAST,
      {
        variant: 'destructive',
        description: handleHTTPError(error).message
      }
    )
  }
}
const addVisibleUser = async (user) => {

  try {

    const userId = Number(user.value)

    await api.addVisibleUser(
      conversationStore.current.uuid,
      userId
    )

    const visibleUsers =
      conversationStore.current.custom_attributes.visible_users || []

    const exists = visibleUsers.some(
      id => Number(id) === userId
    )

    if (!exists) {
      visibleUsers.push(userId)
    }

    conversationStore.current.custom_attributes.visible_users =
      visibleUsers

  } catch (error) {

    emitter.emit(
      EMITTER_EVENTS.SHOW_TOAST,
      {
        variant: 'destructive',
        description: handleHTTPError(error).message
      }
    )
  }
}

const addSelectedVisibleUsers = async () => {

  for (const user of selectedVisibleUsers.value) {
    await addVisibleUser(user)
  }

  selectedVisibleUsers.value = []
}

const sortedVisibleUsers = computed(() => {
  const visibleUsers = [
    ...(conversationStore.current?.custom_attributes?.visible_users || [])
  ]

  const creatorID =
    conversationStore.current?.custom_attributes?.creator_id

  return visibleUsers.sort((a, b) => {
    if (a === creatorID) return -1
    if (b === creatorID) return 1
    return 0
  })
})

const availableUsers = computed(() => {

  const visibleUsers =
    conversationStore.current?.custom_attributes?.visible_users || []

  return usersStore.options.filter(
    user =>
      !visibleUsers.some(
        id => Number(id) === Number(user.value)
      )
  )
})

</script>

<style scoped>
:deep(.accordion-item) {
  @apply border-0 mb-2;
}

:deep(.accordion-trigger) {
  @apply bg-muted p-2 text-sm font-medium rounded mx-2;
}

:deep(.accordion-content) {
  @apply p-4;
}

:deep(.accordion-content--actions) {
  @apply space-y-3 p-4;
}
</style>

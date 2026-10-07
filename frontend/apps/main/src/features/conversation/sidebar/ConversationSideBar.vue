<template>
  <Tabs v-model="activeTab" class="flex flex-col h-full">
    <TabsList class="grid grid-cols-2 mx-4 mt-3">
      <TabsTrigger value="details" class="text-xs">{{ $t('copilot.details') }}</TabsTrigger>
      <TabsTrigger value="copilot" class="text-xs">{{ COPILOT_NAME }}</TabsTrigger>
    </TabsList>

    <TabsContent value="details" class="mt-0">
      <ConversationSideBarContact class="p-4" />
      <Accordion type="multiple" collapsible v-model="accordionState">
        <AccordionItem value="actions" class="accordion-item">
          <AccordionTrigger class="accordion-trigger">
            {{ $t('globals.terms.action', 2) }}
          </AccordionTrigger>

          <!-- Agent, team, priority, and tags assignment -->
          <AccordionContent class="accordion-content--actions">
            <div v-if="conversationStore.current" class="space-y-2">
              <p class="text-sm font-medium">Zuweisungen</p>
              <UserMultiSelect
                :model-value="selectedAssignees"
                :items="agentOptions"
                placeholder="Mitarbeiter hinzufügen"
                @update:modelValue="updateAssignees"
              />
              <AssignmentSaveState
                :state="assigneeSaveState"
                :can-undo="canUndoAssignees"
                @undo="undoAssignees"
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

            <div v-if="conversationStore.current">
              <SelectTag
                :model-value="conversationStore.current.tags || []"
                @update:modelValue="onTagsChange"
                :items="tags.map((tag) => ({ label: tag, value: tag }))"
                :placeholder="t('placeholders.selectTags')"
              />
              <div class="mt-2 flex flex-wrap items-center gap-1">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  class="h-6 gap-1 px-2 text-xs text-muted-foreground"
                  :disabled="isSuggestingTags"
                  @click="suggestTags"
                >
                  <Loader2 v-if="isSuggestingTags" class="h-3 w-3 animate-spin" />
                  <Sparkles v-else class="h-3 w-3" />
                  {{ t('conversation.sidebar.suggestTags') }}
                </Button>
                <button
                  v-for="suggestion in tagSuggestions"
                  :key="suggestion"
                  type="button"
                  class="inline-flex items-center gap-1 rounded-md bg-accent px-2 py-0.5 text-xs text-accent-foreground hover:bg-accent/80"
                  @click="applySuggestedTag(suggestion)"
                >
                  <Plus class="h-3 w-3" />
                  {{ suggestion }}
                </button>
              </div>
            </div>
          </AccordionContent>
        </AccordionItem>

        <!-- Custom: visibility management -->
        <AccordionItem
          value="visibility"
          class="accordion-item"
          v-if="conversationStore.current?.custom_attributes?.visible_users"
        >
          <AccordionTrigger class="accordion-trigger">
            Sichtbarkeit ({{ selectedVisibleUsers.length }})
          </AccordionTrigger>

          <AccordionContent class="accordion-content">
            <div class="mt-3 space-y-2">
              <UserMultiSelect
                :model-value="selectedVisibleUsers"
                :items="usersStore.options"
                :locked-values="lockedVisibleUserIDs"
                :readonly="!canManageVisibility"
                placeholder="Mitarbeiter hinzufügen"
                @update:modelValue="updateVisibleUsers"
              />
              <AssignmentSaveState
                v-if="canManageVisibility"
                :state="visibilitySaveState"
                :can-undo="canUndoVisibleUsers"
                @undo="undoVisibleUsers"
              />
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
    </TabsContent>

    <TabsContent value="copilot" class="flex-1 min-h-0 mt-0">
      <CopilotPanel />
    </TabsContent>
  </Tabs>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { Sparkles, Loader2, Plus } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useTagStore } from '@/stores/tag'
import { useUserStore } from '@/stores/user'
import { COPILOT_NAME } from '@/constants/copilot'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@shared-ui/components/ui/tabs'
import ConversationInfo from './ConversationInfo.vue'
import ConversationSideBarContact from '@/features/conversation/sidebar/ConversationSideBarContact.vue'
import CopilotPanel from '@/features/conversation/sidebar/CopilotPanel.vue'
import { SelectTag } from '@shared-ui/components/ui/select'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '@/stores/customAttributes'
import ContactNotes from '@/features/contact/ContactNotes.vue'
import PreviousConversations from '@/features/conversation/sidebar/PreviousConversations.vue'
import ConversationSideBarPageVisits from '@/features/conversation/sidebar/ConversationSideBarPageVisits.vue'
import AssignmentSaveState from './AssignmentSaveState.vue'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import UserMultiSelect from '@main/components/combobox/UserMultiSelect.vue'
import { TAG_ACTION } from '@/constants/conversation'
import api from '@/api'

const customAttributeStore = useCustomAttributeStore()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tagStore = useTagStore()
const userStore = useUserStore()
const tags = ref([])
const selectedAssigneeIDs = ref([])
const selectedVisibleUserIDs = ref([])
const originalAssigneeIDs = ref([])
const originalVisibleUserIDs = ref([])
const savedAssigneeIDs = ref([])
const savedVisibleUserIDs = ref([])
const assigneeSaveState = ref('idle')
const visibilitySaveState = ref('idle')
const assigneeSaveQueues = new Map()
const visibilitySaveQueues = new Map()
const savedVisibleUserIDsByUUID = new Map()
let assigneeChangeVersion = 0
let visibilityChangeVersion = 0
const accordionState = useStorage('conversation-sidebar-accordion', [])
const activeTab = useStorage('conversation-sidebar-tab', 'details')
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

const isSuggestingTags = ref(false)
const tagSuggestions = ref([])

// Suggestions belong to one conversation; drop them when the sidebar switches so they never leak.
watch(
  () => conversationStore.current?.uuid,
  () => {
    tagSuggestions.value = []
  }
)

const suggestTags = async () => {
  const conv = conversationStore.current
  if (!conv || isSuggestingTags.value) return
  const uuid = conv.uuid
  isSuggestingTags.value = true
  try {
    const resp = await api.aiSuggestTags({ conversation_uuid: uuid })
    if (conversationStore.current?.uuid !== uuid) return
    const current = conversationStore.current.tags || []
    const suggestions = (resp.data.data || []).filter((tag) => !current.includes(tag))
    if (suggestions.length === 0) {
      tagSuggestions.value = []
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('conversation.sidebar.noTagSuggestions')
      })
      return
    }
    tagSuggestions.value = suggestions
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSuggestingTags.value = false
  }
}

const applySuggestedTag = (tag) => {
  const conv = conversationStore.current
  if (!conv) return
  const current = conv.tags || []
  if (!current.includes(tag)) onTagsChange([...current, tag])
  tagSuggestions.value = tagSuggestions.value.filter((suggestion) => suggestion !== tag)
}

const priorityOptions = computed(() => conversationStore.priorityOptions)

const agentOptions = computed(() => {
  const isMe = (option) => String(option.value) === String(userStore.userID)
  const me = usersStore.options.find(isMe)
  if (!me) return usersStore.options
  return [me, ...usersStore.options.filter((option) => !isMe(option))]
})

const normalizeUserIDs = (ids) => [...new Set(ids.map(Number).filter((id) => id > 0))]
const sameUserIDs = (left, right) => {
  const a = normalizeUserIDs(left).sort((x, y) => x - y)
  const b = normalizeUserIDs(right).sort((x, y) => x - y)
  return a.length === b.length && a.every((id, index) => id === b[index])
}
const usersForIDs = (ids) =>
  normalizeUserIDs(ids).map(
    (id) =>
      usersStore.options.find((user) => Number(user.value) === id) || {
        value: id,
        label: `User ${id}`
      }
  )
const selectedAssignees = computed(() => usersForIDs(selectedAssigneeIDs.value))

const isPersonalInboxConversation = (conversation) =>
  conversation?.inbox_access_mode === 'personal' ||
  conversation?.custom_attributes?.access_mode === 'personal'

const personalOwnerIDs = computed(() => {
  const conversation = conversationStore.current
  if (!isPersonalInboxConversation(conversation)) return []
  const ownerIDs = conversation.inbox_owner_user_ids?.length
    ? conversation.inbox_owner_user_ids
    : [conversation.inbox_owner_user_id, conversation.custom_attributes?.owner_user_id]
  return normalizeUserIDs(ownerIDs)
})

const selectedVisibleUsers = computed(() => {
  const conversation = conversationStore.current
  let visibleUserIDs = selectedVisibleUserIDs.value
  const legacyOwnerID = Number(conversation?.custom_attributes?.owner_user_id)
  if (
    personalOwnerIDs.value.length > 0 &&
    legacyOwnerID > 0 &&
    !personalOwnerIDs.value.includes(legacyOwnerID)
  ) {
    visibleUserIDs = visibleUserIDs.filter((id) => Number(id) !== legacyOwnerID)
  }
  return usersForIDs([...visibleUserIDs, ...personalOwnerIDs.value])
})

watch(
  () => conversationStore.current?.uuid,
  () => {
    const conversation = conversationStore.current
    if (!conversation) {
      selectedAssigneeIDs.value = []
      selectedVisibleUserIDs.value = []
      assigneeSaveState.value = 'idle'
      visibilitySaveState.value = 'idle'
      return
    }
    const assigneeIDs = conversation.assigned_user_ids?.length
      ? conversation.assigned_user_ids
      : conversation.assigned_user_id
        ? [conversation.assigned_user_id]
        : []
    const visibleUserIDs = conversation.custom_attributes?.visible_users || []
    selectedAssigneeIDs.value = normalizeUserIDs(assigneeIDs)
    selectedVisibleUserIDs.value = normalizeUserIDs(visibleUserIDs)
    originalAssigneeIDs.value = [...selectedAssigneeIDs.value]
    originalVisibleUserIDs.value = [...selectedVisibleUserIDs.value]
    savedAssigneeIDs.value = [...selectedAssigneeIDs.value]
    savedVisibleUserIDs.value = [...selectedVisibleUserIDs.value]
    savedVisibleUserIDsByUUID.set(conversation.uuid, [...selectedVisibleUserIDs.value])
    assigneeSaveState.value = 'idle'
    visibilitySaveState.value = 'idle'
    assigneeChangeVersion += 1
    visibilityChangeVersion += 1
  },
  { immediate: true }
)

const fetchTags = async () => {
  await tagStore.fetchTags()
  tags.value = tagStore.tags.map((item) => item.name)
}

const canUndoAssignees = computed(
  () => !sameUserIDs(selectedAssigneeIDs.value, originalAssigneeIDs.value)
)

const updateAssignees = (users) => {
  const ids = normalizeUserIDs(users.map((user) => user.value))
  if (sameUserIDs(ids, selectedAssigneeIDs.value)) return
  selectedAssigneeIDs.value = ids
  saveAssignees(ids)
}

const saveAssignees = (assigneeIDs) => {
  const conversation = conversationStore.current
  if (!conversation) return
  const uuid = conversation.uuid
  const ids = normalizeUserIDs(assigneeIDs)
  const version = ++assigneeChangeVersion
  assigneeSaveState.value = 'saving'
  const previous = assigneeSaveQueues.get(uuid) || Promise.resolve()
  const request = previous
    .catch(() => {})
    .then(async () => {
      try {
        await api.setUserAssignees(uuid, ids)
        if (conversationStore.current?.uuid !== uuid) return
        savedAssigneeIDs.value = [...ids]
        conversationStore.current.assigned_user_ids = [...ids]
        conversationStore.current.assigned_user_id = ids[0] || null
        if (version === assigneeChangeVersion) assigneeSaveState.value = 'saved'
      } catch (error) {
        if (conversationStore.current?.uuid !== uuid || version !== assigneeChangeVersion) return
        selectedAssigneeIDs.value = [...savedAssigneeIDs.value]
        conversationStore.current.assigned_user_ids = [...savedAssigneeIDs.value]
        conversationStore.current.assigned_user_id = savedAssigneeIDs.value[0] || null
        assigneeSaveState.value = 'error'
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    })
  assigneeSaveQueues.set(uuid, request)
}

const undoAssignees = () => {
  if (!canUndoAssignees.value) return
  selectedAssigneeIDs.value = [...originalAssigneeIDs.value]
  saveAssignees(selectedAssigneeIDs.value)
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

const assignedUserIDs = computed(() => {
  return selectedAssigneeIDs.value
})

const visibilityManagerIDs = computed(() => {
  const attrs = conversationStore.current?.custom_attributes || {}
  const managers = attrs.visibility_managers || []
  return attrs.creator_id ? [...managers, attrs.creator_id] : managers
})

const isVisibilityManager = (userID) =>
  visibilityManagerIDs.value.some((id) => Number(id) === Number(userID))

const lockedVisibleUserIDs = computed(() => [
  ...personalOwnerIDs.value,
  ...visibilityManagerIDs.value,
  ...assignedUserIDs.value
])

const canManageVisibility = computed(
  () =>
    userStore.roles.includes('Admin') ||
    personalOwnerIDs.value.some((id) => Number(id) === Number(userStore.userID)) ||
    (conversationStore.current?.custom_attributes?.customer_visibility === true &&
      userStore.roles.includes('Kundensupport')) ||
    isVisibilityManager(userStore.userID)
)

const canUndoVisibleUsers = computed(
  () => !sameUserIDs(selectedVisibleUserIDs.value, originalVisibleUserIDs.value)
)

const updateVisibleUsers = (users) => {
  const ownerIDs = personalOwnerIDs.value
  const ids = normalizeUserIDs(users.map((user) => user.value)).filter(
    (id) =>
      !ownerIDs.includes(id) ||
      selectedVisibleUserIDs.value.some((visibleID) => Number(visibleID) === id)
  )
  if (sameUserIDs(ids, selectedVisibleUserIDs.value)) return
  selectedVisibleUserIDs.value = ids
  if (conversationStore.current?.custom_attributes) {
    conversationStore.current.custom_attributes.visible_users = [...ids]
  }
  saveVisibleUsers(ids)
}

const saveVisibleUsers = (visibleUserIDs) => {
  const conversation = conversationStore.current
  if (!conversation) return
  const uuid = conversation.uuid
  const ids = normalizeUserIDs(visibleUserIDs)
  const version = ++visibilityChangeVersion
  visibilitySaveState.value = 'saving'
  const previous = visibilitySaveQueues.get(uuid) || Promise.resolve()
  const request = previous
    .catch(() => {})
    .then(async () => {
      try {
        const currentSavedIDs = [...(savedVisibleUserIDsByUUID.get(uuid) || [])]
        for (const userID of currentSavedIDs.filter((id) => !ids.includes(id))) {
          await api.removeVisibleUser(uuid, userID)
          const savedIDs = savedVisibleUserIDsByUUID.get(uuid) || []
          savedVisibleUserIDsByUUID.set(
            uuid,
            savedIDs.filter((id) => id !== userID)
          )
        }
        const savedAfterRemovals = savedVisibleUserIDsByUUID.get(uuid) || []
        for (const userID of ids.filter((id) => !savedAfterRemovals.includes(id))) {
          await api.addVisibleUser(uuid, userID)
          savedVisibleUserIDsByUUID.set(uuid, [
            ...(savedVisibleUserIDsByUUID.get(uuid) || []),
            userID
          ])
        }
        if (conversationStore.current?.uuid === uuid && version === visibilityChangeVersion) {
          savedVisibleUserIDs.value = [...ids]
          savedVisibleUserIDsByUUID.set(uuid, [...ids])
          visibilitySaveState.value = 'saved'
        }
      } catch (error) {
        if (conversationStore.current?.uuid !== uuid || version !== visibilityChangeVersion) return
        const savedIDs = savedVisibleUserIDsByUUID.get(uuid) || []
        selectedVisibleUserIDs.value = [...savedIDs]
        savedVisibleUserIDs.value = [...savedIDs]
        conversationStore.current.custom_attributes.visible_users = [...savedIDs]
        visibilitySaveState.value = 'error'
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    })
  visibilitySaveQueues.set(uuid, request)
}

const undoVisibleUsers = () => {
  if (!canUndoVisibleUsers.value) return
  selectedVisibleUserIDs.value = [...originalVisibleUserIDs.value]
  conversationStore.current.custom_attributes.visible_users = [...originalVisibleUserIDs.value]
  saveVisibleUsers(selectedVisibleUserIDs.value)
}
</script>

<style scoped>
:deep(.accordion-item) {
  @apply border-0 mb-2;
}

:deep(.accordion-trigger) {
  @apply bg-muted p-2 text-sm font-medium rounded-md mx-2;
}

:deep(.accordion-content) {
  @apply p-4;
}

:deep(.accordion-content--actions) {
  @apply space-y-3 px-4 pt-4 pb-0;
}
</style>

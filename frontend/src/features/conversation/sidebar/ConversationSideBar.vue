<template>
  <div>
    <ConversationSideBarContact class="p-4" />
    <Accordion
      type="multiple"
      collapsible
      :default-value="['Actions', 'Information', 'Previous conversations']"
    >
      <AccordionItem value="Actions" class="border-0 mb-2 mb-2">
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded-lg mx-2">
          Actions
        </AccordionTrigger>
        <AccordionContent class="space-y-4 p-4">
          <ComboBox
            v-model="assignedUserID"
            :items="[{ value: 'none', label: 'None' }, ...usersStore.options]"
            placeholder="Search agent"
            defaultLabel="Assign agent"
            @select="selectAgent"
          >
            <template #item="{ item }">
              <div class="flex items-center gap-3 py-2">
                <Avatar class="w-8 h-8">
                  <AvatarImage
                    :src="item.value === 'none' ? '/default-avatar.png' : item.avatar_url"
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
                    :src="selected?.value === 'none' ? '/default-avatar.png' : selected?.avatar_url"
                    :alt="selected?.value === 'none' ? 'N' : selected?.label?.slice(0, 2)"
                  />
                  <AvatarFallback>
                    {{
                      selected?.value === 'none' ? 'N' : selected?.label?.slice(0, 2)?.toUpperCase()
                    }}
                  </AvatarFallback>
                </Avatar>
                <span class="text-sm">{{ selected?.label || 'Assign agent' }}</span>
              </div>
            </template>
          </ComboBox>

          <ComboBox
            v-model="assignedTeamID"
            :items="[{ value: 'none', label: 'None' }, ...teamsStore.options]"
            placeholder="Search team"
            defaultLabel="Assign team"
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
              <div class="flex items-center gap-3" v-if="selected">
                <div class="w-7 h-7 flex items-center justify-center">
                  {{ selected?.emoji }}
                </div>
                <span class="text-sm">{{ selected?.label || 'Select team' }}</span>
              </div>
            </template>
          </ComboBox>

          <ComboBox
            v-model="priorityID"
            :items="priorityOptions"
            :defaultLabel="'Select priority'"
            placeholder="Select priority"
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
                <span class="text-sm">{{ selected?.label || 'Select priority' }}</span>
              </div>
            </template>
          </ComboBox>

          <SelectTag
            v-if="conversationStore.current"
            v-model="conversationStore.current.tags"
            :items="tags"
            placeholder="Select tags"
          />
        </AccordionContent>
      </AccordionItem>

      <AccordionItem value="Information" class="border-0 mb-2">
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded-lg mx-2">
          Information
        </AccordionTrigger>
        <AccordionContent class="p-4">
          <ConversationInfo />
        </AccordionContent>
      </AccordionItem>

      <AccordionItem value="Previous conversations" class="border-0 mb-2">
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded-lg mx-2">
          Previous conversations
        </AccordionTrigger>
        <AccordionContent class="p-4">
          <div
            v-if="
              conversationStore.current?.previous_conversations?.length === 0 ||
              conversationStore.conversation?.loading
            "
            class="text-center text-sm text-muted-foreground py-4"
          >
            No previous conversations
          </div>
          <div v-else class="space-y-3">
            <router-link
              v-for="conversation in conversationStore.current.previous_conversations"
              :key="conversation.uuid"
              :to="{
                name: 'inbox-conversation',
                params: {
                  uuid: conversation.uuid,
                  type: 'assigned'
                }
              }"
              class="block p-2 rounded-md hover:bg-muted"
            >
              <div class="flex items-center justify-between">
                <div class="flex flex-col">
                  <span class="font-medium text-sm">
                    {{ conversation.contact.first_name }} {{ conversation.contact.last_name }}
                  </span>
                  <span class="text-xs text-muted-foreground truncate max-w-[200px]">
                    {{ conversation.last_message }}
                  </span>
                </div>
                <span class="text-xs text-muted-foreground" v-if="conversation.last_message_at">
                  {{ format(new Date(conversation.last_message_at), 'h') + ' h' }}
                </span>
              </div>
            </router-link>
          </div>
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
import { format } from 'date-fns'
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
import { useToast } from '@/components/ui/toast/use-toast'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'
import { CircleAlert, SignalLow, SignalMedium, SignalHigh, Users } from 'lucide-vue-next'

const { toast } = useToast()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tags = ref([])
let isConversationChange = false

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
    toast({
      title: 'Error',
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
</script>

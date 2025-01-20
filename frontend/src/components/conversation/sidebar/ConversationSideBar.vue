<template>
  <div v-if="conversationStore.current">
    <ConversationSideBarContact :conversation="conversationStore.current" class="p-3" />
    <Accordion type="multiple" collapsible class="border-t" :default-value="['Actions', 'Information']">
      <AccordionItem value="Actions">
        <AccordionTrigger class="bg-accent p-2"> Actions </AccordionTrigger>
        <AccordionContent class="space-y-5 p-3">
          <!-- Agent -->
          <ComboBox
            v-model="assignedUserID"
            :items="usersStore.options"
            placeholder="Search agent"
            defaultLabel="Assign agent"
            @select="selectAgent"
          >
            <template #item="{ item }">
              <div class="flex items-center gap-2">
                <Avatar class="w-8 h-8">
                  <AvatarImage :src="item.avatar_url ?? ''" :alt="item.label.slice(0, 2)" />
                  <AvatarFallback>
                    {{ item.label.slice(0, 2).toUpperCase() }}
                  </AvatarFallback>
                </Avatar>
                <span>{{ item.label }}</span>
              </div>
            </template>

            <template #selected="{ selected }">
              <div v-if="selected" class="flex items-center gap-2">
                <Avatar class="w-7 h-7">
                  <AvatarImage :src="selected.avatar_url ?? ''" :alt="selected.label.slice(0, 2)" />
                  <AvatarFallback>
                    {{ selected.label.slice(0, 2).toUpperCase() }}
                  </AvatarFallback>
                </Avatar>
                <span>{{ selected.label }}</span>
              </div>
              <span v-else>Select user</span>
            </template>
          </ComboBox>

          <!-- Team -->
          <ComboBox
            v-model="assignedTeamID"
            :items="teamsStore.options"
            placeholder="Search team"
            defaultLabel="Assign team"
            @select="selectTeam"
          >
            <template #item="{ item }">
              <div class="flex items-center gap-2 ml-2">
                {{ item.emoji }}
                <span>{{ item.label }}</span>
              </div>
            </template>

            <template #selected="{ selected }">
              <div v-if="selected" class="flex items-center gap-2">
                {{ selected.emoji }}
                <span>{{ selected.label }}</span>
              </div>
              <span v-else>Select team</span>
            </template>
          </ComboBox>

          <!-- Priority  -->
          <ComboBox
            v-model="conversationStore.current.priority"
            :items="conversationStore.priorityOptions"
            :defaultLabel="conversationStore.current.priority ?? 'Select priority'"
            placeholder="Select priority"
            @select="selectPriority"
          />

          <!-- Tags -->
          <SelectTag
            v-model="conversationStore.current.tags"
            :items="tags"
            placeholder="Select tags"
          />
        </AccordionContent>
      </AccordionItem>
      <AccordionItem value="Information">
        <AccordionTrigger class="bg-accent p-2"> Information </AccordionTrigger>
        <AccordionContent class="space-y-5 p-3">
          <ConversationInfo :conversation="conversationStore.current"></ConversationInfo>
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
import ConversationSideBarContact from '@/components/conversation/sidebar/ConversationSideBarContact.vue'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'
import { SelectTag } from '@/components/ui/select'
import { useToast } from '@/components/ui/toast/use-toast'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const { toast } = useToast()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tags = ref([])

onMounted(async () => {
  await fetchTags()
})

watch(
  () => conversationStore.current?.tags,
  () => {
    if (!conversationStore.current?.tags) return
    conversationStore.upsertTags({
      tags: JSON.stringify(conversationStore.current.tags)
    })
  }
)

const assignedUserID = computed(() => String(conversationStore.current.assigned_user_id))
const assignedTeamID = computed(() => String(conversationStore.current.assigned_team_id))

const fetchTags = async () => {
  try {
    const resp = await api.getTags()
    resp.data.data.forEach((item) => {
      tags.value.push(item.name)
    })
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

const handlePriorityChange = (priority) => {
  conversationStore.updatePriority(priority)
}

const selectAgent = (agent) => {
  conversationStore.current.assigned_user_id = agent.value
  handleAssignedUserChange(agent.value)
}

const selectTeam = (team) => {
  conversationStore.current.assigned_team_id = team.value
  handleAssignedTeamChange(team.value)
}

const selectPriority = (priority) => {
  conversationStore.current.priority = priority.label
  handlePriorityChange(priority.label)
}
</script>

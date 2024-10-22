<template>
  <div class="p-3">
    <ConversationSideBarContact :conversation="conversationStore.current"></ConversationSideBarContact>
    <Accordion type="multiple" collapsible class="border-t mt-4" :default-value="['Actions', 'Information']">
      <AccordionItem value="Actions">
        <AccordionTrigger>
          <h4 class="scroll-m-20 text-base font-medium tracking-tight">
            Actions
          </h4>
        </AccordionTrigger>
        <AccordionContent class="space-y-5">
          <!-- Agent -->
          <AssignAgent :agents="agents" :conversation="conversationStore.current" :selectAgent="selectAgent">
          </AssignAgent>
          <!-- Team -->
          <AssignTeam :teams="teams" :conversation="conversationStore.current" :selectTeam="selectTeam">
          </AssignTeam>
          <!-- Priority  -->
          <PriorityChange :priorities="priorities" :conversation="conversationStore.current"
            :selectPriority="selectPriority"></PriorityChange>
          <!-- Tags -->
          <SelectTag v-model="conversationStore.current.tags" :items="tags" placeHolder="Select tags"></SelectTag>
        </AccordionContent>
      </AccordionItem>
      <AccordionItem value="Information">
        <AccordionTrigger>
          <span class="scroll-m-20 text-base font-medium tracking-tight">
            Information
          </span>
        </AccordionTrigger>
        <AccordionContent>
          <ConversationInfo :conversation="conversationStore.current"></ConversationInfo>
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import api from '@/api'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@/components/ui/accordion'
import ConversationInfo from './ConversationInfo.vue'
import ConversationSideBarContact from '@/components/conversation/sidebar/ConversationSideBarContact.vue'
import AssignAgent from './AssignAgent.vue'
import AssignTeam from './AssignTeam.vue'
import PriorityChange from './PriorityChange.vue'
import { SelectTag } from '@/components/ui/select'
import { useToast } from '@/components/ui/toast/use-toast'
import { handleHTTPError } from '@/utils/http'

const { toast } = useToast()
const conversationStore = useConversationStore()
const priorities = ref([])
const agents = ref([])
const teams = ref([])
const tags = ref([])
const tagIDMap = {}

onMounted(async () => {
  await Promise.all([
    fetchUsers(),
    fetchTeams(),
    fetchTags(),
    getPrioritites()
  ])
})

watch(() => conversationStore.current.tags, () => {
  handleUpsertTags()
}, { deep: true })

const handleUpsertTags = () => {
  let tagIDs = conversationStore.current.tags.map((tag) => {
    if (tag in tagIDMap) {
      return tagIDMap[tag]
    }
  })
  conversationStore.upsertTags({
    tag_ids: JSON.stringify(tagIDs)
  })
}

const fetchUsers = async () => {
  try {
    const resp = await api.getUsersCompact()
    agents.value = resp.data.data
  } catch (error) {
    toast({
      title: 'Could not fetch users',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const fetchTeams = async () => {
  try {
    const resp = await api.getTeamsCompact()
    teams.value = resp.data.data
  } catch (error) {
    toast({
      title: 'Could not fetch teams',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const fetchTags = async () => {
  try {
    const resp = await api.getTags()
    resp.data.data.forEach(item => {
      tagIDMap[item.name] = item.id
      tags.value.push(item.name)
    })
  } catch (error) {
    toast({
      title: 'Could not fetch tags',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const getPrioritites = async () => {
  const resp = await api.getPriorities()
  priorities.value = resp.data.data.map((priority) => priority.name)
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

const selectAgent = (id) => {
  conversationStore.current.assigned_user_id = id
  handleAssignedUserChange(id)
}

const selectTeam = (id) => {
  conversationStore.current.assigned_team_id = id
  handleAssignedTeamChange(id)
}

const selectPriority = (priority) => {
  conversationStore.current.priority = priority
  handlePriorityChange(priority)
}
</script>

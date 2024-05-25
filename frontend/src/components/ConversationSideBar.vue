<template>
  <div class="p-3">
    <div>
      <Avatar class="size-20">
        <AvatarImage :src=conversationStore.conversation.data.contact_avatar_url />
        <AvatarFallback>
          {{ conversationStore.conversation.data.contact_first_name.toUpperCase().substring(0, 2) }}
        </AvatarFallback>
      </Avatar>
      <h4 class="text-l ">
        {{ conversationStore.conversation.data.contact_first_name + ' ' +
          conversationStore.conversation.data.contact_last_name }}
      </h4>
      <p class="text-sm text-muted-foreground flex gap-2 mt-1" v-if="conversationStore.conversation.data.contact_email">
        <Mail class="size-3 mt-1"></Mail>
        {{ conversationStore.conversation.data.contact_email }}
      </p>
      <p class="text-sm text-muted-foreground flex gap-2 mt-1"
        v-if="conversationStore.conversation.data.contact_phone_number">
        <Phone class="size-3 mt-1"></Phone>
        {{ conversationStore.conversation.data.contact_phone_number }}
      </p>
    </div>
    <Accordion type="single" collapsible class="border-t mt-4">
      <AccordionItem :value="actionAccordion.title">
        <AccordionTrigger>
          <p>{{ actionAccordion.title }}</p>
        </AccordionTrigger>
        <AccordionContent>

          <!-- Assigned agent -->
          <!-- <Select v-model="conversationStore.conversation.data.assigned_agent_uuid"
            @update:modelValue="handleAssignedAgentChange" id="select-agent">
            <SelectTrigger class="mb-3">
              <SelectValue placeholder="Assigned agent" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectLabel>Assigned agent</SelectLabel>
                <SelectItem :value="agent.uuid" v-for="(agent) in agents" :key="agent.uuid">
                  {{ agent.first_name + ' ' + agent.last_name }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select> -->


          <div class="mb-3">
            <Popover v-model:open="agentSelectDropdownOpen">
              <PopoverTrigger as-child>
                <Button variant="outline" role="combobox" :aria-expanded="agentSelectDropdownOpen"
                  class="w-full justify-between">
                  {{ conversationStore.conversation.data.assigned_agent_uuid
                    ? agents.find((agent) => agent.uuid ===
                      conversationStore.conversation.data.assigned_agent_uuid)?.first_name + ' ' + agents.find((agent) =>
                        agent.uuid === conversationStore.conversation.data.assigned_agent_uuid)?.last_name
                    : "Select agent..." }}

                  <CaretSortIcon class="ml-2 h-4 w-4 shrink-0 opacity-50" />
                </Button>
              </PopoverTrigger>
              <PopoverContent class="p-0 PopoverContent">
                <Command @update:modelValue="handleAssignedAgentChange">
                  <CommandInput class="h-9" placeholder="Search agent..." />
                  <CommandEmpty>No agent found.</CommandEmpty>
                  <CommandList>
                    <CommandGroup>
                      <CommandItem v-for="agent in agents" :key="agent.uuid"
                        :value="agent.first_name + ' ' + agent.last_name" @select="(ev) => {
                          if (typeof ev.detail.value === 'string') {
                            conversationStore.conversation.data.assigned_agent_uuid = ev.detail.value
                          }
                          agentSelectDropdownOpen = false
                        }">
                        {{ agent.first_name + ' ' + agent.last_name }}
                        <CheckIcon :class="cn(
                          'ml-auto h-4 w-4',
                          conversationStore.conversation.data.assigned_agent_uuid === agent.uuid ? 'opacity-100' : 'opacity-0',
                        )" />
                      </CommandItem>
                    </CommandGroup>
                  </CommandList>
                </Command>
              </PopoverContent>
            </Popover>
          </div>

          <!-- Assigned agent end -->

          <!-- Assigned team  -->
          <!-- <Select v-model="conversationStore.conversation.data.assigned_team_uuid"
            @update:modelValue="handleAssignedTeamChange">
            <SelectTrigger class="mb-3">
              <SelectValue placeholder="Assigned team" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectLabel>Assigned team</SelectLabel>
                <SelectItem :value="team.uuid" v-for="(team) in teams" :key="team.uuid">
                  {{ team.name }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select> -->

          <div class="mb-3">
            <Popover v-model:open="teamSelectDropdownOpen">
              <PopoverTrigger as-child>
                <Button variant="outline" role="combobox" :aria-expanded="teamSelectDropdownOpen"
                  class="w-full justify-between">
                  {{ conversationStore.conversation.data.assigned_team_uuid
                    ? teams.find((team) => team.uuid ===
                      conversationStore.conversation.data.assigned_team_uuid)?.name
                    : "Select team..." }}
                  <CaretSortIcon class="ml-2 h-4 w-4 shrink-0 opacity-50" />
                </Button>
              </PopoverTrigger>
              <PopoverContent class="p-0 PopoverContent">
                <Command @update:modelValue="handleAssignedTeamChange">
                  <CommandInput class="h-9" placeholder="Search team..." />
                  <CommandEmpty>No team found.</CommandEmpty>
                  <CommandList>
                    <CommandGroup>
                      <CommandItem v-for="team in teams" :key="team.uuid" :value="team.name" @select="(ev) => {
                        if (ev.detail.value) {
                          const selectedTeamName = ev.detail.value;
                          const selectedTeam = teams.find(team => team.name === selectedTeamName);
                          if (selectedTeam) {
                            conversationStore.conversation.data.assigned_team_uuid = selectedTeam.uuid;
                          }
                        }
                        teamSelectDropdownOpen = false
                      }">
                        {{ team.name }}
                        <CheckIcon :class="cn(
                          'ml-auto h-4 w-4',
                          conversationStore.conversation.data.assigned_team_uuid === team.uuid ? 'opacity-100' : 'opacity-0',
                        )" />
                      </CommandItem>
                    </CommandGroup>
                  </CommandList>
                </Command>
              </PopoverContent>
            </Popover>
          </div>

          <!-- assigned team end -->

          <!-- Priority  -->
          <Select v-model="conversationStore.conversation.data.priority" @update:modelValue="handlePriorityChange">
            <SelectTrigger class="mb-3">
              <SelectValue placeholder="Priority" class="font-medium" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectLabel>Priority</SelectLabel>
                <SelectItem value="Low">
                  Low
                </SelectItem>
                <SelectItem value="Medium">
                  Medium
                </SelectItem>
                <SelectItem value="High">
                  High
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
          <!-- Priority -->

          <!-- Tags -->
          <TagsInput class="px-0 gap-0 w-full" :model-value="tagsSelected" @update:modelValue="handleUpsertTags">
            <div class="flex gap-2 flex-wrap items-center px-3">
              <TagsInputItem v-for="item in tagsSelected" :key="item" :value="item">
                <TagsInputItemText />
                <TagsInputItemDelete />
              </TagsInputItem>
            </div>

            <ComboboxRoot v-model="tagsSelected" v-model:open="tagDropdownOpen" v-model:searchTerm="tagSearchTerm"
              class="w-full">
              <ComboboxAnchor as-child>
                <ComboboxInput placeholder="Add tags..." as-child>
                  <TagsInputInput class="w-full px-3" :class="tagsSelected.length > 0 ? 'mt-2' : ''"
                    @keydown.enter.prevent />
                </ComboboxInput>
              </ComboboxAnchor>

              <ComboboxPortal>
                <CommandList position="popper"
                  class="w-[--radix-popper-anchor-width] rounded-md mt-2 border bg-popover text-popover-foreground shadow-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2">
                  <CommandEmpty />
                  <CommandGroup>
                    <CommandItem v-for="ftag in tagsFiltered" :key="ftag.value" :value="ftag.label" @select.prevent="(ev) => {
                      if (typeof ev.detail.value === 'string') {
                        tagSearchTerm = ''
                        tagsSelected.push(ev.detail.value)
                        tagDropdownOpen = false
                      }

                      if (tagsFiltered.length === 0) {
                        tagDropdownOpen = false
                      }
                    }">
                      {{ ftag.label }}
                    </CommandItem>
                  </CommandGroup>
                </CommandList>
              </ComboboxPortal>
            </ComboboxRoot>
          </TagsInput>
          <!-- Tags end -->

        </AccordionContent>
      </AccordionItem>
    </Accordion>



    <Accordion type="single" collapsible>
      <AccordionItem :value="infoAccordion.title">
        <AccordionTrigger>
          <p>{{ infoAccordion.title }}</p>
        </AccordionTrigger>
        <AccordionContent>

          <div class="flex flex-col gap-1 mb-5">
            <p class="font-medium">Initiated at</p>
            <p>
              {{ format(conversationStore.conversation.data.created_at, "PPpp") }}
            </p>
          </div>

          <div class="flex flex-col gap-1 mb-5">
            <p class="font-medium">
              Resolved at
            </p>
            <p v-if="conversationStore.conversation.data.resolved_at">
              {{ format(conversationStore.conversation.data.resolved_at, "PPpp") }}
            </p>
            <p v-else>
              -
            </p>
          </div>

          <div class="flex flex-col gap-1 mb-5">
            <p class="font-medium">
              Closed at
            </p>
            <p v-if="conversationStore.conversation.data.closed_at">
              {{ format(conversationStore.conversation.data.closed_at, "PPpp") }}
            </p>
            <p v-else>
              -
            </p>
          </div>


          <div class="flex flex-col gap-1 mb-5">
            <p class="font-medium">SLA</p>
            <p>48 hours remaining</p>
          </div>

        </AccordionContent>
      </AccordionItem>
    </Accordion>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { format } from 'date-fns'
import api from '@/api';

import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { CaretSortIcon, CheckIcon } from '@radix-icons/vue'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { ComboboxAnchor, ComboboxInput, ComboboxPortal, ComboboxRoot } from 'radix-vue'
import { CommandEmpty, CommandGroup, CommandInput, Command, CommandItem, CommandList } from '@/components/ui/command'
import { TagsInput, TagsInputInput, TagsInputItem, TagsInputItemDelete, TagsInputItemText } from '@/components/ui/tags-input'
import { Mail, Phone } from "lucide-vue-next"

// Stores, states.
const conversationStore = useConversationStore();
const agents = ref([])
const teams = ref([])
const agentSelectDropdownOpen = ref(false)
const teamSelectDropdownOpen = ref(false)
const tags = ref([])
const tagIDMap = {}
const tagsSelected = ref(conversationStore.conversation.data.tags)
const tagDropdownOpen = ref(false)
const tagSearchTerm = ref('')
const teamSearchTerm = ref('')
const tagsFiltered = computed(() => tags.value.filter(i => !tagsSelected.value.includes(i.label)))
// const agentsFiltered = computed(() => tags.value.filter(i => !tagsSelected.value.includes(i.label)))

const actionAccordion = {
  "title": "Action"
}
const infoAccordion = {
  "title": "Information"
}

// Functions, methods.
onMounted(() => {
  api.getAgents().then((resp) => {
    agents.value = resp.data.data;
  }).catch(error => {
    console.log(error)
  })

  api.getTeams().then((resp) => {
    teams.value = resp.data.data;
  }).catch(error => {
    console.log(error)
  })

  api.getTags().then(async (resp) => {
    let dt = resp.data.data
    dt.forEach(item => {
      tags.value.push({
        label: item.name,
        value: item.id,
      })
      tagIDMap[item.name] = item.id
    })
  })
})

const handleAssignedAgentChange = (v) => {
  conversationStore.updateAssignee("agent", {
    "assignee_uuid": v
  })
}
const handleAssignedTeamChange = (v) => {
  conversationStore.updateAssignee("team", {
    "assignee_uuid": v
  })
}

const handlePriorityChange = (v) => {
  conversationStore.updatePriority(v)
}

const handleUpsertTags = () => {
  let tagIDs = tagsSelected.value.map((tag) => {
    if (tag in tagIDMap) {
      return tagIDMap[tag]
    }
  })
  conversationStore.upsertTags({
    "tag_ids": JSON.stringify(tagIDs)
  })
}

</script>

<style></style>
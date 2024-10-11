<template>
    <Popover v-model:open="open">
        <PopoverTrigger as-child>
            <Button variant="outline" role="combobox" :aria-expanded="open" class="w-full justify-between">
                {{ assignedAgentName }}
                <CaretSortIcon class="ml-2 h-4 w-4 shrink-0 opacity-50" />
            </Button>
        </PopoverTrigger>

        <PopoverContent class="p-0">
            <Command @update:searchTerm="handleFilterAgents">
                <CommandInput class="h-9" placeholder="Search agent" />
                <CommandEmpty>No agent found.</CommandEmpty>
                <CommandList>
                    <CommandGroup>
                        <CommandItem v-for="agent in filteredAgents" :key="agent.id" :value="agent.id"
                            @select="handleSelectAgent(agent.id)">
                            {{ `${agent.first_name} ${agent.last_name}` }}
                            <CheckIcon :class="cn(
                                'ml-auto h-4 w-4',
                                conversation.assigned_user_id === agent.id ? 'opacity-100' : 'opacity-0'
                            )" />
                        </CommandItem>
                    </CommandGroup>
                </CommandList>
            </Command>
        </PopoverContent>
    </Popover>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { CaretSortIcon, CheckIcon } from '@radix-icons/vue'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import {
  CommandEmpty,
  CommandGroup,
  CommandInput,
  Command,
  CommandItem,
  CommandList
} from '@/components/ui/command'

const open = ref(false)
const props = defineProps({
    conversation: Object,
    selectAgent: Function,
    agents: Array,
})
const filteredAgents = ref([])

const assignedAgentName = computed(() => {
    const assignedUserId = props.conversation.assigned_user_id
    if (!assignedUserId) return 'Select agent'
    const agent = props.agents.find(agent => agent.id === assignedUserId)
    return agent ? `${agent.first_name} ${agent.last_name}` : 'Select agent'
})

const handleFilterAgents = (search) => {
    filteredAgents.value = props.agents.filter(agent =>
        `${agent.first_name.toLowerCase()} ${agent.last_name.toLowerCase()}`.includes(search.toLowerCase())
    )
}

const handleSelectAgent = (id) => {
    props.selectAgent(id)
    open.value = false
}

watch(() => props.agents, (newAgents) => {
    filteredAgents.value = [...newAgents]
}, { immediate: true })
</script>

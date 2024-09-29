<template>
    <Popover v-model:open="open">
        <PopoverTrigger as-child>
            <Button variant="outline" role="combobox" :aria-expanded="open" class="w-full justify-between">
                {{ assignedTeamName }}
                <CaretSortIcon class="ml-2 h-4 w-4 shrink-0 opacity-50" />
            </Button>
        </PopoverTrigger>

        <PopoverContent class="p-0">
            <Command @update:searchTerm="handleFilterTeams">
                <CommandInput class="h-9" placeholder="Search team" />
                <CommandEmpty>No team found.</CommandEmpty>
                <CommandList>
                    <CommandGroup>
                        <CommandItem v-for="team in filteredTeams" :key="team.id" :value="team.id" @select="handleSelectTeam(team.id)">
                            {{ team.name }}
                            <CheckIcon :class="cn(
                                'ml-auto h-4 w-4',
                                conversation.assigned_team_id === team.id ? 'opacity-100' : 'opacity-0'
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
    selectTeam: Function,
    teams: Array,
})
const filteredTeams = ref([])

const assignedTeamName = computed(() => {
    const assignedTeamId = props.conversation.assigned_team_id
    if (!assignedTeamId) return 'Select team...'
    const team = props.teams.find(team => team.id === assignedTeamId)
    return team ? team.name : 'Select team...'
})

const handleFilterTeams = (search) => {
    filteredTeams.value = props.teams.filter(team =>
        team.name.toLowerCase().includes(search.toLowerCase())
    )
}

const handleSelectTeam = (id) => {
    props.selectTeam(id)
    open.value = false
}

watch(() => props.teams, (newTeams) => {
    filteredTeams.value = [...newTeams]
}, { immediate: true })
</script>

<template>
    <Popover v-model:open="open">
        <PopoverTrigger as-child>
            <Button variant="outline" role="combobox" :aria-expanded="open" class="w-full justify-between">
                {{
                    selectedPriority
                }}
                <CaretSortIcon class="ml-2 h-4 w-4 shrink-0 opacity-50" />
            </Button>
        </PopoverTrigger>

        <PopoverContent class="p-0 PopoverContent">
            <Command @update:searchTerm="handleFilterPriorities">
                <CommandInput class="h-9" placeholder="Search priority" />
                <CommandEmpty>No priority found.</CommandEmpty>
                <CommandList>
                    <CommandGroup>
                        <CommandItem v-for="priority in filteredPriorities" :key="priority" :value="priority"
                            @select="handleSelectPriority(priority)">
                            {{ priority }}
                            <CheckIcon :class="cn(
                                'ml-auto h-4 w-4',
                                conversation.priority === priority ? 'opacity-100' : 'opacity-0'
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
    priorities: Array,
    conversation: Object,
    selectPriority: Function,
})
const filteredPriorities = ref([])

const selectedPriority = computed(() => {
    return props.conversation.priority
        ? props.priorities.find(priority => priority === props.conversation.priority)
        : 'Select priority'
})

const handleFilterPriorities = (search) => {
    filteredPriorities.value = props.priorities.filter(priority =>
        priority.toLowerCase().includes(search.toLowerCase())
    )
}

watch(() => props.priorities, (newPriorities) => {
    filteredPriorities.value = [...newPriorities]
}, { immediate: true })

const handleSelectPriority = (priority) => {
    props.selectPriority(priority)
    open.value = false
}
</script>
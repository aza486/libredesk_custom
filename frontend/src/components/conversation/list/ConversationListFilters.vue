<template>
    <div class="flex justify-between px-2 py-2 border-b">

        <Tabs v-model="conversationType">
            <TabsList class="w-full flex justify-evenly">
                <TabsTrigger value="assigned" class="w-full"> Assigned </TabsTrigger>
                <TabsTrigger value="team" class="w-full"> Team </TabsTrigger>
                <TabsTrigger value="all" class="w-full"> All </TabsTrigger>
            </TabsList>
        </Tabs>

        <Popover>
            <PopoverTrigger as-child>
                <div class="flex items-center mr-2">
                    <ListFilter size="20" class="mx-auto cursor-pointer"></ListFilter>
                </div>
            </PopoverTrigger>
            <PopoverContent class="w-52">
                <div>
                    <Select v-model="conversationFilter">
                        <SelectTrigger>
                            <SelectValue placeholder="Select a filter" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem value="status_all"> All </SelectItem>
                                <SelectItem value="status_open"> Open </SelectItem>
                                <SelectItem value="status_processing"> Processing </SelectItem>
                                <SelectItem value="status_spam"> Spam </SelectItem>
                                <SelectItem value="status_resolved"> Resolved </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                </div>
            </PopoverContent>
        </Popover>
    </div>
</template>

<script setup>
import { defineModel, watch } from 'vue'
import { ListFilter } from 'lucide-vue-next'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue
} from '@/components/ui/select'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

const conversationType = defineModel('type')
const conversationFilter = defineModel('filter')
const props = defineProps({
    handleFilterChange: {
        type: Function,
        required: true
    },
})

watch(conversationFilter, (newValue) => {
    props.handleFilterChange(newValue)
})

</script>
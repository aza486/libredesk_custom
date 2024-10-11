<template>
    <div class="flex justify-between px-2 py-2 border-b">
        <Tabs v-model="conversationType">
            <TabsList class="w-full flex justify-evenly">
                <TabsTrigger value="assigned" class="w-full"> Assigned </TabsTrigger>
                <TabsTrigger value="unassigned" class="w-full"> Unassigned </TabsTrigger>
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
                    Work in progress.
                </div>
            </PopoverContent>
        </Popover>
    </div>
</template>

<script setup>
import { defineModel } from 'vue'
import { ListFilter } from 'lucide-vue-next'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

const conversationType = defineModel('type')
</script>
<template>
  <div class="flex justify-between px-2 py-2 border-b w-full">
    <Tabs v-model="conversationStore.conversations.type">
      <TabsList class="w-full flex justify-evenly">
        <TabsTrigger value="assigned" class="w-full"> Assigned </TabsTrigger>
        <TabsTrigger value="unassigned" class="w-full"> Unassigned </TabsTrigger>
        <TabsTrigger value="all" class="w-full"> All </TabsTrigger>
      </TabsList>
    </Tabs>
    <Popover v-model:open="open">
      <PopoverTrigger as-child>
        <div class="flex items-center mr-2 relative">
          <span class="absolute inline-flex h-2 w-2 rounded-full bg-primary opacity-75 right-0 bottom-5"
            v-if="conversationStore.conversations.filters.length > 0"></span>
          <ListFilter size="27"
            class="mx-auto cursor-pointer transition-all transform hover:scale-110 hover:bg-secondary hover:bg-opacity-80 p-1 rounded-md" />
        </div>
      </PopoverTrigger>
      <PopoverContent class="w-[450px]">
        <Filter v-model:modelValue="listFilters" :fields="fields" @apply="handleApply" @clear="handleClear" />
      </PopoverContent>
    </Popover>
  </div>
</template>


<script setup>
import { ref, onMounted } from 'vue'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ListFilter } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import Filter from '@/components/common/Filter.vue'
import api from '@/api'

onMounted(() => {
  getStatuses()
})

const conversationStore = useConversationStore()
const open = ref(false)
const listFilters = ref(conversationStore.conversations.filters)
const statuses = ref([])
const emit = defineEmits(["updateFilters"])

const getStatuses = async () => {
  const resp = await api.getStatuses()
  statuses.value = resp.data.data.map(status => ({
    label: status.name,
    value: status.id.toString(),
  }))
}

const fields = ref([
  {
    model: 'conversations',
    label: 'Status',
    value: 'status_id',
    type: 'select',
    options: statuses
  },
  {
    model: 'conversations',
    label: 'Priority',
    value: 'priority_id',
    type: 'select',
    options: [
      { label: 'Low', value: "1" },
      { label: 'Medium', value: "2" },
      { label: 'High', value: "3" },
    ]
  },
  {
    model: 'conversations',
    label: 'Reference number',
    value: 'reference_number',
    type: 'text',
  }
])

const handleApply = (filters) => {
  open.value = false
  emit('updateFilters', filters)
}

const handleClear = () => {
  open.value = false
  emit('updateFilters', [])
}
</script>
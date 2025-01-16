<template>
  <div class="flex justify-end px-2 py-2 border-b w-full">
    <Popover v-model:open="open">
      <PopoverTrigger as-child>
        <div class="flex items-center mr-2 relative">
          <span class="absolute inline-flex h-2 w-2 rounded-full bg-primary opacity-75 right-0 bottom-5 z-20"
            v-if="conversationStore.conversations.filters.length > 0" />
          <ListFilter size="27"
            class="mx-auto cursor-pointer transition-all transform hover:scale-110 hover:bg-secondary hover:bg-opacity-80 p-1 rounded-md z-10" />
        </div>
      </PopoverTrigger>
      <PopoverContent class="w-[450px]">
        <Filter v-model:modelValue="localFilters" :fields="fields" @apply="handleApply" @clear="handleClear" />
      </PopoverContent>
    </Popover>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ListFilter } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import Filter from '@/components/common/FilterBuilder.vue'
import api from '@/api'

const conversationStore = useConversationStore()
const open = ref(false)
const localFilters = ref([])
const statuses = ref([])
const priorities = ref([])
const emit = defineEmits(['updateFilters'])

onMounted(() => {
  fetchInitialData()
  localFilters.value = [...conversationStore.conversations.filters]
})

const fetchInitialData = async () => {
  const [statusesResp, prioritiesResp] = await Promise.all([
    api.getStatuses(),
    api.getPriorities()
  ])

  statuses.value = statusesResp.data.data.map(status => ({
    label: status.name,
    value: status.id.toString(),
  }))

  priorities.value = prioritiesResp.data.data.map(priority => ({
    label: priority.name,
    value: priority.id.toString(),
  }))
}

const fields = ref([
  {
    model: 'conversations',
    label: 'Status',
    value: 'status_id',
    type: 'select',
    options: statuses,
  },
  {
    model: 'conversations',
    label: 'Priority',
    value: 'priority_id',
    type: 'select',
    options: priorities,
  },
  {
    model: 'conversations',
    label: 'Reference number',
    value: 'reference_number',
    type: 'text',
  }
])

const handleApply = (filters) => {
  emit('updateFilters', filters)
  open.value = false
}

const handleClear = () => {
  localFilters.value = []
  emit('updateFilters', [])
  open.value = false
}
</script>

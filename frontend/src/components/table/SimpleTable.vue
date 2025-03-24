<template>
  <table class="min-w-full divide-y divide-gray-200">
    <thead class="bg-gray-50">
      <tr>
        <th
          v-for="(header, index) in headers"
          :key="index"
          scope="col"
          class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
        >
          {{ header }}
        </th>
        <th scope="col" class="relative px-6 py-3"></th>
      </tr>
    </thead>
    <tbody class="bg-white divide-y divide-gray-200">
      <template v-if="data.length === 0">
        <tr>
          <td :colspan="headers.length + 1" class="px-6 py-12 text-center">
            <div class="flex flex-col items-center space-y-4">
              <span class="text-md text-gray-500"> No records found. </span>
            </div>
          </td>
        </tr>
      </template>
      <template v-else>
        <tr v-for="(item, index) in data" :key="index">
          <td
            v-for="key in keys"
            :key="key"
            class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900"
          >
            {{ item[key] }}
          </td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
            <Button size="xs" variant="ghost" @click.prevent="deleteItem(item)">
              <Trash2 class="h-4 w-4" />
            </Button>
          </td>
        </tr>
      </template>
    </tbody>
  </table>
</template>

<script setup>
import { Trash2 } from 'lucide-vue-next'
import { defineProps, defineEmits } from 'vue'
import { Button } from '@/components/ui/button'

defineProps({
  headers: {
    type: Array,
    required: true,
    default: () => []
  },
  keys: {
    type: Array,
    required: true,
    default: () => []
  },
  data: {
    type: Array,
    required: true,
    default: () => []
  }
})

const emit = defineEmits(['deleteItem'])

function deleteItem(item) {
  emit('deleteItem', item)
}
</script>

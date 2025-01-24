<template>
  <div class="flex flex-wrap px-2 py-1">
    <div class="flex flex-wrap gap-2">
      <div
        v-for="action in actions"
        :key="action.type"
        class="flex items-center bg-white border border-gray-200 rounded shadow-sm transition-all duration-300 ease-in-out hover:shadow-md group"
      >
        <div class="flex items-center space-x-2 px-3 py-2">
          <component
            :is="getIcon(action.type)"
            size="16"
            class="text-primary group-hover:text-primary"
          />
          <Tooltip>
            <TooltipTrigger as-child>
              <div
                class="max-w-[12rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-primary group-hover:text-gray-900"
              >
                {{ getDisplayValue(action) }}
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p class="text-sm">{{ getTooltip(action) }}</p>
            </TooltipContent>
          </Tooltip>
        </div>
        <button
          @click.stop="onRemove(action)"
          class="p-2 text-gray-400 hover:text-red-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-opacity-50 rounded transition-colors duration-300 ease-in-out"
          title="Remove action"
        >
          <X size="14" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { X, Users, User, MessageSquare, Tags, Flag } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

defineProps({
  actions: {
    type: Array,
    required: true
  },
  onRemove: {
    type: Function,
    required: true
  }
})

const getIcon = (type) =>
  ({
    assign_team: Users,
    assign_user: User,
    set_status: MessageSquare,
    set_priority: Flag,
    set_tags: Tags
  })[type]

const getDisplayValue = (action) => {
  if (action.display_value?.length) {
    return action.display_value.join(', ')
  }
  return action.value.join(', ')
}

const getTooltip = (action) => {
  switch (action.type) {
    case 'assign_team':
      return `Assign to team: ${getDisplayValue(action)}`
    case 'assign_user':
      return `Assign to user: ${getDisplayValue(action)}`
    case 'set_status':
      return `Set status to: ${getDisplayValue(action)}`
    case 'set_priority':
      return `Set priority to: ${getDisplayValue(action)}`
    case 'set_tags':
      return `Set tags: ${getDisplayValue(action)}`
    default:
      return `Action: ${action.type}, Value: ${getDisplayValue(action)}`
  }
}
</script>

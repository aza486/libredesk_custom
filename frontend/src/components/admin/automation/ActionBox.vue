<template>
  <div class="box border p-5 space-y-5 rounded">
    <div class="space-y-5">
      <div v-for="(action, index) in actions" :key="index" class="space-y-5">
        <div v-if="index > 0">
          <hr class="border-t-2 border-dotted border-gray-300" />
        </div>
        <div class="space-y-3">
          <div class="flex space-x-5 justify-between">
            <Select v-model="action.type" @update:modelValue="(value) => handleFieldChange(value, index)">
              <SelectTrigger class="w-56">
                <SelectValue placeholder="Select action" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectLabel>Conversation</SelectLabel>
                  <SelectItem v-for="(actionItem, key) in conversationActions" :key="key" :value="key">
                    {{ actionItem.label }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>

            <div v-if="action.type && conversationActions[action.type].inputType === 'select'" class="flex-1">
              <Select v-model="action.value" @update:modelValue="(value) => handleValueChange(value, index)">
                <SelectTrigger class="w-56">
                  <SelectValue placeholder="Select value" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem v-for="(act, index) in getDropdownValues(action.type).value" :key="index"
                      :value="act.value.toString()">
                      {{ act.name }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>

            <div class="cursor-pointer" @click.prevent="removeAction(index)">
              <CircleX size="21" />
            </div>
          </div>
          <div v-if="action.type && conversationActions[action.type].inputType === 'richtext'" class="pl-0">
            <QuillEditor theme="snow" v-model:content="action.value" contentType="html"
              @update:content="(value) => handleValueChange(value, index)" class="h-32 mb-12" />
          </div>
        </div>
      </div>
    </div>
    <div>
      <Button variant="outline" @click.prevent="addAction" size="sm">Add action</Button>
    </div>
  </div>
</template>

<script setup>
import { toRefs, ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { CircleX } from 'lucide-vue-next'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { QuillEditor } from '@vueup/vue-quill'
import '@vueup/vue-quill/dist/vue-quill.snow.css'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const props = defineProps({
  actions: {
    type: Object,
    required: true
  }
})

const { actions } = toRefs(props)
const emitter = useEmitter()
const teams = ref([])
const users = ref([])
const statuses = ref([])
const priorities = ref([])
const emit = defineEmits(['update-actions', 'add-action', 'remove-action'])

onMounted(async () => {
  try {
    const [teamsResp, usersResp, statusesResp, prioritiesResp] = await Promise.all([
      api.getTeamsCompact(),
      api.getUsersCompact(),
      api.getStatuses(),
      api.getPriorities(),
    ])

    teams.value = teamsResp.data.data.map(team => ({
      value: team.id,
      name: team.name
    }))

    users.value = usersResp.data.data.map(user => ({
      value: user.id,
      name: user.first_name + ' ' + user.last_name
    }))

    statuses.value = statusesResp.data.data.map(status => ({
      value: status.name,
      name: status.name
    }))

    priorities.value = prioritiesResp.data.data.map(priority => ({
      value: priority.name,
      name: priority.name
    }))
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Something went wrong',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
})

const handleFieldChange = (value, index) => {
  actions.value[index].value = ''
  actions.value[index].type = value
  emitUpdate(index)
}

const handleValueChange = (value, index) => {
  actions.value[index].value = value
  emitUpdate(index)
}

const removeAction = (index) => {
  emit('remove-action', index)
}

const addAction = (index) => {
  emit('add-action', index)
}

const emitUpdate = (index) => {
  emit('update-actions', actions, index)
}

const conversationActions = {
  assign_team: {
    label: 'Assign to team',
    inputType: 'select',
  },
  assign_user: {
    label: 'Assign to user',
    inputType: 'select',
  },
  set_status: {
    label: 'Set status',
    inputType: 'select',
  },
  set_priority: {
    label: 'Set priority',
    inputType: 'select',
  },
  send_private_note: {
    label: 'Send private note',
    inputType: 'richtext',
  },
  reply: {
    label: 'Send reply',
    inputType: 'richtext',
  }
}

const actionDropdownValues = {
  assign_team: teams,
  assign_user: users,
  set_status: statuses,
  set_priority: priorities,
}

const getDropdownValues = (field) => {
  return actionDropdownValues[field] || []
}
</script>
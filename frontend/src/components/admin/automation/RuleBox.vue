<template>
  <div>
    <div class="mb-5">
      <RadioGroup class="flex" :modelValue="ruleGroup.logical_op" @update:modelValue="handleGroupOperator">
        <div class="flex items-center space-x-2">
          <RadioGroupItem value="OR" />
          <Label for="r1">Match <b>ANY</b> of below.</Label>
        </div>
        <div class="flex items-center space-x-2">
          <RadioGroupItem value="AND" />
          <Label for="r1">Match <b>ALL</b> of below.</Label>
        </div>
      </RadioGroup>
    </div>
    <div class="box border p-5 space-y-5 rounded-lg">
      <div class="space-y-5">
        <div v-for="(rule, index) in ruleGroup.rules" :key="rule" class="space-y-5">
          <div v-if="index > 0">
            <hr class="border-t-2 border-dotted border-gray-300" />
          </div>
          <div class="flex justify-between">
            <div class="flex space-x-5">
              <!-- Field selection -->
              <Select v-model="rule.field" @update:modelValue="(value) => handleFieldChange(value, index)">
                <SelectTrigger class="w-56">
                  <SelectValue placeholder="Select field" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectLabel>Conversation</SelectLabel>
                    <SelectItem v-for="(field, key) in conversationFields" :key="key" :value="key">
                      {{ field.label }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>

              <!-- Operator selection -->
              <Select v-model="rule.operator" @update:modelValue="(value) => handleOperatorChange(value, index)">
                <SelectTrigger class="w-56">
                  <SelectValue placeholder="Select operator" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem v-for="(op, key) in getFieldOperators(rule.field)" :key="key" :value="op">
                      {{ op }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            <div class="cursor-pointer" @click.prevent="removeCondition(index)">
              <CircleX size="21" />
            </div>
          </div>

          <!-- Value input based on field type -->
          <div v-if="showInput(index)">
            <!-- Text input -->
            <Input type="text" placeholder="Set value" v-if="inputType(index) === 'text'" v-model="rule.value"
              @update:modelValue="(value) => handleValueChange(value, index)" />

            <!-- Dropdown -->
            <Select v-model="rule.value" @update:modelValue="(value) => handleValueChange(value, index)"
              v-if="inputType(index) === 'select'">
              <SelectTrigger class="w-56">
                <SelectValue placeholder="Select value" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem v-for="(op, key) in getFieldOptions(rule.field)" :key="key" :value="op">
                    {{ op }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>

            <!-- Tag -->
            <TagsInput :defaultValue="fieldValueAsArray(rule.value)" v-if="inputType(index) === 'tag'" @update:modelValue="(value) => handleValueChange(value, index)">
              <TagsInputItem v-for="item in fieldValueAsArray(rule.value)" :key="item" :value="item">
                <TagsInputItemText />
                <TagsInputItemDelete />
              </TagsInputItem>
              <TagsInputInput placeholder="Select values" />
            </TagsInput>
          </div>

          <div class="flex items-center space-x-2">
            <Checkbox id="terms" :defaultChecked="rule.case_sensitive_match"
              @update:checked="(value) => handleCaseSensitiveCheck(value, index)" />
            <label for="terms"> Case sensitive match </label>
          </div>
        </div>
      </div>
      <div>
        <Button variant="outline" size="sm" @click.prevent="addCondition">Add condition</Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { toRefs, ref, onMounted } from 'vue'
import { Checkbox } from '@/components/ui/checkbox'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { TagsInput, TagsInputInput, TagsInputItem, TagsInputItemDelete, TagsInputItemText } from '@/components/ui/tags-input'
import { CircleX } from 'lucide-vue-next'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const props = defineProps({
  ruleGroup: {
    type: Object,
    required: true
  },
  groupIndex: {
    type: Number,
    required: true
  }
})

const emitter = useEmitter()
const statuses = ref([])
const priorities = ref([
  "Low", "Medium", "High"
])
const { ruleGroup } = toRefs(props)

onMounted(async () => {
  try {
    const [statusesResp] = await Promise.all([api.getStatuses()])
    statuses.value = statusesResp.data.data.map(status => (status.name))
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Could not fetch statuses',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
})

const emit = defineEmits(['update-group', 'add-condition', 'remove-condition'])

const handleGroupOperator = (value) => {
  ruleGroup.value.logical_op = value
  emitUpdate()
}

const handleFieldChange = (value, ruleIndex) => {
  // Clear operator and value on field change
  ruleGroup.value.rules[ruleIndex].operator = ''
  ruleGroup.value.rules[ruleIndex].value = ''
  ruleGroup.value.rules[ruleIndex].field = value
  emitUpdate()
}

const handleOperatorChange = (value, ruleIndex) => {
  // Set initial value based on operator.
  if (["contains", "not contains"].includes(value))
    ruleGroup.value.rules[ruleIndex].value = []
  else
    ruleGroup.value.rules[ruleIndex].value = ''
  ruleGroup.value.rules[ruleIndex].operator = value
  emitUpdate()
}

const handleValueChange = (value, ruleIndex) => {
  console.log("value ", value)
  const operator = ruleGroup.value.rules[ruleIndex].operator
  // For 'contains' and 'not contains', join array into a single string
  if (["contains", "not contains"].includes(operator)) {
    ruleGroup.value.rules[ruleIndex].value = Array.isArray(value) ? value.join(',') : value
  } else {
    ruleGroup.value.rules[ruleIndex].value = String(value)
  }
  emitUpdate()
}

const fieldValueAsArray = (value) => {
  return Array.isArray(value) ? value : (value ? value.split(',') : [])
}

const handleCaseSensitiveCheck = (value, ruleIndex) => {
  ruleGroup.value.rules[ruleIndex].case_sensitive_match = value
  emitUpdate()
}

const removeCondition = (index) => {
  emit('remove-condition', props.groupIndex, index)
}

const addCondition = () => {
  emit('add-condition', props.groupIndex)
}

const emitUpdate = () => {
  emit('update-group', ruleGroup, props.groupIndex)
}

const conversationFields = {
  content: { label: 'Content' },
  subject: { label: 'Subject' },
  status: { label: 'Status' },
  priority: { label: 'Priority' },
  assigned_team: { label: 'Assigned team' },
  assigned_user: { label: 'Assigned user' }
}

const fieldOperators = {
  content: ["contains", "not contains", "equals", "not equals", "set", "not set"],
  subject: ["contains", "not contains", "equals", "not equals", "set", "not set"],
  status: ["equals", "not equals", "set", "not set"],
  priority: ["equals", "not equals", "set", "not set"],
  assigned_team: ["set", "not set"],
  assigned_user: ["set", "not set"]
}

const fieldOptions = {
  status: statuses,
  priority: priorities,
}

const getFieldOperators = (field) => {
  return fieldOperators[field] || []
}

const getFieldOptions = (field) => {
  return fieldOptions[field]?.value || []
}

const inputType = (index) => {
  const field = ruleGroup.value.rules[index]?.field
  const operator = ruleGroup.value.rules[index]?.operator
  if (["contains", "not contains"].includes(operator)) {
    return "tag"
  }
  if (["status", "priority"].includes(field)) {
    return "select"
  }
  if (["equals", "not equals"].includes(operator)) {
    return "text"
  }
  return ""
}

const showInput = (index) => {
  const operator = ruleGroup.value.rules[index]?.operator
  return !["set", "not set"].includes(operator)
}
</script>

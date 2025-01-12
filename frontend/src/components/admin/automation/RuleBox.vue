<template>
  <div>
    <div class="mb-5">
      <RadioGroup
        class="flex"
        :modelValue="ruleGroup.logical_op"
        @update:modelValue="handleGroupOperator"
      >
        <div class="flex items-center space-x-2">
          <RadioGroupItem value="OR" />
          <Label>Match <b>ANY</b> of below.</Label>
        </div>
        <div class="flex items-center space-x-2">
          <RadioGroupItem value="AND" />
          <Label>Match <b>ALL</b> of below.</Label>
        </div>
      </RadioGroup>
    </div>

    <div class="space-y-5 rounded-lg" :class="{ 'box border p-5': ruleGroup.rules?.length > 0 }">
      <div class="space-y-5">
        <div v-for="(rule, index) in ruleGroup.rules" :key="rule" class="space-y-5">
          <div v-if="index > 0">
            <hr class="border-t-2 border-dotted border-gray-200" />
          </div>

          <!-- Field -->
          <div class="flex space-x-5 items-start">
            <Select
              v-model="rule.field"
              @update:modelValue="(value) => handleFieldChange(value, index)"
            >
              <SelectTrigger class="w-56">
                <SelectValue placeholder="Select field" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectLabel>Conversation</SelectLabel>
                  <SelectItem v-for="(field, key) in conversationFilters" :key="key" :value="key">
                    {{ field.label }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>

            <!-- Operator -->
            <Select
              v-model="rule.operator"
              @update:modelValue="(value) => handleOperatorChange(value, index)"
            >
              <SelectTrigger class="w-56">
                <SelectValue placeholder="Select operator" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem
                    v-for="(op, key) in getFieldOperators(rule.field)"
                    :key="key"
                    :value="op"
                  >
                    {{ op }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>

            <!-- Value -->
            <div v-if="showInput(index)" class="flex-1">
              <!-- Plain text input -->
              <Input
                type="text"
                placeholder="Set value"
                v-if="inputType(index) === 'text'"
                v-model="rule.value"
                @update:modelValue="(value) => handleValueChange(value, index)"
              />

              <!-- Number input -->
              <Input
                type="number"
                placeholder="Set value"
                v-if="inputType(index) === 'number'"
                v-model="rule.value"
                @update:modelValue="(value) => handleValueChange(value, index)"
              />

              <!-- Select input -->
              <div v-if="inputType(index) === 'select'">
                <ComboBox
                  v-model="rule.value"
                  :items="getFieldOptions(rule.field)"
                  @select="handleValueChange($event, index)"
                >
                  <template #item="{ item }">
                    <div class="flex items-center gap-2 ml-2">
                      <Avatar v-if="rule.field === 'assigned_user'" class="w-7 h-7">
                        <AvatarImage :src="item.avatar_url ?? ''" :alt="item.label.slice(0, 2)" />
                        <AvatarFallback>
                          {{ item.label.slice(0, 2).toUpperCase() }}
                        </AvatarFallback>
                      </Avatar>
                      <span v-if="rule.field === 'assigned_team'">
                        {{ item.emoji }}
                      </span>
                      <span>{{ item.label }}</span>
                    </div>
                  </template>

                  <template #selected="{ selected }">
                    <div v-if="rule?.field === 'assigned_team'">
                      <div v-if="selected" class="flex items-center gap-2">
                        {{ selected.emoji }}
                        <span>{{ selected.label }}</span>
                      </div>
                      <span v-else>Select team</span>
                    </div>

                    <div
                      v-else-if="rule?.field === 'assigned_user'"
                      class="flex items-center gap-2"
                    >
                      <div v-if="selected" class="flex items-center gap-2">
                        <Avatar class="w-7 h-7">
                          <AvatarImage
                            :src="selected.avatar_url ?? ''"
                            :alt="selected.label.slice(0, 2)"
                          />
                          <AvatarFallback>
                            {{ selected.label.slice(0, 2).toUpperCase() }}
                          </AvatarFallback>
                        </Avatar>
                        <span>{{ selected.label }}</span>
                      </div>
                      <span v-else>Select user</span>
                    </div>
                    <span v-else>
                      <span v-if="!selected"> Select</span>
                      <span v-else>{{ selected.label }} </span>
                    </span>
                  </template>
                </ComboBox>
              </div>

              <!-- Tag input -->
              <div v-if="inputType(index) === 'tag'">
                <TagsInput
                  :defaultValue="fieldValueAsArray(rule.value)"
                  @update:modelValue="(value) => handleValueChange(value, index)"
                >
                  <TagsInputItem
                    v-for="item in fieldValueAsArray(rule.value)"
                    :key="item"
                    :value="item"
                  >
                    <TagsInputItemText />
                    <TagsInputItemDelete />
                  </TagsInputItem>
                  <TagsInputInput placeholder="Select values" />
                </TagsInput>
              </div>
            </div>

            <!-- Remove condition -->
            <div class="cursor-pointer mt-2" @click.prevent="removeCondition(index)">
              <X size="16" />
            </div>
          </div>

          <div class="flex items-center space-x-2">
            <Checkbox
              id="terms"
              :defaultChecked="rule.case_sensitive_match"
              @update:checked="(value) => handleCaseSensitiveCheck(value, index)"
            />
            <label> Case sensitive match </label>
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
import { toRefs } from 'vue'
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
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '@/components/ui/tags-input'
import { X } from 'lucide-vue-next'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { useConversationFilters } from '@/composables/useConversationFilters'

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

const { conversationFilters } = useConversationFilters()
const { ruleGroup } = toRefs(props)
const emit = defineEmits(['update-group', 'add-condition', 'remove-condition'])

const handleGroupOperator = (value) => {
  ruleGroup.value.logical_op = value
  emitUpdate()
}

const handleFieldChange = (value, ruleIndex) => {
  ruleGroup.value.rules[ruleIndex].operator = ''
  ruleGroup.value.rules[ruleIndex].value = ''
  ruleGroup.value.rules[ruleIndex].field = value
  emitUpdate()
}

const handleOperatorChange = (value, ruleIndex) => {
  if (['contains', 'not contains'].includes(value)) {
    ruleGroup.value.rules[ruleIndex].value = []
  } else {
    ruleGroup.value.rules[ruleIndex].value = ''
  }
  ruleGroup.value.rules[ruleIndex].operator = value
  emitUpdate()
}

const handleValueChange = (value, ruleIndex) => {
  // Get value from object if it's an object.
  const val = typeof value === 'object' && !Array.isArray(value) ? value.value : value

  // Fetch the rule.
  const rule = ruleGroup.value.rules[ruleIndex]

  // Array values are stored as comma separated string.
  rule.value = ['contains', 'not contains'].includes(rule.operator)
    ? Array.isArray(val)
      ? val.join(',')
      : val
    : String(val)

  emitUpdate()
}

const fieldValueAsArray = (value) => {
  return Array.isArray(value) ? value : value ? value.split(',') : []
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

const getFieldOperators = (field) => {
  return conversationFilters.value[field]?.operators || []
}

const getFieldOptions = (field) => {
  return conversationFilters.value[field]?.options || []
}

const inputType = (index) => {
  const field = ruleGroup.value.rules[index]?.field
  const operator = ruleGroup.value.rules[index]?.operator
  if (['contains', 'not contains'].includes(operator)) return 'tag'
  if (field) return conversationFilters.value[field].type
  return ''
}

const showInput = (index) => {
  const operator = ruleGroup.value.rules[index]?.operator
  return !['set', 'not set'].includes(operator)
}
</script>

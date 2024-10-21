<template>
  <div v-for="(filter, index) in modelValue" :key="index">
    <div class="flex items-center space-x-2 mb-2 flex-row justify-between">
      <div class="w-1/3">
        <Select v-model="filter.field" @update:modelValue="updateFieldModel(filter, $event)">
          <SelectTrigger class="w-full">
            <SelectValue placeholder="Select Field" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem v-for="field in fields" :key="field.value" :value="field.value">
                {{ field.label }}
              </SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      <div class="w-1/3">
        <Select v-model="filter.operator">
          <SelectTrigger class="w-full">
            <SelectValue placeholder="Select Operator" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem v-for="operator in getFieldOperators(filter.field)" :key="operator.value"
                :value="operator.value">
                {{ operator.label }}
              </SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      <div v-if="getFieldType(filter.field) === 'text'" class="w-1/3">
        <Input v-model="filter.value" type="text" placeholder="Value" class="w-full" />
      </div>
      <div v-else-if="getFieldType(filter.field) === 'select'" class="w-1/3">
        <Select v-model="filter.value">
          <SelectTrigger class="w-full">
            <SelectValue placeholder="Select Value" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem v-for="option in getFieldOptions(filter.field)" :key="option.value" :value="option.value">
                {{ option.label }}
              </SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>
      <div v-else-if="getFieldType(filter.field) === 'number'" class="w-1/3">
        <Input v-model="filter.value" type="number" placeholder="Value" class="w-full" />
      </div>
      <button v-if="modelValue.length > 1" @click="removeFilter(index)"
        class="flex items-center justify-center w-3 h-3 rounded-full bg-red-100 hover:bg-red-200 transition-colors">
        <X class="text-slate-400" />
      </button>
    </div>
  </div>
  <div class="flex justify-between mt-4">
    <Button size="sm" @click="addFilter">Add Filter</Button>
    <div class="flex justify-end space-x-4">
      <Button size="sm" @click="applyFilters">Apply</Button>
      <Button size="sm" @click="clearFilters">Clear</Button>
    </div>
  </div>
</template>


<script setup>
import { computed, onMounted } from 'vue'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { X } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const props = defineProps({
  fields: {
    type: Array,
    required: true,
  },
})

const emit = defineEmits(['apply', 'clear'])
const modelValue = defineModel('modelValue', { required: true })
const createFilter = () => ({ model: '', field: '', operator: '', value: '' })

const operatorsByType = {
  text: [
    { label: 'Equals', value: '=' },
    { label: 'Not Equals', value: '!=' },
  ],
  select: [
    { label: 'Equals', value: '=' },
    { label: 'Not Equals', value: '!=' },
  ],
  number: [
    { label: 'Equals', value: '=' },
    { label: 'Not Equals', value: '!=' },
    { label: 'Greater Than', value: '>' },
    { label: 'Less Than', value: '<' },
    { label: 'Greater Than or Equal', value: '>=' },
    { label: 'Less Than or Equal', value: '<=' },
  ],
}

onMounted(() => {
  if (modelValue.value.length === 0) {
    modelValue.value.push(createFilter())
  }
})

const addFilter = () => {
  modelValue.value.push(createFilter())
}

const removeFilter = (index) => {
  modelValue.value.splice(index, 1)
}

const applyFilters = () => {
  if (validFilters.value.length > 0) emit('apply', validFilters.value)
}

const validFilters = computed(() => {
  return modelValue.value.filter(filter => filter.field !== "" && filter.operator != "" && filter.value != "")
})

const clearFilters = () => {
  modelValue.value = []
  emit('clear')
}

const getFieldOperators = computed(() => (fieldValue) => {
  const field = props.fields.find(f => f.value === fieldValue)
  return field ? operatorsByType[field.type] : []
})

const getFieldType = computed(() => (fieldValue) => {
  const field = props.fields.find(f => f.value === fieldValue)
  return field ? field.type : 'text'
})

const getFieldOptions = computed(() => (fieldValue) => {
  const field = props.fields.find(f => f.value === fieldValue)
  return field && field.options ? field.options : []
})

const updateFieldModel = (filter, fieldValue) => {
  const field = props.fields.find(f => f.value === fieldValue)
  if (field) {
    filter.model = field.model
  }
}
</script>
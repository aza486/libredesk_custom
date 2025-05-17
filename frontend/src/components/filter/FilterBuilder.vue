<template>
  <div class="space-y-4">
    <div class="w-[27rem]" v-if="modelValue.length === 0"></div>

    <div
      v-for="(modelFilter, index) in modelValue"
      :key="index"
      class="group flex items-center gap-3"
    >
      <div class="flex gap-2 w-full">
        <!-- Field -->
        <div class="flex-1">
          <Select v-model="modelFilter.field">
            <SelectTrigger class="bg-transparent hover:bg-slate-100 w-full">
              <SelectValue :placeholder="t('form.field.selectField')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="field in fields" :key="field.field" :value="field.field">
                  {{ field.label }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <!-- Operator -->
        <div class="flex-1">
          <Select v-model="modelFilter.operator" v-if="modelFilter.field">
            <SelectTrigger class="bg-transparent hover:bg-slate-100 w-full">
              <SelectValue :placeholder="t('form.field.selectOperator')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="op in getFieldOperators(modelFilter)" :key="op" :value="op">
                  {{ op }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <!-- Value -->
        <div class="flex-1">
          <div v-if="modelFilter.field && modelFilter.operator">
            <template v-if="modelFilter.operator !== 'set' && modelFilter.operator !== 'not set'">
              <ComboBox
                v-if="getFieldOptions(modelFilter).length > 0"
                v-model="modelFilter.value"
                :items="getFieldOptions(modelFilter)"
                :placeholder="t('form.field.select')"
              >
                <template #item="{ item }">
                  <div v-if="modelFilter.field === 'assigned_user_id'">
                    <div class="flex items-center gap-1">
                      <Avatar class="w-6 h-6">
                        <AvatarImage :src="item.avatar_url || ''" :alt="item.label.slice(0, 2)" />
                        <AvatarFallback>{{ item.label.slice(0, 2).toUpperCase() }} </AvatarFallback>
                      </Avatar>
                      <span>{{ item.label }}</span>
                    </div>
                  </div>
                  <div v-else-if="modelFilter.field === 'assigned_team_id'">
                    <div class="flex items-center gap-2 ml-2">
                      <span>{{ item.emoji }}</span>
                      <span>{{ item.label }}</span>
                    </div>
                  </div>
                  <div v-else>
                    {{ item.label }}
                  </div>
                </template>

                <template #selected="{ selected }">
                  <div v-if="!selected">{{ $t('form.field.selectValue') }}</div>
                  <div v-if="modelFilter.field === 'assigned_user_id'">
                    <div class="flex items-center gap-2">
                      <div v-if="selected" class="flex items-center gap-1">
                        <Avatar class="w-6 h-6">
                          <AvatarImage
                            :src="selected.avatar_url || ''"
                            :alt="selected.label.slice(0, 2)"
                          />
                          <AvatarFallback>{{
                            selected.label.slice(0, 2).toUpperCase()
                          }}</AvatarFallback>
                        </Avatar>
                        <span>{{ selected.label }}</span>
                      </div>
                    </div>
                  </div>
                  <div v-else-if="modelFilter.field === 'assigned_team_id'">
                    <div class="flex items-center gap-2">
                      <span v-if="selected">
                        {{ selected.emoji }}
                        <span>{{ selected.label }}</span>
                      </span>
                    </div>
                  </div>
                  <div v-else-if="selected">
                    {{ selected.label }}
                  </div>
                </template>
              </ComboBox>
              <Input
                v-else
                v-model="modelFilter.value"
                class="bg-transparent hover:bg-slate-100"
                :placeholder="t('form.field.value')"
                type="text"
              />
            </template>
          </div>
        </div>
      </div>

      <button @click="removeFilter(index)" class="p-1 hover:bg-slate-100 rounded">
        <X class="w-4 h-4 text-slate-500" />
      </button>
    </div>

    <div class="flex items-center justify-between pt-3">
      <Button variant="ghost" size="sm" @click="addFilter" class="text-slate-600">
        <Plus class="w-3 h-3 mr-1" />
        {{
          $t('globals.messages.add', {
            name: $t('globals.terms.filter')
          })
        }}
      </Button>
      <div class="flex gap-2" v-if="showButtons">
        <Button variant="ghost" @click="clearFilters">{{ $t('globals.buttons.reset') }}</Button>
        <Button @click="applyFilters">{{ $t('globals.buttons.apply') }}</Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Plus, X } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Avatar, AvatarImage, AvatarFallback } from '@/components/ui/avatar'
import { useI18n } from 'vue-i18n'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'

const props = defineProps({
  fields: {
    type: Array,
    required: true
  },
  showButtons: {
    type: Boolean,
    default: true
  }
})
const { t } = useI18n()
const emit = defineEmits(['apply', 'clear'])
const modelValue = defineModel('modelValue', { required: false, default: () => [] })

const createFilter = () => ({ field: '', operator: '', value: '' })

onMounted(() => {
  if (modelValue.value.length === 0) {
    modelValue.value = [createFilter()]
  }
})

const getModel = (field) => {
  const fieldConfig = props.fields.find((f) => f.field === field)
  return fieldConfig?.model || ''
}

// Set model for each filter
watch(
  () => modelValue.value,
  (filters) => {
    filters.forEach((filter) => {
      if (filter.field && !filter.model) {
        filter.model = getModel(filter.field)
      }
    })
  },
  { deep: true }
)

// Reset operator and value when field changes for a filter at a given index
watch(
  () => modelValue.value.map((f) => f.field),
  (newFields, oldFields) => {
    newFields.forEach((field, index) => {
      if (field !== oldFields[index]) {
        modelValue.value[index].operator = ''
        modelValue.value[index].value = ''
      }
    })
  }
)

const addFilter = () => {
  modelValue.value = [...modelValue.value, createFilter()]
}
const removeFilter = (index) => {
  modelValue.value = modelValue.value.filter((_, i) => i !== index)
}
const applyFilters = () => {
  modelValue.value = validFilters.value
  emit('apply', modelValue.value)
}
const clearFilters = () => {
  modelValue.value = []
  emit('clear')
}

const validFilters = computed(() => {
  return modelValue.value.filter((filter) => filter.field && filter.operator && filter.value)
})

const getFieldOptions = (fieldValue) => {
  const field = props.fields.find((f) => f.field === fieldValue.field)
  return field?.options || []
}

const getFieldOperators = (modelFilter) => {
  const field = props.fields.find((f) => f.field === modelFilter.field)
  return field?.operators || []
}
</script>

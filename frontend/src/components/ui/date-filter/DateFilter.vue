<template>
  <div class="flex items-center gap-2">
    <Select v-model="selectedDays" @update:model-value="handleFilterChange">
      <SelectTrigger class="w-[140px] h-8 text-xs">
        <SelectValue
          :placeholder="
            t('globals.messages.select', {
              name: t('globals.terms.day', 2)
            })
          "
        />
      </SelectTrigger>
      <SelectContent class="text-xs">
        <SelectItem value="0">{{ $t('globals.terms.today') }}</SelectItem>
        <SelectItem value="1">{{ $t('filters.last_1_day') }}</SelectItem>
        <SelectItem value="2">{{ $t('filters.last_2_days') }}</SelectItem>
        <SelectItem value="7">{{ $t('filters.last_7_days') }}</SelectItem>
        <SelectItem value="30">{{ $t('filters.last_30_days') }}</SelectItem>
        <SelectItem value="90">{{ $t('filters.last_90_days') }}</SelectItem>
        <SelectItem value="custom">{{ $t('filters.custom_days') }}</SelectItem>
      </SelectContent>
    </Select>
    <div v-if="selectedDays === 'custom'" class="flex items-center gap-2">
      <Input
        v-model="customDaysInput"
        type="number"
        min="1"
        max="365"
        :placeholder="$t('filters.days_placeholder')"
        class="w-20"
        @blur="handleCustomDaysChange"
        @keyup.enter="handleCustomDaysChange"
      />
      <span class="text-xs text-muted-foreground">{{
        $t('globals.terms.day', 2).toLowerCase()
      }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'

const { t } = useI18n()

const emit = defineEmits(['filterChange'])
const selectedDays = ref('30')
const customDaysInput = ref('')

const handleFilterChange = (value) => {
  if (value === 'custom') {
    customDaysInput.value = '30'
    emit('filterChange', 30)
  } else {
    emit('filterChange', parseInt(value))
  }
}

const handleCustomDaysChange = () => {
  const days = parseInt(customDaysInput.value)
  if (days && days > 0 && days <= 365) {
    emit('filterChange', days)
  } else {
    customDaysInput.value = '30'
    emit('filterChange', 30)
  }
}

handleFilterChange(selectedDays.value)
</script>

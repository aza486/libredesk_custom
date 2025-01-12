<template>
  <div class="space-y-5">
    <div>
      <p class="text-sm-muted">{{ helptext }}</p>
    </div>
    <div v-if="type === 'new_conversation'">
      <Select v-model="executionMode">
        <SelectTrigger class="w-[280px]">
          <Settings size="18" />
          <SelectValue>{{
            executionMode === 'first_match'
              ? 'Execute the first matching rule'
              : 'Execute all matching rules'
          }}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="first_match">Execute the first matching rule</SelectItem>
          <SelectItem value="all">Execute all matching rules</SelectItem>
        </SelectContent>
      </Select>
    </div>
    <div>
      <Spinner v-if="isLoading"></Spinner>
      <div class="space-y-5" v-else>
        <draggable v-model="rules" class="space-y-5" item-key="name" @end="onDragEnd">
          <template #item="{ element }">
            <RuleList :rule="element" @delete-rule="deleteRule" @toggle-rule="toggleRule" />
          </template>
        </draggable>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import RuleList from './RuleList.vue'
import { Spinner } from '@/components/ui/spinner'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Settings } from 'lucide-vue-next'
import draggable from 'vuedraggable'
import api from '@/api'

const isLoading = ref(false)
const rules = ref([])
const executionMode = ref('all')
const props = defineProps({
  type: {
    type: String,
    required: true
  },
  helptext: {
    type: String,
    required: false
  }
})

onMounted(() => {
  fetchRules()
})

const fetchRules = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAutomationRules(props.type)
    rules.value = resp.data.data
    executionMode.value = resp.data.data[0]?.execution_mode || 'all'
  } finally {
    isLoading.value = false
  }
}

const deleteRule = async (id) => {
  await api.deleteAutomationRule(id)
  fetchRules()
}

const toggleRule = async (id) => {
  await api.toggleAutomationRule(id)
  fetchRules()
}

const onDragEnd = async () => {
  const weights = {}
  rules.value.forEach((rule, index) => {
    weights[rule.id] = index + 1
  })
  await api.updateAutomationRuleWeights(weights)
}

const updateExecutionMode = async () => {
  await api.updateAutomationRulesExecutionMode({
    mode: executionMode.value
  })
}

watch(executionMode, updateExecutionMode)
</script>

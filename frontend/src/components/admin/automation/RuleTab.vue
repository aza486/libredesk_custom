<template>
  <div class="space-y-5">
    <div>
      <p class="text-sm-muted">{{ helptext }}</p>
    </div>
    <Spinner v-if=isLoading></Spinner>
    <div class="space-y-5" v-else>
      <RuleList v-for="rule in rules" :key="rule.name" :rule="rule" @delete-rule="deleteRule"
        @toggle-rule="toggleRule" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import RuleList from './RuleList.vue'
import { Spinner } from '@/components/ui/spinner'
import api from '@/api'

const isLoading = ref(false)
const rules = ref([])

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
    let resp = await api.getAutomationRules(props.type)
    rules.value = resp.data.data
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
</script>

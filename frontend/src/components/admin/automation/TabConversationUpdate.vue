<template>
    <div class="space-y-5">
        <div>
            <p class="text-muted-foreground text-sm">Rules that run when a conversation is updated</p>
        </div>
        <div v-if="showRuleList" class="space-y-5">
            <RuleList v-for="rule in rules" :key="rule.name" :rule="rule" @delete-rule="deleteRule"  @toggle-rule="toggleRule"/>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import RuleList from './RuleList.vue'
import api from '@/api';

const showRuleList = ref(true)
const rules = ref([])

onMounted(() => {
    fetchRules()
})

const fetchRules = async () => {
    let resp = await api.getAutomationRules("conversation_update")
    rules.value = resp.data.data
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
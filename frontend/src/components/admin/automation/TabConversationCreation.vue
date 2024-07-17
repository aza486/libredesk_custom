<template>
    <div v-if="showRuleList" class="space-y-5">
        <RuleList v-for="rule in filteredRules" :key="rule.name" :rule="rule"/>
    </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import RuleList from './RuleList.vue'
import api from '@/api';

const showRuleList = ref(true)
const rules = ref([])

onMounted(async () => {
    let resp = await api.getAutomationRules()
    rules.value = resp.data.data
})

const filteredRules = computed(() =>rules.value.filter(rule => rule.type === "new_conversation"));
</script>
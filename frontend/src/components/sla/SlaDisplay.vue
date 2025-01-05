<template>
    <div v-if="dueAt">
        <span v-if="actualAt && isAfterDueTime"
            class="bg-red-100 p-1.5 rounded-xl text-xs text-red-700 border border-red-300">
            <AlertCircle class="inline-block w-4 h-4 mr-1" /> {{ label }} Overdue
        </span>
        <span v-else-if="actualAt && !isAfterDueTime" class="text-xs text-green-700">
            <span v-if="showSLAHit">
                <CheckCircle class="inline-block w-4 h-4 mr-1" /> {{ label }} SLA Hit
            </span>
        </span>
        <span v-else-if="sla?.status === 'remaining'"
            class="bg-yellow-100 p-1.5 rounded-xl text-xs text-yellow-700 border border-yellow-300">
            <Clock class="inline-block w-4 h-4 mr-1" /> {{ label }} {{ sla.value }}
        </span>
        <span v-else-if="sla?.status === 'overdue'"
            class="bg-red-100 p-1.5 rounded-xl text-xs text-red-700 border border-red-300">
            <AlertCircle class="inline-block w-4 h-4 mr-1" /> {{ label }} Overdue by {{ sla.value }}
        </span>
    </div>
</template>

<script setup>
import { ref } from 'vue'
import { useSla } from '@/composables/useSla'
import { AlertCircle, CheckCircle, Clock } from 'lucide-vue-next'

const props = defineProps({
    dueAt: String,
    actualAt: String,
    label: String,
    showSLAHit: {
        type: Boolean,
        default: true
    }
})

const { sla, isAfterDueTime } = useSla(ref(props.dueAt), ref(props.actualAt))
</script>

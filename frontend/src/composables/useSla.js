// composables/useSla.js
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { isAfter } from 'date-fns'
import { calculateSla } from '@/utils/sla'

export function useSla (dueAt, actualAt) {
    const sla = ref(null)

    const isAfterDueTime = computed(() => {
        if (!dueAt.value || !actualAt.value) return false
        return isAfter(new Date(actualAt.value), new Date(dueAt.value))
    })

    function updateSla () {
        if (!dueAt.value) {
            sla.value = null
            return
        }
        sla.value = calculateSla(dueAt.value)
    }

    onMounted(() => {
        updateSla()
        // Update the SLA every 30 seconds.
        const intervalId = setInterval(updateSla, 30000)
        onUnmounted(() => {
            clearInterval(intervalId)
        })
    })

    return { sla, isAfterDueTime, updateSla }
}

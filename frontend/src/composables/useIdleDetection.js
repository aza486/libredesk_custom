import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useUserStore } from '@/stores/user'
import { debounce } from '@/utils/debounce'
import { useStorage } from '@vueuse/core'

export function useIdleDetection () {
    const userStore = useUserStore()
    // 4 minutes
    const AWAY_THRESHOLD = 4 * 60 * 1000
    // 1 minute
    const CHECK_INTERVAL = 60 * 1000

    // Store last activity time in localStorage to sync across tabs
    const lastActivity = useStorage('last_active', Date.now())
    const timer = ref(null)

    function resetTimer () {
        if (userStore.user.availability_status === 'away' || userStore.user.availability_status === 'offline') {
            userStore.updateUserAvailability('online', false)
        }
        const now = Date.now()
        if (lastActivity.value < now) {
            lastActivity.value = now
        }
    }

    const debouncedResetTimer = debounce(resetTimer, 200)

    function checkIdle () {
        if (Date.now() - lastActivity.value > AWAY_THRESHOLD &&
            userStore.user.availability_status === 'online') {
            userStore.updateUserAvailability('away', false)
        }
    }

    onMounted(() => {
        window.addEventListener('mousemove', debouncedResetTimer)
        window.addEventListener('keypress', debouncedResetTimer)
        window.addEventListener('click', debouncedResetTimer)
        timer.value = setInterval(checkIdle, CHECK_INTERVAL)
    })

    onBeforeUnmount(() => {
        window.removeEventListener('mousemove', debouncedResetTimer)
        window.removeEventListener('keypress', debouncedResetTimer)
        window.removeEventListener('click', debouncedResetTimer)
        if (timer.value) {
            clearInterval(timer.value)
            timer.value = null
        }
    })

    // Watch for lastActivity changes in localStorage to handle multi-tab sync
    watch(lastActivity, (newVal, oldVal) => {
        if (newVal > oldVal) {
            resetTimer()
        }
    })
}

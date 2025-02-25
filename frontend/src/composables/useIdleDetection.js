import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useUserStore } from '@/stores/user'
import { debounce } from '@/utils/debounce'

export function useIdleDetection () {
    const userStore = useUserStore()
    // 4 minutes
    const AWAY_THRESHOLD = 4 * 60 * 1000
    // 1 minute
    const CHECK_INTERVAL = 60 * 1000
    const lastActivity = ref(Date.now())
    const timer = ref(null)

    function resetTimer () {
        if (userStore.user.availability_status === 'away' || userStore.user.availability_status === 'offline') {
            userStore.updateUserAvailability('online', false)
        }
        lastActivity.value = Date.now()
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
        clearInterval(timer.value)
    })
}
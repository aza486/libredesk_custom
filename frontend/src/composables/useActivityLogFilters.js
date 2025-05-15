import { computed } from 'vue'
import { useUsersStore } from '@/stores/users'
import { FIELD_TYPE, FIELD_OPERATORS } from '@/constants/filterConfig'

export function useActivityLogFilters () {
    const uStore = useUsersStore()
    const activityLogListFilters = computed(() => ({
        actor_id: {
            label: 'Actor',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: uStore.options
        },
        activity_type: {
            label: 'Activity type',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: [{
                label: 'Login',
                value: 'login'
            }, {
                label: 'Logout',
                value: 'logout'
            }, {
                label: 'Away',
                value: 'away'
            }, {
                label: 'Away Reassigned',
                value: 'away_reassigned'
            }, {
                label: 'Online',
                value: 'online'
            }]
        },
    }))
    return {
        activityLogListFilters
    }
}
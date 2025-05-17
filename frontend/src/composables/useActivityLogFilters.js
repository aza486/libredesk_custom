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
                label: 'User login',
                value: 'agent_login'
            }, {
                label: 'User logout',
                value: 'agent_logout'
            }, {
                label: 'User away',
                value: 'agent_away'
            }, {
                label: 'User away reassigned',
                value: 'agent_away_reassigned'
            }, {
                label: 'User online',
                value: 'agent_online'
            }]
        },
    }))
    return {
        activityLogListFilters
    }
}
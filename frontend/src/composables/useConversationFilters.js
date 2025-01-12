import { computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useInboxStore } from '@/stores/inbox'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useSlaStore } from '@/stores/sla'
import { FIELD_TYPE, FIELD_OPERATORS } from '@/constants/filterConfig'

export function useConversationFilters () {
    const cStore = useConversationStore()
    const iStore = useInboxStore()
    const uStore = useUsersStore()
    const tStore = useTeamStore()
    const slaStore = useSlaStore()

    const conversationsListFilters = computed(() => ({
        status_id: {
            label: 'Status',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusesForSelect
        },
        priority_id: {
            label: 'Priority',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.prioritiesForSelect
        },
        assigned_team_id: {
            label: 'Assigned team',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: tStore.forSelect
        },
        assigned_user_id: {
            label: 'Assigned user',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: uStore.forSelect
        },
        inbox_id: {
            label: 'Inbox',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: iStore.forSelect
        }
    }))

    const conversationFilters = computed(() => ({
        content: {
            label: 'Content',
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT
        },
        subject: {
            label: 'Subject',
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT
        },
        status: {
            label: 'Status',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusesForSelect
        },
        priority: {
            label: 'Priority',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.prioritiesForSelect
        },
        assigned_team: {
            label: 'Assigned team',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: tStore.forSelect
        },
        assigned_user: {
            label: 'Assigned agent',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: uStore.forSelect
        },
        hours_since_created: {
            label: 'Hours since created',
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        hours_since_resolved: {
            label: 'Hours since resolved',
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        inbox: {
            label: 'Inbox',
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: iStore.forSelect
        }
    }))

    const conversationActions = computed(() => ({
        assign_team: {
            label: 'Assign to team',
            type: FIELD_TYPE.SELECT,
            options: tStore.forSelect
        },
        assign_user: {
            label: 'Assign to user',
            type: FIELD_TYPE.SELECT,
            options: uStore.forSelect
        },
        set_status: {
            label: 'Set status',
            type: FIELD_TYPE.SELECT,
            options: cStore.statusesForSelect
        },
        set_priority: {
            label: 'Set priority',
            type: FIELD_TYPE.SELECT,
            options: cStore.prioritiesForSelect
        },
        send_private_note: {
            label: 'Send private note',
            type: FIELD_TYPE.RICHTEXT
        },
        send_reply: {
            label: 'Send reply',
            type: FIELD_TYPE.RICHTEXT
        },
        set_sla: {
            label: 'Set SLA',
            type: FIELD_TYPE.SELECT,
            options: slaStore.forSelect
        }
    }))

    return {
        conversationsListFilters,
        conversationFilters,
        conversationActions
    }
}
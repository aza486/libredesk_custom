import { ref } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useInboxStore } from '@/stores/inbox'
import { useAgentStore } from '@/stores/agent'
import { useTeamStore } from '@/stores/team'

const OPERATORS = ["equals", "not equals", "set", "not set"]

export const useConversationFilters = () => {
    const conversationStore = useConversationStore()
    const iStore = useInboxStore()
    const aStore = useAgentStore()
    const tStore = useTeamStore()
    const conversationsListFilters = ref(null)
    const singleConversationFilters = ref(null)

    const initConversationListFilters = async () => {
        await Promise.all([
            conversationStore.fetchStatuses(),
            conversationStore.fetchPriorities(),
            aStore.fetchAgents(),
            tStore.fetchTeams(),
            iStore.fetchInboxes()
        ])

        conversationsListFilters.value = {
            status_id: {
                label: 'Status',
                operators: OPERATORS,
                options: conversationStore.statusesForSelect
            },
            priority_id: {
                label: 'Priority',
                operators: OPERATORS,
                options: conversationStore.prioritiesForSelect
            },
            assigned_team_id: {
                label: 'Assigned team',
                operators: OPERATORS,
                options: tStore.forSelect
            },
            assigned_user_id: {
                label: 'Assigned user',
                operators: OPERATORS,
                options: aStore.forSelect
            },
            inbox_id: {
                label: 'Inbox',
                operators: OPERATORS,
                options: iStore.forSelect,
            },
        }
    }

    const initSingleConversationFilters = () => {
        singleConversationFilters.value = {
            content: {
                label: 'Content',
                operators: OPERATORS
            },
            subject: {
                label: 'Subject',
                operators: OPERATORS
            },
            status_id: {
                label: 'Status',
                operators: OPERATORS,
                options: conversationStore.statusesForSelect
            },
            priority_id: {
                label: 'Priority',
                operators: OPERATORS,
                options: conversationStore.prioritiesForSelect
            },
            assigned_team_id: {
                label: 'Assigned team',
                operators: OPERATORS
            },
            assigned_user_id: {
                label: 'Assigned user',
                operators: OPERATORS
            },
        }
    }

    return {
        conversationsListFilters,
        singleConversationFilters,
        initConversationListFilters,
        initSingleConversationFilters
    }
}
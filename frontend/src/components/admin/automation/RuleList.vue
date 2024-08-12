<template>
    <div class="flex flex-col box px-5 py-6 rounded-lg justify-center">
        <div class="flex justify-between space-y-3">
            <div>
                <span class="sub-title space-x-3 flex justify-center items-center">
                    <div class="text-base">
                        {{ rule.name }}
                    </div>
                    <div class="mb-1">
                        <Badge v-if="!rule.disabled">Enabled</Badge>
                        <Badge v-else variant="secondary">Disabled</Badge>
                    </div>
                </span>
            </div>
            <div>
                <DropdownMenu>
                    <DropdownMenuTrigger as-child>
                        <button>
                            <EllipsisVertical size=21></EllipsisVertical>
                        </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent>
                        <DropdownMenuItem @click="navigateToEditRule(rule.id)">
                            <span>Edit</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem @click="$emit('delete-rule', rule.id)">
                            <span>Delete</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem @click="$emit('toggle-rule', rule.id)" v-if="!rule.disabled">
                            <span>Disable</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem @click="$emit('toggle-rule', rule.id)" v-else>
                            <span>Enable</span>
                        </DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            </div>
        </div>
        <p class="text-sm-muted">
            {{ rule.description }}
        </p>
    </div>
</template>

<script setup>
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { EllipsisVertical } from 'lucide-vue-next';
import { useRouter } from 'vue-router'
import { Badge } from '@/components/ui/badge'

const router = useRouter()
defineEmits(['delete-rule', 'toggle-rule'])

defineProps({
    rule: {
        type: Object,
        required: true,
    },
})

const navigateToEditRule = (id) => {
    router.push({ path: `/admin/automations/${id}/edit` })
}
</script>
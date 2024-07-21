<template>
    <div class="flex flex-col box border p-5 space-y-2">
        <div class="flex justify-between">
            <div>
                <span class="admin-subtitle flex space-x-3 items-center">
                    <div class="text-base">
                        {{ rule.name }}
                    </div>
                    <div>
                        <Badge v-if="!rule.disabled" class="text-[10px] py-0 px-1">Enabled</Badge>
                        <Badge v-else class="text-[10px] py-0 px-1" variant="secondary">Disabled</Badge>
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
        <p class="text-muted-foreground text-sm">
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
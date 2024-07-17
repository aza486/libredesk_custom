<template>
    <div class="flex flex-col box border p-5 space-y-2">
        <div class="flex justify-between">
            <div>
                <span class="admin-subtitle">
                    {{ rule.name }}
                </span>
            </div>
            <div>
                <DropdownMenu>
                    <DropdownMenuTrigger as-child>
                        <Button variant="ghost">
                            <EllipsisVertical size=21></EllipsisVertical>
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent>
                        <DropdownMenuItem @click="editRule(rule.id)">
                            <span>Edit</span>
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
const router = useRouter()

defineProps({
    rule: {
        type: Object,
        required: true,
    },
})

const deleteRule = () => {

}

const editRule = (id) => {
    router.push({ path: `/admin/automations/${id}/edit` })
}
</script>
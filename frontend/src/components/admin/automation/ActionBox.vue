<template>
    <div class="box border p-5 space-y-5 rounded">
        <div class="space-y-5">
            <div v-for="(action, index) in actions" :key="index" class="space-y-5">
                <div v-if="index > 0">
                    <hr class="border-t-2 border-dotted border-gray-300">
                </div>
                <div class="flex space-x-5 justify-between">
                    <Select v-model="action.type"
                        @update:modelValue="(value) => handleFieldChange(value, index)">
                        <SelectTrigger class="w-56">
                            <SelectValue placeholder="Select action" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectLabel>Conversation</SelectLabel>
                                <SelectItem v-for="(actionItem, key) in conversationActions" :key="key" :value="key">
                                    {{ actionItem.label }}
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                    <div class="cursor-pointer" @click="removeAction(index)">
                        <CircleX size="21" />
                    </div>
                </div>
                <div>
                    <Input type="text" placeholder="Set value" :modelValue="action.value"
                        @update:modelValue="(value) => handleValueChange(value, index)" />
                </div>
            </div>
        </div>
        <div>
            <Button variant="outline" @click="addAction" size="sm">Add action</Button>
        </div>
    </div>
</template>

<script setup>
import { toRefs } from 'vue'
import { Button } from '@/components/ui/button'
import { CircleX } from 'lucide-vue-next';
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectLabel,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'

const props = defineProps({
    actions: {
        type: Object,
        required: true,
    },
})

const { actions } = toRefs(props)

const emit = defineEmits(['update-actions', 'add-action', 'remove-action'])

const handleFieldChange = (value, index) => {
    actions.value[index].type = value
    emitUpdate(index)
}

const handleValueChange = (value, index) => {
    actions.value[index].value = value
    emitUpdate(index)
}

const removeAction = (index) => {
    emit('remove-action', index)
}

const addAction = (index) => {
    emit('add-action', index)
}

const emitUpdate = (index) => {
    emit('update-actions', actions, index)
}

const conversationActions = {
    "assign_team": {
        "label": "Assign to team"
    },
    "assign_user": {
        "label": "Assign to user"
    },
    "set_status": {
        "label": "Set status"
    },
    "set_priority": {
        "label": "Set priority"
    },
}
</script>
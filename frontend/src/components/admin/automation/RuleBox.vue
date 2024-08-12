<template>
    <div>
        <div class="mb-5">
            <RadioGroup class="flex" :modelValue="ruleGroup.logical_op" @update:modelValue="handleGroupOperator">
                <div class="flex items-center space-x-2">
                    <RadioGroupItem value="OR" />
                    <Label for="r1">Match <b>ANY</b> of below.</Label>
                </div>
                <div class="flex items-center space-x-2">
                    <RadioGroupItem value="AND" />
                    <Label for="r1">Match <b>ALL</b> of below.</Label>
                </div>
            </RadioGroup>
        </div>
        <div class="box border p-5 space-y-5 rounded-lg">
            <div class="space-y-5">
                <div v-for="(rule, index) in ruleGroup.rules" :key="rule" class="space-y-5">
                    <div v-if="index > 0">
                        <hr class="border-t-2 border-dotted border-gray-300">
                    </div>
                    <div class="flex  justify-between">
                        <div class="flex space-x-5">
                            <Select v-model="rule.field"
                                @update:modelValue="(value) => handleFieldChange(value, index)">
                                <SelectTrigger class="w-56">
                                    <SelectValue placeholder="Select field" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectGroup>
                                        <SelectLabel>Conversation</SelectLabel>
                                        <SelectItem v-for="(field, key) in conversationFields" :key="key" :value="key">
                                            {{ field.label }}
                                        </SelectItem>
                                    </SelectGroup>
                                </SelectContent>
                            </Select>

                            <Select v-model="rule.operator"
                                @update:modelValue="(value) => handleOperatorChange(value, index)">
                                <SelectTrigger class="w-56">
                                    <SelectValue placeholder="Select operator" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectGroup>
                                        <SelectItem v-for="(field, key) in operators" :key="key" :value="key">
                                            {{ field.label }}
                                        </SelectItem>
                                    </SelectGroup>
                                </SelectContent>
                            </Select>
                        </div>
                        <div class="cursor-pointer" @click="removeCondition(index)">
                            <CircleX size="21" />
                        </div>
                    </div>
                    <div>
                        <Input type="text" placeholder="Set value" :modelValue="rule.value"
                            @update:modelValue="(value) => handleValueChange(value, index)" />
                    </div>
                    <div class="flex items-center space-x-2">
                        <Checkbox id="terms" :defaultChecked="rule.case_sensitive_match"
                            @update:checked="(value) => handleCaseSensitiveCheck(value, index)" />
                        <label for="terms">
                            Case sensitive match
                        </label>
                    </div>
                </div>

            </div>
            <div>
                <Button variant="outline" size="sm" @click="addCondition">Add condition</Button>
            </div>
        </div>
    </div>
</template>

<script setup>
import { toRefs } from 'vue'
import { Checkbox } from '@/components/ui/checkbox'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Button } from '@/components/ui/button'
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectLabel,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { CircleX } from 'lucide-vue-next';
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'

const props = defineProps({
    ruleGroup: {
        type: Object,
        required: true,
    },
    groupIndex: {
        Type: Number,
        required: true,
    }
})

const { ruleGroup } = toRefs(props)

const emit = defineEmits(['update-group', 'add-condition', 'remove-condition'])

const handleGroupOperator = (value) => {
    ruleGroup.value.logical_op = value
    emitUpdate()
}

const handleFieldChange = (value, ruleIndex) => {
    ruleGroup.value.rules[ruleIndex].field = value
    emitUpdate()
}

const handleOperatorChange = (value, ruleIndex) => {
    ruleGroup.value.rules[ruleIndex].operator = value
    emitUpdate()
}

const handleValueChange = (value, ruleIndex) => {
    ruleGroup.value.rules[ruleIndex].value = value
    emitUpdate()
}

const handleCaseSensitiveCheck = (value, ruleIndex) => {
    ruleGroup.value.rules[ruleIndex].case_sensitive_match = value
    emitUpdate()
}

const removeCondition = (index) => {
    emit('remove-condition', props.groupIndex, index)
}

const addCondition = () => {
    emit('add-condition', props.groupIndex)
}

const emitUpdate = () => {
    emit('update-group', ruleGroup, props.groupIndex)
}

const conversationFields = {
    "content": {
        "label": "Content"
    },
    "subject": {
        "label": "Subject"
    },
    "status": {
        "label": "Status"
    },
    "priority": {
        "label": "Priority"
    },
    "assigned_team": {
        "label": "Assigned team"
    },
    "assigned_user": {
        "label": "Assigned user"
    }
}

const operators = {
    "contains": {
        "label": "Contains"
    },
    "not_contains": {
        "label": "Not contains"
    },
    "equals": {
        "label": "Equals"
    },
    "not_equals": {
        "label": "Not equals"
    },
    "set": {
        "label": "Set"
    },
    "not_set": {
        "label": "Not set"
    }
}
</script>
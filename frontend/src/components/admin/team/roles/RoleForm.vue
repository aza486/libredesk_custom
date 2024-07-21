<template>
    <form @submit.prevent="onSubmit" class="w-2/3 space-y-6">
        <FormField v-slot="{ componentField }" name="name">
            <FormItem v-auto-animate>
                <FormLabel>Name</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="Agent" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="description">
            <FormItem>
                <FormLabel>Description</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="This role is for all support agents" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <p class="text-xl">Set permissions for this role</p>

        <div v-for="entity in permissions" :key="entity.name" class="space-y-4">
            <p class="text-lg">{{ entity.name }}</p>
            <div class="space-y-2">
                <FormField v-for="permission in entity.permissions" :key="permission.name" v-slot="{ value }"
                    type="checkbox" :name="permission.name">
                    <FormItem class="flex flex-col gap-y-5 space-y-0 border p-4 box">
                        <div class="flex space-x-5">
                            <FormControl>
                                <Checkbox :checked="selectedPermissions.includes(permission.name)"
                                    @update:checked="(newValue) => handleChange(newValue, permission.name)" />
                                <FormLabel>{{ permission.label }}</FormLabel>
                            </FormControl>
                        </div>
                        <div class="space-y-1 leading-none" v-if="permission.subOptions">
                            <div class="ml-6 space-y-1">
                                <FormField v-for="subOption in permission.subOptions" :key="subOption.name"
                                    v-slot="{ value: subValue }" type="checkbox" :name="subOption.name">
                                    <FormItem class="flex flex-row items-start gap-x-3 space-y-0 p-4">
                                        <FormControl>
                                            <Checkbox :checked="selectedPermissions.includes(subOption.name)"
                                                @update:checked="(newValue) => handleChange(newValue, subOption.name)" />
                                        </FormControl>
                                        <div class="space-y-1 leading-none">
                                            <FormLabel>{{ subOption.label }}</FormLabel>
                                        </div>
                                    </FormItem>
                                </FormField>
                            </div>
                        </div>
                    </FormItem>
                </FormField>
            </div>
        </div>

        <Button type="submit" size="sm">{{ submitLabel }}</Button>
    </form>
</template>

<script setup>
import { watch, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { Checkbox } from '@/components/ui/checkbox'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'

const props = defineProps({
    initialValues: {
        type: Object,
        required: false
    },
    submitForm: {
        type: Function,
        required: true
    },
    submitLabel: {
        type: String,
        required: false,
        default: () => 'Submit'
    },
})

const permissions = ref([
    {
        name: 'Conversation',
        permissions: [
            {
                name: 'conversation:reply',
                label: 'Reply to a conversation',
            },
            {
                name: 'conversation:edit_all_properties',
                label: 'Edit conversation properties',
                subOptions: [
                    { name: 'conversation:edit_status', label: 'Edit status' },
                    { name: 'conversation:edit_priority', label: 'Edit priority' },
                    { name: 'conversation:edit_team', label: 'Edit team' },
                    { name: 'conversation:edit_agent', label: 'Edit agent' },
                ],
            },
            {
                name: 'conversation:all',
                label: 'View all conversations',
                subOptions: [
                    { name: 'conversation:team', label: 'View team conversations' },
                    { name: 'conversation:assigned', label: 'View assigned conversations' },
                ],
            },
        ],
    },
    {
        name: 'Admin',
        permissions: [
            {
                name: 'admin:get',
                label: 'Access to the admin panel',
                subOptions: [
                    { name: 'inboxes:manage', label: 'Manage inboxes' },
                    { name: 'users:manage', label: 'Manage users' },
                    { name: 'teams:manage', label: 'Manage teams' },
                    { name: 'roles:manage', label: 'Manage roles' },
                    { name: 'automations:manage', label: 'Manage automations' },
                ],
            },
        ]
    }
])

const selectedPermissions = ref([])

const form = useForm({
    validationSchema: toTypedSchema(formSchema),
    initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
    values.permissions = selectedPermissions.value
    console.log("submitting ", values)
    props.submitForm(values)
})

const handleChange = (value, perm) => {
    if (value) {
        selectedPermissions.value.push(perm)
    } else {
        const index = selectedPermissions.value.indexOf(perm)
        if (index > -1) {
            selectedPermissions.value.splice(index, 1)
        }
    }
}

// Watch for changes in initialValues and update the form.
watch(
    () => props.initialValues,
    (newValues) => {
        if (newValues) {
            form.setValues(newValues)
            selectedPermissions.value = newValues.permissions || []
        }
    },
    { immediate: true }
)
</script>
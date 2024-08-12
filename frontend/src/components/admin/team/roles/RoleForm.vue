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

        <p class="text-base">Set permissions for this role</p>

        <div v-for="entity in permissions" :key="entity.name" class="border box p-4 rounded-lg shadow-sm">
            <p class="text-lg mb-5">{{ entity.name }}</p>
            <div class="space-y-4">
                <FormField v-for="permission in entity.permissions" :key="permission.name" type="checkbox"
                    :name="permission.name">
                    <FormItem class="flex flex-col gap-y-5 space-y-0 rounded-lg">
                        <div class="flex space-x-3">
                            <FormControl>
                                <Checkbox :checked="selectedPermissions.includes(permission.name)"
                                    @update:checked="(newValue) => handleChange(newValue, permission.name)" />
                                <FormLabel>{{ permission.label }}</FormLabel>
                            </FormControl>
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
            { name: 'conversation:reply', label: 'Reply to conversations' },
            { name: 'conversation:edit_all_properties', label: 'Edit all conversation properties' },
            { name: 'conversation:edit_status', label: 'Edit conversation status' },
            { name: 'conversation:edit_priority', label: 'Edit conversation priority' },
            { name: 'conversation:edit_team', label: 'Edit conversation team' },
            { name: 'conversation:edit_user', label: 'Edit conversation user' },
            { name: 'conversation:view_all', label: 'View all conversations' },
            { name: 'conversation:view_team', label: 'View team conversations' },
            { name: 'conversation:view_assigned', label: 'View assigned conversations' }
        ]
    },
    {
        name: 'Admin',
        permissions: [
            { name: 'admin:access', label: 'Access the admin panel' },
            { name: 'settings:manage_general', label: 'Manage general settings' },
            { name: 'settings:manage_file', label: 'Manage file upload settings' },
            { name: 'login:manage', label: 'Manage login settings' },
            { name: 'inboxes:manage', label: 'Manage inboxes' },
            { name: 'users:manage', label: 'Manage users' },
            { name: 'teams:manage', label: 'Manage teams' },
            { name: 'roles:manage', label: 'Manage roles' },
            { name: 'automations:manage', label: 'Manage automations' },
            { name: 'templates:manage', label: 'Manage templates' }
        ]
    },
    {
        name: 'Dashboard',
        permissions: [
            { name: 'dashboard:view_global', label: 'Access global dashboard' },
            { name: 'dashboard:view_team_self', label: 'Access dashboard of teams the user is part of' }
        ]
    }
]);

const selectedPermissions = ref([])

const form = useForm({
    validationSchema: toTypedSchema(formSchema),
    initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
    values.permissions = selectedPermissions.value
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
    { deep: true }
)
</script>

<template>
    <form @submit="onSubmit" class="w-2/3 space-y-6">
        <FormField v-slot="{ componentField }" name="provider_url">
            <FormItem v-auto-animate>
                <FormLabel>Provider URL</FormLabel>
                <FormControl>
                    <Input type="url" placeholder="Provider URL" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="client_id">
            <FormItem v-auto-animate>
                <FormLabel>Client ID</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="Client ID" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="client_secret">
            <FormItem v-auto-animate>
                <FormLabel>Client Secret</FormLabel>
                <FormControl>
                    <Input type="password" placeholder="Client Secret" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="redirect_uri">
            <FormItem v-auto-animate>
                <FormLabel>Redirect URI</FormLabel>
                <FormControl>
                    <Input type="url" placeholder="Redirect URI" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <Button type="submit" size="sm"> {{ submitLabel }} </Button>
    </form>
</template>

<script setup>
import { watch } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { oidcLoginFormSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import {
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@/components/ui/form'
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
        default: () => 'Save'
    },
})

const form = useForm({
    validationSchema: toTypedSchema(oidcLoginFormSchema),
})

const onSubmit = form.handleSubmit((values) => {
    props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
    () => props.initialValues,
    (newValues) => {
        form.setValues(newValues)
    },
    { deep: true }
)
</script>
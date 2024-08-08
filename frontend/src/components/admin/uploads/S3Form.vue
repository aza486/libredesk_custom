<template>
    <form @submit="onS3FormSubmit" class="w-2/3 space-y-6">
        <FormField v-slot="{ componentField }" name="provider">
            <FormItem>
                <FormLabel>Provider</FormLabel>
                <FormControl>
                    <Select v-bind="componentField" v-model="componentField.modelValue"
                        @update:modelValue="handleProviderUpdate">
                        <SelectTrigger>
                            <SelectValue placeholder="Select a provider" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem value="s3">
                                    S3
                                </SelectItem>
                                <SelectItem value="localfs">
                                    Local filesystem
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="region">
            <FormItem v-auto-animate>
                <FormLabel>Region</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="ap-south-1" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="access_key">
            <FormItem v-auto-animate>
                <FormLabel>AWS access key</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="AWS access key" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="access_secret">
            <FormItem v-auto-animate>
                <FormLabel>AWS access secret</FormLabel>
                <FormControl>
                    <Input type="password" placeholder="AWS access secret" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="bucket_type">
            <FormItem>
                <FormLabel>Bucket type</FormLabel>
                <FormControl>
                    <Select v-bind="componentField" v-model="componentField.modelValue">
                        <SelectTrigger>
                            <SelectValue placeholder="Select bucket type" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectGroup>
                                <SelectItem value="public">
                                    Public
                                </SelectItem>
                                <SelectItem value="private">
                                    Private
                                </SelectItem>
                            </SelectGroup>
                        </SelectContent>
                    </Select>
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="bucket">
            <FormItem v-auto-animate>
                <FormLabel>Bucket</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="Bucket" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="bucket_path">
            <FormItem v-auto-animate>
                <FormLabel>Bucket path</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="Bucket path" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="upload_expiry">
            <FormItem v-auto-animate>
                <FormLabel>Upload expiry</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="24h" v-bind="componentField" />
                </FormControl>
                <FormDescription>Only applicable for private buckets.</FormDescription>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="url">
            <FormItem v-auto-animate>
                <FormLabel>S3 backend URL</FormLabel>
                <FormControl>
                    <Input type="url" placeholder="https://ap-south-1.s3.amazonaws.com" v-bind="componentField"
                        defaultValue="https://ap-south-1.s3.amazonaws.com" />
                </FormControl>
                <FormDescription>Only change if using a custom S3 compatible backend like Minio.</FormDescription>
                <FormMessage />
            </FormItem>
        </FormField>
        <Button type="submit" size="sm"> {{ submitLabel }} </Button>
    </form>
</template>

<script setup>
import { watch, defineEmits } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { s3FormSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import {
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
    FormDescription
} from '@/components/ui/form'
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
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

const emit = defineEmits(['provider-update'])

const s3Form = useForm({
    validationSchema: toTypedSchema(s3FormSchema),
})

const onS3FormSubmit = s3Form.handleSubmit((values) => {
    props.submitForm(values)
})

const handleProviderUpdate = (value) => {
    emit('provider-update', value)
}

// Watch for changes in initialValues and update the form.
watch(
    () => props.initialValues,
    (newValues) => {
        s3Form.setValues(newValues)
    },
    { deep: true, immediate: true }
)

</script>

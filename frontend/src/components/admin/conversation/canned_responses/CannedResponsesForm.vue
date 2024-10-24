<template>
    <form class="space-y-6">
        <FormField v-slot="{ componentField }" name="title">
            <FormItem>
                <FormLabel>Title</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="" v-bind="componentField" />
                </FormControl>
                <FormDescription></FormDescription>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField, handleInput }" name="content">
            <FormItem>
                <FormLabel>Content</FormLabel>
                <FormControl>
                    <QuillEditor theme="snow" v-model:content="componentField.modelValue" contentType="html"
                        @update:content="handleInput"></QuillEditor>
                </FormControl>
                <FormDescription></FormDescription>
                <FormMessage />
            </FormItem>
        </FormField>

        <!-- Form submit button slot -->
        <slot name="footer"></slot>
    </form>
</template>

<script setup>
import { QuillEditor } from '@vueup/vue-quill'
import '@vueup/vue-quill/dist/vue-quill.snow.css';
import {
    FormControl,
    FormDescription,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
</script>
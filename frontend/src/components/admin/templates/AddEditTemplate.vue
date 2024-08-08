<template>
    <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>
    <TemplateForm :initial-values="template" :submitForm="submitForm" />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import TemplateForm from './TemplateForm.vue'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'

const template = ref({})
const router = useRouter()

const props = defineProps({
    id: {
        type: String,
        required: true
    }
})

const submitForm = async (values) => {
    if (props.id) {
        await api.updateTemplate(props.id, values)
    } else {
        await api.createTemplate(values)
        router.push("/admin/templates")
    }
}

const breadCrumLabel = () => {
    return props.id ? "Edit" : 'New';
}

const breadcrumbLinks = [
    { path: '/admin/templates', label: 'Templates' },
    { path: '#', label: breadCrumLabel() }
]

onMounted(async () => {
    if (props.id) {
        try {
            const resp = await api.getTemplate(props.id)
            template.value = resp.data.data
        } catch (error) {
            console.log(error)
        }
    }
})
</script>
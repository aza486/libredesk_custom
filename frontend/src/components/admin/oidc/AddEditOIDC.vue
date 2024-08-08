<template>
    <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>
    <OIDCForm :initial-values="oidc" :submitForm="submitForm" />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import OIDCForm from './OIDCForm.vue'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'

const oidc = ref({})
const router = useRouter()

const props = defineProps({
    id: {
        type: String,
        required: true
    }
})

const submitForm = async (values) => {
    if (props.id) {
        await api.updateOIDC(props.id, values)
    } else {
        await api.createOIDC(values)
        router.push("/admin/oidc")
    }
}

const breadCrumLabel = () => {
    return props.id ? "Edit" : 'New';
}

const breadcrumbLinks = [
    { path: '/admin/oidc', label: 'OIDC' },
    { path: '#', label: breadCrumLabel() }
]

onMounted(async () => {
    if (props.id) {
        try {
            const resp = await api.getOIDC(props.id)
            oidc.value = resp.data.data
        } catch (error) {
            console.log(error)
        }
    }
})

</script>
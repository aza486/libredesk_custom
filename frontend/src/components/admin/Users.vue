<template>
    <div class="flex justify-end mb-4">
        <form>
            <Dialog>
                <DialogTrigger as-child>
                    <Button>
                        Add user
                    </Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[425px]">
                    <DialogHeader>
                        <DialogTitle>Create a new user</DialogTitle>
                        <DialogDescription>
                        </DialogDescription>
                    </DialogHeader>
                    <AddUsersForm></AddUsersForm>
                </DialogContent>
            </Dialog>
        </form>
    </div>
    <div class="w-full">
        <DataTable :columns="columns" :data="data" />
    </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { columns } from '@/components/admin/UsersDataTableColumns.js'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/admin/DataTable.vue'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'
import AddUsersForm from '@/components/admin/AddUsersForm.vue'
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import api from '@/api';
const { toast } = useToast()


const data = ref([])

const getData = async () => {
    try {
        const response = await api.getUsers()
        data.value = response.data.data
    } catch (error) {
        toast({
            title: 'Uh oh! Could not fetch users.',
            variant: 'destructive',
            description: handleHTTPError(error).message,
        });
    }
}

onMounted(async () => {
    getData()
})

</script>
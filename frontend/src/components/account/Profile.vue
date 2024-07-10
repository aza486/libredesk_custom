<script setup>
import { h, ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { useUserStore } from '@/stores/user'

import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
  FormControl,
  FormDescription,
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { toast } from '@/components/ui/toast'
import AvatarUploader from '@/components/account/AvatarUploader.vue'

const formSchema = toTypedSchema(z.object({
  username: z.string().min(2).max(50),
  bio: z.string().optional(),
  file: z.any().optional(),
}))

const userStore = useUserStore()

const { isFieldDirty, handleSubmit } = useForm({
  validationSchema: formSchema,
})

const avatarImageFile = ref(null)
const avatarImageURL = ref(null)

const onAvatarImageChange = ({ file, url }) => {
  avatarImageFile.value = file
  avatarImageURL.value = url

}

const onAvatarImageDelete = () => {
  avatarImageURL.value = null
  avatarImageFile.value = null
}

const onSubmit = handleSubmit((values) => {
  toast({
    title: 'You submitted the following values:',
    description: h('pre', { class: 'mt-2 w-[340px] rounded-md bg-slate-950 p-4' }, h('code', { class: 'text-white' }, JSON.stringify(values, null, 2))),
  })
})

</script>

<template>
  <Form class="w-2/3 space-y-6" @submit="onSubmit">
    <FormField v-slot="{ componentField }" name="bio">
      <FormItem v-auto-animate>
        <FormLabel>Bio</FormLabel>
        <FormControl>
          <Textarea placeholder="Tell us a little bit about yourself" class="resize-none" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <Button type="submit">
      Update
    </Button>
  </Form>
</template>

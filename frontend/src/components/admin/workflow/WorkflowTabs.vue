<template>
    <Tabs default-value="conversation_creation">
        <TabsList class="grid w-full grid-cols-3">
            <TabsTrigger value="conversation_creation">
                Conversation creation
            </TabsTrigger>
            <TabsTrigger value="conversation_updates">
                Conversation updates
            </TabsTrigger>
            <TabsTrigger value="time_triggers">
                Time triggers
            </TabsTrigger>
        </TabsList>
        <TabsContent value="conversation_creation">
            <Card>
                <CardHeader>
                    <CardTitle>Account</CardTitle>
                    <CardDescription>
                        Make changes to your account here. Click save when you're done.
                    </CardDescription>
                </CardHeader>
                <CardContent class="space-y-2">
                    <div class="space-y-1">
                        <Label for="name">Name</Label>
                        <Input id="name" default-value="Pedro Duarte" />
                    </div>
                    <div class="space-y-1">
                        <Label for="username">Username</Label>
                        <Input id="username" default-value="@peduarte" />
                    </div>
                </CardContent>
                <CardFooter>
                    <Button>Save changes</Button>
                </CardFooter>
            </Card>
        </TabsContent>
        <TabsContent value="conversation_updates">
            <Card>
                <CardHeader>
                    <CardTitle>Password</CardTitle>
                    <CardDescription>
                        Change your password here. After saving, you'll be logged out.
                    </CardDescription>
                </CardHeader>
                <CardContent class="space-y-2">
                    <div class="space-y-1">
                        <Label for="current">Current password</Label>
                        <Input id="current" type="password" />
                    </div>
                    <div class="space-y-1">
                        <Label for="new">New password</Label>
                        <Input id="new" type="password" />
                    </div>
                </CardContent>
                <CardFooter>
                    <Button>Save password</Button>
                </CardFooter>
            </Card>
        </TabsContent>
        <TabsContent value="time_triggers">
            <Card>
                <CardHeader>
                    <CardTitle>Password</CardTitle>
                    <CardDescription>
                        Change your password here. After saving, you'll be logged out.
                    </CardDescription>
                </CardHeader>
                <CardContent class="space-y-2">
                    yo
                </CardContent>
                <CardFooter>
                    <Button>Save password</Button>
                </CardFooter>
            </Card>
        </TabsContent>
    </Tabs>
</template>

<script setup>
import { Button } from '@/components/ui/button'
import {
    Card,
    CardContent,
    CardDescription,
    CardFooter,
    CardHeader,
    CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
    Tabs,
    TabsContent,
    TabsList,
    TabsTrigger,
} from '@/components/ui/tabs'
</script>

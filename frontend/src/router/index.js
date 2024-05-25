import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import ConversationsView from '../views/ConversationView.vue'
import UserLoginView from '../views/UserLoginView.vue'

const routes = [
  {
    path: '/',
    redirect: '/dashboard'
  },
  {
    path: '/dashboard',
    name: "dashboard",
    component: DashboardView
  },
  {
    path: '/conversations/:uuid?',
    name: "conversations",
    component: ConversationsView,
    props: true,
  },
  {
    path: '/login',
    name: "login",
    component: UserLoginView
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: routes,
})

export default router

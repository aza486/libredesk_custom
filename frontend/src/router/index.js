import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import ConversationsView from '../views/ConversationView.vue'
import UserLoginView from '../views/UserLoginView.vue'
import AccountView from "@/views/AccountView.vue"

const routes = [
  {
    path: '/',
    name: "login",
    component: UserLoginView
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
    path: '/account/:page?',
    name: 'account',
    component: AccountView,
    props: true,
    beforeEnter: (to, from, next) => {
      if (!to.params.page) {
        next({ ...to, params: { ...to.params, page: 'profile' } });
      } else {
        next();
      }
    },
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: routes,
})

export default router

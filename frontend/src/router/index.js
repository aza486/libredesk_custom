import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import ConversationsView from '../views/ConversationView.vue'
import UserLoginView from '../views/UserLoginView.vue'
import AccountView from "@/views/AccountView.vue"
import AdminView from "@/views/AdminView.vue"
import TeamUsers from "@/components/admin/Users.vue"
import AddInbox from "@/components/admin/AddInbox.vue"

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
  },
  {
    path: '/admin/:page?',
    name: 'admin',
    component: AdminView,
    props: true,
    beforeEnter: (to, from, next) => {
      if (!to.params.page) {
        next({ ...to, params: { ...to.params, page: 'inboxes' } });
      } else {
        next();
      }
    },
    children: [
      {
        path: 'users',
        component: TeamUsers
      },
      {
        path: 'new',
        component: AddInbox
      },
    ]
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: routes,
})

export default router

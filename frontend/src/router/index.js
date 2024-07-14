import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import ConversationsView from '../views/ConversationView.vue'
import UserLoginView from '../views/UserLoginView.vue'
import AccountView from '@/views/AccountView.vue'
import AdminView from '@/views/AdminView.vue'
import Inbox from '@/components/admin/Inbox.vue'
import Team from '@/components/admin/team/Team.vue'
import Teams from '@/components/admin/team/Teams.vue'
import Users from '@/components/admin/team/Users.vue'
import Workflow from '@/components/admin/workflow/Workflow.vue'

const routes = [
  {
    path: '/',
    name: 'login',
    component: UserLoginView
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: DashboardView
  },
  {
    path: '/conversations/:uuid?',
    name: 'conversations',
    component: ConversationsView,
    props: true
  },
  {
    path: '/account/:page?',
    name: 'account',
    component: AccountView,
    props: true,
    beforeEnter: (to, from, next) => {
      if (!to.params.page) {
        next({ ...to, params: { ...to.params, page: 'profile' } })
      } else {
        next()
      }
    }
  },
  {
    path: '/admin/inboxes',
    name: 'admin',
    component: AdminView,
    children: [
      {
        path: '',
        component: Inbox
      },
      {
        path: 'new',
        component: () => import('@/components/admin/NewInbox.vue')
      },
      {
        path: ':id/edit',
        props: true,
        component: () => import('@/components/admin/EditInbox.vue')
      }
    ]
  },
  {
    path: '/admin/team',
    name: 'team',
    component: AdminView,
    children: [
      {
        path: '',
        component: Team
      },
      {
        path: 'users',
        component: Users
      },
      {
        path: 'users/new',
        component: () => import('@/components/admin/team/AddUsersForm.vue')
      },
      {
        path: 'teams/new',
        component: () => import('@/components/admin/team/AddTeamForm.vue')
      },
      {
        path: 'users/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/EditUserForm.vue')
      },
      {
        path: 'teams',
        component: Teams
      },
      {
        path: 'teams/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/EditTeamForm.vue')
      }
    ]
  },
  {
    path: '/admin/workflow',
    name: 'workflow',
    component: AdminView,
    children: [
      {
        path: '',
        component: Workflow
      },
    ]
  },
  // Fallback to dashboard.
  {
    path: '/:pathMatch(.*)*',
    redirect: (to) => {
      console.log(`Redirecting to dashboard from: ${to.fullPath}`)
      return '/dashboard'
    }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: routes
})

export default router

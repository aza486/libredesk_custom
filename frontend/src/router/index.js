import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import ConversationsView from '../views/ConversationView.vue'
import UserLoginView from '../views/UserLoginView.vue'
import AccountView from '@/views/AccountView.vue'
import AdminView from '@/views/AdminView.vue'
import Inbox from '@/components/admin/inbox/Inbox.vue'
import Team from '@/components/admin/team/TeamSection.vue'
import Teams from '@/components/admin/team/teams/TeamsCard.vue'
import Users from '@/components/admin/team/users/UsersCard.vue'
import Automation from '@/components/admin/automation/Automation.vue'

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
        component: () => import('@/components/admin/inbox/NewInbox.vue')
      },
      {
        path: ':id/edit',
        props: true,
        component: () => import('@/components/admin/inbox/EditInbox.vue')
      }
    ]
  },
  {
    path: '/admin/teams',
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
        component: () => import('@/components/admin/team/users/AddUserForm.vue')
      },
      {
        path: 'teams/new',
        component: () => import('@/components/admin/team/teams/AddTeamForm.vue')
      },
      {
        path: 'users/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/users/EditUserForm.vue')
      },
      {
        path: 'teams',
        component: Teams
      },
      {
        path: 'teams/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/teams/EditTeamForm.vue')
      },
      {
        path: 'roles',
        component: () => import('@/components/admin/team/roles/RolesCard.vue')
      },
      {
        path: 'roles/new',
        component: () => import('@/components/admin/team/roles/NewRole.vue')
      },
      {
        path: 'roles/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/roles/EditRole.vue')
      },
    ]
  },
  {
    path: '/admin/automations',
    name: 'automation',
    component: AdminView,
    children: [
      {
        path: '',
        component: Automation
      },
      {
        path: ':id/edit',
        props: true,
        component: () => import('@/components/admin/automation/CreateOrEditRule.vue')
      },
      {
        path: 'new',
        props: true,
        component: () => import('@/components/admin/automation/CreateOrEditRule.vue')
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

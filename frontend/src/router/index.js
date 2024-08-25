import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import ConversationsView from '../views/ConversationView.vue'
import UserLoginView from '../views/UserLoginView.vue'
import AccountView from '@/views/AccountView.vue'
import AdminView from '@/views/AdminView.vue'
import Inbox from '@/components/admin/inbox/InboxPage.vue'
import Team from '@/components/admin/team/TeamPage.vue'
import Teams from '@/components/admin/team/teams/TeamsPage.vue'
import Users from '@/components/admin/team/users/UsersCard.vue'
import Automation from '@/components/admin/automation/AutomationPage.vue'
import Uploads from '@/components/admin/uploads/UploadsPage.vue'
import General from '@/components/admin/general/GeneralPage.vue'
import Templates from '@/components/admin/templates/TemplatesPage.vue'
import OIDC from '@/components/admin/oidc/OIDCPage.vue'

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
    path: '/admin',
    name: 'admin',
    component: AdminView,
    children: [
      {
        path: 'inboxes',
        component: Inbox
      },
      {
        path: 'inboxes/new',
        component: () => import('@/components/admin/inbox/NewInbox.vue')
      },
      {
        path: 'inboxes/:id/edit',
        props: true,
        component: () => import('@/components/admin/inbox/EditInbox.vue')
      },
      {
        path: 'teams',
        component: Team
      },
      {
        path: 'teams/users',
        component: Users
      },
      {
        path: 'teams/users/new',
        component: () => import('@/components/admin/team/users/AddUserForm.vue')
      },
      {
        path: 'teams/users/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/users/EditUserForm.vue')
      },
      {
        path: 'teams/teams',
        component: Teams
      },
      {
        path: 'teams/teams/new',
        component: () => import('@/components/admin/team/teams/AddTeamForm.vue')
      },
      {
        path: 'teams/teams/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/teams/EditTeamForm.vue')
      },
      {
        path: 'teams/roles',
        component: () => import('@/components/admin/team/roles/RolesPage.vue')
      },
      {
        path: 'teams/roles/new',
        component: () => import('@/components/admin/team/roles/NewRole.vue')
      },
      {
        path: 'teams/roles/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/roles/EditRole.vue')
      },
      {
        path: 'automations',
        component: Automation
      },
      {
        path: 'automations/new',
        props: true,
        component: () => import('@/components/admin/automation/CreateOrEditRule.vue')
      },
      {
        path: 'automations/:id/edit',
        props: true,
        component: () => import('@/components/admin/automation/CreateOrEditRule.vue')
      },
      {
        path: 'uploads',
        component: Uploads
      },
      {
        path: 'general',
        component: General
      },
      {
        path: 'templates',
        component: Templates
      },
      {
        path: 'templates/:id/edit',
        props: true,
        component: () => import('@/components/admin/templates/AddEditTemplate.vue')
      },
      {
        path: 'templates/new',
        component: () => import('@/components/admin/templates/AddEditTemplate.vue')
      },
      {
        path: 'oidc',
        component: OIDC
      },
      {
        path: 'oidc/:id/edit',
        props: true,
        component: () => import('@/components/admin/oidc/AddEditOIDC.vue')
      },
      {
        path: 'oidc/new',
        component: () => import('@/components/admin/oidc/AddEditOIDC.vue')
      },
      {
        path: 'conversations',
        component: () => import('@/components/admin/conversation/ConversationPage.vue')
      },
      {
        path: 'conversations/tags',
        component: () => import('@/components/admin/conversation/tags/TagsPage.vue')
      },
      {
        path: 'conversations/priority',
        component: () => import('@/components/admin/conversation/priority/PriorityPage.vue')
      },
      {
        path: 'conversations/status',
        component: () => import('@/components/admin/conversation/status/StatusPage.vue')
      }
    ]
  },
  // Fallback to dashboard.
  {
    path: '/:pathMatch(.*)*',
    redirect: (to) => {
      alert(`Redirecting to dashboard from: ${to.fullPath}`)
      return '/dashboard'
    }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: routes
})

export default router

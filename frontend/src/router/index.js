import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import ConversationsView from '../views/ConversationView.vue'
import UserLoginView from '../views/UserLoginView.vue'
import AccountView from '@/views/AccountView.vue'
import AdminView from '@/views/AdminView.vue'

const routes = [
  {
    path: '/',
    name: 'login',
    component: UserLoginView,
    meta: { title: 'Login' }
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: DashboardView,
    meta: { title: 'Dashboard' }
  },
  {
    path: '/conversations/:uuid?',
    name: 'conversations',
    component: ConversationsView,
    props: true,
    meta: { title: 'Conversations' }
  },
  {
    path: '/account/:page?',
    name: 'account',
    component: AccountView,
    props: true,
    meta: { title: 'Account' },
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
    meta: { title: 'Admin' },
    children: [
      {
        path: 'inboxes',
        component: () => import('@/components/admin/inbox/Inbox.vue'),
        meta: { title: 'Admin - Inboxes' }
      },
      {
        path: 'inboxes/new',
        component: () => import('@/components/admin/inbox/NewInbox.vue'),
        meta: { title: 'Admin - New Inbox' }
      },
      {
        path: 'inboxes/:id/edit',
        props: true,
        component: () => import('@/components/admin/inbox/EditInbox.vue'),
        meta: { title: 'Admin - Edit Inbox' }
      },
      {
        path: 'notification',
        component: () => import('@/components/admin/notification/NotificationSetting.vue'),
        meta: { title: 'Admin - Notification Settings' }
      },
      {
        path: 'teams',
        component: () => import('@/components/admin/team/Team.vue'),
        meta: { title: 'Admin - Teams' }
      },
      {
        path: 'teams/users',
        component: () => import('@/components/admin/team/users/UsersCard.vue'),
        meta: { title: 'Admin - Users' }
      },
      {
        path: 'teams/users/new',
        component: () => import('@/components/admin/team/users/AddUserForm.vue'),
        meta: { title: 'Admin - Add User' }
      },
      {
        path: 'teams/users/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/users/EditUserForm.vue'),
        meta: { title: 'Admin - Edit User' }
      },
      {
        path: 'teams/teams',
        component: () => import('@/components/admin/team/teams/Teams.vue'),
        meta: { title: 'Admin - Teams Management' }
      },
      {
        path: 'teams/teams/new',
        component: () => import('@/components/admin/team/teams/AddTeamForm.vue'),
        meta: { title: 'Admin - Add Team' }
      },
      {
        path: 'teams/teams/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/teams/EditTeamForm.vue'),
        meta: { title: 'Admin - Edit Team' }
      },
      {
        path: 'teams/roles',
        component: () => import('@/components/admin/team/roles/Roles.vue'),
        meta: { title: 'Admin - Roles' }
      },
      {
        path: 'teams/roles/new',
        component: () => import('@/components/admin/team/roles/NewRole.vue'),
        meta: { title: 'Admin - Add Role' }
      },
      {
        path: 'teams/roles/:id/edit',
        props: true,
        component: () => import('@/components/admin/team/roles/EditRole.vue'),
        meta: { title: 'Admin - Edit Role' }
      },
      {
        path: 'automations',
        component: () => import('@/components/admin/automation/Automation.vue'),
        meta: { title: 'Admin - Automations' }
      },
      {
        path: 'automations/new',
        props: true,
        component: () => import('@/components/admin/automation/CreateOrEditRule.vue'),
        meta: { title: 'Admin - Create Automation' }
      },
      {
        path: 'automations/:id/edit',
        props: true,
        component: () => import('@/components/admin/automation/CreateOrEditRule.vue'),
        meta: { title: 'Admin - Edit Automation' }
      },
      {
        path: 'general',
        component: () => import('@/components/admin/general/General.vue'),
        meta: { title: 'Admin - General Settings' }
      },
      {
        path: 'templates',
        component: () => import('@/components/admin/templates/Templates.vue'),
        meta: { title: 'Admin - Templates' }
      },
      {
        path: 'templates/:id/edit',
        props: true,
        component: () => import('@/components/admin/templates/AddEditTemplate.vue'),
        meta: { title: 'Admin - Edit Template' }
      },
      {
        path: 'templates/new',
        component: () => import('@/components/admin/templates/AddEditTemplate.vue'),
        meta: { title: 'Admin - Add Template' }
      },
      {
        path: 'oidc',
        component: () => import('@/components/admin/oidc/OIDC.vue'),
        meta: { title: 'Admin - OIDC' }
      },
      {
        path: 'oidc/:id/edit',
        props: true,
        component: () => import('@/components/admin/oidc/AddEditOIDC.vue'),
        meta: { title: 'Admin - Edit OIDC' }
      },
      {
        path: 'oidc/new',
        component: () => import('@/components/admin/oidc/AddEditOIDC.vue'),
        meta: { title: 'Admin - Add OIDC' }
      },
      {
        path: 'conversations',
        component: () => import('@/components/admin/conversation/Conversation.vue'),
        meta: { title: 'Admin - Conversations' }
      },
      {
        path: 'conversations/tags',
        component: () => import('@/components/admin/conversation/tags/Tags.vue'),
        meta: { title: 'Admin - Conversation Tags' }
      },
      {
        path: 'conversations/statuses',
        component: () => import('@/components/admin/conversation/status/Status.vue'),
        meta: { title: 'Admin - Conversation Statuses' }
      },
      {
        path: 'conversations/canned-responses',
        component: () => import('@/components/admin/conversation/canned_responses/CannedResponses.vue'),
        meta: { title: 'Admin - Canned Responses' }
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

// Global navigation guard to set document title
router.beforeEach((to, from, next) => {
  document.title = to.meta.title || 'Default Title'
  next()
})

export default router

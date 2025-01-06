import { createRouter, createWebHistory } from 'vue-router'
import App from '../App.vue'
import OuterApp from '../OuterApp.vue'
import DashboardView from '../views/DashboardView.vue'
import ConversationsView from '../views/ConversationView.vue'
import UserLoginView from '../views/UserLoginView.vue'
import AccountView from '@/views/AccountView.vue'
import AdminView from '@/views/AdminView.vue'
import ResetPasswordView from '@/views/ResetPasswordView.vue'
import SetPasswordView from '@/views/SetPasswordView.vue'

const routes = [
  {
    path: '/',
    component: OuterApp,
    children: [
      {
        path: '',
        name: 'login',
        component: UserLoginView,
        meta: { title: 'Login' }
      },
      {
        path: 'reset-password',
        name: 'reset-password',
        component: ResetPasswordView,
        meta: { title: 'Reset Password' }
      },
      {
        path: 'set-password',
        name: 'set-password',
        component: SetPasswordView,
        meta: { title: 'Set Password' }
      }
    ]
  },
  {
    path: '/',
    component: App,
    children: [
      {
        path: '/reports',
        name: 'reports',
        redirect: '/reports/overview',
        children: [
          {
            path: 'overview',
            name: 'overview',
            component: DashboardView,
            meta: { title: 'Overview' }
          },
        ]
      },
      {
        path: '/inboxes',
        name: 'inboxes',
        redirect: '/inboxes/assigned',
        meta: { title: 'Inboxes' },
        children: [
          {
            path: ':type(assigned|unassigned|all)',
            name: 'inbox',
            component: ConversationsView,
            props: route => ({ type: route.params.type, uuid: route.params.uuid }),
            meta: route => ({ title: `${route.params.type.charAt(0).toUpperCase()}${route.params.type.slice(1)} inbox` }),
            children: [
              {
                path: 'conversation/:uuid',
                name: 'inbox-conversation',
                component: ConversationsView,
                props: true,
                meta: { title: 'Conversation' }
              }
            ]
          }
          
        ]
      },
      {
        path: '/teams',
        name: 'teams',
        redirect: '/teams/:teamID',
        meta: { title: 'Teams' },
        children: [
          {
            path: ':teamID',
            name: 'team-inbox',
            props: true,
            component: ConversationsView,
            meta: route => ({ title: `Team ${route.params.teamID} inbox` }),
            children: [
              {
                path: 'conversation/:uuid',
                name: 'team-inbox-conversation',
                component: ConversationsView,
                props: true,
                meta: { title: 'Conversation' }
              }
            ]
          }
        ]
      },
      {
        path: '/views',
        name: 'views',
        redirect: '/views/:viewID',
        meta: { title: 'Views' },
        children: [
          {
            path: ':viewID',
            name: 'view-inbox',
            props: true,
            component: ConversationsView,
            meta: route => ({ title: `View ${route.params.viewID} inbox` }),
            children: [
              {
                path: 'conversation/:uuid',
                name: 'view-inbox-conversation',
                component: ConversationsView,
                props: true,
                meta: { title: 'Conversation' }
              }
            ]
          }
        ]
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
        redirect: '/admin/general',
        component: AdminView,
        meta: { title: 'Admin' },
        children: [
          {
            path: 'business-hours',
            component: () => import('@/components/admin/business_hours/BusinessHours.vue'),
            meta: { title: 'Admin - Business Hours' },
            children: [
              {
                path: 'new',
                component: () => import('@/components/admin/business_hours/CreateOrEditBusinessHours.vue'),
                meta: { title: 'Admin - Add Business Hours' }
              },
              {
                path: ':id/edit',
                name: 'edit-business-hours',
                props: true,
                component: () => import('@/components/admin/business_hours/CreateOrEditBusinessHours.vue'),
                meta: { title: 'Admin - Edit Business Hours' }
              },
            ]
          },
          {
            path: 'sla',
            component: () => import('@/components/admin/sla/SLA.vue'),
            meta: { title: 'Admin - SLA' },
            children: [
              {
                path: 'new',
                component: () => import('@/components/admin/sla/CreateEditSLA.vue'),
                meta: { title: 'Admin - Add SLA' }
              },
              {
                path: ':id/edit',
                props: true,
                component: () => import('@/components/admin/sla/CreateEditSLA.vue'),
                meta: { title: 'Admin - Edit SLA' }
              },
            ]
          },
          {
            path: 'inboxes',
            component: () => import('@/components/admin/inbox/Inbox.vue'),
            meta: { title: 'Admin - Inboxes' },
            children: [
              {
                path: 'new',
                component: () => import('@/components/admin/inbox/NewInbox.vue'),
                meta: { title: 'Admin - New Inbox' }
              },
              {
                path: ':id/edit',
                props: true,
                component: () => import('@/components/admin/inbox/EditInbox.vue'),
                meta: { title: 'Admin - Edit Inbox' }
              },
            ],
          },
          {
            path: 'notification',
            component: () => import('@/components/admin/notification/NotificationSetting.vue'),
            meta: { title: 'Admin - Notification Settings' }
          },
          {
            path: 'teams',
            meta: { title: 'Admin - Teams' },
            children: [
              {
                path: 'users',
                component: () => import('@/components/admin/team/users/UsersCard.vue'),
                meta: { title: 'Admin - Users' },
                children: [
                  {
                    path: 'new',
                    component: () => import('@/components/admin/team/users/AddUserForm.vue'),
                    meta: { title: 'Admin - Create User' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    component: () => import('@/components/admin/team/users/EditUserForm.vue'),
                    meta: { title: 'Admin - Edit User' }
                  },
                ]
              },
              {
                path: 'teams',
                component: () => import('@/components/admin/team/teams/Teams.vue'),
                meta: { title: 'Admin - Teams Management' },
                children: [

                  {
                    path: 'new',
                    component: () => import('@/components/admin/team/teams/CreateTeamForm.vue'),
                    meta: { title: 'Admin - Create Team' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    component: () => import('@/components/admin/team/teams/EditTeamForm.vue'),
                    meta: { title: 'Admin - Edit Team' }
                  },
                ]
              },
              {
                path: 'roles',
                component: () => import('@/components/admin/team/roles/Roles.vue'),
                meta: { title: 'Admin - Roles' },
                children: [
                  {
                    path: 'new',
                    component: () => import('@/components/admin/team/roles/NewRole.vue'),
                    meta: { title: 'Admin - Create Role' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    component: () => import('@/components/admin/team/roles/EditRole.vue'),
                    meta: { title: 'Admin - Edit Role' }
                  }
                ]
              },
            ]
          },
          {
            path: 'automations',
            component: () => import('@/components/admin/automation/Automation.vue'),
            meta: { title: 'Admin - Automations' },
            children: [
              {
                path: 'new',
                props: true,
                component: () => import('@/components/admin/automation/CreateOrEditRule.vue'),
                meta: { title: 'Admin - Create Automation' }
              },
              {
                path: ':id/edit',
                props: true,
                component: () => import('@/components/admin/automation/CreateOrEditRule.vue'),
                meta: { title: 'Admin - Edit Automation' }
              }
            ]
          },
          {
            path: 'general',
            component: () => import('@/components/admin/general/General.vue'),
            meta: { title: 'Admin - General Settings' }
          },
          {
            path: 'templates',
            component: () => import('@/components/admin/templates/Templates.vue'),
            meta: { title: 'Admin - Templates' },
            children: [
              {
                path: ':id/edit',
                props: true,
                component: () => import('@/components/admin/templates/AddEditTemplate.vue'),
                meta: { title: 'Admin - Edit Template' }
              },
              {
                path: 'new',
                component: () => import('@/components/admin/templates/AddEditTemplate.vue'),
                meta: { title: 'Admin - Add Template' }
              }
            ]
          },
          {
            path: 'oidc',
            component: () => import('@/components/admin/oidc/OIDC.vue'),
            meta: { title: 'Admin - OIDC' },
            children: [
              {
                path: ':id/edit',
                props: true,
                component: () => import('@/components/admin/oidc/AddEditOIDC.vue'),
                meta: { title: 'Admin - Edit OIDC' }
              },
              {
                path: 'new',
                component: () => import('@/components/admin/oidc/AddEditOIDC.vue'),
                meta: { title: 'Admin - Add OIDC' }
              }
            ]
          },
          {
            path: 'conversations',
            meta: { title: 'Admin - Conversations' },
            children: [
              {
                path: 'tags',
                component: () => import('@/components/admin/conversation/tags/Tags.vue'),
                meta: { title: 'Admin - Conversation Tags' }
              },
              {
                path: 'statuses',
                component: () => import('@/components/admin/conversation/status/Status.vue'),
                meta: { title: 'Admin - Conversation Statuses' }
              },
              {
                path: 'canned-responses',
                component: () => import('@/components/admin/conversation/canned_responses/CannedResponses.vue'),
                meta: { title: 'Admin - Canned Responses' }
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: (to) => {
      // TODO: Remove this alert and redirect to 404 page
      alert(`Redirecting to overview from: ${to.fullPath}`)
      return '/reports/overview'
    }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: routes
})

router.beforeEach((to, from, next) => {
  document.title = to.meta.title || ''
  next()
})

export default router

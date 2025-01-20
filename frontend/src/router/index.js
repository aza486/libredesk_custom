import { createRouter, createWebHistory } from 'vue-router'
import App from '../App.vue'
import OuterApp from '../OuterApp.vue'
import DashboardView from '../views/DashboardView.vue'
import ConversationsView from '../views/ConversationView.vue'
import SearchView from '../views/SearchView.vue'
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
        meta: { title: 'Inbox', hidePageHeader: true },
        children: [
          {
            path: 'search',
            name: 'search',
            component: SearchView,
            meta: { title: 'Search', hidePageHeader: true },
          },
          {
            path: ':type(assigned|unassigned|all)',
            name: 'inbox',
            component: ConversationsView,
            props: route => ({ type: route.params.type, uuid: route.params.uuid }),
            meta: {
              title: 'Inbox',
              type: route => route.params.type === 'assigned' ? 'My inbox' : route.params.type
            },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'inbox-conversation',
                component: ConversationsView,
                props: true,
                meta: {
                  title: 'Inbox',
                  type: route => route.params.type === 'assigned' ? 'My inbox' : route.params.type,
                  hidePageHeader: true
                }
              }
            ]
          }
        ]
      },
      {
        path: '/teams',
        name: 'teams',
        redirect: '/teams/:teamID',
        meta: { title: 'Team inbox', hidePageHeader: true },
        children: [
          {
            path: ':teamID',
            name: 'team-inbox',
            props: true,
            component: ConversationsView,
            meta: { title: `Team inbox` },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'team-inbox-conversation',
                component: ConversationsView,
                props: true,
                meta: { title: 'Team inbox', hidePageHeader: true }
              }
            ]
          }
        ]
      },
      {
        path: '/views',
        name: 'views',
        redirect: '/views/:viewID',
        meta: { title: 'View', hidePageHeader: true },
        children: [
          {
            path: ':viewID',
            name: 'view-inbox',
            props: true,
            component: ConversationsView,
            meta: { title: `Views` },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'view-inbox-conversation',
                component: ConversationsView,
                props: true,
                meta: { title: 'Views', hidePageHeader: true }
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
            meta: { title: 'Business Hours' },
            children: [
              {
                path: 'new',
                component: () => import('@/components/admin/business_hours/CreateOrEditBusinessHours.vue'),
                meta: { title: 'New Business Hours' }
              },
              {
                path: ':id/edit',
                name: 'edit-business-hours',
                props: true,
                component: () => import('@/components/admin/business_hours/CreateOrEditBusinessHours.vue'),
                meta: { title: 'Edit Business Hours' }
              },
            ]
          },
          {
            path: 'sla',
            component: () => import('@/components/admin/sla/SLA.vue'),
            meta: { title: 'SLA' },
            children: [
              {
                path: 'new',
                component: () => import('@/components/admin/sla/CreateEditSLA.vue'),
                meta: { title: 'New SLA' }
              },
              {
                path: ':id/edit',
                props: true,
                component: () => import('@/components/admin/sla/CreateEditSLA.vue'),
                meta: { title: 'Edit SLA' }
              },
            ]
          },
          {
            path: 'inboxes',
            component: () => import('@/components/admin/inbox/Inbox.vue'),
            meta: { title: 'Inboxes' },
            children: [
              {
                path: 'new',
                component: () => import('@/components/admin/inbox/NewInbox.vue'),
                meta: { title: 'New Inbox' }
              },
              {
                path: ':id/edit',
                props: true,
                component: () => import('@/components/admin/inbox/EditInbox.vue'),
                meta: { title: 'Edit Inbox' }
              },
            ],
          },
          {
            path: 'notification',
            component: () => import('@/components/admin/notification/NotificationSetting.vue'),
            meta: { title: 'Notification Settings' }
          },
          {
            path: 'teams',
            meta: { title: 'Teams' },
            children: [
              {
                path: 'users',
                component: () => import('@/components/admin/team/users/UsersCard.vue'),
                meta: { title: 'Users' },
                children: [
                  {
                    path: 'new',
                    component: () => import('@/components/admin/team/users/AddUserForm.vue'),
                    meta: { title: 'Create User' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    component: () => import('@/components/admin/team/users/EditUserForm.vue'),
                    meta: { title: 'Edit User' }
                  },
                ]
              },
              {
                path: 'teams',
                component: () => import('@/components/admin/team/teams/Teams.vue'),
                meta: { title: 'Teams' },
                children: [

                  {
                    path: 'new',
                    component: () => import('@/components/admin/team/teams/CreateTeamForm.vue'),
                    meta: { title: 'Create Team' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    component: () => import('@/components/admin/team/teams/EditTeamForm.vue'),
                    meta: { title: 'Edit Team' }
                  },
                ]
              },
              {
                path: 'roles',
                component: () => import('@/components/admin/team/roles/Roles.vue'),
                meta: { title: 'Roles' },
                children: [
                  {
                    path: 'new',
                    component: () => import('@/components/admin/team/roles/NewRole.vue'),
                    meta: { title: 'Create Role' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    component: () => import('@/components/admin/team/roles/EditRole.vue'),
                    meta: { title: 'Edit Role' }
                  }
                ]
              },
            ]
          },
          {
            path: 'automations',
            component: () => import('@/components/admin/automation/Automation.vue'),
            meta: { title: 'Automations' },
            children: [
              {
                path: 'new',
                props: true,
                component: () => import('@/components/admin/automation/CreateOrEditRule.vue'),
                meta: { title: 'Create Automation' }
              },
              {
                path: ':id/edit',
                props: true,
                component: () => import('@/components/admin/automation/CreateOrEditRule.vue'),
                meta: { title: 'Edit Automation' }
              }
            ]
          },
          {
            path: 'general',
            component: () => import('@/components/admin/general/General.vue'),
            meta: { title: 'General' }
          },
          {
            path: 'templates',
            component: () => import('@/components/admin/templates/Templates.vue'),
            meta: { title: 'Templates' },
            children: [
              {
                path: ':id/edit',
                name: 'edit-template',
                props: true,
                component: () => import('@/components/admin/templates/CreateEditTemplate.vue'),
                meta: { title: 'Edit Template' }
              },
              {
                path: 'new',
                name: 'new-template',
                props: true,
                component: () => import('@/components/admin/templates/CreateEditTemplate.vue'),
                meta: { title: 'New Template' }
              }
            ]
          },
          {
            path: 'oidc',
            component: () => import('@/components/admin/oidc/OIDC.vue'),
            meta: { title: 'OIDC' },
            children: [
              {
                path: ':id/edit',
                props: true,
                component: () => import('@/components/admin/oidc/CreateEditOIDC.vue'),
                meta: { title: 'Edit OIDC' }
              },
              {
                path: 'new',
                component: () => import('@/components/admin/oidc/CreateEditOIDC.vue'),
                meta: { title: 'New OIDC' }
              }
            ]
          },
          {
            path: 'conversations',
            meta: { title: 'Conversations' },
            children: [
              {
                path: 'tags',
                component: () => import('@/components/admin/conversation/tags/Tags.vue'),
                meta: { title: 'Tags' }
              },
              {
                path: 'statuses',
                component: () => import('@/components/admin/conversation/status/Status.vue'),
                meta: { title: 'Statuses' }
              },
              {
                path: 'Macros',
                component: () => import('@/components/admin/conversation/macros/Macros.vue'),
                meta: { title: 'Macros' },
                children: [
                  {
                    path: 'new',
                    component: () => import('@/components/admin/conversation/macros/CreateMacro.vue'),
                    meta: { title: 'Create Macro' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    component: () => import('@/components/admin/conversation/macros/EditMacro.vue'),
                    meta: { title: 'Edit Macro' }
                  },
                ]
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
  document.title = to.meta.title + ' | LibreDesk'
  next()
})

export default router

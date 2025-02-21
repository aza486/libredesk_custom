import { createRouter, createWebHistory } from 'vue-router'
import App from '../App.vue'
import OuterApp from '../OuterApp.vue'
import DashboardView from '../views/reports/DashboardView.vue'
import InboxLayout from '../layouts/inbox/InboxLayout.vue'
import SearchView from '../views/search/SearchView.vue'
import UserLoginView from '../views/login/UserLoginView.vue'
import AccountLayout from '@/layouts/account/AccountLayout.vue'
import AdminLayout from '@/layouts/admin/AdminLayout.vue'
import ResetPasswordView from '@/views/outerapp/ResetPasswordView.vue'
import SetPasswordView from '@/views/outerapp/SetPasswordView.vue'
import InboxView from '@/views/inbox/InboxView.vue'
import ConversationDetailView from '@/views/conversation/ConversationDetailView.vue'

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
        path: '/inboxes/teams/:teamID',
        name: 'teams',
        props: true,
        component: InboxLayout,
        meta: { title: 'Team inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'team-inbox',
            component: InboxView,
            props: true,
            meta: { title: 'Team inbox' }
          },
          {
            path: 'conversation/:uuid',
            name: 'team-inbox-conversation',
            component: ConversationDetailView,
            props: true,
            meta: { title: 'Team inbox', hidePageHeader: true }
          }
        ]
      },
      {
        path: '/inboxes/views/:viewID',
        name: 'views',
        props: true,
        component: InboxLayout,
        meta: { title: 'View inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'view-inbox',
            component: InboxView,
            props: true,
            meta: { title: 'View inbox' }
          },
          {
            path: 'conversation/:uuid',
            name: 'view-inbox-conversation',
            component: ConversationDetailView,
            props: true,
            meta: { title: 'View inbox', hidePageHeader: true }
          }
        ]
      },
      {
        path: 'inboxes/search',
        name: 'search',
        component: SearchView,
        meta: { title: 'Search', hidePageHeader: true },
      },
      {
        path: '/inboxes/:type(assigned|unassigned|all)?',
        name: 'inboxes',
        redirect: '/inboxes/assigned',
        component: InboxLayout,
        props: true,
        meta: { title: 'Inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'inbox',
            component: InboxView,
            props: true,
            meta: {
              title: 'Inbox',
              type: route => route.params.type === 'assigned' ? 'My inbox' : route.params.type
            },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'inbox-conversation',
                component: ConversationDetailView,
                props: true,
                meta: {
                  title: 'Inbox',
                  type: route => route.params.type === 'assigned' ? 'My inbox' : route.params.type,
                  hidePageHeader: true
                }
              },
            ]
          },
        ]
      },
      {
        path: '/account/:page?',
        name: 'account',
        redirect: '/account/profile',
        component: AccountLayout,
        props: true,
        meta: { title: 'Account' },
        children: [
          {
            path: 'profile',
            name: 'profile',
            component: () => import('@/views/account/profile/ProfileEditView.vue'),
            meta: { title: 'Edit Profile' }
          }
        ]
      },
      {
        path: '/admin',
        name: 'admin',
        redirect: '/admin/general',
        component: AdminLayout,
        meta: { title: 'Admin' },
        children: [
          {
            path: 'general',
            component: () => import('@/views/admin/general/General.vue'),
            meta: { title: 'General' }
          },
          {
            path: 'business-hours',
            component: () => import('@/views/admin/business-hours/BusinessHours.vue'),
            meta: { title: 'Business Hours' },
            children: [
              {
                path: '',
                name: 'business-hours-list',
                component: () => import('@/views/admin/business-hours/BusinessHoursList.vue'),
              },
              {
                path: 'new',
                name: 'new-business-hours',
                component: () => import('@/views/admin/business-hours/CreateOrEditBusinessHours.vue'),
                meta: { title: 'New Business Hours' }
              },
              {
                path: ':id/edit',
                name: 'edit-business-hours',
                props: true,
                component: () => import('@/views/admin/business-hours/CreateOrEditBusinessHours.vue'),
                meta: { title: 'Edit Business Hours' }
              },
            ]
          },
          {
            path: 'sla',
            component: () => import('@/views/admin/sla/SLA.vue'),
            meta: { title: 'SLA' },
            children: [
              {
                path: '',
                name: 'sla-list',
                component: () => import('@/views/admin/sla/SLAList.vue'),
              },
              {
                path: 'new',
                name: 'new-sla',
                component: () => import('@/views/admin/sla/CreateEditSLA.vue'),
                meta: { title: 'New SLA' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-sla',
                component: () => import('@/views/admin/sla/CreateEditSLA.vue'),
                meta: { title: 'Edit SLA' }
              },
            ]
          },
          {
            path: 'inboxes',
            component: () => import('@/views/admin/inbox/InboxView.vue'),
            meta: { title: 'Inboxes' },
            children: [
              {
                path: '',
                name: 'inbox-list',
                component: () => import('@/views/admin/inbox/InboxList.vue'),
              },
              {
                path: 'new',
                name: 'new-inbox',
                component: () => import('@/views/admin/inbox/NewInbox.vue'),
                meta: { title: 'New Inbox' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-inbox',
                component: () => import('@/views/admin/inbox/EditInbox.vue'),
                meta: { title: 'Edit Inbox' }
              },
            ],
          },
          {
            path: 'notification',
            component: () => import('@/features/admin/notification/NotificationSetting.vue'),
            meta: { title: 'Notification Settings' }
          },
          {
            path: 'teams',
            meta: { title: 'Teams' },
            children: [
              {
                path: 'users',
                component: () => import('@/views/admin/users/Users.vue'),
                meta: { title: 'Users' },
                children: [
                  {
                    path: '',
                    name: 'user-list',
                    component: () => import('@/views/admin/users/UserList.vue'),
                  },
                  {
                    path: 'new',
                    name: 'new-user',
                    component: () => import('@/views/admin/users/CreateUser.vue'),
                    meta: { title: 'Create User' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    component: () => import('@/views/admin/users/EditUser.vue'),
                    meta: { title: 'Edit User' }
                  },
                ]
              },
              {
                path: 'teams',
                component: () => import('@/views/admin/teams/Teams.vue'),
                meta: { title: 'Teams' },
                children: [
                  {
                    path: '',
                    name: 'team-list',
                    component: () => import('@/views/admin/teams/TeamList.vue'),
                  },
                  {
                    path: 'new',
                    name: 'new-team',
                    component: () => import('@/views/admin/teams/CreateTeamForm.vue'),
                    meta: { title: 'Create Team' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-team',
                    component: () => import('@/views/admin/teams/EditTeamForm.vue'),
                    meta: { title: 'Edit Team' }
                  },
                ]
              },
              {
                path: 'roles',
                component: () => import('@/views/admin/roles/Roles.vue'),
                meta: { title: 'Roles' },
                children: [
                  {
                    path: '',
                    name: 'role-list',
                    component: () => import('@/views/admin/roles/RoleList.vue'),
                  },
                  {
                    path: 'new',
                    name: 'new-role',
                    component: () => import('@/views/admin/roles/NewRole.vue'),
                    meta: { title: 'Create Role' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-role',
                    component: () => import('@/views/admin/roles/EditRole.vue'),
                    meta: { title: 'Edit Role' }
                  }
                ]
              },
            ]
          },
          {
            path: 'automations',
            component: () => import('@/views/admin/automations/Automation.vue'),
            name: 'automations',
            meta: { title: 'Automations' },
            children: [
              {
                path: 'new',
                props: true,
                name: 'new-automation',
                component: () => import('@/views/admin/automations/CreateOrEditRule.vue'),
                meta: { title: 'Create Automation' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-automation',
                component: () => import('@/views/admin/automations/CreateOrEditRule.vue'),
                meta: { title: 'Edit Automation' }
              }
            ]
          },
          {
            path: 'templates',
            component: () => import('@/views/admin/templates/Templates.vue'),
            name: 'templates',
            meta: { title: 'Templates' },
            children: [
              {
                path: ':id/edit',
                name: 'edit-template',
                props: true,
                component: () => import('@/views/admin/templates/CreateEditTemplate.vue'),
                meta: { title: 'Edit Template' }
              },
              {
                path: 'new',
                name: 'new-template',
                props: true,
                component: () => import('@/views/admin/templates/CreateEditTemplate.vue'),
                meta: { title: 'New Template' }
              }
            ]
          },
          {
            path: 'sso',
            component: () => import('@/views/admin/oidc/OIDC.vue'),
            name: 'sso',
            meta: { title: 'SSO' },
            children: [
              {
                path: '',
                name: 'sso-list',
                component: () => import('@/views/admin/oidc/OIDCList.vue'),
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-sso',
                component: () => import('@/views/admin/oidc/CreateEditOIDC.vue'),
                meta: { title: 'Edit SSO' }
              },
              {
                path: 'new',
                name: 'new-sso',
                component: () => import('@/views/admin/oidc/CreateEditOIDC.vue'),
                meta: { title: 'New SSO' }
              }
            ]
          },
          {
            path: 'conversations',
            meta: { title: 'Conversations' },
            children: [
              {
                path: 'tags',
                component: () => import('@/views/admin/tags/TagsView.vue'),
                meta: { title: 'Tags' }
              },
              {
                path: 'statuses',
                component: () => import('@/views/admin/status/StatusView.vue'),
                meta: { title: 'Statuses' }
              },
              {
                path: 'Macros',
                component: () => import('@/views/admin/macros/Macros.vue'),
                meta: { title: 'Macros' },
                children: [
                  {
                    path: '',
                    name: 'macro-list',
                    component: () => import('@/views/admin/macros/MacroList.vue'),
                  },
                  {
                    path: 'new',
                    name: 'new-macro',
                    component: () => import('@/views/admin/macros/CreateMacro.vue'),
                    meta: { title: 'Create Macro' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-macro',
                    component: () => import('@/views/admin/macros/EditMacro.vue'),
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
  document.title = to.meta.title + ' - Libredesk'
  next()
})

export default router

export const reportsNavItems = [
    {
        title: 'Overview',
        href: '/reports/overview',
        permission: 'reports:manage'
    }
]

export const adminNavItems = [
    {
        title: 'Workspace',
        children: [
            {
                title: 'General',
                href: '/admin/general',
                permission: 'general_settings:manage'
            },
            {
                title: 'Business Hours',
                href: '/admin/business-hours',
                permission: 'business_hours:manage'
            },
            {
                title: 'SLA',
                href: '/admin/sla',
                permission: 'sla:manage'
            }
        ]
    },
    {
        title: 'Conversations',
        children: [
            {
                title: 'Tags',
                href: '/admin/conversations/tags',
                permission: 'tags:manage'
            },
            {
                title: 'Macros',
                href: '/admin/conversations/macros',
                permission: 'macros:manage'
            },
            {
                title: 'Statuses',
                href: '/admin/conversations/statuses',
                permission: 'status:manage'
            }
        ]
    },
    {
        title: 'Inboxes',
        children: [
            {
                title: 'Inboxes',
                href: '/admin/inboxes',
                permission: 'inboxes:manage'
            }
        ]
    },
    {
        title: 'Teammates',
        children: [
            {
                title: 'Users',
                href: '/admin/teams/users',
                permission: 'users:manage'
            },
            {
                title: 'Teams',
                href: '/admin/teams/teams',
                permission: 'teams:manage'
            },
            {
                title: 'Roles',
                href: '/admin/teams/roles',
                permission: 'roles:manage'
            }
        ]
    },
    {
        title: 'Automations',
        children: [
            {
                title: 'Automations',
                href: '/admin/automations',
                permission: 'automations:manage'
            }
        ]
    },
    {
        title: 'Notifications',
        children: [
            {
                title: 'Email',
                href: '/admin/notification',
                permission: 'notification_settings:manage'
            }
        ]
    },
    {
        title: 'Templates',
        children: [
            {
                title: 'Templates',
                href: '/admin/templates',
                permission: 'templates:manage'
            }
        ]
    },
    {
        title: 'Security',
        children: [
            {
                title: 'SSO',
                href: '/admin/sso',
                permission: 'oidc:manage'
            }
        ]
    }
]

export const accountNavItems = [
    {
        title: 'Profile',
        href: '/account/profile',
        description: 'Update your profile'
    }
]

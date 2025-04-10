export const reportsNavItems = [
    {
        titleKey: 'navigation.overview',
        href: '/reports/overview',
        permission: 'reports:manage'
    }
]

export const adminNavItems = [
    {
        titleKey: 'navigation.workspace',
        children: [
            {
                titleKey: 'navigation.generalSettings',
                href: '/admin/general',
                permission: 'general_settings:manage'
            },
            {
                titleKey: 'navigation.businessHours',
                href: '/admin/business-hours',
                permission: 'business_hours:manage'
            },
            {
                titleKey: 'navigation.slaPolicies',
                href: '/admin/sla',
                permission: 'sla:manage'
            }
        ]
    },
    {
        titleKey: 'navigation.conversations',
        children: [
            {
                titleKey: 'navigation.tags',
                href: '/admin/conversations/tags',
                permission: 'tags:manage'
            },
            {
                titleKey: 'navigation.macros',
                href: '/admin/conversations/macros',
                permission: 'macros:manage'
            },
            {
                titleKey: 'navigation.statuses',
                href: '/admin/conversations/statuses',
                permission: 'status:manage'
            }
        ]
    },
    {
        titleKey: 'navigation.inboxes',
        children: [
            {
                titleKey: 'navigation.inboxes',
                href: '/admin/inboxes',
                permission: 'inboxes:manage'
            }
        ]
    },
    {
        titleKey: 'navigation.teammates',
        children: [
            {
                titleKey: 'navigation.agents',
                href: '/admin/teams/agents',
                permission: 'users:manage'
            },
            {
                titleKey: 'navigation.teams',
                href: '/admin/teams/teams',
                permission: 'teams:manage'
            },
            {
                titleKey: 'navigation.roles',
                href: '/admin/teams/roles',
                permission: 'roles:manage'
            }
        ]
    },
    {
        titleKey: 'navigation.automations',
        children: [
            {
                titleKey: 'navigation.automations',
                href: '/admin/automations',
                permission: 'automations:manage'
            }
        ]
    },
    {
        titleKey: 'navigation.notifications',
        children: [
            {
                titleKey: 'navigation.email',
                href: '/admin/notification',
                permission: 'notification_settings:manage'
            }
        ]
    },
    {
        titleKey: 'navigation.templates',
        children: [
            {
                titleKey: 'navigation.templates',
                href: '/admin/templates',
                permission: 'templates:manage'
            }
        ]
    },
    {
        titleKey: 'navigation.security',
        children: [
            {
                titleKey: 'navigation.sso',
                href: '/admin/sso',
                permission: 'oidc:manage'
            }
        ]
    }
]

export const accountNavItems = [
    {
        titleKey: 'navigation.profile',
        href: '/account/profile',
        description: 'Update your profile'
    }
]

export const contactNavItems = [
    {
        titleKey: 'navigation.allContacts',
        href: '/contacts',
    }
]
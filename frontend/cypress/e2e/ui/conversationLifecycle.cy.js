describe('Conversation lifecycle', () => {
  const stamp = Date.now()
  const inboxName = `Lifecycle Inbox ${stamp}`
  const teamName = `Lifecycle Team ${stamp}`
  const agentFirstName = 'Lifecycle'
  const agentLastName = `Agent${stamp}`
  const agentName = `${agentFirstName} ${agentLastName}`
  const tagName = `lifecycle-${stamp}`
  const contactLastName = `Customer${stamp}`
  const contactName = `Lifecycle ${contactLastName}`
  const subject = `Lifecycle subject ${stamp}`
  const noteBody = `Internal note ${stamp}`
  const replyBody = `Public reply ${stamp}`

  const smtpHost = Cypress.env('SMTP_HOST') || '127.0.0.1'
  const smtpPort = Number(Cypress.env('SMTP_PORT') || 1025)

  let conversationUuid
  let inboxId
  let teamId
  let agentId
  let tagId

  const openConversation = () => {
    cy.intercept('GET', '**/messages?page=*').as('loadMessages')
    cy.visit(`/inboxes/all/conversation/${conversationUuid}`)
    cy.wait('@loadMessages')
  }

  // The open set of sidebar sections is remembered per browser, so blind toggling closes it later.
  const openActionsSection = () => {
    cy.contains('button', 'Actions').then(($trigger) => {
      if ($trigger.attr('aria-expanded') !== 'true') cy.wrap($trigger).click()
    })
  }

  // The list column also renders a status dropdown; this one is the header badge.
  const statusBadge = () => cy.get('[data-testid="conversation-status"]')

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/inboxes', {
      name: inboxName,
      channel: 'email',
      enabled: true,
      from: `Lifecycle <lifecycle+${stamp}@cypress.test>`,
      config: {
        auth_type: 'password',
        imap: [],
        smtp: [
          {
            host: smtpHost,
            port: smtpPort,
            auth_protocol: 'none',
            max_conns: 2,
            idle_timeout: '5s',
            pool_wait_timeout: '5s',
            max_msg_retries: 1,
            tls_type: 'none'
          }
        ]
      }
    }).then(({ body }) => {
      const inboxID = body.data.id
      inboxId = inboxID
      cy.api('POST', '/api/v1/teams', {
        name: teamName,
        emoji: '🐝',
        conversation_assignment_type: 'Round robin',
        timezone: 'Asia/Kolkata'
      }).then((res) => {
        teamId = res.body.data.id
      })
      cy.api('POST', '/api/v1/agents', {
        first_name: agentFirstName,
        last_name: agentLastName,
        email: `lifecycle.agent.${stamp}@example.com`,
        roles: ['Agent'],
        enabled: true,
        send_welcome_email: false
      }).then((res) => {
        agentId = res.body.data.id
      })
      cy.api('POST', '/api/v1/tags', { name: tagName }).then((res) => {
        tagId = res.body.data.id
      })
      cy.api('POST', '/api/v1/conversations', {
        inbox_id: inboxID,
        contact_email: `lifecycle.customer.${stamp}@example.com`,
        first_name: 'Lifecycle',
        last_name: contactLastName,
        subject,
        content: '<p>Customer opened this conversation.</p>',
        initiator: 'contact'
      }).then((res) => {
        conversationUuid = res.body.data.uuid
        expect(conversationUuid, 'conversation uuid').to.be.a('string').and.not.be.empty
      })
    })
  })

  // Conversations have no delete endpoint, so the conversation itself stays.
  // Teardown never asserts: a failed cleanup must not mask the real failure.
  after(() => {
    cy.login()
    const drop = (path) => cy.api('DELETE', path, null, { failOnStatusCode: false })
    if (tagId) drop(`/api/v1/tags/${tagId}`)
    if (agentId) drop(`/api/v1/agents/${agentId}`)
    if (teamId) drop(`/api/v1/teams/${teamId}`)
    if (inboxId) drop(`/api/v1/inboxes/${inboxId}`)
  })

  beforeEach(() => {
    cy.viewport(1440, 900) // desktop layout: list, conversation and sidebar all on screen
    cy.login()
  })

  it('opens the conversation from the inbox list', () => {
    cy.visit('/inboxes/all')
    cy.contains(contactName).click()

    cy.location('pathname').should('include', conversationUuid)
    cy.contains('Customer opened this conversation.').should('be.visible')
    statusBadge().should('contain.text', 'Open')
  })

  it('assigns the conversation to an agent', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/assignees/users`).as('assignAgent')

    openConversation()
    openActionsSection()
    cy.contains('p', /^Zuweisungen$/)
      .parent()
      .within(() => {
        cy.contains('button[role="combobox"]', 'Mitarbeiter hinzufügen').click()
      })
    cy.get('[role="option"]').contains(agentName).click()

    cy.wait('@assignAgent').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body.assignee_ids).to.deep.eq([agentId])
    })
    cy.get(`button[aria-label="${agentName} entfernen"]`).should('be.visible')
    cy.api('GET', `/api/v1/conversations/${conversationUuid}`)
      .its('body.data.assigned_user_ids')
      .should('deep.equal', [agentId])
  })

  it('assigns the conversation to a team', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/assignee/team`).as('assignTeam')

    openConversation()
    openActionsSection()
    cy.contains('button[role="combobox"]', 'Select team').click()
    cy.get('[role="option"]').contains(teamName).click()

    cy.wait('@assignTeam').its('response.statusCode').should('eq', 200)
    cy.contains('button[role="combobox"]', teamName).should('exist')
  })

  it('changes the priority', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/priority`).as('setPriority')

    openConversation()
    openActionsSection()
    cy.contains('button[role="combobox"]', 'Select priority').click()
    cy.get('[role="option"]').contains('High').click()

    cy.wait('@setPriority').its('response.statusCode').should('eq', 200)
    cy.contains('button[role="combobox"]', 'High').should('exist')

    cy.api('GET', `/api/v1/conversations/${conversationUuid}`)
      .its('body.data.priority')
      .should('eq', 'High')
  })

  it('adds and removes a tag', () => {
    cy.intercept('POST', `**/conversations/${conversationUuid}/tags`).as('setTags')

    openConversation()
    openActionsSection()
    cy.get('input[placeholder="Select tags"]').click()
    cy.get('[role="option"]').contains(tagName).click()

    cy.wait('@setTags').its('response.statusCode').should('eq', 200)
    cy.api('GET', `/api/v1/conversations/${conversationUuid}`)
      .its('body.data.tags')
      .should('include', tagName)

    cy.contains('[data-radix-vue-collection-item]', tagName).find('button').click()
    cy.wait('@setTags').its('response.statusCode').should('eq', 200)
    cy.api('GET', `/api/v1/conversations/${conversationUuid}`)
      .its('body.data.tags')
      .should('not.include', tagName)
  })

  it('posts a private note that renders as a note, not a reply', () => {
    cy.intercept('POST', `**/conversations/${conversationUuid}/messages`).as('sendNote')

    // Hold the real prefetch response until the user has explicitly chosen a note.
    // Hydration must not silently turn that choice into a public reply.
    let releaseDrafts
    const draftsGate = new Cypress.Promise((resolve) => {
      releaseDrafts = resolve
    })
    cy.intercept('GET', '**/api/v1/drafts', (request) => {
      request.continue(() => draftsGate)
    }).as('draftHydration')
    openConversation()
    cy.contains('[role="tab"]', 'Private note').click()
    cy.contains('[role="tab"]', 'Private note').should('have.attr', 'aria-selected', 'true')
    cy.then(() => {
      releaseDrafts()
    })
    cy.wait('@draftHydration')
    cy.contains('[role="tab"]', 'Private note').should('have.attr', 'aria-selected', 'true')
    cy.get('.tiptap.ProseMirror').first().click()
    cy.get('.tiptap.ProseMirror').first().type(noteBody)
    cy.contains('button', /^Send$/).click()

    cy.wait('@sendNote').then(({ request, response }) => {
      expect(request.body.private).to.eq(true)
      expect(response.statusCode).to.eq(200)
      expect(response.body.data.private).to.eq(true)
      const uuid = response.body.data.uuid
      cy.get(`[data-message-uuid="${uuid}"]`)
        .should('contain.text', noteBody)
        .find('[data-message-private="true"]')
        .should('be.visible')
      cy.api('GET', `/api/v1/conversations/${conversationUuid}/messages/${uuid}`)
        .its('body.data.private')
        .should('eq', true)
    })
  })

  it('sends a reply that appears in the thread', () => {
    cy.intercept('POST', `**/conversations/${conversationUuid}/messages`).as('sendReply')

    openConversation()
    // The recipient comes from the loaded thread, sending early fails with "recipient required".
    cy.get('input[placeholder="Email addresses separated by comma"]')
      .first()
      .should('not.have.value', '')
    cy.get('.tiptap.ProseMirror').first().click()
    cy.get('.tiptap.ProseMirror').first().type(replyBody)
    cy.contains('button', /^Send$/).click() // exact: the split button next to it is "send and set status"

    cy.wait('@sendReply').then(({ request, response }) => {
      expect(request.body.private).to.eq(false)
      expect(response.statusCode).to.eq(200)
      expect(response.body.data.private).to.eq(false)
      cy.get(`[data-message-uuid="${response.body.data.uuid}"]`)
        .should('contain.text', replyBody)
        .find('[data-message-private="false"]')
        .should('be.visible')
    })
  })

  it('resolves the conversation and it moves out of the open list', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/status`).as('setStatus')
    cy.intercept('GET', '**/conversations/all?*', (request) => {
      const filters = JSON.parse(request.query.filters || '[]')
      const status = filters.find((filter) => filter.model === 'conversation_statuses')?.value
      if (status === 'Open') request.alias = 'openList'
      if (status === 'Resolved') request.alias = 'resolvedList'
    })

    openConversation()
    statusBadge().click()
    cy.get('[role="menuitem"][data-status="Resolved"]').click()
    cy.wait('@setStatus').its('response.statusCode').should('eq', 200)
    statusBadge().should('have.attr', 'data-status', 'Resolved').and('contain.text', 'Resolved')
    cy.api('GET', `/api/v1/conversations/${conversationUuid}`)
      .its('body.data.status')
      .should('eq', 'Resolved')

    // Wait for the list to actually load, else "not.exist" passes on an empty list.
    cy.visit('/inboxes/all')
    cy.get('[data-testid="conversation-list-status"]').click()
    cy.get('[role="menuitem"][data-status="Open"]').click()
    cy.wait('@openList').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      expect(response.body.data.results.map((ticket) => ticket.uuid)).not.to.include(
        conversationUuid
      )
    })
    cy.get('[data-testid="conversation-list-status"]').should('have.attr', 'data-status', 'Open')
    cy.contains(contactName).should('not.exist')

    cy.get('[data-testid="conversation-list-status"]').click()
    cy.get('[role="menuitem"][data-status="Resolved"]').click()
    cy.wait('@resolvedList').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      const ticket = response.body.data.results.find((ticket) => ticket.uuid === conversationUuid)
      expect(ticket, 'persisted resolved ticket in the resolved list').to.exist
      expect(ticket.status).to.eq('Resolved')
    })
    cy.get('[data-testid="conversation-list-status"]').should(
      'have.attr',
      'data-status',
      'Resolved'
    )
    cy.contains(contactName).should('exist')
  })
})

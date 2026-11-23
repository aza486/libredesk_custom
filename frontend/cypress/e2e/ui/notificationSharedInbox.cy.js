// Run only against the disposable validation app and its synthetic mailbox.
describe('Shared notification and internal mailbox', () => {
  const smtpKeys = ['host', 'port', 'username', 'password', 'max_conns', 'idle_timeout', 'wait_timeout', 'auth_protocol', 'max_msg_retries', 'tls_type', 'tls_skip_verify', 'hello_hostname']
  let smtpBefore
  beforeEach(() => { cy.viewport(1280, 850); cy.login() })

  it('selects a mailbox without a team recipient and reloads without restart', () => {
    cy.api('GET', '/api/v1/settings/notifications/email').then(({ body }) => { smtpBefore = body.data })
    cy.intercept('PUT', '**/api/v1/settings/notifications/email').as('saveSharedMailbox')
    cy.visit('/admin/notification')
    cy.get('[data-testid="notification-inbox"]').click()
    cy.get('[role="option"]').contains('Browser system inbox — browser-system@example.test').click()
    cy.get('input[name="team_contact_email"]').should('not.exist')
    cy.get('input[name="host"]').should('not.exist')
    cy.get('button[type="submit"]').click()
    cy.wait('@saveSharedMailbox').its('response.statusCode').should('eq', 200)
    cy.reload()
    cy.get('[data-testid="notification-inbox"]').should('contain', 'Browser system inbox')
    cy.get('input[name="team_contact_email"]').should('not.exist')
    cy.api('GET', '/api/v1/settings/notifications/email').then(({ body }) => {
      for (const key of smtpKeys) expect(body.data[`notification.email.${key}`], `preserved ${key}`).to.eq(smtpBefore[`notification.email.${key}`])
      expect(body.data['notification.email.email_address']).to.eq('browser-system@example.test')
      expect(body.data['notification.email.inbox_uuid']).to.match(/^[0-9a-f-]{36}$/)
    })
    cy.request('/api/v1/settings/general').then(({ body }) => { expect(body.data['app.restart_required']).not.to.eq(true) })
  })

  it('shows the existing mailbox selector on a narrow screen', () => {
    cy.viewport(390, 844)
    cy.visit('/admin/notification')
    cy.get('[data-testid="notification-inbox"]').should('be.visible').click()
    cy.get('[role="option"]').contains('Browser system inbox').should('be.visible')
    cy.get('[role="option"]').should('not.contain', 'Private browser inbox')
  })

  it('creates a real team-only ticket and stores a visible follow-up without email controls', () => {
    let teamID, uuid
    const teamName = `Stored team ${Date.now()}`
    cy.api('POST', '/api/v1/teams', { name: teamName, emoji: '🧪', conversation_assignment_type: 'Manual', timezone: 'UTC' }).then(({ body }) => { teamID = body.data.id })
    cy.visit('/inboxes/all')
    cy.contains('New conversation').click()
    cy.get('[role="dialog"]').contains('button', 'Internes Ticket').click()
    cy.get('[role="dialog"]').contains('button[role="combobox"]', 'Select team').click()
    cy.get('[role="option"]').contains(teamName).click()
    cy.get('[role="dialog"] input[name="subject"]').type('Stored-only browser ticket')
    cy.get('[role="dialog"] .tiptap.ProseMirror').type('Original stored team message')
    cy.intercept('POST', '**/api/v1/conversations').as('createTeam')
    cy.get('[role="dialog"] button[type="submit"]').click()
    cy.wait('@createTeam').then(({ response, request }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body).not.to.have.property('contact_email')
      expect(response.body.data.contact_id).to.eq(0)
      expect(response.body.data.assigned_team_id).to.eq(teamID)
      uuid = response.body.data.uuid
      cy.visit(`/inboxes/all/conversation/${uuid}`)
    })
    cy.contains('Original stored team message').should('be.visible')
    cy.get('[data-testid="stored-internal-message"]').should('contain', 'no email delivery')
    cy.contains('button', 'Store internally').should('be.visible')
    cy.contains('label', /^TO:$/).should('not.exist')
    cy.get('.tiptap.ProseMirror').last().type('Stored browser follow-up')
    cy.intercept('POST', '**/conversations/*/messages').as('storeInternal')
    cy.contains('button', 'Store internally').click()
    cy.wait('@storeInternal').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      expect(response.body.data.status).to.eq('stored')
      expect(response.body.data.type).to.eq('internal')
    })
    cy.contains('Stored browser follow-up').should('be.visible')
    cy.then(() => cy.api('GET', `/api/v1/conversations/${uuid}/messages`)).then(({ body }) => {
      const messages = body.data.results.filter(message => message.type !== 'activity')
      expect(messages).to.have.length(2)
      for (const message of messages) {
        expect(message.type).to.eq('internal')
        expect(message.status).to.eq('stored')
        expect(message.private).to.eq(false)
        expect(message.meta).not.to.have.property('to')
      }
    })
  })
})

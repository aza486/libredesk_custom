// These checks use only synthetic API data and the configured SMTP test sink.
// No workflow export or external AI service is required.
describe('Post-upgrade custom UI regressions', () => {
  const stamp = Date.now()
  const message = `Post-upgrade inbound ${stamp}`
  const draft = `Post-upgrade unsent draft ${stamp}`
  const address = `stabilization.${stamp}@example.test`
  let inboxId
  let uuid

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/inboxes', {
      name: `Post-upgrade inbox ${stamp}`,
      channel: 'email',
      enabled: true,
      from: `Stabilization <stabilization.${stamp}@example.test>`,
      config: {
        auth_type: 'password',
        imap: [],
        smtp: [
          {
            host: Cypress.env('SMTP_HOST') || '127.0.0.1',
            port: Number(Cypress.env('SMTP_PORT') || 1025),
            auth_protocol: 'none',
            tls_type: 'none',
            max_conns: 2,
            max_msg_retries: 1,
            idle_timeout: '5s',
            pool_wait_timeout: '5s'
          }
        ]
      }
    }).then(({ body }) => {
      inboxId = body.data.id
      cy.api('POST', '/api/v1/conversations', {
        inbox_id: inboxId,
        contact_email: `stabilization.customer.${stamp}@example.test`,
        first_name: 'Stabilization',
        subject: `Post-upgrade subject ${stamp}`,
        content: `<p>${message}</p>`,
        initiator: 'contact'
      }).then(({ body: conversation }) => {
        uuid = conversation.data.uuid
      })
    })
  })

  beforeEach(() => {
    cy.viewport(1440, 900)
    cy.login()
  })

  after(() => {
    cy.login()
    cy.api(
      'DELETE',
      `/api/v1/service-email-addresses?address=${encodeURIComponent(address)}`,
      null,
      { failOnStatusCode: false }
    )
    if (inboxId) cy.api('DELETE', `/api/v1/inboxes/${inboxId}`, null, { failOnStatusCode: false })
  })

  it('retains an admin conversation deep link while the real profile response is delayed', () => {
    cy.intercept('GET', '**/api/v1/agents/me', (req) => {
      req.continue((res) => res.setDelay(750))
    }).as('profile')
    cy.visit(`/inboxes/all/conversation/${uuid}`)
    cy.wait('@profile').its('response.statusCode').should('eq', 200)
    cy.location('pathname').should('eq', `/inboxes/all/conversation/${uuid}`)
    cy.contains(message).should('be.visible')
  })

  it('preserves an unsent Tiptap draft when expanding and restoring the composer', () => {
    cy.visit(`/inboxes/all/conversation/${uuid}`)
    cy.contains(message).should('be.visible')
    cy.get('.tiptap.ProseMirror').first().click()
    cy.get('.tiptap.ProseMirror').first().type(draft)
    cy.get('button[aria-label="Expanded reply editor"]').click()
    cy.get('[role="dialog"] .tiptap.ProseMirror').should('contain.text', draft)
    cy.get('button[aria-label="Restore editor"]').click()
    cy.get('.tiptap.ProseMirror').first().should('contain.text', draft)
    cy.api('GET', `/api/v1/conversations/${uuid}/messages?page=1`).then(({ body }) => {
      expect(JSON.stringify(body.data), 'draft has not been sent').not.to.contain(draft)
    })
  })

  it('shares the service-address registry between sidebar and admin and confirms removal', () => {
    cy.visit('/inboxes/assigned')
    cy.contains('button', 'Manage addresses').click()
    cy.get('[role="dialog"] input[type="email"]').type(address)
    cy.get('[role="dialog"] button[type="submit"]').click()
    cy.get('[role="dialog"]').contains(address).should('be.visible')
    cy.api('GET', '/api/v1/service-email-addresses').its('body.data').should('include', address)

    cy.visit('/admin/service-emails')
    cy.contains(address).should('be.visible')
    cy.get(`button[aria-label="Remove ${address}"]`).click()
    cy.get('[role="alert"]')
      .contains('button', /^Remove$/)
      .click()
    cy.contains(address).should('not.exist')
    cy.api('GET', '/api/v1/service-email-addresses').its('body.data').should('not.include', address)
  })
})

// All customer data is synthetic. Microphone, transcription and generation are
// mocked; this verifies UI composition, not a real provider or microphone.
describe('AI generation and shared voice input', () => {
  const stamp = Date.now()
  const draft = `Unsent voice regression ${stamp}`
  let uuid, inboxID, originalUUID, noteUUID
  let transcript = 'Please keep the reply short.'
  const fullscreen = '[role="dialog"]:has(button[aria-label="Restore editor"])'
  const startVoice = () =>
    cy.get('button[aria-label="Start voice input"]').filter(':visible').click()
  const stopVoice = () =>
    cy.get('button[aria-label="Stop recording and transcribe"]').filter(':visible').click()
  const pressEscape = () => cy.focused().trigger('keydown', { key: 'Escape', bubbles: true })
  const openGenerator = () => cy.get('button[title="Generate reply"]').filter(':visible').click()

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/inboxes', {
      name: `Voice test ${stamp}`,
      channel: 'email',
      enabled: true,
      from: 'Voice <voice@example.test>',
      config: {
        auth_type: 'password',
        imap: [],
        smtp: [
          {
            host: Cypress.env('SMTP_HOST') || 'mailhog',
            port: 1025,
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
      inboxID = body.data.id
      cy.api('POST', '/api/v1/conversations', {
        inbox_id: inboxID,
        contact_email: `voice.customer.${stamp}@example.test`,
        first_name: 'Voice',
        subject: `Voice test ${stamp}`,
        content: '<p>When will order 42 arrive?</p>',
        initiator: 'contact'
      }).then(({ body }) => {
        uuid = body.data.uuid
        cy.api('GET', `/api/v1/conversations/${uuid}/messages?page=1`).then(({ body }) => {
          originalUUID = body.data.results.find((m) => m.type === 'incoming').uuid
        })
        cy.api('POST', `/api/v1/conversations/${uuid}/messages`, {
          sender_type: 'agent',
          private: true,
          message: '<p>Synthetic AI suggestion</p>',
          attachments: [],
          mentions: [],
          cc: [],
          bcc: [],
          to: [],
          echo_id: ''
        }).then(({ body }) => {
          noteUUID = body.data.uuid
        })
      })
    })
  })
  beforeEach(() => {
    cy.viewport(Cypress.currentTest.title.includes('small fullscreen') ? 390 : 1440, 1000)
    cy.login()
    cy.intercept('GET', '**/ai/transcribe/config', { body: { data: { enabled: true } } })
    cy.intercept('POST', '**/ai/transcribe', (req) =>
      req.reply({ body: { data: { text: transcript } } })
    ).as('transcribe')
    cy.intercept('POST', '**/ai/generate-reply', {
      body: { data: '<p>Reviewed suggestion</p>' }
    }).as('generate')
    cy.intercept('GET', `**/conversations/${uuid}/messages?*`, (req) => {
      req.continue((res) => {
        const note = res.body.data.results.find((m) => m.uuid === noteUUID)
        if (note)
          note.meta = {
            ai_analysis: {
              kind: 'n8n_ai_analysis',
              version: 1,
              original_message_uuid: originalUUID,
              suggested_message: 'Synthetic AI suggestion'
            }
          }
      })
    })
    cy.visit(`/inboxes/all/conversation/${uuid}`, {
      onBeforeLoad(win) {
        win.__voiceTracksStopped = 0
        Object.defineProperty(win.navigator, 'mediaDevices', {
          configurable: true,
          value: {
            getUserMedia: async () => ({
              getTracks: () => [{ stop: () => win.__voiceTracksStopped++ }]
            })
          }
        })
        win.MediaRecorder = class {
          static isTypeSupported(type) {
            return type.startsWith('audio/webm')
          }
          state = 'inactive'
          start() {
            this.state = 'recording'
          }
          stop() {
            this.state = 'inactive'
            this.ondataavailable?.({ data: new win.Blob(['synthetic audio']) })
            this.onstop?.()
          }
        }
      }
    })
    cy.contains('When will order 42 arrive?').should('be.visible')
    if (Cypress.currentTest.title.includes('small fullscreen')) {
      cy.get('button:has(svg.lucide-pencil)').should('be.visible').click()
    }
    cy.get('.tiptap.ProseMirror').first().clear()
    cy.get('.tiptap.ProseMirror').first().type(draft)
  })
  after(() => {
    cy.login()
    if (inboxID) cy.api('DELETE', `/api/v1/inboxes/${inboxID}`, null, { failOnStatusCode: false })
  })

  it('dictates in expanded composer, closes only prompt with Escape and preserves unsent draft', () => {
    cy.get('button[aria-label="Expanded reply editor"]').click()
    openGenerator()
    cy.contains('Create or edit an AI reply').should('be.visible')
    startVoice()
    cy.contains('Recording (maximum 2 minutes)').should('be.visible')
    stopVoice()
    cy.wait('@transcribe')
    cy.get('#ai-generate-instructions').should('have.value', transcript)
    cy.get('@generate.all').should('have.length', 0)
    cy.get(`${fullscreen} .tiptap`).should('contain.text', draft)
    cy.get('#ai-generate-instructions').focus()
    pressEscape()
    cy.get('#ai-generate-instructions').should('not.exist')
    cy.get(fullscreen).should('be.visible')
    cy.get(`${fullscreen} .tiptap`).should('contain.text', draft)
    openGenerator()
    startVoice()
    cy.contains('button', 'Verwerfen').should('not.exist')
    cy.contains('button', 'Discard').click()
    cy.window().its('__voiceTracksStopped').should('be.greaterThan', 0)
    openGenerator()
    cy.contains('button', 'Create').should('be.enabled')
    cy.contains('button', 'Discard').click()
    cy.get(fullscreen).should('be.visible')
    // Existing status dropdown is still keyboard accessible after voice input.
    cy.get(`${fullscreen} header button`).first().click()
    cy.get('[role="menu"]').should('be.visible')
    pressEscape()
    cy.get(fullscreen).should('be.visible')
    cy.get('button[aria-label="Restore editor"]').click()
    cy.contains('[role="tab"]', 'Juno').click()
    cy.get('textarea[placeholder="Ask anything…"]').should('be.visible')
    cy.api('GET', `/api/v1/conversations/${uuid}/messages?page=1`).then(({ body }) => {
      expect(JSON.stringify(body.data)).not.to.contain(draft)
    })
  })

  it('previews then explicitly adopts while preserving draft until adoption', () => {
    openGenerator()
    cy.get('#ai-generate-instructions').type('Correct spelling only')
    cy.contains('button', 'Apply').click()
    cy.wait('@generate')
      .its('request.body')
      .should('include', { conversation_uuid: uuid, mode: 'edit' })
    cy.contains('[aria-label="Reply preview"]', 'Reviewed suggestion').should('be.visible')
    cy.get('.tiptap').should('contain.text', draft)
    cy.contains('button', 'Adopt suggestion').click()
    cy.get('.tiptap').should('contain.text', 'Reviewed suggestion')
    cy.get('.tiptap').type('{ctrl}z')
    cy.get('.tiptap').should('contain.text', draft)
  })

  it('dictates in Juno, style instructions and feedback without sending or saving', () => {
    cy.intercept('POST', '**/ai/copilot', (req) => {
      throw new Error(`Voice unexpectedly sent to Juno: ${req.url}`)
    })
    cy.contains('[role="tab"]', 'Juno').click()
    startVoice()
    stopVoice()
    cy.wait('@transcribe')
    cy.get('textarea[placeholder="Ask anything…"]').should('have.value', transcript)
    cy.window()
      .its('__voiceTracksStopped')
      .then((stoppedBeforeSwitch) => {
        startVoice()
        cy.contains('[role="tab"]', 'Details').click()
        cy.window().its('__voiceTracksStopped').should('be.greaterThan', stoppedBeforeSwitch)
      })
    cy.get('button[aria-label="Expanded reply editor"]').click()
    cy.contains('button', 'Stil anpassen').click()
    cy.get('#ai-style-instructions').type('Manual instruction')
    startVoice()
    stopVoice()
    cy.wait('@transcribe')
    cy.get('#ai-style-instructions').should('have.value', `Manual instruction ${transcript}`)
    cy.get('.tiptap').should('contain.text', draft)
    cy.intercept('POST', '**/ai-style-adjust', {
      body: {
        success: true,
        request_id: 'synthetic',
        adapted_text: 'Adjusted preview',
        profile_source: 'preset'
      }
    }).as('style')
    cy.contains('button', 'Anpassen').click()
    cy.wait('@style')
    cy.contains('[aria-label="Antwortvorschau"]', 'Adjusted preview').should('be.visible')
    cy.contains('button', 'Letzte Änderung rückgängig').click()
    cy.contains('[aria-label="Antwortvorschau"]', 'Synthetic AI suggestion').should('be.visible')
    cy.contains('button', 'Verwerfen').click()
    cy.contains('button', 'Write own reply').click()
    cy.get('#ai-feedback-reason').type('Manual reason')
    startVoice()
    stopVoice()
    cy.wait('@transcribe')
    cy.get('#ai-feedback-reason').should('have.value', `Manual reason ${transcript}`)
    cy.contains('button', 'Weiter zur Antwort').should('be.enabled')
    cy.contains('button', 'Abbrechen').click()
    cy.get(fullscreen).should('be.visible')
    cy.get('.tiptap').should('contain.text', draft)
  })
  it('keeps prompt controls reachable on a small fullscreen viewport', () => {
    openGenerator()
    cy.get('#ai-generate-instructions').should('be.visible').type('Short reply')
    startVoice()
    stopVoice()
    cy.wait('@transcribe')
    cy.get('#ai-generate-instructions').should('have.value', `Short reply ${transcript}`)
    cy.contains('button', 'Discard').click()
    cy.get(fullscreen).should('be.visible')
    cy.contains(`${fullscreen} [role="tab"]`, 'Reply').click()
    cy.get(`${fullscreen} .tiptap`).should('be.visible').and('contain.text', draft)
    cy.get('button[aria-label="Restore editor"]').click()
    cy.contains('button', draft).should('be.visible')
  })
})

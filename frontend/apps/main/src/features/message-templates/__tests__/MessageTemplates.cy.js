import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import mitt from 'mitt'
import { initI18n } from '@main/i18n'
import de from '../../../../../../../i18n/de-DE.json'
import Harness from './fixtures/MessageTemplatesHarness.vue'
const template = {
  id: 1,
  uuid: 'test-template',
  name: 'Gelangensbestätigung',
  scope: 'personal',
  revision: 1,
  content:
    '<p>Sehr geehrte/r [Name],</p><p>Vielen Dank.<br>Bestätigung</p><ul><li><strong>Vorteil</strong> <em>Kursiv</em> <u>Unterstrichen</u></li></ul><ol><li>Nummeriert</li></ol><p><a href="https://example.test/preise">Informationen und Preise</a></p>'
}
const openMenu = () => cy.contains('button', 'Vorlagen').click()
const press = (key) => cy.focused().trigger('keydown', { key, bubbles: true })
const menu = () => cy.get('[role="menu"]')
function mount(templates = [template], support = true) {
  cy.intercept('GET', '**/api/v1/message-templates?*', {
    data: { templates, can_customer_support: support }
  }).as('list')
  cy.intercept('GET', '**/api/v1/message-templates/test-template?*', { data: template }).as('get')
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { render: () => null } }]
  })
  const i18n = initI18n({ legacy: false, locale: 'de-DE', messages: { 'de-DE': de } })
  cy.mount(Harness, {
    global: {
      plugins: [createPinia(), router, i18n],
      config: { globalProperties: { emitter: mitt() } }
    }
  })
}
describe('Nachrichtenvorlagen im Antworteditor', () => {
  beforeEach(() => cy.viewport(1280, 1000))
  it('portals upward without clipping or layout shift; inserts at cursor and keeps draft, recipients, mode and signature', () => {
    mount()
    cy.get('[data-test=normal-composer] .tiptap').click()
    cy.get('[data-test=normal-composer] .tiptap').type('{end}')
    cy.get('[data-test=normal-composer]').then(($editor) => {
      const bounds = $editor[0].getBoundingClientRect().toJSON()
      openMenu()
      cy.wait('@list')
      menu().should('have.attr', 'data-side', 'top').should('be.visible')
      cy.get('[data-test=normal-composer]').should(($now) =>
        expect($now[0].getBoundingClientRect().toJSON()).to.deep.equal(bounds)
      )
    })
    menu().find('input').should('not.exist')
    menu()
      .find('.overflow-y-auto')
      .should(($scroll) => expect($scroll[0].scrollHeight).to.be.at.most($scroll[0].clientHeight))
    cy.contains('[role=menuitem]', 'Gelangensbestätigung').click()
    cy.wait('@get')
    menu().should('not.exist')
    cy.get('.tiptap').should('contain.text', 'Bestehender Entwurf').and('contain.text', '[Name]')
    cy.get('.tiptap strong').should('have.text', 'Vorteil')
    cy.get('.tiptap em').should('have.text', 'Kursiv')
    cy.get('.tiptap u').should('have.text', 'Unterstrichen')
    cy.get('.tiptap ol li').should('have.text', 'Nummeriert')
    cy.get('.tiptap a').should('have.attr', 'href', 'https://example.test/preise')
    cy.focused().should('have.class', 'tiptap')
    cy.get('[data-test=recipients]').should(
      'have.text',
      'customer@example.test|cc@example.test|bcc@example.test'
    )
    cy.get('[data-test=signature]').should('have.text', '72')
    cy.get('[data-test=mode]').should('have.text', 'reply')
  })
  it('opens downward in fullscreen, uses local search and only scrolls long lists; Escape keeps editor and private-note mode', () => {
    mount(
      Array.from({ length: 35 }, (_, i) => ({
        ...template,
        uuid: String(i),
        name: `Vorlage ${String(i).padStart(2, '0')}`
      }))
    )
    cy.get('[aria-label="Vergrößerter Antworteditor"]').click()
    cy.contains('[role=tab]', 'Private Notiz').click()
    openMenu()
    cy.wait('@list')
    menu().should('have.attr', 'data-side', 'bottom')
    menu()
      .find('.overflow-y-auto')
      .should(($scroll) =>
        expect($scroll[0].scrollHeight).to.be.greaterThan($scroll[0].clientHeight)
      )
    cy.contains('[role=menuitem]', 'Vorlagen bearbeiten').should('be.visible')
    menu().find('input').type('Vorlage 12')
    menu().find('[role=menuitem]').should('have.length', 2)
    press('Escape')
    menu().should('not.exist')
    cy.get('.template-fullscreen-composer').should('be.visible')
    cy.get('[data-test=mode]').should('have.text', 'private_note')
    cy.get('.template-fullscreen-composer .tiptap').should('contain.text', 'Bestehender Entwurf')
  })
  it('keeps draft while managing templates; confirms unsaved edits and closes only the nested overlay', () => {
    mount()
    cy.get('[aria-label="Vergrößerter Antworteditor"]').click()
    cy.contains('[role=tab]', 'Private Notiz').click()
    openMenu()
    cy.wait('@list')
    cy.contains('[role=menuitem]', 'Vorlagen bearbeiten').click()
    cy.contains('Vorlagen verwalten').should('be.visible')
    cy.contains('button', 'Neue Vorlage').click()
    cy.get('#message-template-name').type('Ungespeichert')
    cy.get('[aria-label="Vorlagenverwaltung schließen"]').click()
    cy.contains('Ungespeicherte Änderungen verwerfen?').should('be.visible')
    cy.contains('[role=alertdialog] button', 'Verwerfen').click()
    cy.contains('Vorlagen verwalten').should('not.exist')
    cy.get('.template-fullscreen-composer').should('be.visible')
    cy.get('.template-fullscreen-composer .tiptap').should('contain.text', 'Bestehender Entwurf')
    cy.get('[data-test=mode]').should('have.text', 'private_note')
  })
  it('appends to an adopted AI draft, keeps private notes private and denies stale content after revocation', () => {
    mount()
    cy.get('[data-test=ai-adopt]').click()
    cy.contains('[role=tab]', 'Private Notiz').click()
    openMenu()
    cy.wait('@list')
    cy.contains('[role=menuitem]', 'Gelangensbestätigung').click()
    cy.wait('@get')
    cy.get('.tiptap')
      .should('contain.text', 'Übernommener KI-Entwurf')
      .and('contain.text', '[Name]')
    cy.get('[data-test=mode]').should('have.text', 'private_note')
    cy.intercept('GET', '**/api/v1/message-templates/test-template?*', {
      statusCode: 404,
      body: { message: 'Rechte entzogen' }
    })
    openMenu()
    cy.wait('@list')
    cy.contains('[role=menuitem]', 'Gelangensbestätigung').click()
    cy.contains('[role=alert]', 'Rechte entzogen').should('be.visible')
    cy.get('.tiptap').find('ul').should('have.length', 1)
  })
  it('supports keyboard selection, outside click, and hiding the global group in private inboxes', () => {
    mount([template], false)
    openMenu()
    cy.wait('@list')
    menu().should('contain.text', 'Meine Vorlagen').and('not.contain.text', 'Kundensupport')
    press('ArrowDown')
    press('Enter')
    cy.wait('@get')
    menu().should('not.exist')
    openMenu()
    cy.wait('@list')
    cy.get('[data-test=normal-composer] input').first().click({ force: true })
    menu().should('not.exist')
  })
  it('flips within a constrained viewport and uses theme colors in dark mode', () => {
    cy.viewport(1000, 600)
    mount(
      Array.from({ length: 35 }, (_, i) => ({ ...template, uuid: String(i), name: `Vorlage ${i}` }))
    )
    openMenu()
    cy.wait('@list')
    menu()
      .should('have.attr', 'data-side', 'bottom')
      .should(($menu) => {
        const bounds = $menu[0].getBoundingClientRect()
        expect(bounds.top).to.be.at.least(0)
        expect(bounds.bottom).to.be.at.most(600)
      })
    menu()
      .invoke('css', 'background-color')
      .then((light) => {
        cy.document().then((doc) => doc.documentElement.classList.add('dark'))
        menu().invoke('css', 'background-color').should('not.equal', light)
      })
    cy.contains('[role=menuitem]', 'Vorlagen bearbeiten').should('be.visible')
    cy.document().then((doc) => doc.documentElement.classList.remove('dark'))
  })
  it('clears the open menu when employee identity changes', () => {
    mount()
    openMenu()
    cy.wait('@list')
    cy.get('[data-test=switch-identity]').click({ force: true })
    menu().should('not.exist')
    cy.get('.tiptap').should('have.text', 'Bestehender Entwurf')
  })
})

import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import mitt from 'mitt'
import { initI18n } from '@main/i18n'
import en from '../../../../../../i18n/en-US.json'
import Harness from './fixtures/FullscreenLightboxHarness.vue'

const viewer = '.vel-modal'
const fullscreen = '.lightbox-fullscreen-harness'
const openImage = (index = 0) => {
  cy.get('[data-test=history] img').eq(index).click('bottom', { scrollBehavior: false })
  cy.get(viewer).should('be.visible')
  cy.focused().closest('[role=dialog]').should('not.have.class', 'lightbox-fullscreen-harness')
}
const image = () => cy.get(`${viewer} img[style]`)
const press = (key) => cy.focused().trigger('keydown', { key, bubbles: true })
const expand = () => {
  cy.get('[data-test=expand]').click()
  // Capture layout only after the real dialog's entry animation has settled.
  cy.get(fullscreen).should(($dialog) => {
    expect($dialog[0].getBoundingClientRect().left).to.equal(16)
    expect($dialog[0].getBoundingClientRect().top).to.equal(16)
  })
}

// Real Radix dialogs/focus scopes, actual split layout, attachment controls and Tiptap.
// No backend or login is used. Fail on any attempted API request.
describe('Image lightbox above fullscreen composer', () => {
  let initialOverflow
  beforeEach(() => {
    cy.viewport(1440, 900)
    cy.document().then((doc) => {
      initialOverflow = doc.body.style.overflow
    })
    cy.intercept('**/api/**', (req) => {
      throw new Error(`Unexpected backend request: ${req.url}`)
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { render: () => null } }]
    })
    const i18n = initI18n({ legacy: false, locale: 'en-US', messages: { 'en-US': en } })
    cy.mount(Harness, {
      global: {
        plugins: [createPinia(), router, i18n],
        config: { globalProperties: { emitter: mitt() } }
      }
    })
  })

  it('closes only the image with X, reopens and restores fullscreen with draft/note/history intact', () => {
    expand()
    cy.get(`${fullscreen} .tiptap`).clear()
    cy.get(`${fullscreen} .tiptap`).type('Unsent browser regression draft')
    cy.contains(`${fullscreen} [role=tab]`, 'Private note').click()
    cy.get('[data-test=history]').scrollTo(0, 40)
    cy.get(fullscreen).then(($dialog) => {
      const dialog = $dialog[0]
      const editor = dialog.querySelector('.tiptap')
      const history = dialog.querySelector('[data-test=history]')
      openImage()
      cy.get('.btn__close').click()
      cy.get(viewer).should('not.exist')
      cy.get(fullscreen).should(($current) => {
        expect($current[0]).to.equal(dialog)
        expect($current[0].querySelector('.tiptap')).to.equal(editor)
        expect($current[0].querySelector('[data-test=history]')).to.equal(history)
        expect(history.scrollTop).to.equal(40)
      })
    })
    cy.get(`${fullscreen} .tiptap`).should('contain.text', 'Unsent browser regression draft')
    cy.get('[data-test=mode-state]').should('have.text', 'private_note')
    openImage(1)
    cy.get('.btn__close').click()
    cy.get('[aria-label="Restore editor"]').click()
    cy.get(fullscreen).should('not.exist')
    cy.document().should((doc) => {
      expect(doc.body.style.overflow).to.equal(initialOverflow)
    })
    cy.get('[data-test=draft-state]').should('contain.text', 'Unsent browser regression draft')
    cy.get('[data-test=mode-state]').should('have.text', 'private_note')
  })

  it('arbitrates Escape by overlay level and restores focus to the fullscreen scope', () => {
    expand()
    openImage()
    press('Escape')
    cy.get(viewer).should('not.exist')
    cy.get(fullscreen).should('be.visible')
    cy.focused().closest(fullscreen).should('exist')
    press('Escape')
    cy.get(fullscreen).should('not.exist')
  })

  it('navigates by pointer and keyboard without remounts, double steps or layout changes', () => {
    expand()
    cy.get(fullscreen).then(($dialog) => {
      const dialog = $dialog[0]
      const editor = dialog.querySelector('.tiptap')
      const bounds = dialog.getBoundingClientRect().toJSON()
      openImage()
      image()
        .invoke('attr', 'src')
        .then((first) => {
          cy.get('.btn__next').click()
          image()
            .invoke('attr', 'src')
            .then((second) => {
              expect(second).not.to.equal(first)
              cy.get('.btn__prev').click()
              image().should('have.attr', 'src', first)
              press('ArrowRight')
              image().should('have.attr', 'src', second)
              press('ArrowLeft')
              image().should('have.attr', 'src', first)
            })
        })
      cy.get('.btn__close').click()
      cy.get(fullscreen).should(($current) => {
        expect($current[0]).to.equal(dialog)
        expect(dialog.querySelector('.tiptap')).to.equal(editor)
        expect(dialog.getBoundingClientRect().toJSON()).to.deep.equal(bounds)
      })
    })
  })

  it('keeps Tab focus inside the viewer and preserves normal fullscreen keyboard handling afterwards', () => {
    expand()
    openImage()
    press('Tab')
    cy.get(`${fullscreen} .tiptap`).then(($editor) => {
      $editor[0].focus()
    })
    cy.focused().closest('[role=dialog]').should('not.have.class', 'lightbox-fullscreen-harness')
    press('Escape')
    cy.get(fullscreen).should('be.visible')
    // No active viewer: navigation is no longer prevented by the removed global handler.
    cy.focused().then(($focused) => {
      const event = new KeyboardEvent('keydown', {
        key: 'ArrowRight',
        bubbles: true,
        cancelable: true
      })
      $focused[0].dispatchEvent(event)
      expect(event.defaultPrevented).to.equal(false)
    })
  })

  it('opens, navigates and closes normally outside fullscreen, including mask and reopening', () => {
    openImage(1)
    cy.get(`${viewer} .vel-img`)
      .should('be.visible')
      .then(($img) => {
        const transform = $img[0].parentElement.style.transform
        cy.get(viewer).trigger('wheel', { deltaY: -100 })
        cy.get(`${viewer} .vel-img`).should(($zoomed) => {
          expect($zoomed[0].parentElement.style.transform).not.to.equal(transform)
        })
      })
    image()
      .invoke('attr', 'src')
      .then((second) => {
        press('ArrowRight')
        image().should('not.have.attr', 'src', second)
        cy.get(viewer).click('topLeft')
        cy.get(viewer).should('not.exist')
        openImage(1)
        image().should('have.attr', 'src', second)
      })
    press('Escape')
    cy.get(viewer).should('not.exist')
    cy.get(fullscreen).should('not.exist')
    cy.document().should((doc) => {
      expect(doc.body.style.overflow).to.equal(initialOverflow)
    })
  })
})

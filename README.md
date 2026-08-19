# LibreDesk Custom

> A customized LibreDesk instance extended for an internal customer-support and AI-assisted email workflow.

This repository contains my customized version of [LibreDesk](https://github.com/abhinavxd/libredesk), an open-source, self-hosted customer support platform.

The project was extended and adapted for a specific internal workflow around **customer emails, ticket management, AI-assisted processing and human approval**.

## ⚠️ Project Scope & Dependencies

This repository contains the **LibreDesk application and my custom modifications**, but it is only one part of the complete system.

The fully functional internal solution consists of several connected components:

```text
Customer Email
      ↓
   LibreDesk
      ↓
      n8n
      ↓
  AI Processing
      ↓
 PostgreSQL
      ↓
 Human Approval
      ↓
   LibreDesk
      ↓
 Customer Response
```

The corresponding **n8n workflows and PostgreSQL databases are private and are therefore not included in this repository**.

This means that cloning this repository provides the LibreDesk application and its visible code-level customizations, but **does not reproduce the complete production system**.

Some functionality may therefore be unavailable or behave differently in a standalone installation.

This separation is intentional because the private components contain **internal business logic, customer-related data structures, workflow configurations and other non-public information**.

## About the Project

Instead of building a customer-support system from scratch, I used LibreDesk as an existing open-source foundation and focused on extending it for a real-world use case.

The complete system combines:

* Customer email handling
* Ticket management
* AI-assisted processing
* Automated workflows
* Product database integration
* Human approval
* Internal support processes

The project therefore combines **frontend development, backend/API integration, Docker, PostgreSQL, automation and AI workflows**.

## My Contributions

### Frontend & LibreDesk

* Customized the Vue.js interface
* Modified navigation and inbox views
* Added and adjusted ticket counters and badges
* Customized tags, teams and ticket states
* Adapted UI elements to the internal workflow
* Integrated LibreDesk into the surrounding automation architecture

### Automation & AI

The customized LibreDesk instance is connected to a private **n8n automation environment**.

The workflow can:

1. Receive incoming customer emails
2. Anonymize relevant personal data
3. Store and process email information
4. Classify requests using AI
5. Query internal product information
6. Generate a suggested response
7. Present the result for human approval
8. Send the approved response
9. Update the ticket status

The automation and database components are intentionally kept private and are not part of this repository.

### Infrastructure

The complete environment involves:

* Docker & Docker Compose
* PostgreSQL
* Redis
* Linux server administration
* Git & GitHub
* API integrations
* Webhooks
* n8n automation
* AI services
* Backup and recovery workflows

## What I Learned

Working on LibreDesk gave me practical experience with a larger existing codebase rather than only developing isolated applications from scratch.

I learned how to:

* Navigate and understand an unfamiliar codebase
* Extend an existing Vue application
* Debug frontend/API communication
* Work with Dockerized applications
* Work with PostgreSQL and Redis
* Design automation workflows with n8n
* Integrate AI services with existing software
* Connect multiple services into one workflow
* Use Git for ongoing development
* Think about reliability, backups and production environments

One of the biggest lessons was learning that software development is often less about building everything yourself and more about **understanding existing systems and extending them without breaking their architecture**.

## 🤖 AI Transparency

AI tools were used as part of the development process.

I primarily used AI for:

* Understanding unfamiliar code
* Explaining errors
* Debugging
* Exploring implementation approaches
* Reviewing code and architecture
* Working through API and automation problems

AI did **not independently design or implement the project**.

The architecture, workflow design, UI changes, integrations, testing and final implementation were developed and validated by me.

AI was used as a development and learning assistant. Generated suggestions were reviewed, adapted and tested before being integrated.

## 🔒 Privacy & Confidential Components

The repository intentionally does **not** contain:

* Production customer data
* Private PostgreSQL databases
* Internal n8n workflows
* Credentials or API keys
* Internal business logic that is not required for the LibreDesk application
* Other confidential infrastructure configuration

The public repository therefore represents the **LibreDesk/custom-development part of the project**, while the complete production architecture remains private.

## Original Project

This project is based on **LibreDesk** by Abhinav Raut.

Original repository:

https://github.com/abhinavxd/libredesk

LibreDesk is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.

This repository contains modifications and extensions made for my specific use case. The original project and its respective authors remain credited.

## Status

🚧 **Active development**

The LibreDesk customizations and the connected internal automation system continue to evolve.

---

### Author

**Daniel Podjapolski**

Media Design · Frontend Development · Automation · AI Workflows

### Based on

**LibreDesk**
by **Abhinav Raut**

Licensed under **AGPL-3.0**

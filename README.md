# LibreDesk Custom

> A customized LibreDesk instance extended for an internal customer-support and email automation workflow.

This repository contains my customized version of [LibreDesk](https://github.com/abhinavxd/libredesk), an open-source, self-hosted customer support platform.

The project was extended and adapted to fit a specific internal workflow around **customer emails, ticket management and AI-assisted response generation**.

## About the Project

Instead of building a customer-support system from scratch, I used LibreDesk as an existing open-source foundation and focused on extending it for a real-world use case.

The main goal was to connect:

* Customer email handling
* Ticket management
* AI-assisted processing
* Automated workflows
* Human approval
* Internal support processes

The project therefore combines **frontend development, backend/API integration, Docker, PostgreSQL, automation and AI workflows**.

## My Contributions

My work primarily focused on adapting and extending the existing LibreDesk application.

### Frontend

* Customized the Vue.js interface
* Modified navigation and inbox views
* Added and adjusted ticket counters and badges
* Customized tags, teams and ticket states
* Improved the handling of incoming customer requests
* Adapted UI elements to the internal workflow

### Automation & AI

LibreDesk was integrated into a larger automation workflow using **n8n**.

The workflow can:

1. Receive incoming customer emails
2. Anonymize relevant personal data
3. Store and process email information
4. Classify customer requests using AI
5. Query internal product information
6. Generate a suggested response
7. Present the result for human approval
8. Send the approved response
9. Update the ticket status

This creates a workflow where AI assists employees rather than replacing the human approval process.

### Infrastructure

The project also involved working with:

* Docker & Docker Compose
* PostgreSQL
* Redis
* Linux server administration
* Git & GitHub
* API integrations
* Webhooks
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
* Connect AI services with existing software
* Use Git for ongoing development and version control
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

The decisions regarding the architecture, workflow, UI changes, integrations, testing and final implementation were made and validated by me.

AI was used as a development and learning assistant, with generated suggestions being reviewed, adapted and tested before use.

## Original Project

This project is based on **LibreDesk** by Abhinav Raut.

Original repository:

https://github.com/abhinavxd/libredesk

LibreDesk is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.

This repository contains modifications and extensions made for my specific use case. The original project and its respective authors remain credited.

## Status

🚧 **Active development**

The project continues to evolve as the internal workflow, automation and LibreDesk customizations are improved.

---

### Author

**Daniel Podjapolski**

Media Design · Frontend Development · Automation · UI/UX

### Based on

**LibreDesk**
by **Abhinav Raut**

Licensed under **AGPL-3.0**

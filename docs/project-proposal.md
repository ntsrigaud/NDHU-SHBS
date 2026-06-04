# NDHU Second-Hand Book Store Project Proposal

**Team:** Neil Taison Rigaud, Jn Neil Alexander, Sley Hortes, Jaime Medina.  
**Institution:** National Dong Hwa University.  
**Course:** Software Engineering, April 2026.

---

### I. Project Overview
The Second-Hand Book Store (SHBS) is a web-based application for **National Dong Hwa University (NDHU)** students to buy and sell used books. It addresses the high cost of textbooks by providing a trusted, campus-exclusive marketplace verified through **NDHU Single Sign-On (SSO)**. Key features include **AI-powered book condition classification** and automated metadata extraction to reduce information asymmetry and increase buyer confidence. Unlike the existing Facebook group, SHBS offers book-specific focus, structured search/filtering, and auditable transaction records.

### II. Scope of Work
The system utilizes a **three-tier client-server architecture**:
*   **Frontend:** React-based Multi-Page Application (MPA).
*   **Backend:** Go-based REST API server.
*   **Database:** PostgreSQL (relational) and MongoDB (unstructured).

**Functional Areas:**
1.  **Authentication:** NDHU SSO and JWT-based session management.
2.  **Marketplace:** Responsive book wall with department, price, and condition filters.
3.  **Seller Features:** Multi-image upload, AI-extracted metadata (title/author/ISBN), and listing management.
4.  **Messaging:** Context-specific buyer-seller chat scoped to listings.
5.  **Commerce:** Persistent shopping cart, simulated checkout, and automatic de-listing.
6.  **AI Integration:** Condition classification (Good/Moderate/Poor) with confidence scores.
7.  **Notifications:** In-app alerts for orders and messages.

### III. Technical Approach & Technology Stack
The project adopts a **microservice-based architecture** for scalability and maintainability. Services communicate via RESTful APIs through a **Traefik** reverse proxy.
*   **Frontend:** Vite, React, Zustand, TailwindCSS.
*   **Backend:** Go (Golang).
*   **AI/ML:** LSTM and frame-level defect detection models trained via Roboflow.
*   **DevOps:** Docker containerization and GitHub Actions for CI/CD.

### IV. Team Composition
*   **Jn Neil Alexander (Product Manager):** Facilitates Scrum and coordinates communication.
*   **Sley Hortes (Software Architect):** Defines architecture and coding standards.
*   **Jaime Medina (Software Developer & UI/UX Lead):** Converts requirements into code.
*   **Neil Taison Rigaud (DevOps & QA Lead):** Manages CI/CD, staging, and quality assurance.

### V. AI Assistance Scope
The team utilizes **Claude** and **GitHub Copilot** for specific tasks such as diagram generation (Mermaid syntax), understanding libraries (Traefik/Clerk), generating Go boilerplate, and suggesting test cases.

### VI. Project Timeline (Agile/Scrum)
*   **Sprint 0 (Week 1):** Repository setup, CI/CD pipeline, and basic scaffolding.
*   **Sprint 1 (Weeks 2-3):** SSO login, user profiles, and the marketplace book wall.
*   **Sprint 2 (Weeks 3-4):** Book uploads, AI integration, and messaging.
*   **Sprint 3 (Week 5):** Cart, simulated checkout, and notification system.
*   **Sprint 4 (Week 6):** Hardening, bug fixes, and final staging deployment.

### VII. Testing and Quality Assurance
Testing includes **Vitest** for the frontend and **Testcontainers-go** for the backend to ensure logic works with real database instances.
*   **Metrics:** Minimum 80% backend code coverage.
*   **Quality Gates:** Pull requests must pass unit tests, linting, and build checks.
*   **Defect Management:** Issues are triaged from **P1 (Critical)** to **P4 (Low)**.

### VIII. Risk Assessment
*   **SSO Integration:** Mitigation involves early contact with NDHU IT or fallback to local auth.
*   **Timeline:** If microservice complexity is too high, services will be consolidated into a single Go binary.
*   **AI Metadata:** Errors are mitigated by requiring seller review before listings go live.

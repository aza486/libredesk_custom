# Libredesk

Fully open-source, self-hosted customer support desk. Single binary app.

> This project is currently in **alpha**. Features and APIs may change and are not yet fully tested.

Visit [libredesk.io](https://libredesk.io) for more info. Check out the [**Live demo**](https://demo.libredesk.io/).

![libredesk-hero-2](https://github.com/user-attachments/assets/0d437a70-92d5-46f1-8936-28b8774d9e45)


## Developer Setup

#### Prerequisites

- **go**
- **pnpm**
- **PostgreSQL >= 13**
- **Redis**

1. **Clone the repository**:

   ```bash
   git clone https://github.com/abhinavxd/libredesk.git
   cd libredesk
   ```

2. **Configure the Application**:

   - Copy the sample configuration file `config.toml.sample` to `config.toml`:
    
       ```bash
       cp config.toml.sample config.toml
       ```
   - Edit the `config.toml` file to configure your database and Redis connection settings.

3. **Run in Development Mode**:

   - Backend: `make run-backend`
   - Frontend: `make run-frontend`


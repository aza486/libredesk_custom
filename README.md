<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>


# Libredesk

Fully open-source, self-hosted customer support desk. Single binary app.

> This project is currently in **alpha**. Features and APIs may change and are not yet fully tested.

Visit [libredesk.io](https://libredesk.io) for more info. Check out the [**Live demo**](https://demo.libredesk.io/).

![Screenshot_20250220_231723](https://github.com/user-attachments/assets/55e0ec68-b624-4442-8387-6157742da253)


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

2. **Create config file**:

   - Copy the sample configuration file `config.toml.sample` to `config.toml`:
    
       ```bash
       cp config.toml.sample config.toml
       ```
   - Edit the `config.toml` file to configure your postgres and redis connection settings.

3. **Run in Development Mode**:

   - Backend: `make run-backend`
   - Frontend: `make run-frontend`


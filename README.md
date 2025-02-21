<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>


# Libredesk

Open source, self-hosted customer support desk. Single binary app.

Visit [libredesk.io](https://libredesk.io) for more info. Check out the [**Live demo**](https://demo.libredesk.io/).

![Screenshot_20250220_231723](https://github.com/user-attachments/assets/55e0ec68-b624-4442-8387-6157742da253)


> [!CAUTION]
> This project is currently in **alpha**. Features and APIs may change and are not yet fully tested.


## Developer Setup

#### Prerequisites

- **go**
- **pnpm**
- **postgreSQL >= 13**
- **redis**

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

3. **Run in development mode**:

   - Backend: `make run-backend`
   - Frontend: `make run-frontend`


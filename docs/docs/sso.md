# Setting up SSO

Libredesk supports external OpenID Connect providers (e.g., Google, Keycloak) for signing in users.

!!! note
    User accounts must be created in Libredesk manually; signup is not supported.

## Generic Configuration Steps

Since each provider’s configuration might differ, consult your provider’s documentation for any additional or divergent settings.

1. **Provider Setup:**  
   In your provider’s admin console, create a new OpenID Connect application/client. Retrieve:
      - **Client ID**
      - **Client Secret**

2. **Libredesk Configuration:**  
   In Libredesk, navigate to **Security > SSO** and click **New SSO**. Enter:
      - **Provider URL** (e.g., the URL of your OpenID provider)
      - **Client ID**
      - **Client Secret**
      - A descriptive **Name** for the connection

3. **Redirect URL:**  
   After saving, copy the generated **Callback URL** from Libredesk and add it as a valid redirect URI in your provider’s client settings.
   
## Provider Examples

### Keycloak

1. Log in to your Keycloak Admin Console.

2. In Keycloak, navigate to **Clients** and click **Create**:

      - **Client ID** (e.g., `libredesk-app`)
      - **Client Protocol**: `openid-connect`
      - **Root URL** and **Web Origins**: your app domain (e.g., `https://ticket.example.com`)
      - Under **Authentication flow**, uncheck everything except **Standard flow**
      - Click **Save**

3. Go to the **Credentials** tab:
      - Ensure **Client Authenticator** is set to `Client Id and Secret`
      - Note down the generated **Client Secret**

4. In Libredesk, go to **Security > SSO** and click **New SSO**:
      - **Provider URL** (e.g., `https://keycloak.example.com/realms/yourrealm`)
      - **Name** (e.g., `Keycloak`)
      - **Client ID**
      - **Client Secret**
      - Click **Save**

5. After saving, click on the three dots and choose **Edit** to open the newly SSO entry.

6. Copy the generated **Callback URL** from Libredesk.

7. Back in Keycloak, edit the client and add the **Callback URL** to **Valid Redirect URIs**:
      - e.g., `https://ticket.example.com/api/v1/oidc/1/finish`

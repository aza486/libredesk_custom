# Setting up SSO

Libredesk supports external OpenID Connect providers (e.g., Google, Keycloak) for signing in users.

!!! Note
    User accounts must be created in Libredesk manually; signup is not supported.

## Generic Configuration Steps

1. **Provider Setup:**  
   In your provider’s admin console, create a new OpenID Connect application/client. Retrieve the **Client ID** and **Client Secret**.

2. **Libredesk Configuration:**  
   In Libredesk, navigate to **Security > SSO** and click **New SSO**. Enter the following:
   - **Provider URL** (e.g., the URL of your OpenID provider)
   - **Client ID**
   - **Client Secret**
   - A descriptive **Name** for the connection

3. **Redirect URL:**  
   After saving, copy the generated **Callback URL** from Libredesk and add it as a valid redirect URI in your provider’s client settings.

4. **Provider-Specific Adjustments:**  
   Since each provider’s configuration might differ, please consult your provider’s documentation for any additional or divergent settings.

## Provider Examples

### Keycloak

1. Log in to your Keycloak Admin Console.

2. In Keycloak, navigate to **Clients** and click **Create**:
   - Set **Client ID** (e.g., `libredesk-app`)
   - Set **Client Protocol** to `openid-connect`
   - Set **Root URL** and **Web Origins** to your app domain (e.g., `https://ticket.example.com`)
   - Uncheck everything besides "Standard flow" at "Authentication flow"
   - Save

3. Go to the **Credentials** tab:
   - Ensure **Client Authenticator** is set to `Client Id and Secret`
   - Note down the generated **Client Secret**

4. In Libredesk, go to **Security > SSO** and click **New SSO**:
   - Enter **Provider URL** (e.g., `https://keycloak.example.com/realms/yourrealm`)
   - Enter a descriptive **Name** (e.g., `Keycloak`)
   - Paste the **Client ID** and **Client Secret**
   - Save
   - After you save, click on the three dots and choose "Edit" to open the just created SSO.
   - Copy the generated **Callback URL** from Libredesk.

6. Back in Keycloak, edit the client and add the **Callback URL** to **Valid Redirect URIs**:
   - e.g., `https://ticket.example.com/api/v1/oidc/1/finish`

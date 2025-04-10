# Setting up SSO

You can use external OpenID Providers like Google, Keycloak, ... to allow your users to sign into Libredesk.

!!! Keep in mind
    You still have to create the user accounts in Libredesk, they do not get created automatically.

## Google

There is currently no documentation for it. Feel free to change that!

## Keycloak

1. Go to your Keycloak Admin Console and log in.
2. Open your Libredesk Dashboard and navigat to "Security" > "SSO".
3. Click the "New SSO" button to add a new SSO.
4. Navigate to "Clients" on the left menu.
5. Click "Create" to add a new client.
6. Enter a Client ID (e.g. `libredesk-app`) and select OpenID Connect as the protocol.
7. Set the "Root URL" (e.g. `https://ticket.example.com`).
8. Set the "Web origins" (e.g. `https://ticket.example.com`).
9. Uncheck everything besides "Standard flow" at "Authentication flow" and save your changes.
10. Go to the "Credentials" tab and change the "Client Authenticator" to `Client Id and Secret`.
11. Enter `https://keycloak.example.com/realms/yourrealm` as the "Provider URL" and choose a name (e.g. `Keycloak`) in Libredesk.
12. Copy and paste the Client ID and secret into Libredesk and save.
13. Afer you save, click on the three dots and choose "Edit" to open the just created SSO.
14. Copy the "Callback URL" and paste it into the "Valid redirect URIs" field inside Keycloak (e.g. `https://ticket.example.com/api/v1/oidc/1/finish`).

## Other providers

There is currently no documentation for it. Feel free to change that!

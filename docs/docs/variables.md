# Avaiable variables

!!! warning "Outgoing email template required"

    The current version requires that the used outgoing email template contains `{{ template "content" . }}`

!!! bug "Email variables"

    All email variables currently return `{<actual email> true true}` instead of `<actual email>` which you might expect. 

## Conversation Variables
{{ .Conversation.ReferenceNumber }}  - The unique reference number of the conversation
{{ .Conversation.Subject }}          - The subject of the conversation
{{ .Conversation.Priority }}         - The priority level of the conversation
{{ .Conversation.UUID }}             - The unique identifier of the conversation

## Agent Variables
{{ .Agent.FirstName }}               - First name of the agent
{{ .Agent.LastName }}                - Last name of the agent
{{ .Agent.FullName }}                - Full name of the agent
{{ .Agent.Email }}                   - Email address of the agent

## Contact Variables
{{ .Contact.FirstName }}             - First name of the contact/customer
{{ .Contact.LastName }}              - Last name of the contact/customer
{{ .Contact.FullName }}              - Full name of the contact/customer
{{ .Contact.Email }}                 - Email address of the contact/customer

## Recipient Variables
{{ .Recipient.FirstName }}           - First name of the recipient
{{ .Recipient.LastName }}            - Last name of the recipient
{{ .Recipient.FullName }}            - Full name of the recipient
{{ .Recipient.Email }}               - Email address of the recipient

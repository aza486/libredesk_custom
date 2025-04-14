# Templating

Templating in outgoing emails allows you to personalize content by embedding dynamic expressions like {{ .Recipient.FullName }}. These expressions reference fields from the conversation, contact, and recipient objects, making it easy to generate customized messages. This section documents all available template variables and provides a basic example to demonstrate their usage.

## Email template expressions

There are several template expressions that can be used in the outgoing email template, They are written in the form `{{ .Recipient.FullName }}`.


| Variable                        | Value                                                  |
|---------------------------------|--------------------------------------------------------|
| {{ .Conversation.ReferenceNumber }} | The unique reference number of the conversation     |
| {{ .Conversation.Subject }}         | The subject of the conversation                     |
| {{ .Conversation.UUID }}           | The unique identifier of the conversation            |

### Contact fields
| Variable                     | Value                              |
|------------------------------|------------------------------------|
| {{ .Contact.FirstName }}     | First name of the contact/customer |
| {{ .Contact.LastName }}      | Last name of the contact/customer  |
| {{ .Contact.FullName }}      | Full name of the contact/customer  |
| {{ .Contact.Email }}         | Email address of the contact/customer |

### Recipient Variables
| Variable                       | Value                             |
|--------------------------------|-----------------------------------|
| {{ .Recipient.FirstName }}     | First name of the recipient       |
| {{ .Recipient.LastName }}      | Last name of the recipient        |
| {{ .Recipient.FullName }}      | Full name of the recipient        |
| {{ .Recipient.Email }}         | Email address of the recipient    |


### Example outgoing email template
```html
Dear {{ .Recipient.FirstName }}
{{ template "content" . }}
Best regards,
```
Here the `{{ template "content" . }}` is a placeholder for the content of the outgoing email. It will be replaced with the actual content of the email when it is sent. The `{{ .Recipient.FirstName }}` will be replaced with the first name of the recipient when the email is sent, this way you don't have to hardcode the name in the template. The same applies to the other variables.

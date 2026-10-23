package conversation

import "errors"

// ErrIncomingInboxBusy leaves the mail on the IMAP server for the next scan.
var ErrIncomingInboxBusy = errors.New("incoming email inbox is being imported; retry on next scan")

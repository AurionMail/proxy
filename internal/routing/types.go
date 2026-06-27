package routing

type ResolveRequest struct {
	Rcpt string `json:"rcpt"`
}

type ResolveResponse struct {
	IdentityEmail string `json:"identity_email"`
	PublicKey     string `json:"public_key"`
}

type MessageContext struct {
	From          string
	OriginalRcpts []string
	PublicKeys    map[string]string
}

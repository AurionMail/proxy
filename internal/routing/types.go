package routing

type ResolveRequest struct {
	Email string `json:"email"`
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

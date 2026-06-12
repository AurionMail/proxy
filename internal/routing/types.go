package routing

type ResolveRequest struct {
    Rcpt string `json:"rcpt"`
}

type ResolveResponse struct {
    IdentityEmail string   `json:"identity_email"`
    PublicKey     string   `json:"public_key"`
    Recipients    []string `json:"recipients"`
}


type MessageContext struct {
    From          string
    OriginalRcpts []string
    IdentityEmail string
    PublicKey     string
    FinalRcpts    []string
}

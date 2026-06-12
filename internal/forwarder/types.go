package forwarder

import "aurion/proxy/internal/routing"

type ForwardJob struct {
    Ctx     *routing.MessageContext
    Message []byte
    Attempt int
    MaxRetry int
}

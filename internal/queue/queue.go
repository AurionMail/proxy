package queue

import (
    "errors"

    "aurion/proxy/internal/forwarder"
    "aurion/proxy/internal/routing"
)

var (
    forwardQueue chan *forwarder.ForwardJob
    forwarderSMTP *forwarder.SMTPForwarder

    ErrQueueFull = errors.New("queue: full")
)

func InitQueue(size int, smtpAddr string) {
    forwardQueue = make(chan *forwarder.ForwardJob, size)
    forwarderSMTP = forwarder.NewSMTPForwarder(smtpAddr)
}

func Enqueue(ctx *routing.MessageContext, msg []byte) error {
    job := &forwarder.ForwardJob{
        Ctx:      ctx,
        Message:  msg,
        Attempt:  0,
        MaxRetry: 5,
    }

    select {
    case forwardQueue <- job:
        return nil
    default:
        return ErrQueueFull
    }
}

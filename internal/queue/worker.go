package queue

import (
	"log"
	"time"
)

func StartWorkers(n int) {
	for i := 0; i < n; i++ {
		go worker()
	}
}

func worker() {
	for job := range forwardQueue {
		err := forwarderSMTP.Forward(job)
		if err == nil {
			log.Printf("forward succeeded for %s", job.Ctx.OriginalRcpts[0])
			continue
		}

		job.Attempt++
		if job.Attempt > job.MaxRetry {
			log.Printf("forward permanently failed for %s: %v", job.Ctx.OriginalRcpts[0], err)
			continue
		}

		delay := time.Duration(1<<job.Attempt) * time.Second
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		time.AfterFunc(delay, func() {
			forwardQueue <- job
		})
	}
}

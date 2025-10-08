package scheduler

import (
	"context"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron *cron.Cron
}

func New() *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds()),
	}
}

func (s *Scheduler) AddJob(spec string, job func(context.Context) error) error {
	_, err := s.cron.AddFunc(spec, func() {
		ctx := context.Background()
		_ = job(ctx)
	})
	return err
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// Package cron provides a source that will invoke the function based on a
// cron schedule.
//
// The source must be configured with a string that contains a cronspec with
// the following structure:
//
// seconds(optional) minutes hours day-of-month month day-of-week
//
// The cronspec also supports descriptions such as @monthly and @weekly.
//
// The implementation is based on github.com/robfig/cron/v3.
package cron

import (
	"context"

	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/robfig/cron/v3"
)

type cronSource struct {
	cronspec string
}

const (
	parserOption = cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor
)

func (cs *cronSource) Serve(ctx context.Context, f fn.Fn) error {
	c := cron.New(cron.WithParser(cron.NewParser(parserOption)))

	_, err := c.AddFunc(cs.cronspec, func() {
		f.Invoke(ctx, map[string]interface{}{})
	})
	if err != nil {
		return err
	}

	c.Start()

	<-ctx.Done()
	c.Stop()

	return nil
}

func (cs *cronSource) ConfigureString(cronspec string) error {
	parser := cron.NewParser(parserOption)
	_, err := parser.Parse(cronspec)
	if err != nil {
		return err
	}

	cs.cronspec = cronspec
	return nil
}

func (cs *cronSource) RequiresConfig() bool {
	return true
}

// New creates and returns a cron source. It must be configured with the desired
// cronspec.
func New() run.Source {
	return &cronSource{}
}

package giu

import (
	"time"

	"github.com/robfig/cron/v3"
)

type CronParams struct {
	ConcurrentMode int
	Location       string
}

const (
	CRON_CONCURRENT_MODE_CONCURRENT int = iota
	CRON_CONCURRENT_MODE_SKIP
	CRON_CONCURRENT_MODE_DELAY
)

func NewCron(params CronParams) *cron.Cron {
	options := []cron.Option{}
	if params.Location != "" {
		tl, err := time.LoadLocation(params.Location)
		if err != nil {
			options = append(options, cron.WithLocation(tl))
		}
	}
	switch params.ConcurrentMode {
	case CRON_CONCURRENT_MODE_SKIP:
		options = append(options, cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	case CRON_CONCURRENT_MODE_DELAY:
		options = append(options, cron.WithChain(cron.DelayIfStillRunning(cron.DefaultLogger)))
	default:

	}
	return cron.New(options...)
}

type ScheduleParams struct {
	Tag         string
	Schedule    string
	WithSeconds bool
}

// NewSchedule creates a new Schedule.
func NewSchedule(params ScheduleParams) (cron.Schedule, error) {
	var s cron.Schedule
	var err error
	if params.WithSeconds {
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		s, err = parser.Parse(params.Schedule)
		if err != nil {
			return nil, err
		}
	} else {
		s, err = cron.ParseStandard(params.Schedule)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

type CronJob struct {
	Schedule cron.Schedule
	Func     func()
}

func (cj *CronJob) Run() {
	cj.Func()
}

func NewCronJob(params CronJob) cron.Job {
	return cron.FuncJob(params.Func)
}

func AddCronJob(c *cron.Cron, jobs []*CronJob) []cron.EntryID {
	ids := make([]cron.EntryID, 0)
	for _, job := range jobs {
		id := c.Schedule(job.Schedule, job)
		ids = append(ids, id)
	}
	return ids
}

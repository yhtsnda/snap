package schedule

import (
	"time"

	"github.com/intelsdilabs/pulse/control"
	"github.com/intelsdilabs/pulse/core"
	"github.com/intelsdilabs/pulse/core/cdata"
)

const (
	ScheduleActive ScheduleState = iota
	ScheduleEnded
	ScheduleError
)

var metricManager ManagesMetric

type Schedule interface {
	Wait(time.Time) ScheduleResponse
	Validate() error
}

type ScheduleState int

type ScheduleResponse interface {
	State() ScheduleState
	Error() error
	MissedIntervals() int
}

type ManagesMetric interface {
	SubscribeMetric(metric []string, ver int, cd *cdata.ConfigDataNode) (*cdata.ConfigDataNode, control.SubscriptionError)
	UnsubscribeMetric(metric []string, ver int)
}

type TaskErrors interface {
	Errors() []error
}

type taskErrors struct {
	errs []error
}

func (t *taskErrors) Errors() []error {
	return t.errs
}

type scheduler struct {
}

func (scheduler *scheduler) CreateTask(mts []core.MetricType, s Schedule, cd *cdata.ConfigDataNode) (*Task, TaskErrors) {
	te := &taskErrors{
		errs: make([]error, 0),
	}

	//validate Schedule
	if err := s.Validate(); err != nil {
		te.errs = append(te.errs, err)
		return nil, te
	}

	//subscribe to MT
	//if we encounter an error we will unwind successful subscriptions
	subscriptions := make([]*metricType, 0)
	for _, m := range mts {
		config, err := metricManager.SubscribeMetric(m.Namespace(), m.Version(), cd)
		if err == nil {
			mt := newMetricType(m, config)
			//mtc = append(mtc, mt)
			subscriptions = append(subscriptions, mt)
		} else {
			te.errs = append(te.errs, err.Errors()...)
		}
	}

	if len(te.errs) > 0 {
		//unwind successful subscriptions
		for _, sub := range subscriptions {
			metricManager.UnsubscribeMetric(sub.Namespace(), sub.Version())
		}
		return nil, te
	}

	task := NewTask(s, subscriptions)
	return task, nil
}

func New(m ManagesMetric) *scheduler {
	metricManager = m
	return &scheduler{}
}

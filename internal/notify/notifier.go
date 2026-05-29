package notify

import (
	"review-view/internal/model"
)

// Notifier 推送扫描结果给目标用户
type Notifier interface {
	Send(task *model.Task, project *model.Project, user *model.User) error
}

// Dispatcher 组合多个 Notifier，任一失败仅记录日志
type Dispatcher struct {
	notifiers []Notifier
}

func NewDispatcher(ns ...Notifier) *Dispatcher {
	return &Dispatcher{notifiers: ns}
}

func (d *Dispatcher) Send(task *model.Task, project *model.Project, user *model.User) error {
	if !user.NotifyEnabled {
		return nil
	}
	for _, n := range d.notifiers {
		_ = n.Send(task, project, user)
	}
	return nil
}

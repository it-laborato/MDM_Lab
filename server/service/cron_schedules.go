package service

import (
	"context"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

// TriggerCronSchedule attempts to trigger an ad-hoc run of the named cron schedule.
func (svc *Service) TriggerCronSchedule(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.CronSchedules{}, mdmlab.ActionWrite); err != nil {
		return err
	}
	return svc.cronSchedulesService.TriggerCronSchedule(name)
}

package trigger

import (
	"context"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/update"
)

// StartUpdates starts periodic updates
func StartUpdates(ctx context.Context, updater update.Updater, db data.Database, idPeriodMapping map[int]time.Duration, onError func(err error)) (forceUpdate func()) {
	errors := make(chan error)
	var smallestPeriod time.Duration
	for id, record := range db.SelectAll() {
		if record.Settings.IPMethod != constants.PROVIDER &&
			smallestPeriod != 0 &&
			idPeriodMapping[id] < smallestPeriod {
			smallestPeriod = idPeriodMapping[id]
		}
	}
	triggerUpdatePublicIP := make(chan struct{})
	if smallestPeriod > 0 { // at least one external IP update
		go func() {
			ticker := time.NewTicker(smallestPeriod)
			defer ticker.Stop()
			for {
				select {
				case <-triggerUpdatePublicIP:
					if err := updater.UpdatePublicIP(); err != nil {
						errors <- err
					}
				case <-ticker.C:
					if err := updater.UpdatePublicIP(); err != nil {
						errors <- err
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	triggers := make([]chan struct{}, len(idPeriodMapping))
	for id, period := range idPeriodMapping {
		triggers[id] = make(chan struct{})
		go func(id int, period time.Duration) {
			ticker := time.NewTicker(period)
			defer ticker.Stop()
			for {
				select {
				case <-triggers[id]:
					if err := updater.Update(id); err != nil {
						errors <- err
					}
				case <-ticker.C:
					if err := updater.Update(id); err != nil {
						errors <- err
					}
				case <-ctx.Done():
					return
				}
			}
		}(id, period)
	}
	// collects errors only
	go func() {
		for {
			select {
			case err := <-errors:
				onError(err)
			case <-ctx.Done():
				return
			}
		}
	}()
	updateRecordsFn := func() {
		for i := range triggers { // TODO lock something for them to wait for the public ip update
			triggers[i] <- struct{}{}
		}
	}
	if smallestPeriod > 0 {
		return func() {
			triggerUpdatePublicIP <- struct{}{}
			updateRecordsFn()
		}
	}
	return updateRecordsFn
}

package rt

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// watcher holds a set of schedules and "watches"
// them for new jobs to send to the scheduler
type watcher struct {
	schedules    map[string]Schedule
	scheduleFunc func(*Job) *Result

	lock      sync.RWMutex
	startOnce sync.Once
}

func newWatcher(scheduleFunc func(*Job) *Result) *watcher {
	w := &watcher{
		schedules:    map[string]Schedule{},
		scheduleFunc: scheduleFunc,
		lock:         sync.RWMutex{},
		startOnce:    sync.Once{},
	}

	return w
}

func (w *watcher) watch(sched Schedule) {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.schedules[uuid.New().String()] = sched

	// we only want to start the ticker if something is actually set up
	// to be scheduled, so we put it behind a sync.Once
	w.startOnce.Do(func() {
		go func() {
			ticker := time.Tick(time.Second)

			// loop forever and check each schedule for new jobs
			// repeating every second
			for {
				remove := []string{}

				w.lock.RLock()
				for uuid, s := range w.schedules {
					if s.Done() {
						// set the schedule to be removed if it's done
						remove = append(remove, uuid)
					} else {
						if job := s.Check(); job != nil {
							// schedule the job and discard the result
							w.scheduleFunc(job).Discard()
						}
					}
				}
				w.lock.RUnlock()

				w.lock.Lock()
				for _, uuid := range remove {
					delete(w.schedules, uuid)
				}
				w.lock.Unlock()

				<-ticker
			}
		}()
	})
}

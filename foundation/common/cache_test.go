package common

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLoadingCache_Get(t *testing.T) {
	type args struct {
		putKey     string
		getKey     string
		failLoader bool
	}

	type test struct {
		name string
		args args
		want Value[int]
	}

	tests := []test{
		{
			name: "Test LoadingCache Get",
			args: args{
				putKey: "count",
				getKey: "count",
			},
			want: Value[int]{
				State: EntryReady,
				Value: 1,
				Error: nil,
			},
		},
		{
			name: "Test LoadingCache Get invalid key",
			args: args{
				putKey: "count",
				getKey: "invalid",
			},
			want: Value[int]{
				State: EntryError,
				Value: 0,
				Error: ErrNotExists,
			},
		},
		{
			name: "Test LoadingCache failed load operation",
			args: args{
				putKey:     "count",
				getKey:     "count",
				failLoader: true,
			},
			want: Value[int]{
				State: EntryError,
				Value: 0,
				Error: ErrNotExists,
			},
		},
	}

	for _, tc := range tests {
		// instructs the loader function to complete
		complete := make(chan struct{}, 1)
		// signals the test to proceed
		step := make(chan struct{}, 1)

		cache := NewLoadingCache[int](NewTreeStore[int]())
		count := 0

		cache.Put(tc.args.putKey, func() (int, error) {
			<-complete
			count += 1
			if tc.args.failLoader {
				// value should not be overwritten on failure to load
				return -1, tc.want.Error
			}

			return count, nil

		})

		var value Value[int]
		go func() {
			step <- struct{}{}
			value = cache.Get(tc.args.getKey)

			assert.Equal(t, tc.want.State, value.State)
			assert.Equal(t, tc.want.Value, value.Value)
			assert.ErrorIs(t, value.Error, tc.want.Error)

			step <- struct{}{}
		}()

		<-step

		// Ensure unrelated keys aren't blocked by loading func
		other := uuid.NewString()
		cache.Put(other, func() (int, error) {
			return -1, nil
		})
		assert.Equal(t, -1, cache.Get(other).Value)

		// cache.Get should be blocked awaiting a response
		assert.Zero(t, value.Value)

		// complete loader func
		complete <- struct{}{}
		close(complete)

		<-step

		// should not invoke loader again; deadlock detector will panic since there are no more references to complete
		value = cache.Get(tc.args.getKey)

		assert.Equal(t, tc.want.Value, value.Value)
		assert.Equal(t, tc.want.State, value.State)
		assert.ErrorIs(t, value.Error, tc.want.Error)
	}
}

func TestLoadingCache_Refresh(t *testing.T) {
	type args struct {
		putKey     string
		getKey     string
		failLoader bool
	}

	type test struct {
		name string
		args args
		want Value[int]
	}

	tests := []test{
		{
			name: "Test LoadingCache Get",
			args: args{
				putKey: "count",
				getKey: "count",
			},
			want: Value[int]{
				State: EntryReady,
				Value: 2,
				Error: nil,
			},
		},
		{
			name: "Test LoadingCache Get invalid key",
			args: args{
				putKey: "count",
				getKey: "invalid",
			},
			want: Value[int]{
				State: EntryError,
				Value: 0,
				Error: ErrNotExists,
			},
		},
		{
			name: "Test LoadingCache failed load operation",
			args: args{
				putKey:     "count",
				getKey:     "count",
				failLoader: true,
			},
			want: Value[int]{
				State: EntryError,
				Value: 1,
				Error: ErrNotExists,
			},
		},
	}

	for _, tc := range tests {
		// instructs the loader function to complete
		complete := make(chan struct{}, 1)
		// signals the test to proceed
		step := make(chan struct{}, 1)

		cache := NewLoadingCache[int](NewTreeStore[int]())

		count := -2
		cache.Put(tc.args.putKey, func() (int, error) {
			if count == -2 {
				count = 1
				return count, nil
			}

			<-complete

			count += 1
			if tc.args.failLoader {
				// value should not be overwritten on failure to load
				return -1, tc.want.Error
			}

			return count, nil

		})

		// value initialization should always succeed
		value := cache.Get(tc.args.putKey)

		go func() {
			step <- struct{}{}
			_ = cache.Refresh(tc.args.putKey)
		}()

		<-step

		go func() {
			step <- struct{}{}
			value = cache.Get(tc.args.getKey)
		}()

		<-step
		close(step)

		complete <- struct{}{}
		close(complete)

		value = cache.Get(tc.args.getKey)
		assert.Equal(t, tc.want.State, value.State)
		assert.Equal(t, tc.want.Value, value.Value)
		assert.ErrorIs(t, value.Error, tc.want.Error)
	}
}

func TestLoadingCache_Cancel(t *testing.T) {
	type args struct {
		putKey     string
		getKey     string
		failLoader bool
	}

	type test struct {
		name string
		args args
		want Value[int]
	}

	tests := []test{
		{
			name: "Test LoadingCache Cancel",
			args: args{
				putKey: "count",
				getKey: "count",
			},
			want: Value[int]{
				State: EntryCanceled,
				Value: 0,
				Error: ErrCanceled,
			},
		},
		{
			name: "Test LoadingCache Cancel invalid key",
			args: args{
				putKey: "count",
				getKey: "invalid",
			},
			want: Value[int]{
				State: EntryError,
				Value: 0,
				Error: ErrNotExists,
			},
		},
		{
			name: "Test LoadingCache failed load operation",
			args: args{
				putKey:     "count",
				getKey:     "count",
				failLoader: true,
			},
			want: Value[int]{
				State: EntryCanceled,
				Value: 0,
				Error: ErrCanceled,
			},
		},
	}

	for _, tc := range tests {
		result := make(chan Value[int], 1)
		// instructs the loader function to complete
		complete := make(chan struct{}, 1)
		// signals the test to proceed
		step := make(chan struct{}, 1)

		cache := NewLoadingCache[int](NewTreeStore[int]())

		cache.Put(tc.args.putKey, func() (int, error) {
			<-complete
			if tc.args.failLoader {
				// value should not be overwritten on failure to load
				return -1, tc.want.Error
			}

			return 2, nil
		})

		go func() {
			step <- struct{}{}
			result <- cache.Get(tc.args.getKey)
		}()

		// cache.Get() parked, load func still blocked
		<-step

		// wake up cache.Get(key) watchers, cancel update
		cache.Cancel(tc.args.getKey)
		value := <-result

		close(step)
		// close unblocks receiver
		close(complete)
		close(result)

		assert.Equal(t, tc.want.State, value.State)
		assert.Equal(t, tc.want.Value, value.Value)
		assert.ErrorIs(t, value.Error, tc.want.Error)
	}
}

func TestLoadingCache_Drop(t *testing.T) {
	type args struct {
		putKey     string
		getKey     string
		failLoader bool
	}

	type test struct {
		name string
		args args
		want Value[int]
	}

	tests := []test{
		{
			name: "Test LoadingCache Drop",
			args: args{
				putKey: "count",
				getKey: "count",
			},
			want: Value[int]{
				State: EntryCanceled,
				Value: 0,
				Error: nil,
			},
		},
		{
			name: "Test LoadingCache Drop, invalid key",
			args: args{
				putKey: "count",
				getKey: "invalid",
			},
			want: Value[int]{
				State: EntryError,
				Value: 0,
				Error: ErrNotExists,
			},
		},
		{
			name: "Test LoadingCache Drop, failed load operation",
			args: args{
				putKey:     "count",
				getKey:     "count",
				failLoader: true,
			},
			want: Value[int]{
				State: EntryCanceled,
				Value: 0,
				Error: nil,
			},
		},
	}

	for _, tc := range tests {
		// signals async cache.Get() has returned
		result := make(chan Value[int], 1)
		// instructs the loader function to complete
		complete := make(chan struct{}, 1)
		// signals the test to proceed
		step := make(chan struct{}, 1)

		cache := NewLoadingCache[int](NewTreeStore[int]())

		cache.Put(tc.args.putKey, func() (int, error) {
			<-complete
			if tc.args.failLoader {
				// value should not be overwritten on failure to load
				return -1, tc.want.Error
			}

			return 2, nil
		})

		go func() {
			step <- struct{}{}
			result <- cache.Get(tc.args.getKey)
		}()

		// cache.Get() scheduled, load func still blocked
		<-step
		close(step)

		// wake up cache.Get(key) watchers, cancel update
		cache.Drop(tc.args.getKey)

		// pending request should be canceled
		value := <-result
		close(result)

		assert.Equal(t, tc.want.State, value.State)
		assert.Equal(t, tc.want.Value, value.Value)
		assert.ErrorIs(t, value.Error, tc.want.Error)

		// unblock loader func
		complete <- struct{}{}
		close(complete)

		// subsequent requests should fail because key is no longer present
		value = cache.Get(tc.args.getKey)
		assert.Equal(t, EntryError, value.State)
		assert.Equal(t, 0, value.Value)
		assert.ErrorIs(t, value.Error, ErrNotExists)
	}
}

func TestLoadingCache_DuplicateRegistration(t *testing.T) {
	wg := sync.WaitGroup{}
	// synchronized collection to capture response
	errs := make(chan error, 10)

	cache := NewLoadingCache[int](NewTreeStore[int]())

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			errs <- cache.Put("key", func() (int, error) {
				return 1, nil
			})
			wg.Done()
		}()
	}

	// wait for all routines to complete
	wg.Wait()
	close(errs)

	errCount := 0
	for err := range errs {
		// record presence of target error
		if IsError(err, ErrExists) {
			errCount += 1
		}
	}

	// ensure all but one put failed with the correct error type
	assert.Equal(t, 9, errCount)
}

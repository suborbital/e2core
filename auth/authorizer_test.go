package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/suborbital/e2core/common"
)

type CompareAssertionFunc[T any] func(t *testing.T, actual T) bool

func TestAuthorizerCache_ConcurrentRequests(t *testing.T) {
	type args struct {
		token string
	}

	type test struct {
		name       string
		args       args
		handler    http.HandlerFunc
		assertOpts CompareAssertionFunc[uint64]
		assertErr  assert.ErrorAssertionFunc
	}

	tests := []test{
		{
			name: "Ensure duplicate requests are pipelined",
			args: args{
				token: "token",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(&TenantInfo{
					AuthorizedParty: "tester",
					Organization:    "acct",
					Environment:     "env",
					Tenant:          "123",
				})
			},
			assertOpts: func(t *testing.T, actual uint64) bool {
				return assert.Equal(t, uint64(1), actual)
			},
			assertErr: func(_ assert.TestingT, err error, _ ...interface{}) bool {
				return err == nil
			},
		},
		{
			name: "Ensure non-credentialed requests are not dispatched",
			args: args{
				token: "",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(&TenantInfo{})
			},
			assertOpts: func(t *testing.T, actual uint64) bool {
				return assert.Equal(t, uint64(0), actual)
			},
			assertErr: func(_ assert.TestingT, err error, _ ...interface{}) bool {
				return common.IsError(err, common.ErrAccess)
			},
		},
		{
			name: "Ensure denied requests return ErrAccess",
			args: args{
				token: "token",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			// Although each request is effectively sent at the same time this is ultimately up to the scheduler.
			// We allow for up to 10% of the requests to go through to avoid flakiness in CI environments.
			assertOpts: func(t *testing.T, actual uint64) bool {
				return assert.LessOrEqual(t, actual, uint64(100))
			},
			assertErr: func(_ assert.TestingT, err error, _ ...interface{}) bool {
				return common.IsError(err, common.ErrAccess) || common.IsError(err, common.ErrCanceled)
			},
		},
	}

	for _, tc := range tests {
		var opts uint64 = 0
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddUint64(&opts, 1)
			tc.handler(w, r)
		}))

		authorizer := &AuthzClient{
			httpClient: svr.Client(),
			location:   svr.URL + "/api/v2/tenant/%s",
			cache:      newAuthorizationCache(common.StableTime(time.Now()), 10*time.Minute),
		}

		var err error
		wg := sync.WaitGroup{}
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				_, err = authorizer.Authorize(NewAccessToken(tc.args.token), "env.app", "namespace", "mod")
				wg.Done()
			}()
		}
		wg.Wait()

		svr.Close()

		tc.assertErr(t, err)
		tc.assertOpts(t, opts)
	}
}

func TestAuthorizerCache(t *testing.T) {
	type args struct {
		token      string
		identifier string
		namespace  string
		mod        string
	}

	type test struct {
		name       string
		args       []args
		handler    http.HandlerFunc
		assertOpts CompareAssertionFunc[uint64]
		wantErr    error
	}

	tests := []test{
		{
			name: "Ensure each unique request is dispatched",
			args: []args{
				{
					token:      "token",
					identifier: "env.123",
					namespace:  "default",
					mod:        "mod",
				},
				{
					token:      "token2",
					identifier: "env.123",
					namespace:  "default",
					mod:        "mod",
				},
				{
					token:      "token",
					identifier: "env.abc",
					namespace:  "default",
					mod:        "mod",
				},
				{
					token:      "token",
					identifier: "env.123",
					namespace:  "not-default",
					mod:        "mod",
				},
				{
					token:      "token",
					identifier: "env.123",
					namespace:  "default",
					mod:        "mod-2",
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				var authzReq *TenantInfo
				_ = json.NewDecoder(r.Body).Decode(&authzReq)
				_ = json.NewEncoder(w).Encode(&TenantInfo{
					AuthorizedParty: "tester",
					Organization:    "acct",
					Environment:     authzReq.Environment,
					Tenant:          authzReq.Tenant,
				})
			},
			assertOpts: func(t *testing.T, actual uint64) bool {
				return assert.Equal(t, uint64(5), actual)
			},
			wantErr: nil,
		},
		{
			name: "Ensure failed requests aren't cached",
			args: []args{
				{
					token:      "token",
					identifier: "env.123",
					namespace:  "default",
					mod:        "mod",
				},
				{
					token:      "token",
					identifier: "env.123",
					namespace:  "default",
					mod:        "mod",
				},
				{
					token:      "token",
					identifier: "env.123",
					namespace:  "default",
					mod:        "mod",
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			assertOpts: func(t *testing.T, actual uint64) bool {
				return assert.Equal(t, uint64(3), actual)
			},
			wantErr: common.ErrAccess,
		},
		{
			name: "Ensure success after failure",
			args: []args{
				{
					token:      "token",
					identifier: "env.123",
					namespace:  "default",
					mod:        "mod",
				},
				{
					token:      "denied",
					identifier: "env.123",
					namespace:  "default",
					mod:        "mod",
				},
				{
					token:      "token",
					identifier: "env.123",
					namespace:  "default",
					mod:        "mod",
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				token := ExtractAccessToken(r.Header)
				if token.Value() == "denied" {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusOK)
					var authzReq *TenantInfo
					_ = json.NewDecoder(r.Body).Decode(&authzReq)
					_ = json.NewEncoder(w).Encode(&TenantInfo{
						AuthorizedParty: "tester",
						Organization:    "acct",
						Environment:     authzReq.Environment,
						Tenant:          authzReq.Tenant,
					})
				}
			},
			assertOpts: func(t *testing.T, actual uint64) bool {
				return assert.Equal(t, uint64(2), actual)
			},
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		var opts uint64 = 0
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddUint64(&opts, 1)
			tc.handler(w, r)
		}))

		authorizer := &AuthzClient{
			httpClient: svr.Client(),
			location:   svr.URL,
			cache:      newAuthorizationCache(common.StableTime(time.Now()), 10*time.Minute),
		}

		var err error
		for _, arg := range tc.args {
			_, err = authorizer.Authorize(NewAccessToken(arg.token), arg.identifier, arg.namespace, arg.mod)
		}

		svr.Close()

		assert.ErrorIs(t, err, tc.wantErr)
		assert.True(t, tc.assertOpts(t, opts))
	}
}

func TestAuthorizerCache_ExpiringEntry(t *testing.T) {
	type args struct {
		ttl time.Duration
	}

	type test struct {
		name       string
		args       args
		handler    http.HandlerFunc
		assertOpts CompareAssertionFunc[uint64]
		wantErr    error
	}

	tests := []test{
		{
			name: "Ensure expired tokens are refreshed",
			args: args{ttl: 1 * time.Second},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(&TenantInfo{
					AuthorizedParty: "tester",
					Organization:    "acct",
					Environment:     "env",
					Tenant:          "123",
				})
			},
			assertOpts: func(t *testing.T, actual uint64) bool {
				return assert.Equal(t, uint64(2), actual)
			},
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		var opts uint64 = 0
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddUint64(&opts, 1)
			tc.handler(w, r)
		}))

		clock := common.StableTime(time.Now())

		authzCache := newAuthorizationCache(clock, 10*time.Minute)
		authzCache.clock = clock

		authorizer := &AuthzClient{
			httpClient: svr.Client(),
			location:   svr.URL,
			cache:      authzCache,
		}

		// 1 auth op
		_, err := authorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")
		_, err = authorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")
		_, err = authorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")

		clock.Tick(authzCache.ttl + 1*time.Second)

		// 2 auth op
		_, err = authorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")
		_, err = authorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")
		_, err = authorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")

		svr.Close()

		assert.ErrorIs(t, err, tc.wantErr)
		assert.True(t, tc.assertOpts(t, opts))
	}
}

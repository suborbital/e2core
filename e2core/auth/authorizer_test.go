package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suborbital/e2core/foundation/common"
)

type CompareAssertionFunc[T any] func(t *testing.T, actual T) bool

func TestAuthorizerGoCache_ConcurrentRequests(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		handler    http.HandlerFunc
		assertOpts CompareAssertionFunc[uint64]
		assertErr  assert.ErrorAssertionFunc
	}{
		{
			name:  "Ensure duplicate requests are pipelined",
			token: "token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(&TenantInfo{
					AuthorizedParty: "tester",
					Environment:     "env",
					ID:              "tnt",
					Name:            "fnName",
				})
			},
			assertOpts: func(t *testing.T, actual uint64) bool {
				return assert.Equalf(t, uint64(1), actual, "expected %d, got %d", 1, actual)
			},
			assertErr: func(_ assert.TestingT, err error, _ ...interface{}) bool {
				return err == nil
			},
		},
		{
			name:  "Ensure non-credentialed requests are not dispatched",
			token: "",
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
			name:  "Ensure denied requests return ErrAccess",
			token: "token",
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var opts uint64 = 0
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddUint64(&opts, 1)
				test.handler(w, r)
			}))

			apiAuthorizer := &APIAuthorizer{
				httpClient: svr.Client(),
				location:   svr.URL + "/environment/v1/tenant/%s",
			}

			// NewGoCacheAuthorizer always returns nil error.
			goCacheAuthorizer, err := NewBigCacheAuthorizer(apiAuthorizer, DefaultConfig)
			require.NoError(t, err, "initialising new big cache authorizer")

			wg := sync.WaitGroup{}
			for i := 0; i < 30; i++ {
				wg.Add(1)
				go func() {
					_, err = goCacheAuthorizer.Authorize(NewAccessToken(test.token), "env.app", "namespace", "mod")
					wg.Done()
				}()
			}
			wg.Wait()

			svr.Close()

			test.assertErr(t, err)
			test.assertOpts(t, opts)
		})

	}
}

func TestAuthorizerGoCache(t *testing.T) {
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
				ident := r.RequestURI[strings.LastIndex(r.RequestURI, "/")+1:]
				env, tenant, _ := strings.Cut(ident, ".")
				_ = json.NewEncoder(w).Encode(&TenantInfo{
					AuthorizedParty: "tester",
					Environment:     env,
					ID:              tenant,
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
			wantErr: ErrUnauthorized,
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
					ident := r.RequestURI[strings.LastIndex(r.RequestURI, "/")+1:]
					env, tenant, _ := strings.Cut(ident, ".")
					_ = json.NewEncoder(w).Encode(&TenantInfo{
						AuthorizedParty: "tester",
						Environment:     env,
						ID:              tenant,
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
		t.Run(tc.name, func(t *testing.T) {
			var opts uint64 = 0
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddUint64(&opts, 1)
				tc.handler(w, r)
			}))

			apiAuthorizer := &APIAuthorizer{
				httpClient: svr.Client(),
				location:   svr.URL + "/api/v2/tenant/%s",
			}

			goCacheAuthorizer, _ := NewGoCacheAuthorizer(apiAuthorizer)

			var err error
			for _, arg := range tc.args {
				_, err = goCacheAuthorizer.Authorize(NewAccessToken(arg.token), arg.identifier, arg.namespace, arg.mod)
			}

			svr.Close()

			assert.ErrorIs(t, err, tc.wantErr)
			assert.True(t, tc.assertOpts(t, opts))
		})

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
					Environment:     "env",
					ID:              "123",
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

		authorizer := &APIAuthorizer{
			httpClient: svr.Client(),
			location:   svr.URL + "/environment/v1/tenant/%s",
		}

		bigCacheAuthorizer, err := NewBigCacheAuthorizer(authorizer, bigcache.Config{
			Shards:             4,
			LifeWindow:         3 * time.Second,
			CleanWindow:        time.Second,
			MaxEntriesInWindow: 200,
			MaxEntrySize:       500,
		})

		// 1 auth op
		_, err = bigCacheAuthorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")
		_, err = bigCacheAuthorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")
		_, err = bigCacheAuthorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")

		time.Sleep(4 * time.Second)

		// 2 auth op
		_, err = bigCacheAuthorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")
		_, err = bigCacheAuthorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")
		_, err = bigCacheAuthorizer.Authorize(NewAccessToken("token"), "env.app", "namespace", "mod")

		svr.Close()

		assert.ErrorIs(t, err, tc.wantErr)
		assert.True(t, tc.assertOpts(t, opts))
	}
}

package capabilities

import "testing"

func TestDefaultCache(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		Rules: CacheRules{
			AllowSet:    true,
			AllowGet:    true,
			AllowDelete: true,
		},
	}

	cache := SetupCache(config)

	t.Run("set enabled", func(t *testing.T) {
		if err := cache.Set("foo", []byte("bar"), 0); err != nil {
			t.Error("error occurred, should not have")
		}
	})

	t.Run("get enabled", func(t *testing.T) {
		val, err := cache.Get("foo")
		if err != nil {
			t.Error("error occurred, should not have")
		}

		if string(val) != "bar" {
			t.Error("got incorrect value, expected 'bar': " + string(val))
		}
	})

	t.Run("delete enabled", func(t *testing.T) {
		if err := cache.Delete("foo"); err != nil {
			t.Error("error occurred, should not have")
		}
	})
}

func TestDisabledCache(t *testing.T) {
	config := CacheConfig{
		Enabled: false,
		Rules: CacheRules{
			AllowSet:    true,
			AllowGet:    true,
			AllowDelete: true,
		},
	}

	cache := SetupCache(config)

	t.Run("set disabled", func(t *testing.T) {
		if err := cache.Set("foo", []byte("bar"), 0); err == nil {
			t.Error("error did not occur, should have")
		}
	})

	t.Run("get disabled", func(t *testing.T) {
		val, err := cache.Get("foo")
		if err == nil {
			t.Error("error did not occur, should have")
		}

		if string(val) != "" {
			t.Error("got incorrect value, expected '': " + string(val))
		}
	})

	t.Run("delete disabled", func(t *testing.T) {
		if err := cache.Delete("foo"); err == nil {
			t.Error("error did not occur, should have")
		}
	})
}

func TestDisabledGet(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		Rules: CacheRules{
			AllowSet:    true,
			AllowGet:    false,
			AllowDelete: true,
		},
	}

	cache := SetupCache(config)

	t.Run("set enabled", func(t *testing.T) {
		if err := cache.Set("foo", []byte("bar"), 0); err != nil {
			t.Error("error occurred, should not have")
		}
	})

	t.Run("get disabled", func(t *testing.T) {
		val, err := cache.Get("foo")
		if err == nil {
			t.Error("error did not occur, should have")
		}

		if string(val) != "" {
			t.Error("got incorrect value, expected '': " + string(val))
		}
	})

	t.Run("delete enabled", func(t *testing.T) {
		if err := cache.Delete("foo"); err != nil {
			t.Error("error occurred, should not have")
		}
	})
}

func TestDisabledSet(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		Rules: CacheRules{
			AllowSet:    false,
			AllowGet:    true,
			AllowDelete: true,
		},
	}

	cache := SetupCache(config)

	t.Run("set disabled", func(t *testing.T) {
		if err := cache.Set("foo", []byte("bar"), 0); err == nil {
			t.Error("error did not occur, should have")
		}
	})

	t.Run("get enabled no value", func(t *testing.T) {
		val, err := cache.Get("foo")
		if err == nil {
			t.Error("error did not occur, should have")
		}

		if string(val) != "" {
			t.Error("got incorrect value, expected '': " + string(val))
		}
	})

	t.Run("delete enabled no value", func(t *testing.T) {
		if err := cache.Delete("foo"); err != nil {
			t.Error("error occurred, should not have")
		}
	})
}

func TestDisabledDelete(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		Rules: CacheRules{
			AllowSet:    true,
			AllowGet:    true,
			AllowDelete: false,
		},
	}

	cache := SetupCache(config)

	t.Run("set enabled", func(t *testing.T) {
		if err := cache.Set("foo", []byte("bar"), 0); err != nil {
			t.Error("error occurred, should not have")
		}
	})

	t.Run("delete disabled", func(t *testing.T) {
		if err := cache.Delete("foo"); err == nil {
			t.Error("error did not occur, should have")
		}
	})

	t.Run("get enabled", func(t *testing.T) {
		val, err := cache.Get("foo")
		if err != nil {
			t.Error("error occurred, should not have")
		}

		if string(val) != "bar" {
			t.Error("got incorrect value, expected 'bar': " + string(val))
		}
	})
}

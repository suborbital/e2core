package tunnel

import (
	"fmt"
	"testing"
)

func TestBalancer(t *testing.T) {
	balancer := NewBalancer()

	balancer.Add("asdf")
	balancer.Add("hjkl")
	balancer.Add("qwer")

	order := []string{"asdf", "hjkl", "qwer"}

	for i := 0; i < 12; i++ {
		next := balancer.Next()

		shouldBe := order[i%3]

		if next != shouldBe {
			t.Error(fmt.Errorf("expected %s, got %s", shouldBe, next))
		}
	}

	balancer.Add("uiop")
	balancer.Add("zxcv")

	order = append(order, "uiop", "zxcv")

	for i := 0; i < 15; i++ {
		next := balancer.Next()

		shouldBe := order[i%5]

		if next != shouldBe {
			t.Error(fmt.Errorf("expected %s, got %s", shouldBe, next))
		}
	}

	balancer.Remove("asdf")
	balancer.Remove("hjkl")

	order = order[2:]

	for i := 0; i < 12; i++ {
		next := balancer.Next()

		shouldBe := order[i%3]

		if next != shouldBe {
			t.Error(fmt.Errorf("expected %s, got %s", shouldBe, next))
		}
	}
}

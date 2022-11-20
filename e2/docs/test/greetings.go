package greetings

import "fmt"

func Hello(who string) {
	if who == "Boo!" {
		fmt.Printf("Ahhh!")
	} else {
		fmt.Printf("Hello, %s!\n", who)
	}
}

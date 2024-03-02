package main

type User struct{}

func main() {
	var provider func(some string) (User, error)

	processUser(provider)
}

func processUser(userProvider func(some string) (User, error)) {
	userProvider("123")
}

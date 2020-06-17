package protocol

func assert(condition bool, message string) {
	if condition == false {
		panic("Assertion failed: " + message)
	}
}

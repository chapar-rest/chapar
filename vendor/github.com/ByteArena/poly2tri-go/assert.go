package poly2tri

func Assert(condition bool, message string) {
	if !condition {
		if message == "" {
			message = "Assert failed"
		}

		panic(message)
	}
}

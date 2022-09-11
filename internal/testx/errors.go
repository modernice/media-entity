package testx

func PanicOn(errs <-chan error) {
	for err := range errs {
		if err != nil {
			panic(err)
		}
	}
}

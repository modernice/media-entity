package testx

import (
	"context"
	"errors"
)

func PanicOn(errs <-chan error) {
	for err := range errs {
		if err != nil && !errors.Is(err, context.Canceled) {
			panic(err)
		}
	}
}

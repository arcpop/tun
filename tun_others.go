// +build !linux

package tun

import (
	"errors"
)

func newTun(name string) (TunInterface, error) {
    return nil, errors.New("Not implemented for this OS")
}
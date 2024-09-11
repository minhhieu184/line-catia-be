package services

import (
	"github.com/mroth/weightedrand/v2"
)

type ServiceGacha[T any] struct {
	chooser *weightedrand.Chooser[T, int]
}

func NewServiceGacha[T any](choices []weightedrand.Choice[T, int]) (*ServiceGacha[T], error) {
	chooser, err := weightedrand.NewChooser(choices...)
	if err != nil {
		return nil, err
	}

	return &ServiceGacha[T]{chooser}, nil
}

func (service *ServiceGacha[T]) Pick() T {
	return service.chooser.Pick()
}

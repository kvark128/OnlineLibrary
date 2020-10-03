package config

import (
	"fmt"
)

type Services []Service

func (s Services) SetService(service Service) {
	s = append(s, service)
	s.SetCurrentService(len(s) - 1)
	Conf.Services = s
}

func (s Services) Service(index int) Service {
	return s[index]
}

func (s Services) RemoveService(index int) {
	copy(s[index:], s[index+1:])
	Conf.Services = Conf.Services[:len(Conf.Services)-1]
}

func (s Services) SetCurrentService(index int) {
	s[0], s[index] = s[index], s[0]
}

func (s Services) CurrentService() (Service, int, error) {
	if len(s) > 0 {
		return s[0], 0, nil
	}
	return Service{}, 0, fmt.Errorf("services list is empty")
}

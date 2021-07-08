package manager

import (
	"fmt"

	"github.com/kvark128/OnlineLibrary/internal/config"
	daisy "github.com/kvark128/daisyonline"
)

type Library struct {
	*daisy.Client
	service           *config.Service
	serviceAttributes *daisy.ServiceAttributes
}

func NewLibrary(service *config.Service) (*Library, error) {
	client := daisy.NewClient(service.URL, config.HTTPTimeout)
	success, err := client.LogOn(service.Credentials.Username, service.Credentials.Password)
	if err != nil {
		return nil, fmt.Errorf("logOn operation: %w", err)
	}

	if !success {
		return nil, fmt.Errorf("logOn operation returned false")
	}

	serviceAttributes, err := client.GetServiceAttributes()
	if err != nil {
		return nil, fmt.Errorf("getServiceAttributes operation: %w", err)
	}

	success, err = client.SetReadingSystemAttributes(&config.ReadingSystemAttributes)
	if err != nil {
		return nil, fmt.Errorf("setReadingSystemAttributes operation: %w", err)
	}

	if !success {
		return nil, fmt.Errorf("setReadingSystemAttributes operation returned false")
	}

	library := &Library{
		Client:            client,
		service:           service,
		serviceAttributes: serviceAttributes,
	}

	return library, nil
}

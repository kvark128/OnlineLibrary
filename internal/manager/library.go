package manager

import (
	"fmt"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	daisy "github.com/kvark128/daisyonline"
)

type Library struct {
	*daisy.Client
	service           *config.Service
	serviceAttributes *daisy.ServiceAttributes
}

func NewLibrary(service *config.Service) (*Library, error) {
	client := daisy.NewClient(service.URL, time.Second*10)
	if ok, err := client.LogOn(service.Credentials.Username, service.Credentials.Password); err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("The LogOn operation returned false")
	}

	serviceAttributes, err := client.GetServiceAttributes()
	if err != nil {
		return nil, err
	}

	_, err = client.SetReadingSystemAttributes(&config.ReadingSystemAttributes)
	if err != nil {
		return nil, err
	}

	library := &Library{
		Client:            client,
		service:           service,
		serviceAttributes: serviceAttributes,
	}

	return library, nil
}

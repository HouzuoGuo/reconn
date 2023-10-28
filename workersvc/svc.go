package workersvc

import (
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// Config has the configuration of the GPU worker service itself and its external dependencies.
type Config struct {
	// ServiceBusQueue is the name of azure service bus queue.
	ServiceBusQueue string
	// ServiceBusClient is the azure service bus client.
	ServiceBusClient *azservicebus.Client
	// ServiceBusSender is the azure service bus receiver client.
	ServiceBusReceiver *azservicebus.Receiver
}

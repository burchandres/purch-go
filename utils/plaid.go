package utils

import (
	"github.com/plaid/plaid-go/v40/plaid"
	
	"sync"
)

var (
	plaidClient *plaid.APIClient
	plaidOnce   sync.Once
)


func GetPlaidClient() *plaid.APIClient {
	plaidOnce.Do(func() {
		config := GetConfig()
		// setup plaid configuration
		configuration := plaid.NewConfiguration()
		configuration.AddDefaultHeader("PLAID-CLIENT-ID", config.PlaidClientID)
		configuration.AddDefaultHeader("PLAID-SECRET", config.PlaidSecret)
		var env plaid.Environment
		if config.PlaidEnv == "" || config.PlaidEnv == "sandbox" {
			env = plaid.Sandbox
		} else {
			env = plaid.Production
		}
		configuration.UseEnvironment(env)
		// return plaid api client to be used
		plaidClient = plaid.NewAPIClient(configuration)
	})
	
	return plaidClient
}
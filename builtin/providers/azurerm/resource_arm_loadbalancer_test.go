package azurerm

import (
	//"fmt"
	//"net/http"
	"testing"
	//"github.com/hashicorp/terraform/helper/acctest"
	//"github.com/hashicorp/terraform/helper/resource"
	//"github.com/hashicorp/terraform/terraform"
)

func TestResourceAzureRMLoadbalancerPrivateIpAddressAllocation_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "Static",
			ErrCount: 0,
		},
		{
			Value:    "Dynamic",
			ErrCount: 0,
		},
		{
			Value:    "STATIC",
			ErrCount: 0,
		},
		{
			Value:    "static",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateLoadbalancerPrivateIpAddressAllocation(tc.Value, "azurerm_loadbalancer")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Loadbalancer private_ip_address_allocation to trigger a validation error")
		}
	}
}

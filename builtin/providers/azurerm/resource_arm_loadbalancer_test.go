package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
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
		_, errors := validateLoadbalancerPrivateIpAddressAllocation(tc.Value, "azurerm_lb")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Loadbalancer private_ip_address_allocation to trigger a validation error")
		}
	}
}

func TestAccAzureRMLoadbalancer_basic(t *testing.T) {
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadbalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadbalancer_basic(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadbalancerExists("azurerm_lb.test"),
				),
			},
		},
	})
}

func TestAccAzureRMLoadbalancer_tags(t *testing.T) {
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadbalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadbalancer_basic(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadbalancerExists("azurerm_lb.test"),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.Environment", "production"),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.Purpose", "AcceptanceTests"),
				),
			},
			{
				Config: testAccAzureRMLoadbalancer_updatedTags(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadbalancerExists("azurerm_lb.test"),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.Purpose", "AcceptanceTests"),
				),
			},
		},
	})
}

func testCheckAzureRMLoadbalancerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		loadbalancerName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for loadbalancer: %s", loadbalancerName)
		}

		conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

		resp, err := conn.Get(resourceGroup, loadbalancerName, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on loadBalancerClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Loadbalancer %q (resource group: %q) does not exist", loadbalancerName, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMLoadbalancerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_lb" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Loadbalancer still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

func testAccAzureRMLoadbalancer_basic(rInt int) string {
	return fmt.Sprintf(`

resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_lb" "test" {
    name = "arm-test-loadbalancer-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    tags {
    	Environment = "production"
    	Purpose = "AcceptanceTests"
    }

}`, rInt, rInt)
}

func testAccAzureRMLoadbalancer_updatedTags(rInt int) string {
	return fmt.Sprintf(`

resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_lb" "test" {
    name = "arm-test-loadbalancer-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    tags {
    	Purpose = "AcceptanceTests"
    }

}`, rInt, rInt)
}

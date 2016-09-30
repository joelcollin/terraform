package azurerm

import "github.com/hashicorp/terraform/helper/schema"

func resourceArmLoadbalancerBackendAddressPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadbalancerBackendAddressPoolCreate,
		Read:   resourceArmLoadbalancerBackendAddressPoolRead,
		Delete: resourceArmLoadbalancerBackendAddressPoolDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"loadbalancer_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"backend_ip_configurations": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"load_balancing_rules": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceArmLoadbalancerBackendAddressPoolCreate(d *schema.ResourceData, meta interface{}) error {
	return resourceArmLoadbalancerBackendAddressPoolRead(d, meta)
}

func resourceArmLoadbalancerBackendAddressPoolRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceArmLoadbalancerBackendAddressPoolDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

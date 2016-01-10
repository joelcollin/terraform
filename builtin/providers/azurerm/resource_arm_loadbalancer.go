package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
)

func resourceArmLoadbalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadbalancerCreate,
		Read:   resourceArmLoadbalancerRead,
		Update: resourceArmLoadbalancerCreate,
		Delete: resourceArmLoadbalancerDelete,

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

			"frontend_ip_configuration": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"subnet_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"private_ip_address": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"public_ip_address_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"private_ip_address_allocation": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateLoadbalancerPrivateIpAddressAllocation,
						},

						"load_balancer_rules": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"inbound_nat_rules": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
				Set: resourceArmLoadbalancerFrontEndIpConfigurationHash,
			},

			"backend_address_pool": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
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
				},
				Set: schema.HashString,
			},

			"load_balancing_rule": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"frontend_ip_configuration_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"backend_address_pool_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},

						"frontend_port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"backend_port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"probe_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"enable_floating_ip": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"idle_timeout_in_minutes": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						"load_distribution": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceArmLoadbalancerLoadBalancingRuleHash,
			},

			"probe": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"protocol": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},

						"port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"request_path": {
							Type:     schema.TypeString,
							Required: true,
						},

						"interval_in_seconds": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"number_of_probes": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"load_balance_rules": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
				Set: resourceArmLoadbalancerProbeHash,
			},

			"inbound_nat_rule": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"frontend_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"backend_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"frontend_ip_configuration_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"backend_ip_configuration_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: resourceArmLoadbalancerInboundNatRulesHash,
			},
		},
	}
}

func resourceArmLoadbalancerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	loadBalancerClient := client.loadBalancerClient

	log.Printf("[INFO] preparing arguments for Azure ARM Loadbalancer creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	properties := network.LoadBalancerPropertiesFormat{}

	if _, ok := d.GetOk("network_security_group_id"); ok {
		frontend_ip_configurations, feIpcErr := expandAzureRmLoadbalancerFrontendIpConfigurations(d)
		if feIpcErr != nil {
			return fmt.Errorf("Error Building list of Loadbalancer Frontend IP Configurations: %s", feIpcErr)
		}
		properties.FrontendIPConfigurations = &frontend_ip_configurations
	}

	if _, ok := d.GetOk("backend_address_pool"); ok {
		backend_address_pools, feIpcErr := expandAzureRmLoadbalancerBackendAddressPools(d)
		if feIpcErr != nil {
			return fmt.Errorf("Error Building list of Loadbalancer Backend Address Pools: %s", feIpcErr)
		}
		properties.BackendAddressPools = &backend_address_pools
	}

	if _, ok := d.GetOk("load_balancing_rule"); ok {
		rules, rulesErr := expandAzureRmLoadbalancerBalancingRules(d)
		if rulesErr != nil {
			return fmt.Errorf("Error Building list of Loadbalancer LoadBalanceRules: %s", rulesErr)
		}
		properties.LoadBalancingRules = &rules
	}

	if _, ok := d.GetOk("probe"); ok {
		probes, probesErr := expandAzureRmLoadbalancerProbes(d)
		if probesErr != nil {
			return fmt.Errorf("Error Building list of Loadbalancer Probes: %s", probesErr)
		}
		properties.Probes = &probes
	}

	if _, ok := d.GetOk("inbound_nat_rule"); ok {
		natRules, natRulesErr := expandAzureRmLoadbalancerInboundNatRules(d)
		if natRulesErr != nil {
			return fmt.Errorf("Error Building list of Loadbalancer InboundNatRules: %s", natRulesErr)
		}
		properties.InboundNatRules = &natRules
	}

	loadbalancer := network.LoadBalancer{
		Name:       &name,
		Location:   &location,
		Properties: &properties,
	}

	_, err := loadBalancerClient.CreateOrUpdate(resGroup, name, loadbalancer, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := loadBalancerClient.Get(resGroup, name, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Loadbalancer %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	log.Printf("[DEBUG] Waiting for LoadBalancer (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: loadbalancerStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Loadbalancer (%s) to become available: %s", name, err)
	}

	return resourceArmLoadbalancerRead(d, meta)
}

func resourceArmLoadbalancerRead(d *schema.ResourceData, meta interface{}) error {
	loadBalancerClient := meta.(*ArmClient).loadBalancerClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["loadBalancers"]
	resGroup := id.ResourceGroup

	resp, err := loadBalancerClient.Get(resGroup, name, "")
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Loadbalancer %s: %s", name, err)
	}

	load_balancer := *resp.Properties

	if load_balancer.FrontendIPConfigurations != nil {
		d.Set("frontend_ip_configuration", flattenLoadBalancerFrontendIpConfiguration(load_balancer.FrontendIPConfigurations))
	}

	if load_balancer.BackendAddressPools != nil {
		d.Set("backend_address_pool", flattenLoadBalancerBackendAddressPools(load_balancer.BackendAddressPools))
	}

	if load_balancer.LoadBalancingRules != nil {
		d.Set("load_balancing_rule", flattenLoadBalancerLoadBalancingRules(load_balancer.LoadBalancingRules))
	}

	if load_balancer.Probes != nil {
		d.Set("probe", flattenLoadBalancerProbes(load_balancer.Probes))
	}

	if load_balancer.InboundNatRules != nil {
		d.Set("inbound_nat_rule", flattenLoadBalancerInboundNatRules(load_balancer.InboundNatRules))
	}

	return nil
}

func resourceArmLoadbalancerDelete(d *schema.ResourceData, meta interface{}) error {
	loadBalancerClient := meta.(*ArmClient).loadBalancerClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["loadBalancers"]

	_, err = loadBalancerClient.Delete(resGroup, name, make(chan struct{}))

	return err
}

func loadbalancerStateRefreshFunc(client *ArmClient, resourceGroupName string, loadbalancer string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.loadBalancerClient.Get(resourceGroupName, loadbalancer, "")
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in loadbalancerStateRefreshFunc to Azure ARM for Loadbalancer '%s' (RG: '%s'): %s", loadbalancer, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}

func resourceArmLoadbalancerFrontEndIpConfigurationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["private_ip_address_allocation"].(string)))

	return hashcode.String(buf.String())
}

func resourceArmLoadbalancerInboundNatRulesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["frontend_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["backend_port"].(int)))

	return hashcode.String(buf.String())
}

func resourceArmLoadbalancerLoadBalancingRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["frontend_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["backend_port"].(int)))

	return hashcode.String(buf.String())
}

func resourceArmLoadbalancerProbeHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["interval_in_seconds"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["number_of_probes"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["request_path"].(string)))

	return hashcode.String(buf.String())
}

func validateLoadbalancerPrivateIpAddressAllocation(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	allocations := map[string]bool{
		"static":  true,
		"dynamic": true,
	}

	if !allocations[value] {
		errors = append(errors, fmt.Errorf("Loadbalancer Allocations can only be Static or Dynamic"))
	}
	return
}

func expandAzureRmLoadbalancerFrontendIpConfigurations(d *schema.ResourceData) ([]network.FrontendIPConfiguration, error) {
	configs := d.Get("frontend_ip_configuration").(*schema.Set).List()
	frontEndConfigs := make([]network.FrontendIPConfiguration, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		private_ip_allocation_method := data["private_ip_address_allocation"].(string)
		properties := network.FrontendIPConfigurationPropertiesFormat{
			PrivateIPAllocationMethod: network.IPAllocationMethod(private_ip_allocation_method),
		}

		if v := data["private_ip_address"].(string); v != "" {
			properties.PrivateIPAddress = &v
		}

		if v := data["public_ip_address_id"].(string); v != "" {
			properties.PublicIPAddress = &network.PublicIPAddress{
				ID: &v,
			}
		}

		if v := data["subnet_id"].(string); v != "" {
			properties.Subnet = &network.Subnet{
				ID: &v,
			}
		}

		name := data["name"].(string)
		frontEndConfig := network.FrontendIPConfiguration{
			Name:       &name,
			Properties: &properties,
		}

		frontEndConfigs = append(frontEndConfigs, frontEndConfig)
	}

	return frontEndConfigs, nil
}

func expandAzureRmLoadbalancerProbes(d *schema.ResourceData) ([]network.Probe, error) {
	probesConfig := d.Get("probe").(*schema.Set).List()
	probes := make([]network.Probe, 0, len(probesConfig))

	for _, configRaw := range probesConfig {
		data := configRaw.(map[string]interface{})

		port := data["port"].(int)
		request_path := data["request_path"].(string)
		interval_in_seconds := data["interval_in_seconds"].(int)
		number_of_probes := data["number_of_probes"].(int)

		properties := network.ProbePropertiesFormat{
			Port:              azure.Int32(int32(port)),
			RequestPath:       &request_path,
			IntervalInSeconds: azure.Int32(int32(interval_in_seconds)),
			NumberOfProbes:    azure.Int32(int32(number_of_probes)),
		}

		if v := data["protocol"].(string); v != "" {
			properties.Protocol = network.ProbeProtocol(v)
		}

		name := data["name"].(string)
		probe := network.Probe{
			Name:       &name,
			Properties: &properties,
		}

		probes = append(probes, probe)
	}

	return probes, nil
}

func expandAzureRmLoadbalancerBackendAddressPools(d *schema.ResourceData) ([]network.BackendAddressPool, error) {
	pools := d.Get("frontend_ip_configuration").(*schema.Set).List()
	backendPools := make([]network.BackendAddressPool, 0, len(pools))

	for _, configRaw := range pools {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		backendpool := network.BackendAddressPool{
			Name: &name,
		}

		backendPools = append(backendPools, backendpool)
	}

	return backendPools, nil
}

func expandAzureRmLoadbalancerInboundNatRules(d *schema.ResourceData) ([]network.InboundNatRule, error) {
	configs := d.Get("inbound_nat_rule").(*schema.Set).List()
	natRules := make([]network.InboundNatRule, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		protocol := data["protocol"].(string)
		frontend_port := data["frontend_port"].(int)
		backend_port := data["backend_port"].(int)
		properties := network.InboundNatRulePropertiesFormat{
			Protocol:     network.TransportProtocol(protocol),
			FrontendPort: azure.Int32(int32(frontend_port)),
			BackendPort:  azure.Int32(int32(backend_port)),
		}

		if v := data["frontend_ip_configuration_id"].(string); v != "" {
			feIpConfig := network.SubResource{
				ID: &v,
			}

			properties.FrontendIPConfiguration = &feIpConfig
		}

		name := data["name"].(string)
		natrule := network.InboundNatRule{
			Name:       &name,
			Properties: &properties,
		}

		natRules = append(natRules, natrule)
	}

	return natRules, nil
}

func expandAzureRmLoadbalancerBalancingRules(d *schema.ResourceData) ([]network.LoadBalancingRule, error) {
	configs := d.Get("load_balancing_rule").(*schema.Set).List()
	natRules := make([]network.LoadBalancingRule, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		frontend_port := data["frontend_port"].(int)
		backend_port := data["backend_port"].(int)
		protocol := data["protocol"].(string)
		properties := network.LoadBalancingRulePropertiesFormat{
			FrontendPort: azure.Int32(int32(frontend_port)),
			BackendPort:  azure.Int32(int32(backend_port)),
			Protocol:     network.TransportProtocol(protocol),
		}

		if v, ok := data["enable_floating_ip"]; ok {
			floating_ip := v.(bool)
			properties.EnableFloatingIP = &floating_ip
		}

		if v, ok := data["idle_timeout_in_minutes"]; ok {
			timeout := v.(int)
			properties.IdleTimeoutInMinutes = azure.Int32(int32(timeout))
		}

		if v, ok := data["load_distribution"]; ok {
			load_distribution := v.(string)
			properties.LoadDistribution = network.LoadDistribution(load_distribution)
		}

		if v, ok := data["probe_id"]; ok {
			id := v.(string)

			probe := network.SubResource{
				ID: &id,
			}
			properties.Probe = &probe
		}

		if v := data["frontend_ip_configuration_id"].(string); v != "" {
			feIpConfig := network.SubResource{
				ID: &v,
			}

			properties.FrontendIPConfiguration = &feIpConfig
		}

		if v := data["backend_address_pool_id"].(string); v != "" {
			beAp := network.SubResource{
				ID: &v,
			}

			properties.BackendAddressPool = &beAp
		}

		name := data["name"].(string)
		natrule := network.LoadBalancingRule{
			Name:       &name,
			Properties: &properties,
		}

		natRules = append(natRules, natrule)
	}

	return natRules, nil
}

func flattenLoadBalancerFrontendIpConfiguration(ipConfigs *[]network.FrontendIPConfiguration) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(*ipConfigs))
	for _, config := range *ipConfigs {
		ipConfig := make(map[string]interface{})
		ipConfig["name"] = *config.Name
		ipConfig["private_ip_address_allocation"] = config.Properties.PrivateIPAllocationMethod

		if config.Properties.Subnet != nil {
			ipConfig["subnet_id"] = *config.Properties.Subnet.ID
		}

		if *config.Properties.PrivateIPAddress != "" {
			ipConfig["private_ip_address"] = *config.Properties.PrivateIPAddress
		}

		if config.Properties.PublicIPAddress != nil {
			ipConfig["public_ip_address_id"] = *config.Properties.PublicIPAddress.ID
		}

		if config.Properties.LoadBalancingRules != nil {
			load_balancing_rules := make([]string, 0, len(*config.Properties.LoadBalancingRules))
			for _, rule := range *config.Properties.LoadBalancingRules {
				load_balancing_rules = append(load_balancing_rules, *rule.ID)
			}

			ipConfig["load_balancer_rules"] = load_balancing_rules

		}

		if config.Properties.InboundNatRules != nil {
			inbound_nat_rules := make([]string, 0, len(*config.Properties.InboundNatRules))
			for _, rule := range *config.Properties.InboundNatRules {
				inbound_nat_rules = append(inbound_nat_rules, *rule.ID)
			}

			ipConfig["inbound_nat_rules"] = inbound_nat_rules

		}

		result = append(result, ipConfig)
	}
	return result
}

func flattenLoadBalancerBackendAddressPools(pools *[]network.BackendAddressPool) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(*pools))
	for _, p := range *pools {
		pool := make(map[string]interface{})
		pool["name"] = *p.Name

		if p.Properties.BackendIPConfigurations != nil {
			backend_ip_configurations := make([]string, 0, len(*p.Properties.BackendIPConfigurations))
			for _, configs := range *p.Properties.BackendIPConfigurations {
				backend_ip_configurations = append(backend_ip_configurations, *configs.ID)
			}

			pool["backend_ip_configurations"] = backend_ip_configurations

		}

		if p.Properties.LoadBalancingRules != nil {
			load_balancing_rules := make([]string, 0, len(*p.Properties.LoadBalancingRules))
			for _, rule := range *p.Properties.LoadBalancingRules {
				load_balancing_rules = append(load_balancing_rules, *rule.ID)
			}

			pool["backend_ip_configurations"] = load_balancing_rules

		}

		result = append(result, pool)
	}
	return result
}

func flattenLoadBalancerLoadBalancingRules(rules *[]network.LoadBalancingRule) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(*rules))
	for _, rule := range *rules {
		lbRule := make(map[string]interface{})
		lbRule["name"] = *rule.Name
		lbRule["protocol"] = rule.Properties.Protocol
		lbRule["frontend_port"] = *rule.Properties.FrontendPort
		lbRule["backend_port"] = *rule.Properties.BackendPort

		if rule.Properties.EnableFloatingIP != nil {
			lbRule["enable_floating_ip"] = *rule.Properties.EnableFloatingIP
		}

		if rule.Properties.IdleTimeoutInMinutes != nil {
			lbRule["idle_timeout_in_minutes"] = *rule.Properties.IdleTimeoutInMinutes
		}

		if rule.Properties.FrontendIPConfiguration != nil {
			lbRule["frontend_ip_configuration_id"] = *rule.Properties.FrontendIPConfiguration.ID
		}

		if rule.Properties.BackendAddressPool != nil {
			lbRule["backend_address_pool_id"] = *rule.Properties.BackendAddressPool.ID
		}

		if rule.Properties.Probe != nil {
			lbRule["probe_id"] = *rule.Properties.Probe.ID
		}

		if rule.Properties.LoadDistribution != "" {
			lbRule["load_distribution"] = rule.Properties.LoadDistribution
		}

		result = append(result, lbRule)
	}
	return result
}

func flattenLoadBalancerProbes(probes *[]network.Probe) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(*probes))
	for _, p := range *probes {
		probe := make(map[string]interface{})
		probe["name"] = *p.Name
		probe["port"] = *p.Properties.Port
		probe["request_path"] = *p.Properties.RequestPath
		probe["interval_in_seconds"] = *p.Properties.IntervalInSeconds
		probe["number_of_probes"] = *p.Properties.NumberOfProbes

		if p.Properties.Protocol != "" {
			probe["protocol"] = p.Properties.Protocol
		}

		if p.Properties.LoadBalancingRules != nil {
			load_balancer_rules := make([]string, 0, len(*p.Properties.LoadBalancingRules))
			for _, rule := range *p.Properties.LoadBalancingRules {
				load_balancer_rules = append(load_balancer_rules, *rule.ID)
			}

			probe["load_balance_rules"] = load_balancer_rules

		}

		result = append(result, probe)
	}
	return result
}

func flattenLoadBalancerInboundNatRules(rules *[]network.InboundNatRule) []map[string]interface{} {

	result := make([]map[string]interface{}, 0, len(*rules))
	for _, rule := range *rules {
		natRule := make(map[string]interface{})
		natRule["name"] = *rule.Name
		natRule["protocol"] = rule.Properties.Protocol
		natRule["frontend_port"] = *rule.Properties.FrontendPort
		natRule["backend_port"] = *rule.Properties.BackendPort

		if rule.Properties.FrontendIPConfiguration != nil {
			natRule["frontend_ip_configuration_id"] = *rule.Properties.FrontendIPConfiguration.ID
		}

		if rule.Properties.BackendIPConfiguration != nil {
			natRule["backend_ip_configuration_id"] = *rule.Properties.BackendIPConfiguration.ID
		}

		result = append(result, natRule)
	}
	return result
}

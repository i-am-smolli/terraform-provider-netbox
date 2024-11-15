package netbox

import (
	"strconv"
	"strings"

	"github.com/fbreckle/go-netbox/netbox/client"
	"github.com/fbreckle/go-netbox/netbox/client/ipam"
	"github.com/fbreckle/go-netbox/netbox/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var resourceNetboxIPRangeStatusOptions = []string{"active", "reserved", "deprecated"}

func resourceNetboxIPRange() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetboxIPRangeCreate,
		Read:   resourceNetboxIPRangeRead,
		Update: resourceNetboxIPRangeUpdate,
		Delete: resourceNetboxIPRangeDelete,

		Description: `:meta:subcategory:IP Address Management (IPAM):From the [official documentation](https://docs.netbox.dev/en/stable/features/ipam/#ip-ranges):

> This model represents an arbitrary range of individual IPv4 or IPv6 addresses, inclusive of its starting and ending addresses. For instance, the range 192.0.2.10 to 192.0.2.20 has eleven members. (The total member count is available as the size property on an IPRange instance.) Like prefixes and IP addresses, each IP range may optionally be assigned to a VRF and/or tenant.`,

		Schema: map[string]*schema.Schema{
			"start_address": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The first address of the IP range. Needs CIDR notation.",
			},
			"end_address": {
				Type:         schema.TypeString,
				Required: 	  true,
				Description:  "The last address of the IP range. Needs CIDR notation.",
			},
			"status": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "active",
				ValidateFunc: validation.StringInSlice(resourceNetboxIPRangeStatusOptions, false),
				Description:  buildValidValueDescription(resourceNetboxIPRangeStatusOptions),
			},
			"tenant_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The ID of the tenant which this range belongs to.",
			},
			"role_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The ID of the role attached to this range.",
			},
			"vrf_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The ID of the VRF which this range belongs to.",
			},
			"description": {
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {	
					return strings.TrimSpace(oldValue) == strings.TrimSpace(newValue)
				},
				DiffSuppressOnRefresh: true,
				
				Type:         schema.TypeString,
				Optional: 	  true,
				Description:  "Brief description of the IP range.",
			},
			"comments": {
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {	
					return strings.TrimSpace(oldValue) == strings.TrimSpace(newValue)
				},
				DiffSuppressOnRefresh: true,

				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Comments about the IP range. Multi-line comments are supported.",
			},
			tagsKey: tagsSchema,
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceNetboxIPRangeCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*client.NetBoxAPI)
	data := models.WritableIPRange{}

	data.StartAddress = strToPtr(d.Get("start_address").(string))
	data.EndAddress = strToPtr(d.Get("end_address").(string))

	data.Status = d.Get("status").(string)
	data.Description = getOptionalStr(d, "description", true)

	data.Tags, _ = getNestedTagListFromResourceDataSet(api, d.Get(tagsKey))

	params := ipam.NewIpamIPRangesCreateParams().WithData(&data)
	res, err := api.Ipam.IpamIPRangesCreate(params, nil)
	if err != nil {
		return err
	}
	d.SetId(strconv.FormatInt(res.GetPayload().ID, 10))

	return resourceNetboxIPRangeUpdate(d, m)
}

func resourceNetboxIPRangeRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*client.NetBoxAPI)
	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	params := ipam.NewIpamIPRangesReadParams().WithID(id)

	res, err := api.Ipam.IpamIPRangesRead(params, nil)
	if err != nil {
		if errresp, ok := err.(*ipam.IpamIPRangesReadDefault); ok {
			errorcode := errresp.Code()
			if errorcode == 404 {
				// If the ID is updated to blank, this tells Terraform the resource no longer exists (maybe it was destroyed out of band). Just like the destroy callback, the Read function should gracefully handle this case. https://www.terraform.io/docs/extend/writing-custom-providers.html
				d.SetId("")
				return nil
			}
		}
		return err
	}

	if res.GetPayload().StartAddress != nil {
		d.Set("start_address", res.GetPayload().StartAddress)
	}

	if res.GetPayload().EndAddress != nil {
		d.Set("end_address", res.GetPayload().EndAddress)
	}

	if res.GetPayload().Status != nil {
		d.Set("status", res.GetPayload().Status.Value)
	}

	if res.GetPayload().Vrf != nil {
		d.Set("vrf_id", res.GetPayload().Vrf.ID)
	}

	d.Set("description", res.GetPayload().Description)

	d.Set("comments", res.GetPayload().Comments)

	if res.GetPayload().Tenant != nil {
		d.Set("tenant_id", res.GetPayload().Tenant.ID)
	}

	if res.GetPayload().Role != nil {
		d.Set("role_id", res.GetPayload().Role.ID)
	}

	d.Set(tagsKey, getTagListFromNestedTagList(res.GetPayload().Tags))

	return nil
}

func resourceNetboxIPRangeUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*client.NetBoxAPI)

	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	data := models.WritableIPRange{}

	data.StartAddress = strToPtr(d.Get("start_address").(string))
	data.EndAddress = strToPtr(d.Get("end_address").(string))

	data.Status = d.Get("status").(string)
	data.Description = getOptionalStr(d, "description", true)
	data.Comments = getOptionalStr(d, "comments", true)

	if vrfID, ok := d.GetOk("vrf_id"); ok {
		data.Vrf = int64ToPtr(int64(vrfID.(int)))
	}

	if tenantID, ok := d.GetOk("tenant_id"); ok {
		data.Tenant = int64ToPtr(int64(tenantID.(int)))
	}

	if roleID, ok := d.GetOk("role_id"); ok {
		data.Role = int64ToPtr(int64(roleID.(int)))
	}

	data.Tags, _ = getNestedTagListFromResourceDataSet(api, d.Get(tagsKey))

	params := ipam.NewIpamIPRangesUpdateParams().WithID(id).WithData(&data)
	_, err := api.Ipam.IpamIPRangesUpdate(params, nil)
	if err != nil {
		return err
	}
	return resourceNetboxIPRangeRead(d, m)
}

func resourceNetboxIPRangeDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*client.NetBoxAPI)
	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	params := ipam.NewIpamIPRangesDeleteParams().WithID(id)
	_, err := api.Ipam.IpamIPRangesDelete(params, nil)
	if err != nil {
		if errresp, ok := err.(*ipam.IpamIPRangesDeleteDefault); ok {
			if errresp.Code() == 404 {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	return nil
}

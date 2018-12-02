package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/transfer"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsTransferServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsTransferServerCreate,
		Read:   resourceAwsTransferServerRead,
		Update: resourceAwsTransferServerUpdate,
		Delete: resourceAwsTransferServerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"invocation_role": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},

			"url": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"identity_provider_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  transfer.IdentityProviderTypeServiceManaged,
				ValidateFunc: validation.StringInSlice([]string{
					transfer.IdentityProviderTypeServiceManaged,
					transfer.IdentityProviderTypeApiGateway,
				}, false),
			},

			"logging_role": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsTransferServerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn
	tags := tagsFromMapTransferServer(d.Get("tags").(map[string]interface{}))

	createOpts := &transfer.CreateServerInput{
		Tags: tags,
	}

	var identityProviderDetails *transfer.IdentityProviderDetails
	if attr, ok := d.GetOk("invocation_role"); ok {
		identityProviderDetails.SetInvocationRole(attr.(string))
	}

	if attr, ok := d.GetOk("url"); ok {
		identityProviderDetails.SetUrl(attr.(string))
	}
	if identityProviderDetails != nil {
		createOpts.SetIdentityProviderDetails(identityProviderDetails)
	}

	if attr, ok := d.GetOk("identity_provider_type"); ok {
		createOpts.SetIdentityProviderType(attr.(string))
	}

	if attr, ok := d.GetOk("logging_role"); ok {
		createOpts.SetIdentityProviderType(attr.(string))
	}

	log.Printf("[DEBUG] Create Transfer Server Option: %#v", createOpts)

	resp, err := conn.CreateServer(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating Transfer Server: %s", err)
	}

	d.SetId(*resp.ServerId)

	return resourceAwsTransferServerRead(d, meta)
}

func resourceAwsTransferServerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn

	descOpts := &transfer.DescribeServerInput{
		ServerId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Describe Transfer Server Option: %#v", descOpts)

	resp, err := conn.DescribeServer(descOpts)
	if err != nil {
		if isAWSErr(err, transfer.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] Transfer Server (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	endpoint := fmt.Sprintf("%s.server.transfer.%s.amazonaws.com", d.Id(), meta.(*AWSClient).region)
	log.Printf("[DEBUG] managed endpoint : %s", endpoint)

	d.Set("arn", resp.Server.Arn)
	d.Set("endpoint", endpoint)
	d.Set("invocation_role", "")
	d.Set("url", "")
	if resp.Server.IdentityProviderDetails != nil {
		d.Set("invocation_role", aws.StringValue(resp.Server.IdentityProviderDetails.InvocationRole))
		d.Set("url", aws.StringValue(resp.Server.IdentityProviderDetails.Url))
	}
	d.Set("identity_provider_type", resp.Server.IdentityProviderType)
	d.Set("logging_role", resp.Server.LoggingRole)

	if err := d.Set("tags", tagsToMapTransferServer(resp.Server.Tags)); err != nil {
		return fmt.Errorf("Error setting tags: %s", err)
	}
	return nil
}

func resourceAwsTransferServerUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn
	updateFlag := false
	updateOpts := &transfer.UpdateServerInput{
		ServerId: aws.String(d.Id()),
	}

	if d.HasChange("logging_role") {
		updateFlag = true
		updateOpts.SetLoggingRole(d.Get("logging_role").(string))
	}

	if d.HasChange("invocation_role") || d.HasChange("url") {
		var identityProviderDetails *transfer.IdentityProviderDetails
		updateFlag = true
		if attr, ok := d.GetOk("invocation_role"); ok {
			identityProviderDetails.SetInvocationRole(attr.(string))
		}

		if attr, ok := d.GetOk("url"); ok {
			identityProviderDetails.SetUrl(attr.(string))
		}
		updateOpts.SetIdentityProviderDetails(identityProviderDetails)
	}

	if updateFlag {
		_, err := conn.UpdateServer(updateOpts)
		if err != nil {
			if isAWSErr(err, transfer.ErrCodeResourceNotFoundException, "") {
				log.Printf("[WARN] Transfer Server (%s) not found, removing from state", d.Id())
				d.SetId("")
				return nil
			}
			return fmt.Errorf("error updating Transfer Server (%s): %s", d.Id(), err)
		}
	}

	if err := setTagsTransferServer(conn, d); err != nil {
		return fmt.Errorf("Error update tags: %s", err)
	}

	return resourceAwsTransferServerRead(d, meta)
}

func resourceAwsTransferServerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).transferconn

	delOpts := &transfer.DeleteServerInput{
		ServerId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Delete Transfer Server Option: %#v", delOpts)

	_, err := conn.DeleteServer(delOpts)
	if err != nil {
		if isAWSErr(err, transfer.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] Transfer Server (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	return nil
}

package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsBackupVault() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsBackupVaultCreate,
		Read:   resourceAwsBackupVaultRead,
		Delete: resourceAwsBackupVaultDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`[a-z0-9\-]+`), "must contain alphanumeric characters or underscores"),
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"kms_key_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"recovery_points": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceAwsBackupVaultCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).backupconn

	input := &backup.CreateBackupVaultInput{
		BackupVaultName: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("tags"); ok {
		input.BackupVaultTags = tagsFromMapGeneric(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("kms_key_arn"); ok {
		input.EncryptionKeyArn = aws.String(v.(string))
	}

	_, err := conn.CreateBackupVault(input)
	if err != nil {
		return fmt.Errorf("error creating Backup Vault (%s): %s", d.Id(), err)
	}

	d.SetId(d.Get("name").(string))

	return resourceAwsBackupVaultRead(d, meta)
}

func resourceAwsBackupVaultRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).backupconn

	input := &backup.DescribeBackupVaultInput{
		BackupVaultName: aws.String(d.Id()),
	}

	resp, err := conn.DescribeBackupVault(input)
	if err != nil {
		return fmt.Errorf("error reading Backup Vault (%s): %s", d.Id(), err)
	}

	d.Set("kms_key_arn", resp.EncryptionKeyArn)
	d.Set("arn", resp.BackupVaultArn)
	d.Set("recovery_points", resp.NumberOfRecoveryPoints)

	tresp, err := conn.ListTags(&backup.ListTagsInput{
		ResourceArn: aws.String(*resp.BackupVaultArn),
	})

	if err != nil {
		log.Printf("[DEBUG] Error retrieving tags for ARN: %s", aws.StringValue(resp.BackupVaultArn))
	}

	var tags map[string]*string
	if len(tresp.Tags) > 0 {
		tags = tresp.Tags
	}
	d.Set("tags", tags)

	return nil
}

func resourceAwsBackupVaultDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).backupconn

	input := &backup.DeleteBackupVaultInput{
		BackupVaultName: aws.String(d.Get("name").(string)),
	}

	_, err := conn.DeleteBackupVault(input)
	if err != nil {
		return fmt.Errorf("error deleting Backup Vault (%s): %s", d.Id(), err)
	}

	return nil
}

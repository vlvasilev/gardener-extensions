package infrastructure

import (
	alicloudv1alpha1 "github.com/gardener/gardener-extensions/controllers/provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/gardener/gardener-extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

const (
	DefaultVPCID        = "${alicloud_vpc.vpc.id}"
	DefaultNATGatewayID = "${alicloud_nat_gateway.nat_gateway.id}"
	DefaultSNATTableID  = "${alicloud_nat_gateway.nat_gateway.snat_table_ids}"
)

func ComputeTerraformerChartInput(
	infra *extensionsv1alpha1.Infrastructure,
	config *alicloudv1alpha1.InfrastructureConfig,
	cluster *controller.Cluster,
) (*TerraformerChartInput, error) {
	vpcInput, err := FetchTerraformerVPCChartInput(config.Networks.VPC)
	if err != nil {
		return nil, err
	}

	return &TerraformerChartInput{
		SSHPublicKey: infra.Spec.SSHPublicKey,
		Region:       infra.Spec.Region,
		Zones:        config.Networks.Zones,
		ClusterName:  cluster.Shoot.Name,
		VPC:          *vpcInput,
	}, nil
}

func FetchTerraformerVPCChartInput(vpc alicloudv1alpha1.VPC) (*TerraformerVPCChartInput, error) {
	if vpc.ID == nil {
		return &TerraformerVPCChartInput{
			Create:       true,
			ID:           DefaultVPCID,
			NATGatewayID: DefaultNATGatewayID,
			SNATTableID:  DefaultSNATTableID,
			CIDR:         string(*vpc.CIDR),
		}, nil
	}
}

type TerraformerVPCChartInput struct {
	Create       bool
	ID           string
	CIDR         string
	NATGatewayID string
	SNATTableID  string
}

type TerraformerChartInput struct {
	Zones        []alicloudv1alpha1.Zone
	SSHPublicKey []byte
	ClusterName  string
	VPC          TerraformerVPCChartInput
	Region       string
}

func ComputeTerraformerChartValues(input *TerraformerChartInput) map[string]interface{} {
	var zones []map[string]interface{}

	for _, zone := range input.Zones {
		zones = append(zones, map[string]interface{}{
			"name": zone.Name,
			"cidr": map[string]interface{}{
				"worker": zone.Worker,
			},
		})
	}

	return map[string]interface{}{
		"alicloud": map[string]interface{}{
			"region": input.Region,
		},
		"create": map[string]interface{}{
			"vpc": input.VPC.Create,
		},
		"vpc": map[string]interface{}{
			"cidr":         input.VPC.CIDR,
			"id":           input.VPC.ID,
			"natGatewayID": input.VPC.NATGatewayID,
			"snatTableID":  input.VPC.SNATTableID,
		},
		"clusterName":  input.ClusterName,
		"sshPublicKey": input.SSHPublicKey,
		"zones":        zones,
	}
}

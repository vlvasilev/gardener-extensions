package infrastructure

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	kutils "k8s.io/apimachinery/pkg/util/runtime"
	"strconv"
	"strings"

	"github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/apis/aws/v1alpha1"
	awstypes "github.com/gardener/gardener-extensions/controllers/provider-aws/pkg/aws"
	"github.com/gardener/gardener-extensions/pkg/controller"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/aws"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/operation/terraformer"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (c *actuator) injectStatusError(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, lastOperationType extensionsv1alpha1.LastOperationType, description string, progress int, err error) {
	infrastructure.Status.ObservedGeneration = infrastructure.Generation
	infrastructure.Status.LastOperation, infrastructure.Status.LastError = controller.ReconcileError(lastOperationType, fmt.Sprintf("%s: %v", description, err), progress)
	if err := c.client.Status().Update(ctx, infrastructure); err != nil {
		c.logger.Error(err, "Could not update infrastructure infrastructure status after update error", "infrastructure", infrastructure.Name)
	}
}

func (c *actuator) injectStatusSuccess(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, lastOperationType extensionsv1alpha1.LastOperationType, description string) error {
	infrastructure.Status.ObservedGeneration = infrastructure.Generation
	infrastructure.Status.LastOperation, infrastructure.Status.LastError = controller.ReconcileSucceeded(lastOperationType, description)
	return c.client.Status().Update(ctx, infrastructure)
}

func getZones(infra *extensionsv1alpha1.Infrastructure, infraProviderConfig *v1alpha1.InfrastructureConfig) []map[string]interface{} {
	zones := []map[string]interface{}{}
	for zoneIndex, zone := range infra.Spec.Zones {
		zones = append(zones, map[string]interface{}{
			"name": zone,
			"cidr": map[string]interface{}{
				"worker":   infraProviderConfig.Networks.Workers[zoneIndex],
				"public":   infraProviderConfig.Networks.Public[zoneIndex],
				"internal": infraProviderConfig.Networks.Internal[zoneIndex],
			},
		})
	}
	return zones
}

func (c *actuator) getTerraformer(purpose, namespace, name string) (*terraformer.Terraformer, error) {
	serverVersion, err := c.kubernetes.Discovery().ServerVersion()

	if err != nil {
		return nil, err
	}
	tfImage, err := awstypes.ImageVector.FindImage(awstypes.TerraformerImageName, serverVersion.GitVersion, serverVersion.GitVersion)
	if err != nil {
		return nil, err
	}

	return terraformer.NewForConfig(logger.NewLogger("info"), c.config, purpose, namespace, name, tfImage.String())
}

func (c *actuator) destroyKubernetesLoadBalancersAndSecurityGroups(namespace string, tf *terraformer.Terraformer, awsClient aws.ClientInterface) error {
	if _, err := tf.GetState(); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	vpcIDKey := "vpc_id"
	stateVariables, err := tf.GetStateOutputVariables(vpcIDKey)
	if err != nil {
		if terraformer.IsVariablesNotFoundError(err) {
			c.logger.Info("Skipping explicit AWS load balancer and security group deletion because not all variables have been found in the Terraform state.")
			return nil
		}
		return err
	}
	vpcID := stateVariables[vpcIDKey]
	// Find load balancers and security groups.
	loadBalancers, err := awsClient.ListKubernetesELBs(vpcID, namespace)
	if err != nil {
		return err
	}
	securityGroups, err := awsClient.ListKubernetesSecurityGroups(vpcID, namespace)
	if err != nil {
		return err
	}

	// Destroy load balancers and security groups.
	for _, loadBalancerName := range loadBalancers {
		if err := awsClient.DeleteELB(loadBalancerName); err != nil {
			return err
		}
	}
	for _, securityGroupID := range securityGroups {
		if err := awsClient.DeleteSecurityGroup(securityGroupID); err != nil {
			return err
		}
	}
	return nil
}

func (c *actuator) injectProviderStateIntoStatus(ctx context.Context, tf *terraformer.Terraformer, infrastructure *extensionsv1alpha1.Infrastructure) error {
	outputVarKeys := []string{
		awstypes.VPCIDKey,
		awstypes.SSHKeyName,
		awstypes.IAMInstanceProfileNodes,
		awstypes.NodesRole,
		awstypes.SecurityGroupsNodes,
	}

	for zoneIndex := range infrastructure.Spec.Zones {
		outputVarKeys = append(outputVarKeys, fmt.Sprintf("%s%d", awstypes.SubnetNodesPrefix, zoneIndex))
		outputVarKeys = append(outputVarKeys, fmt.Sprintf("%s%d", awstypes.SubnetPublicPrefix, zoneIndex))
	}

	values, err := tf.GetStateOutputVariables(outputVarKeys...)
	if err != nil {
		return err
	}

	var (
		instanceProfiles = []v1alpha1.InstanceProfile{
			{
				Purpose: "instance profile for nodes",
				Name:    values[awstypes.IAMInstanceProfileNodes],
			},
		}
		roles = []v1alpha1.Role{
			{
				Purpose: "role for nodes",
				ARN:     values[awstypes.NodesRole],
			},
		}
		subnets = func(values map[string]string) []v1alpha1.Subnet {
			var subnetsToReturn []v1alpha1.Subnet
			for key, value := range values {
				if strings.HasPrefix(key, awstypes.SubnetPublicPrefix) {
					zoneID, err := strconv.Atoi(strings.TrimPrefix(key, awstypes.SubnetPublicPrefix))
					kutils.Must(err)

					subnetsToReturn = append(subnetsToReturn, v1alpha1.Subnet{
						Name: key,
						ID:   value,
						Zone: infrastructure.Spec.Zones[zoneID],
					})
				}
				if strings.HasPrefix(key, awstypes.SubnetNodesPrefix) {
					zoneID, err := strconv.Atoi(strings.TrimPrefix(key, awstypes.SubnetNodesPrefix))
					kutils.Must(err)
					subnetsToReturn = append(subnetsToReturn, v1alpha1.Subnet{
						Name: key,
						ID:   value,
						Zone: infrastructure.Spec.Zones[zoneID],
					})
				}
			}
			return subnetsToReturn
		}(values)
		securityGroups = []v1alpha1.SecurityGroup{
			{
				Name: awstypes.SecurityGroupsNodes,
				ID:   values[awstypes.SecurityGroupsNodes],
			},
		}
	)

	infrastructure.Status.ProviderStatus = &runtime.RawExtension{
		Object: &v1alpha1.InfrastructureStatus{
			VPC: v1alpha1.VPC{
				ID:             values[awstypes.VPCIDKey],
				Subnets:        subnets,
				SecurityGroups: securityGroups,
			},
			EC2: v1alpha1.EC2{
				KeyName: values[awstypes.SSHKeyName],
			},
			IAM: v1alpha1.IAM{
				InstanceProfiles: instanceProfiles,
				Roles:            roles,
			},
		},
	}
	return c.client.Status().Update(ctx, infrastructure)
}

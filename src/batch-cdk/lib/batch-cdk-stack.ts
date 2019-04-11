import cdk = require('@aws-cdk/cdk');
import batch = require("@aws-cdk/aws-batch");
import iam = require("@aws-cdk/aws-iam");
import ec2 = require("@aws-cdk/aws-ec2");

export class BatchCdkStack extends cdk.Stack {
	constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
		super(scope, id, props);

		// The code that defines your stack goes here

		// iam
		let batch_service_role = new iam.Role(this, "BatchServiceRole", {
			managedPolicyArns: [
				'arn:aws:iam::aws:policy/service-role/AWSBatchServiceRole'
			],
			roleName: this.node.getContext("batch_service_role_name"),
			assumedBy: new iam.ServicePrincipal('batch.amazonaws.com')
		});


		let batch_instance_role = new iam.Role(this, "BatchInstanceRole", {
			managedPolicyArns: [
				'arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role'
			],
			roleName: this.node.getContext("batch_instance_role_name"),
			assumedBy: new iam.ServicePrincipal('ec2.amazonaws.com')
		});
		let batch_job_role = new iam.Role(this, "BatchJobRole",{
			managedPolicyArns: [
				'arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role'
			],
			roleName: this.node.getContext("batch_job_role_name"),
			assumedBy: new iam.ServicePrincipal('ecs-tasks.amazonaws.com')
		});

		new iam.Policy(this, "DenyActionOnUserPolicy",{
			policyName: 'DenyActionsOnUsersPolicy',
			statements: [
				new iam.PolicyStatement().deny().addAction("*").addResource("arn:aws:iam::*:user/*")
			],
			roles: [
				batch_instance_role,
				batch_job_role
			]
		});

		let spot_fleet_role = new iam.Role(this, "SpotFleetRole", {
			managedPolicyArns: [
				'arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole'
			],
			roleName: this.node.getContext("spot_fleet_role_name"),
			assumedBy: new iam.ServicePrincipal('spotfleet.amazonaws.com')
		});

		/* It seems there is a bug exists. */
		// const vpc = ec2.VpcNetwork.importFromContext(this, "defaultVPC",{
		// 	isDefault: true
		// });

		let subnet = new ec2.CfnSubnet(this, "subnet", {
			vpcId: this.node.getContext("default_vpc_id"),
			cidrBlock: this.node.getContext("subnet_cidr"),
			tags: [{
				key: "Name",
				value: this.node.getContext("subnet_name")
			}]
		});

		let eip = new ec2.CfnEIP(this, "elasticIP", {
				domain: 'vpc'
		});
		eip.node.apply(new cdk.Tag("Name", this.node.getContext("eip_name")));

		let security_group = new ec2.CfnSecurityGroup(this, "securityGroup", {
			vpcId: this.node.getContext("default_vpc_id"),
			groupDescription: this.node.getContext("security_group_name"),
			groupName: this.node.getContext("security_group_name")
		});
		security_group.node.apply(new cdk.Tag("Name", this.node.getContext("security_group_name")));

		let network_interface = new ec2.CfnNetworkInterface(this, "network_interface", {
			subnetId: subnet.subnetId,
			groupSet: [
				security_group.securityGroupId
			],
		});
		network_interface.node.apply(new cdk.Tag("Name", this.node.getContext("network_interface_name")));

		new ec2.CfnEIPAssociation(this, "EIPAssociate", {
			allocationId: eip.eipAllocationId,
			networkInterfaceId: network_interface.ref
		});
		let batch_environment = new batch.CfnComputeEnvironment(this, "batchCompute", {
			computeEnvironmentName: this.node.getContext("batch_env_name"),
			serviceRole: batch_service_role.roleArn,
			type: 'MANAGED',
			computeResources: {
				bidPercentage: 40,
				desiredvCpus: 0,
				minvCpus: 0,
				maxvCpus: 0,
				instanceRole: batch_instance_role.roleArn,
				instanceTypes: [
						"m4.large",
						"m3.medium",
						"m3.large",
						"m5.large"
				],
				securityGroupIds: [ security_group.securityGroupId ],
				subnets: [ subnet.subnetId ],
				type: "SPOT",
				spotIamFleetRole: spot_fleet_role.roleArn
			}
		});

		new batch.CfnJobQueue(this, "BatchQueue", {
			jobQueueName: this.node.getContext("batch_queue_name"),
			priority: 2,
			computeEnvironmentOrder: [
				{
					computeEnvironment: batch_environment.computeEnvironmentArn,
					order: 1
				}
			]
		});
		new batch.CfnJobDefinition(this, "BatchJobDefine", {
			jobDefinitionName: this.node.getContext("batch_job_definition_name"),
			retryStrategy: {attempts: 1},
			timeout: {attemptDurationSeconds: 3600},
			containerProperties: {
				command: [ "bash", "-c", "outer_build" ],
				environment: [
					{
						name: "S3_OBJECT_URL",
						value: "dummy_value"
					}
				],
				jobRoleArn: batch_job_role.roleArn,
				image: this.node.getContext("docker_image_addr"),
				memory: 1024,
				vcpus: 1
			},
			type: "container"
		});

		new cdk.CfnOutput(this, "AuthBuildWithBatchPublicIP", {
			description: "The public Ip that auth build batch cluster will use, therefore which is need to be added to BitBucket whitelist",
			value: eip.eipIp
		})

	}
}

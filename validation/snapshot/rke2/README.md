# RKE2 Snapshot Configs

## Table of Contents
1. [Prerequisites](../README.md)
2. [Tests Cases](#Test-Cases)
3. [Configurations](#Configurations)
4. [Configuration Defaults](#defaults)
5. [Logging Levels](#Logging)
6. [Back to general snapshot](../README.md)

## Test Cases
All of the test cases in this package are listed below, keep in mind that all configuration for these tests have built in defaults [Configuration Defaults](#defaults). These tests will provision a cluster if one is not provided via the rancher.ClusterName field.


### Recurring Snapshot Test

#### Description:
The recurring snapshot test verifies that a cluster can create a series of snapshots. All configurations are not required if an already provisioned cluster is provided to the test.

#### Required Configurations:
1. [Cloud Credential](#cloud-credential-config)
2. [Cluster Config](#cluster-config)
3. [Machine Config](#machine-config)

#### Table Tests:
1. `RKE2_Recurring_Restores`

#### Run Commands:
1. `gotestsum --format standard-verbose --packages=github.com/rancher/tests/validation/snapshot/rke2 --junitfile results.xml --jsonfile results.json -- -tags=validation -run TestSnapshotRecurringTestSuite/TestSnapshotRecurringRestores -timeout=1h -v`


### Snapshot Restore Test

#### Description:
The snapshot restore test validates that snapshots can be created and restored without any failures or longterm disruption to workloads.

#### Required Configurations:
1. [Cloud Credential](#cloud-credential-config)
2. [Cluster Config](#cluster-config)
3. [Machine Config](#machine-config)

#### Table Tests:
1. `RKE2_Restore_ETCD`
2. `RKE2_Restore_ETCD_K8sVersion`
3. `RKE2_Restore_ETCD`

#### Run Commands:
1. `gotestsum --format standard-verbose --packages=github.com/rancher/tests/validation/snapshot/rke2 --junitfile results.xml --jsonfile results.json -- -tags=validation -run TestSnapshotRestoreTestSuite/TestSnapshotRestore -timeout=1h -v`


### Snapshot Retention Test

#### Description:
The snapshot retention test validates that the configured number of snapshots are retained and older snapshots are deleted as expected.

#### Required Configurations:
1. [Cloud Credential](#cloud-credential-config)
2. [Cluster Config](#cluster-config)
3. [Machine Config](#machine-config)

#### Table Tests:
1. `RKE2_Snapshot_Retention`

#### Run Commands:
1. `gotestsum --format standard-verbose --packages=github.com/rancher/tests/validation/snapshot/rke2 --junitfile results.xml --jsonfile results.json -- -tags=validation -run TestSnapshotRetentionTestSuite/TestSnapshotRetention -timeout=1h -v`


### Snapshot Windows Test

#### Description:
The snapshot windows test verifies that snapshots can be created and restored on a cluster containing windows nodes

#### Required Configurations:
1. [Cloud Credential](#cloud-credential-config)
2. [Cluster Config](#cluster-config)
3. [Machine Config](#machine-config)

#### Table Tests:
1. `RKE2_Windows_Restore`

#### Run Commands:
1. `gotestsum --format standard-verbose --packages=github.com/rancher/tests/validation/snapshot/rke2 --junitfile results.xml --jsonfile results.json -- -tags=validation -run TestSnapshotRestoreWindowsTestSuite/TestSnapshotRestoreWindows -timeout=1h -v`


### Snapshot S3 Test

#### Description:
The snapshot S3 test validates that snapshots can be stored and restored from an S3 bucket.

#### Required Configurations:
1. [Cloud Credential](#cloud-credential-config)
2. [Cluster Config](#cluster-config)
3. [Machine Config](#machine-config)
4. S3 configuration in etcd section of cluster config

#### Table Tests:
1. `RKE2_Snapshot_S3`

#### Run Commands:
1. `gotestsum --format standard-verbose --packages=github.com/rancher/tests/validation/snapshot/rke2 --junitfile results.xml --jsonfile results.json -- -tags=validation -run TestSnapshotS3TestSuite/TestSnapshotS3 -timeout=1h -v`

### Dualstack Snapshot Restore Test

#### Description:
The dualstack snapshot restore test validates that a cluster configured for dualstack networking can create and restore snapshots successfully.

#### Required Configurations:
1. [Cloud Credential](#cloud-credential-config)
2. [Cluster Config](#cluster-config)
3. [Machine Config](#machine-config)

#### Table Tests:
1. `RKE2_Dualstack_Snapshot_Restore`

#### Run Commands:
1. `gotestsum --format standard-verbose --packages=github.com/rancher/tests/validation/snapshot/rke2/dualstack --junitfile results.xml --jsonfile results.json -- -tags=validation -run TestSnapshotDualstackRestoreTestSuite/TestSnapshotDualstackRestore -timeout=1h -v`


### IPv6 Snapshot Tests

#### Description:
The IPv6 snapshot tests validate snapshot creation and restore functionality on clusters configured with IPv6 networking.

#### Required Configurations:
1. [Cloud Credential](#cloud-credential-config)
2. [Cluster Config](#cluster-config) (with IPv6 settings)
3. [Machine Config](#machine-config)

#### Table Tests:
1. `RKE2_IPv6_Restore_ETCD`
2. `RKE2_IPv6_Restore_ETCD_K8sVersion`
3. `RKE2_IPv6_Restore_Upgrade_Strategy`

#### Run Commands:
1. `gotestsum --format standard-verbose --packages=github.com/rancher/tests/validation/snapshot/rke2/ipv6 --junitfile results.xml --jsonfile results.json -- -tags=validation -run TestSnapshotIPv6RestoreTestSuite/TestSnapshotIPv6Restore -timeout=2h -v`

## Configurations

### Cluster Config
clusterConfig is needed to the run the all RKE2 tests. If no cluster config is provided all values have defaults.

**nodeProviders is only needed for custom cluster tests; the framework only supports custom clusters through aws/ec2 instances.**
```yaml
clusterConfig:
  machinePools:
  - machinePoolConfig:
      etcd: true
      controlplane: false
      worker: false
      quantity: 1
  - machinePoolConfig:
      etcd: false
      controlplane: true
      worker: false
      quantity: 1
  - machinePoolConfig:
      etcd: false
      controlplane: false
      worker: true
      quantity: 1
  kubernetesVersion: ""               #Permutable in dynamic tests. Leave blank for the latest kubernetes version
  provider: "aws"                     #Permutable in dynamic tests.
  nodeProvider: "ec2"
  hardened: false
  compliance: false                   #Set this to true for rancher versions with compliance
  psact: ""                           #either rancher-privileged|rancher-restricted|rancher-baseline
  
  etcd:
    disableSnapshot: false
    snapshotScheduleCron: "0 */5 * * *"
    snapshotRetain: 3
    s3:
      bucket: ""
      endpoint: "s3.us-east-2.amazonaws.com"
      endpointCA: ""
      folder: ""
      region: "us-east-2"
      skipSSLVerify: true
```

### Cloud Credential Config
Cloud credentials for various cloud providers.

#### AWS
```yaml
awsCredentials:                       #required (all) for AWS
  secretKey: ""
  accessKey: ""
  defaultRegion: ""
```

#### Digital Ocean
```yaml
digitalOceanCredentials:             #required (all) for DO
  accessToken": ""
```

#### Linode
```yaml
linodeCredentials:                    #required (all) for Linode
  token: ""
```

#### Azure
```yaml
azureCredentials:                     #required (all) for Azure
  clientId: ""
  clientSecret: ""
  subscriptionId": ""
  environment: "AzurePublicCloud"
```

#### Harvester
```yaml
harvesterCredentials:                 #required (all) for Harvester
  clusterId: ""
  clusterType: ""
  kubeconfigContent: ""
```

#### Google
```yaml
googleCredentials:                    #required (all) for Google
  authEncodedJson: ""
```

#### VSphere
```yaml
vmwarevsphereCredentials:             #required (all) for VMware
  password: ""
  username: ""
  vcenter: ""
  vcenterPort: ""
```

### Machine Config
Machine config is needed for tests that provision node driver clusters. 

#### AWS Machine Config
```yaml
awsMachineConfigs:                            #default              
  region: "us-east-2"
  awsMachineConfig:
  - roles: ["etcd","controlplane","worker"]
    ami: ""                                   #required
    instanceType: "t3a.medium"
    sshUser: "ubuntu"                         #required
    vpcId: ""                                 #required
    volumeType: "gp3"                         
    zone: "a"
    retries: "5"                              
    rootSize: "100"                            
    securityGroup: [""]                       #required                       
```

#### Digital Ocean Machine Config
```yaml
doMachineConfigs:                              #required (all)
  region: ""
  doMachineConfig:
  - roles: ["etcd","controlplane","worker"]
    image: "ubuntu-20-04-x64"
    backups: false
    ipv6: false
    monitoring: false
    privateNetworking: false
    size: "s-2vcpu-4gb"
    sshKeyContents: ""
    sshKeyFingerprint: ""
    sshPort: "22"
    sshUser: ""
    tags: ""
    userdata: ""
```

#### Linode Machine Config
```yaml
linodeMachineConfigs:                           #required (all)
  region: "us-west"
  linodeMachineConfig:
  - roles: ["etcd","controlplane","worker"]
    authorizedUsers: ""
    createPrivateIp: true
    dockerPort: "2376"
    image: "linode/ubuntu22.04"
    instanceType: "g6-standard-8"
    rootPass: ""
    sshPort: "22"
    sshUser: ""
    stackscript: ""
    stackscriptData: ""
    swapSize: "512"
    tags: ""
    uaPrefix: "Rancher"
```

#### Azure Machine Config
```yaml
azureMachineConfigs:                            #required (all)
  environment: "AzurePublicCloud"
  azureMachineConfig:
  - roles: ["etcd","controlplane","worker"]
    availabilitySet: "docker-machine"
    diskSize: "30"
    faultDomainCount: "3"
    image: "canonical:UbuntuServer:22.04-LTS:latest"
    location: "westus"
    managedDisks: false
    noPublicIp: false
    nsg: ""
    openPort: ["6443/tcp", "2379/tcp", "2380/tcp", "8472/udp", "4789/udp", "9796/tcp", "10256/tcp", "10250/tcp", "10251/tcp", "10252/tcp"]
    resourceGroup: "docker-machine"
    size: "Standard_D2_v2"
    sshUser: ""
    staticPublicIp: false
    storageType: "Standard_LRS"
    subnet: ""
    subnetPrefix: "x.x.x.x/xx"
    updateDomainCount: "5"
    usePrivateIp: false
    vnet: ""
```

#### Harvester Machine Config
```yaml
harvesterMachineConfigs:                        #required (all)
  vmNamespace: "default"
  harvesterMachineConfig:
  - roles: ["etcd","controlplane","worker"]
    diskSize: "40"
    cpuCount: "2"
    memorySize: "8"
    networkName: ""
    imageName: ""
    sshUser: ""
    diskBus: "virtio
```

#### Vsphere Machine Config
```yaml
vmwarevsphereMachineConfigs:                    #required (all)
    datacenter: "/<datacenter>"
    hostSystem: "/<datacenter>/path-to-host"
    datastore: "/<datacenter>/path-to-datastore" 
    datastoreURL: "ds:///<url>"             
    folder: "/<datacenter>/path-to-vm-folder" 
    pool: "/<datacenter>/path-to-resource-pool" 
    vmwarevsphereMachineConfig:
    - cfgparam: ["disk.enableUUID=TRUE"]
      cloudConfig: "#cloud-config\n\n"
      customAttribute: []
      tag: []
      roles: ["etcd","controlplane",worker]
      creationType: "template"
      os: "linux"
      cloneFrom: "/<datacenter>/path-to-linux-image"
      cloneFromWindows: "/<datacenter>/path-to-windows-image"
      contentLibrary: ""                                        
      datastoreCluster: ""
      network: ["/<datacenter>/path-to-vm-network"]
      sshUser: ""
      sshPassword: ""                                           
      sshUserGroup: ""
      sshPort: "22"
      cpuCount: "4"
      diskSize: "40000"
      memorySize: "8192"
```

#### Custom Cluster Config
Custom clusters are only supported on AWS.
```yaml
  awsEC2Configs:
    region: "us-east-2"
    awsSecretAccessKey: ""
    awsAccessKeyID: ""
    awsEC2Config:
      - instanceType: "t3a.medium"
        awsRegionAZ: ""
        awsAMI: ""
        awsSecurityGroups: [""]
        awsSubnetID: ""
        awsSSHKeyName: ""
        awsCICDInstanceTag: "rancher-validation"
        awsIAMProfile: ""
        awsUser: "ubuntu"
        volumeSize: 50
        roles: ["etcd", "controlplane", "worker"]
      - instanceType: "t3a.xlarge"
        awsAMI: ""
        awsSecurityGroups: [""]
        awsSubnetID: ""
        awsSSHKeyName: ""
        awsCICDInstanceTag: "rancher-validation"
        awsUser: "Administrator"
        volumeSize: 50
        roles: ["windows"]
```

## Defaults
This package contains a defaults folder which contains default test configuration data for non-sensitive fields. The goal of this data is to: 
1. Reduce the number of fields the user needs to provide in the cattle_config file. 
2. Reduce the amount of yaml data that needs to be stored in our pipelines.
3. Make it easier to run tests

Any data the user provides will override these defaults which are stored here: [defaults](defaults/defaults.yaml). 

## Logging
This package supports several logging levels. You can set the logging levels via the cattle config and all levels above the provided level will be logged while all logs below that logging level will be omitted. 

```yaml
logging:
   level: "trace" #trace debug, info, warning, error
```

## Additional
1. If the tests passes immediately without warning, try adding the `-count=1` or run `go clean -cache`. This will avoid previous results from interfering with the new test run.
2. All of the tests utilize parallelism when running for more finite control of how things are run in parallel use the -p and -parallel.
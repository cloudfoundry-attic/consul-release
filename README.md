# consul-release
---

This is a [BOSH](http://bosh.io) release for [consul](https://github.com/hashicorp/consul).

* [CI](https://mega.ci.cf-app.com/pipelines/consul)
* [Roadmap](https://www.pivotaltracker.com/n/projects/1382120)

###Contents

1. [Deploying](#deploying)
2. [Running Tests](#running-tests)

## Deploying

In order to deploy consul-release you must follow the standard steps for deploying software with BOSH.

We assume you have already deployed and targeted a BOSH director. For more instructions on how to do that please see the [BOSH documentation](http://bosh.io/docs).

### 1. Uploading a stemcell

Find the "BOSH Lite Warden" stemcell you wish to use. [bosh.io](https://bosh.io/stemcells) provides a resource to find and download stemcells.  Then run `bosh upload release STEMCELL_URL_OR_PATH_TO_DOWNLOADED_STEMCELL`.

### 2. Creating a release

From within the consul-release director run `bosh create release --force` to create a development release.

### 3. Uploading a release

Once you've created a development release run `bosh upload release` to upload your development release to the director.

### 4. Generating a deployment manifest

We provide a set of scripts and templates to generate a simple deployment manifest. You should use these as a starting point for creating your own manifest, but they should not be considered comprehensive or production-ready.

In order to automatically generate a manifest you must have installed [spiff](https://github.com/cloudfoundry-incubator/spiff).  Once installed, manifests can be generated using `./scripts/generate_consul_deployment_manifest [STUB LIST]` with the provided stubs:

1. director_uuid_stub
	
	The director_uuid_stub provides the uuid for the currently targeted BOSH director.
	```yaml
	---
	director_uuid: DIRECTOR_UUID
	```
2. instance_count_stub

	The instance count stub provides the ability to overwrite the number of instances of consul to deploy. The minimal deployment of consul is shown below:
	```yaml
	---
	instance_count_overrides:
	  consul_z1:
	    instances: 1
	  consul_z2:
	    instances: 0
	```

	NOTE: at no time should you deploy only 2 instances of consul.
3. persistent_disk_stub

	The persistent disk stub allows you to override the size of the persistent disk used in each instance of the consul job. If you wish to use the default settings provide a stub with only an empty hash:
	```yaml
	---
	persistent_disk_overrides: {}
	```
	
	To override disk sizes the format is as follows
	```yaml
	---
	persistent_disk_overrides:
	  consul_z1: 1234
	  consul_z2: 1234	
	```
	
4. iaas_settings

	The IaaS settings stub contains IaaS-specific settings, including networks, cloud properties, and compilation properties. Please see the BOSH documentation for setting up networks and subnets on your IaaS of choice. We currently allow for three network configurations on your IaaS: consul1, consul2, and compilation. You must also specify the stemcell to deploy against as well as the version (or latest).
	
We provide [default stubs for a BOSH-Lite deployment](https://github.com/cloudfoundry-incubator/consul-release/blob/master/manifest-generation/bosh-lite-stubs).  Specifically:

* instance_count_stub: [manifest-generation/bosh-lite-stubs/instance-count-overrides.yml](manifest-generation/bosh-lite-stubs/instance-count-overrides.yml)
* persistent_disk_stub: [manifest-generation/bosh-lite-stubs/persistent-disk-overrides.yml](manifest-generation/bosh-lite-stubs/persistent-disk-overrides.yml)
* iaas_settings: [manifest-generation/bosh-lite-stubs/iaas-settings-consul.yml](manifest-generation/bosh-lite-stubs/iaas-settings-consul.yml)

[Optional]

1. If you wish to override the name of the release and the deployment (default: consul) you can provide a release_name_stub with the following format:
	
	```yaml
	---
	name_overrides:
	  release_name: NAME
	  deployment_name: NAME
	```

Output the result of the above command to a file: `./scripts/generate_consul_deployment_manifest [STUB LIST] > OUTPUT_MANIFEST_PATH`.

### 5. Deploy.

Run `bosh -d OUTPUT_MANIFEST_PATH deploy`.

## Confab Tests

Run the `confab` tests by executing the `src/confab/scripts/test` executable.

## Acceptance Tests

The acceptance tests deploy a new consul cluster and exercise a variety of features, including scaling the number of nodes, as well as destructive testing to verify resilience.

### Prerequisites

The following should be installed on the local machine:

- Consul
- Golang (>= 1.5)

If using homebrew, these can be installed with:

```
brew install consul go
```

### Network setup

#### BOSH-Lite

Make sure you’ve run `bin/add-route`.
This will setup some routing rules to give the tests access to the consul VMs.

#### AWS

You will want to run your tests from a VM within the same subnet as determined in your iaas-settings stub.
This assumes you are using a private subnet within a VPC.

### Environment setup

This repository assumes that it is the root of your `GOPATH`. You can set this up by doing the following:

```shell
source .envrc
```

Or if you have `direnv` installed:

```shell
direnv allow
```

### Running the CONSATS

Run all the tests with:

```
CONSATS_CONFIG=[config_file.json] ./scripts/test_default
```

Run a specific set of tests with:

```
CONSATS_CONFIG=[config_file.json] ./scripts/test <some test packages>
```

The `CONSATS_CONFIG` environment variable points to a configuration file which specifies the endpoint of the BOSH director and the path to your `iaas_settings` stub.

See below for more information on the contents of this configuration file.

### CONSATS config

An example config json for BOSH-lite would look like:

```json
cat > integration_config.json << EOF
{
  "bosh_target": "192.168.50.4",
  "iaas_settings_consul_stub_path": "${PWD}/src/acceptance-tests/manifest-generation/bosh-lite-stubs/iaas-settings-consul.yml",
  "iaas_settings_turbulence_stub_path": "${PWD}/src/acceptance-tests/manifest-generation/bosh-lite-stubs/iaas-settings-turbulence.yml",
  "turbulence_properties_stub_path": "${PWD}/src/acceptance-tests/manifest-generation/bosh-lite-stubs/turbulence/property-overrides.yml",
  "cpi_release_location": "https://bosh.io/d/github.com/cppforlife/bosh-warden-cpi-release?v=28",
  "cpi_release_name": "bosh-warden-cpi",
  "bind_address": "192.168.50.1",
  "turbulence_release_location": "http://bosh.io/d/github.com/cppforlife/turbulence-release?v=0.4"
}
EOF
export CONSATS_CONFIG=$PWD/integration_config.json
```

NOTE: when specifying locations on the filesystem, it is best to provide an absolute path.

The full set of config parameters is explained below:
* `bosh_target` (required) Public BOSH IP address that will be used to host test environment.
* `bind_address` (required) IP that the local consul node will use to connect to the cluster. See note below for info about bosh-lite
* `iaas_settings_consul_stub_path` (required) Stub containing iaas settings for the consul deployment.
* `iaas_settings_turbulence_stub_path` (required for turbulence tests) Stub containing iaas setting for the turbulence deployment.
* `turbulence_properties_stub_path` (required for turbulence tests) Stub containing property overrides for the turbulence deployment.
* `turbulence_release_location` (required for turbulence tests) Location of the turbulence release to use for the tests (version 0.4 or higher required).
* `cpi_release_location` (required for turbulence tests) CPI for the current BOSH director being used to deploy tests with (version 28 or higher required).
* `cpi_release_name` (required for turbulence tests) Name for the `cpi_release_location` parameter
* `bosh_operation_timeout` (optional) Time to wait for BOSH commands to exit before erroring out. (default time is 5 min if not specified)
* `turbulence_operation_timeout` (optional) Time to wait for Turbulence operations to succeed before erroring out. (default time is 5 min if not specified)

Note: When running against bosh-lite the IP specified for `bind_address` must be in the 192.168.50.0/24 subnet. This is due how the consul agent and servers communicate 
and determine whether a source is trusted. This differs from most bosh-lite networking where the 10.244.0.0/24 subnet is used.

Note: You must ensure that the stemcells specified in `iaas_settings_consul` and `iaas_settings_turbulence_stub_path` are already uploaded to the director at `bosh_target`.

Note: The ruby `bundler` gem is used to install the correct version of the `bosh_cli`, as well as to decrease the `bosh` startup time. 

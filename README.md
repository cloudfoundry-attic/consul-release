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

## Running Tests

We have written a test suite that exercises spinning up single/multiple consul server instances, scaling them,
and perform rolling deploys. If you have already installed Go, you can run `CONSATS_CONFIG=[config_file.json] ./scripts/test_default`.
`./scripts/test_default` calls `./scripts/test`, which installs all dependancies and appends the release directory
to the gopath. `./scripts/test_default` passes all test suites to `./scripts/test`. If you wish to run a specific test suite
you can call `./scripts/test` directly like so: `CONSATS_CONFIG=[config_file.json] ./scripts/test src/acceptance_tests/deploy/`.

The CONSATS_CONFIG environment variable points to a configuration file which specifies the endpoint of the BOSH
director and the path to your iaas_settings stub. An example config json for BOSH-lite would look like:

```json
cat > integration_config.json << EOF
{
  "bosh_target": "192.168.50.4",
  "iaas_settings_consul_stub_path": "./src/acceptance-tests/manifest-generation/bosh-lite-stubs/iaas-settings-consul.yml",
  "iaas_settings_turbulence_stub_path": "./src/acceptance-tests/manifest-generation/bosh-lite-stubs/iaas-settings-turbulence.yml",
  "turbulence_properties_stub_path": "./src/acceptance-tests/manifest-generation/bosh-lite-stubs/turbulence/property-overrides.yml",
  "cpi_release_url": "https://bosh.io/d/github.com/cppforlife/bosh-warden-cpi-release?v=21",
  "cpi_release_name": "bosh-warden-cpi",
  "bind_address": "192.168.50.1",
  "turbulence_release_url": "http://bosh.io/d/github.com/cppforlife/turbulence-release?v=0.4"
}
EOF
export CONSATS_CONFIG=$PWD/integration_config.json
```

The full set of config parameters is explained below:
* `bosh_target` (required) Public BOSH IP address that will be used to host test environment.
* `bind_address` (required) IP that the local consul node will use to connect to the cluster. See note below for info about bosh-lite
* `iaas_settings_consul_stub_path` (required) Stub containing iaas settings for the consul deployment.
* `iaas_settings_turbulence_stub_path` (required for turbulence tests) Stub containing iaas setting for the turbulence deployment.
* `turbulence_properties_stub_path` (required for turbulence tests) Stub containing property overrides for the turbulence deployment.
* `cpi_release_url` (required for turbulence tests) CPI for the current BOSH director being used to deploy tests with.
* `cpi_release_name` (required for turbulence tests) Name for the `cpi_release_url` parameter
* `bosh_operation_timeout` (optional) Time to wait for BOSH commands to exit before erroring out. (default time is 5 min if not specified)
* `turbulence_operation_timeout` (optional) Time to wait for Turbulence operations to succeed before erroring out. (default time is 5 min if not specified)

Note: When running against bosh-lite the IP specified for `bind_address` must be in the 192.168.50.0/24 subnet. This is due how the consul agent and servers communicate 
and determine whether a source is trusted. This differs from most bosh-lite networking where the 10.244.0.0/24 subnet is used.

Note: You must ensure that the stemcells specified in `iaas_settings_consul` and `iaas_settings_turbulence_stub_path` are already uploaded to the director at `bosh_target`.

Note: The ruby `bundler` gem is used to install the correct version of the `bosh_cli`, as well as to decrease the `bosh` startup time. 

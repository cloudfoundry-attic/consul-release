# consul-release
---

This is a [BOSH](http://bosh.io) release for [consul](https://github.com/hashicorp/consul).

* [CI](https://mega.ci.cf-app.com/pipelines/consul)
* [Roadmap](https://www.pivotaltracker.com/n/projects/1382120)

###Contents

* [Using Consul](#using-consul)
* [Deploying](#deploying)
* [Confab Tests](#confab-tests)
* [Acceptance Tests](#acceptance-tests)
* [Known Issues](#known-issues)
* [Disaster Recovery](#disaster-recovery)

## Using Consul

Consul is a distributed key-value store that provides a host of applications.
It can be used to provide service discovery, key-value configuration,
and distributed locks within cloud infrastructure environments.

### Within CloudFoundry

Principally, Consul is used to provide service discovery for many of
the components. Components can register services with Consul, making these
services available to other CloudFoundry components. A component looking to
discover other services would run a Consul agent locally, and lookup services
using DNS names. Consul transparently updates the DNS records across the cluster
as services start and stop, or pass/fail their health checks.

Additionally, Consul is able to store key-value data across its distributed
cluster. CloudFoundry makes use of this feature by storing some simple
configuration data, making it reachable across all nodes in the cluster.

CloudFoundry also makes some use of Consul's distributed locks.
This feature is used to ensure that one, and only one, component is able to
perform some critical action at a time.

### Fault Tolerance and Data Durability

Consul is a distributed data-store and as such, conforms to some form of fault
tolerance under disadvantageous conditions. Broadly, these tolerances are
described by the [CAP Theorem](https://en.wikipedia.org/wiki/CAP_theorem),
specifying that a distributed computer system cannot provide all three of the
guarantees outlined in the theorem (consistency, availability,
and partition tolerance). In the default configuration, Consul has a preference
to guarantee consistency and partition tolerance over availability. This means
that under network partioning, the cluster can become unavailable. The
unavailability of the cluster can result in the inability to write to the
key-value store, maintain or acquire distributed locks, or discover other
services. Consul makes this tradeoff with a preference for consistency of the
stored data in the case of network partitions. The Consul team has published
some [results](https://www.consul.io/docs/internals/jepsen.html) from their
testing of Consul's fault tolerance.

This behavior means that Consul may not be the best choice for persisting
critically important data. Not having explicitly supported backup-and-restore
workflows also makes guaranteeing data durability difficult.

## Deploying

In order to deploy consul-release you must follow the standard steps for deploying software with BOSH.

We assume you have already deployed and targeted a BOSH director. For more instructions on how to do that please see the [BOSH documentation](http://bosh.io/docs).

### 1. Uploading a stemcell

Find the "BOSH Lite Warden" stemcell you wish to use. [bosh.io](https://bosh.io/stemcells) provides a resource to find and download stemcells.  Then run `bosh upload release STEMCELL_URL_OR_PATH_TO_DOWNLOADED_STEMCELL`.

### 2. Creating a release

From within the consul-release director run `bosh create release --force` to create a development release.

### 3. Uploading a release

Once you've created a development release run `bosh upload release` to upload your development release to the director.

### 4. Using a sample deployment manifest

We provide a set of sample deployment manifests that can be used as a starting point for creating your own manifest, but they should not be considered comprehensive. They are located in manifests/aws and manifests/bosh-lite.

### 5. Deploy.

Run `bosh -d OUTPUT_MANIFEST_PATH deploy`.

## Confab Tests

Run the `confab` tests by executing the `src/confab/scripts/test` executable.

## Acceptance Tests

The acceptance tests deploy a new consul cluster and exercise a variety of features, including scaling the number of nodes, as well as destructive testing to verify resilience.

### Prerequisites

The following should be installed on the local machine:

- jq
- Consul
- Golang (>= 1.5)

If using homebrew, these can be installed with:

```
brew install consul go jq
```

### Network setup

#### BOSH-Lite

Make sure you’ve run `bin/add-route`.
This will setup some routing rules to give the tests access to the consul VMs.

#### AWS

You will want to run CONSATS from a VM within the same subnet specified in your manifest.
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

#### Running locally

Run all the tests with:

```
CONSATS_CONFIG=[config_file.json] ./scripts/test
```

Run a specific set of tests with:

```
CONSATS_CONFIG=[config_file.json] ./scripts/test <some test packages>
```

The `CONSATS_CONFIG` environment variable points to a configuration file which specifies the endpoint of the BOSH director.
When specifying location of the CONSATS_CONFIG, it must be an absolute path on the filesystem.

See below for more information on the contents of this configuration file.

### CONSATS config

An example config json for BOSH-lite would look like:

```json
cat > integration_config.json << EOF
{
  "bosh":{
    "target": "192.168.50.4",
    "username": "admin",
    "password": "admin"
  }
}
EOF
export CONSATS_CONFIG=$PWD/integration_config.json
```

The full set of config parameters is explained below:
* `bosh.target` (required) Public BOSH IP address that will be used to host test environment
* `bosh.username` (required) Username for the BOSH director login
* `bosh.password` (required) Password for the BOSH director login
* `bosh.director_ca_cert` BOSH Director CA Cert
* `aws.subnet` Subnet ID for AWS deployments
* `aws.access_key_id` Key ID for AWS deployments
* `aws.secret_access_key` Secret Access Key for AWS deployments
* `aws.default_key_name` Default Key Name for AWS deployments
* `aws.default_security_groups` Security groups for AWS deployments
* `aws.region` Region for AWS deployments
* `registry.host` Host for the BOSH registry
* `registry.port` Port for the BOSH registry
* `registry.username` Username for the BOSH registry
* `registry.password` Password for the BOSH registry

#### Running as BOSH errand

##### Dependencies

The `acceptance-tests` BOSH errand assumes that the BOSH director has already uploaded the correct versions of the dependent releases.
The required releases are:
* [turbulence-release](http://bosh.io/releases/github.com/cppforlife/turbulence-release?version=0.4)
* [consul-release](http://bosh.io/releases/github.com/cloudfoundry-incubator/consul-release) or `bosh create release && bosh upload release`

For BOSH-Lite:
* [bosh-warden-cpi-release](http://bosh.io/releases/github.com/cppforlife/bosh-warden-cpi-release?version=28)

For AWS:
* [bosh-aws-cpi-release](http://bosh.io/releases/github.com/cloudfoundry-incubator/bosh-aws-cpi-release?version=39)

##### Creating a consats deployment manifest

We provide an example deployment manifest for running the errand on AWS.
The manifest can be used by replacing all of the placeholder values in the file `manifests/aws/consats.yml`.

##### Deploying the errand

Run `bosh deployment manifests/aws/consats.yml`.
Run `bosh deploy`.

##### Running the errand

Run `bosh run errand acceptance-tests`

## Known Issues

### 1-node clusters

It is not recommended to run a 1-node cluster in any "production" environment.
Having a 1-node cluster does not ensure any amount of data persistence.

WARNING: Scaling your cluster to or from a 1-node configuration may result in data loss.

## Disaster Recovery

In the event that the consul cluster ends up in a bad state that is difficult
to debug, you have the option of stopping the consul agent on each server node,
removing its data store, and then restarting the process:

```
monit stop consul_agent (on all server nodes in consul cluster)
rm -rf /var/vcap/store/consul_agent/* (on all server nodes in consul cluster)
monit start consul_agent (one-by-one on each server node in consul cluster)
```

There are often more graceful ways to solve specific issues, but it is hard
to document all of the possible failure modes and recovery steps. As long as
your consul cluster does not contain critical data that cannot be repopulated,
this option is safe and will probably get you unstuck.

Additional information about outage recovery can be found on the consul
[documentation page](https://www.consul.io/docs/guides/outage.html).

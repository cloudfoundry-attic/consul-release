# consul-release
---

This is a [bosh](http://bosh.io) release for [consul](https://github.com/hashicorp/consul).

* CI is currently being built
* [Roadmap](https://www.pivotaltracker.com/n/projects/1382120)

###Contents

1. [Usage](#usage)
2. [Deploying](#deploying)
3. [Running Tests](#running-tests)
4. [Advanced](#advanced)


#Usage

* Currently this release is not consumed by Cloud Foundry as a stand-alone bosh release. This is expected to change in the near future
* Consul cannot yet be deployed independantly of Cloudfoundry. This will happen very soon.

---
#Deploying
In order to deploy consul-release you must follow the standard steps for deploying software with bosh.

We assume you have already deployed and targeted a bosh director. For more instructions on how to do that please see the [bosh documentation](http://bosh.io/docs).

###1. Uploading a stemcell
Find the "BOSH Lite Warden" stemcell you wish to use. [bosh.io](https://bosh.io/stemcells) provides a resource to find and download stemcells.  Then run `bosh upload release PATH_TO_DOWNLOADED_STEMCELL`.

###2. Creating a release
From within the consul-release director run `bosh create release --force` to create a development release.

###3. Uploading a release
Once you've created a development release run `bosh upload release` to upload your development release to the director.

###4. Generating a deployment manifest
We provide a set of scripts and templates to generate a simple deployment manifest. You should use these as a starting point for creating your own manifest, but they should not be considered comprehensive or production-ready.

In order to automatically generate a manifest you must have installed the following dependencies:

1. [Spiff](https://github.com/cloudfoundry-incubator/spiff)

Once installed, manifests can be generated using `./scripts/generate_consul_deployment_manifest [STUB LIST]` with the provided stubs:

1. uuid_stub
	
	The uuid_stub provides the uuid for the currently targeted bosh director.
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

	Remember, at no time should you deploy only 2 instances of consul.
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

	The iaas settings stub contains iaas specific settings, including networks, cloud properties, and compilation properties. Please see the bosh documentation for setting up networks and subnets on your IaaS of choice. We currently allow for three network configurations on your IaaS: consul1, consul22, and compilation. You must also specify the stemcell to deploy against as well as the version (or latest).
	
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

###5. Deploy
Output the result of the above command to a file: `./scripts/generate_consul_deployment_manifest [STUB LIST] > consul.yml`.  Then run `bosh -d consul.yml deploy`.
	
---
#Advanced

### Blobstore Creation

Users who simply wish to consume and deploy consul-release do not need to create
their own bosh blobstore. If, however, you plan on forking and developing
your own consul-release seperate from the Cloudfoundry organization you must
configure your own aws account and blobstore. `scripts/configure-aws` creates
an s3 bucket and associated IAM user and fills the details of those resources
out in `final.yml` and `private.yml` files. The `configure-aws` script takes 2 inputs, in order.
The first argument is the location of your private deployment directory which contains an aws_environment file.
For on the deployment directories please see [Deployment Directory Details](https://github.com/cloudfoundry/mega-ci#deployment-directory-details).
The second argument is the location of a directory where you would like to store
your bosh `private.yml` file. For more on bosh `private.yml` files please see
[adding blobs on bosh.io](https://bosh.io/docs/create-release.html#blobs).

The cloudformation template used to configure the aws stack with your blobstore
can be found in `templates/aws/consul-blobs-bucket.json`. Due to s3 bucket names
being globally unique, you must first change the [bucket name](link to line with bucket-name)
to your desired name in the cloudformation template.

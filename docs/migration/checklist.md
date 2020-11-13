# Migration Checklist

## Build and Deploy
  - [ ] Dockerfile
  - [ ] build_deploy.sh
  - [ ] pr_check.sh
  - [ ] App interface entries for [Jenkins jobs](https://github.com/RedHatInsights/clowder/tree/master/docs/migration#create-pr-check-and-build-master-jenkins-jobs-in-app-interface)
  - [ ] saas-deploy file [for Bonfire](https://github.com/RedHatInsights/clowder/tree/master/docs/migration#create-new-saas-deploy-file)
#### Build and Deploy Parameters
  - [ ] ENV_NAME
  - [ ] CLOWDER_ENABLED
  - [ ] MIN_REPLICAS
## Code changes to conusme in [app config](https://github.com/RedHatInsights/clowder/tree/master/docs/migration#code-changes-to-consume-configuration)
  - [ ] Kafka topics and bootstrap url
  - [ ] Object store buckets (minio, s3)
  - [ ] Object store library default to minio not boto
  - [ ] RDS Databases
  - [ ] Redis Databases
  - [ ] Logging via Cloudwatch
  - [ ] Web path and port 
  - [ ] Metrics path and port  
  - [ ] Dependent services (rbac, etc) 
## Clowdapp.yml resources and [types](https://github.com/RedHatInsights/clowder/blob/master/apis/cloud.redhat.com/v1alpha1/clowdapp_types.go)
  - [ ] Image spec
  - [ ] Resource requirements
  - [ ] Command arguments
  - [ ] Environment vars
  - [ ] Liveness and Readiness probes
  - [ ] Volumes and volume mounts
  - [ ] Kafka topics
  - [ ] Web port
  - [ ] Databases
  - [ ] In-memory Db
  - [ ] Dependent services (rbac, etc)
## Backwards Compatability
  - [ ] e2e build disabled (after build pipeline is validated)
  - [ ] Disable 'quay copier' build config for app
  - [ ] Update qa/smoke images to pull from quay
  - [ ] Disable the Prod env option in your app's jenkins deploy

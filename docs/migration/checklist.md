# Migration Checklist

### The Source Repo has 
  - [ ] Dockerfile
  - [ ] build_deploy.sh
  - [ ] pr_check.sh
  - [ ] e2e build disabled (after build pipeline is validated)
### App config (check if not applicable)
  - [ ] Kafka topics and bootstrap url
  - [ ] Object store buckets (minio, s3)
  - [ ] Databases (RDS, Redis)
  - [ ] Logging via Cloudwatch
  - [ ] Web path and port 
  - [ ] Metrics path and port  
  - [ ] Dependent services (rbac, etc) 
### Clowdapp resource
  - [ ] Image spec
  - [ ] Resource requirements
  - [ ] Command arguments
  - [ ] Environment vars
  - [ ] Liveness and Readiness probes
  - [ ] Volumes and volume mounts
### Parameters
  - [ ] ENV_NAME
  - [ ] CLOWDER_ENABLED
  - [ ] MIN_REPLICAS

# Migration Checklist

> Suggested order is top to bottom

## App Changes - Settings, Config, and ClowdApp (if applicable)
  - [ ] database (RDS) is connected via clowder config and is present in ClowdApp
  - [ ] inMemoryDb (Redis) is connected via clowder config and is present in ClowdApp
  - [ ] kafka is connected via clowder, topics are present in Clowdapp
  - [ ] logging is sent to cloudwatch via clowder
  - [ ] objectstores are connected via clowder, minio is used instead of boto
  - [ ] webport is present in Clowdapp
  - [ ] dependent services are present in Clowdapp
  - [ ] CLOWDER_ENABLED can toggle backwards compatability in app config
  - [ ] ENV_NAME is parameterized and set to required
  - [ ] CLOWDER_ENABLED is parameterized and set to required
  - [ ] MIN_REPLICAS is parameterized

## Build and Deploy
  - [ ] Dockerfile is up to date
  - [ ] build_deploy.sh is pushing to quay
  - [ ] pr_check.sh is building in ephemeral env and passing local tests
  - [ ] App interface entries for [Jenkins jobs are running](https://github.com/RedHatInsights/clowder/tree/master/docs/migration#create-pr-check-and-build-master-jenkins-jobs-in-app-interface)
  - [ ] saas-deploy file [for Bonfire is enabled](https://github.com/RedHatInsights/clowder/tree/master/docs/migration#create-new-saas-deploy-file)

## Backwards Compatability
  - [ ] e2e builds are disabled
  - [ ] 'quay copier' is disabled in build config for app
  - [ ] qa/smoke images pull from quay
  - [ ] Prod env option is disabled in your app's jenkins deploy


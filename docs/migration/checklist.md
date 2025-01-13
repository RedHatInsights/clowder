# Migration Checklist

Suggested order is top to bottom. App changes with PR approved by team, 
Build changes with PR approved by team, once validated, turn off e2e builds 
and setup backwards compatability

## App Changes - Settings, Config, and ClowdApp (if applicable)
* [ ] Database (RDS) is connected via clowder config and is present in ``ClowdApp``
* [ ] InMemoryDb (Redis) is connected via clowder config and is present in ``ClowdApp``
* [ ] Kafka is connected via clowder, topics are present in ``Clowdapp``
* [ ] Logging is sent to cloudwatch via clowder
* [ ] Objectstores are connected via clowder, minio is used instead of boto
* [ ] Webport is present in ``Clowdapp``
* [ ] Dependent services are present in ``Clowdapp``
* [ ] Optional dependent services are present in ``Clowdapp``
* [ ] ``CLOWDER_ENABLED`` can toggle backwards compatability in app config
* [ ] ``ENV_NAME`` is parameterized and set to required
* [ ] ``MIN_REPLICAS`` is parameterized

## Build and Deploy
* [ ] ``Dockerfile`` is up to date
* [ ] ``build_deploy.sh`` is pushing to quay
* [ ] ``pr_check.sh`` is building in ephemeral env and passing local tests
* [ ] App interface entries for [Jenkins jobs are running](https://github.com/RedHatInsights/clowder/blob/master/docs/migration/migration.md#create-pr-check-and-build-master-jenkins-jobs-in-app-interface)
* [ ] saas-deploy file [for Bonfire is enabled](https://github.com/RedHatInsights/clowder/blob/master/docs/migration/migration.md#create-new-saas-deploy-file)

## Backwards Compatibility
* [ ] e2e builds are disabled
* [ ] ``quay copier`` is disabled in build config for app
* [ ] qa/smoke images pull from quay
* [ ] Prod env option is disabled in app's jenkins deploy

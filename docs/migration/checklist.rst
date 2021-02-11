Migration Checklist
===================

   Suggested order is top to bottom. App changes with PR approved by
   team, Build changes with PR approved by team, once validated, turn
   off e2e builds and setup backwards compatability

App Changes - Settings, Config, and ClowdApp
--------------------------------------------
-  database (RDS) is connected via clowder config and is present in
   ClowdApp
-  inMemoryDb (Redis) is connected via clowder config and is present
   in ClowdApp
-  kafka is connected via clowder, topics are present in Clowdapp
-  logging is sent to cloudwatch via clowder
-  objectstores are connected via clowder, minio is used instead of
   boto
-  webservices are present in Clowdapp
-  dependent services are present in Clowdapp
-  CLOWDER_ENABLED can toggle backwards compatability in app config
-  ENV_NAME is parameterized and set to required
-  CLOWDER_ENABLED is parameterized and set to required
-  MIN_REPLICAS is parameterized

Build and Deploy
----------------

-  dockerfile is up to date
-  build_deploy.sh is pushing to quay
-  pr_check.sh is building in ephemeral env and passing local tests
-  app interface entries for `jenkins jobs are running`_
-  saas-deploy file `for Bonfire is enabled`_

Backwards Compatibility
-----------------------

-  e2e builds are disabled
-  ‘quay copier’ is disabled in build config for app
-  qa/smoke images pull from quay
-  Prod env option is disabled in your app’s jenkins deploy

.. _jenkins jobs are running: https://github.com/RedHatInsights/clowder/tree/master/docs/migration#create-pr-check-and-build-master-jenkins-jobs-in-app-interface
.. _for Bonfire is enabled: https://github.com/RedHatInsights/clowder/tree/master/docs/migration#create-new-saas-deploy-file

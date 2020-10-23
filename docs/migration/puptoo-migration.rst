Puptoo Migration to Clowder and Bonfire
=======================================

* Code changes
    * Need to be able to configure app with and without clowder
    * Feature flags?
* Create new deployment template
    * Lives in code repo
    * Consists of ClowdApp and any ConfigMap or Secret required for app to run
    * Real secrets will live in vault and be injected via app-interface
* saas-deploy file changes
    * Need a new saas-deploy file for clowder-based deployments
* Add image build jenkins job to app-interface
    * Public github repos use ci-ext, Gitlab repos use ci-int
    * Private github repos NOT SUPPORTED
* Add build_deploy.sh and pr_check.sh to project
    * copypasta some examples
* Add ephemeral target to saas-deploy file
    * copypasta

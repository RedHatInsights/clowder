..  _confighashprovider:

ConfigHash Provider
===================

The **ConfigHash Provider** is responsible for creating the configuration hash annotation on the
``Deployment`` resource. Changes to configuration then modify the deployment resource's template
annotations and thereby restart pods, forcing them to pick up the new configuration.

There is no configuration for this provider.

# Dependencies Provider

The **Dependencies Provider** is responsible for passing the list of dependent
enpoints through into the app configuration.

## ClowdApp Configuration

There are two kinds of dependency, optional and mandatory. With mandatory
dependencies, the application will not complete reconciliation unless the
dependency exists. With an optional dependency, the application will complete
reconciliation and will have its configuration updated should the dependency
come online. This is performed via a config hash annotation update on the
deployment template.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  # Other App Config
  dependencies:
  - app_name1
  optionalDependencies:
  - app_name2
```

## ClowdEnv Configuration

There are no configuration options for this provider.

## Generated App Configuration

The Endpoint appear in the cdappconfig.json with the following structure. 

A client helper is available for the endpoints and privateEndpoints.

### JSON structure

```yaml
{
  "endpoints": [
    {
      "name": "deployment1",
      "app": "app_name1",
      "hostname": "deployment1.svc",
      "port": 8000
    },
    {
      "name": "deployment2",
      "app": "app_name2",
      "hostname": "deployment2.svc",
      "port": 8000
    },
  ],
  "privateEndpoints": [
    {
      "name": "deployment1",
      "app": "app_name1",
      "hostname": "deployment1.svc",
      "port": 10000
    },
  ]
}
```

### Client access

### Client helpers

The following helpers present a nested dictionary type structure allowing the
client to look up an endpoint via the app/deployment name. As an example:

```python
LoadedConfig.dependencyEndpoints["app_name1"]["deployment1"]
```

For supported languages, the dependency configuration is access via the
following attribute names.

Language   | Attribute Name   
--|--                   
Python     | ``LoadedConfig.dependencyEndpoints``
Go         | ``LoadedConfig.DependencyEndpoints``
Javascript | ``LoadedConfig.dependencyEndpoints``
Ruby       | ``LoadedConfig.dependencyEndpoints``


Private endpoints are accessible via these attribute names.

Language   | Attribute Name        
--|--                     
Python     | ``LoadedConfig.privateDependencyEndpoints``
Go         | ``LoadedConfig.PrivateDependencyEndpoints``
Javascript | ``LoadedConfig.privateDependencyEndpoints``
Ruby       | ``LoadedConfig.privateDependencyEndpoints``


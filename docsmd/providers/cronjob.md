# CronJob Provider

The **CronJob Provider** is responsible for creating CronJob resources from
`Job` requests in the `ClowdApp` spec.

## ClowdApp Configuration

To request a CronJob, a `Job` spec is created and uses a `Schedule` field
to specify that it is a `CronJob`, rather than a standard Job.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  # Other App Config
  jobs:
  - name: inventory
    schedule: "* * * * */5"
    podSpec:
    image: quay.io/psav/clowder-hello
```

## ClowdEnv Configuration

There is no Environment configuration for the CronJob provider.

## Generated App Configuration

There is no App configuration generated by this provider.

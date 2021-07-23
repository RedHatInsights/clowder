package objectstore

import (
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
)

func objectStoreEquals(actual *config.ObjectStoreConfig, expected *config.ObjectStoreConfig) string {
	oneNil, otherNil := actual == nil, expected == nil

	if oneNil && otherNil {
		return ""
	}

	if oneNil != otherNil {
		return "One object is nil"
	}

	actualLen, expectedLen := len(actual.Buckets), len(expected.Buckets)

	if actualLen != expectedLen {
		return fmt.Sprintf("Different number of buckets %d; expected %d", actualLen, expectedLen)
	}

	for i, bucket := range actual.Buckets {
		expectedBucket := expected.Buckets[i]
		if bucket.Name != expectedBucket.Name {
			return fmt.Sprintf("Bad bucket name %s; expected %s", bucket.Name, expectedBucket.Name)
		}
		if *bucket.AccessKey != *expectedBucket.AccessKey {
			return fmt.Sprintf(
				"%s: Bad accessKey '%s'; expected '%s'",
				bucket.Name,
				*bucket.AccessKey,
				*expectedBucket.AccessKey,
			)
		}
		if *bucket.SecretKey != *expectedBucket.SecretKey {
			return fmt.Sprintf(
				"%s: Bad secretKey %s; expected %s",
				bucket.Name,
				*bucket.SecretKey,
				*expectedBucket.SecretKey,
			)
		}
		if bucket.RequestedName != expectedBucket.RequestedName {
			return fmt.Sprintf(
				"%s: Bad requestedName %s; expected %s",
				bucket.Name,
				bucket.RequestedName,
				expectedBucket.RequestedName,
			)
		}
	}

	if actual.Port != expected.Port {
		return fmt.Sprintf("Bad port %d; expected %d", actual.Port, expected.Port)
	}

	if actual.Hostname != expected.Hostname {
		return fmt.Sprintf("Bad hostname %s; expected %s", actual.Hostname, expected.Hostname)
	}

	if actual.AccessKey != expected.AccessKey {
		return fmt.Sprintf("Bad accessKey %s; expected %s", *actual.AccessKey, *expected.AccessKey)
	}

	if actual.SecretKey != expected.SecretKey {
		return fmt.Sprintf("Bad secretKey %s; expected %s", *actual.SecretKey, *expected.SecretKey)
	}

	return ""
}

package apikey_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestApikey(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Apikey Suite")
}

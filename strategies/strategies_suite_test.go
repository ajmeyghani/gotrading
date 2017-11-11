package strategies_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStrategies(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Strategies Suite")
}

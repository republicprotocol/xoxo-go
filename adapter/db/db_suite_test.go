package db_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLeveldb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "db Suite")
}

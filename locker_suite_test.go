package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLocker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Locker Test Suite")
}

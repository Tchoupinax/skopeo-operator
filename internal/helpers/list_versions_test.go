package helpers_test

import (
	"github.com/Tchoupinax/skopeo-operator/internal/helpers"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parse time", func() {
	It("should correctly find image for 3.x", func() {
		var expectedValue = []string{
			"3.7",
			"3.6",
			"v3.6",
			"3.5",
			"3.4",
			"3.3",
			"3.2",
			"3.1",
			"3.0",
		}
		Expect(helpers.ListVersion(logr.Logger{}, "quay.io/nginx/nginx-ingress", "3.x", false)).To(Equal(expectedValue))
	})

	It("should correctly find image for 3.3.x", func() {
		var expectedValue = []string{
			"3.3.2",
			"3.3.1",
			"3.3.0",
		}
		Expect(helpers.ListVersion(logr.Logger{}, "quay.io/nginx/nginx-ingress", "3.3.x", false)).To(Equal(expectedValue))
	})

	It("should correctly find image with release candidate", func() {
		var expectedValue = []string{
			"v2.13.0-rc5",
			"v2.13.0-rc4",
			"v2.13.0-rc3",
			"v2.13.0-rc2",
			"v2.13.0-rc1",
		}
		Expect(helpers.ListVersion(logr.Logger{}, "quay.io/argoproj/argocd", "2.13.x", true)).To(Equal(expectedValue))
	})

	It("generate regex", func() {
		Expect(helpers.GenerateRegex("2", false)).To(Equal(`^v?2$`))
		Expect(helpers.GenerateRegex("2.x", false)).To(Equal(`^v?2.\d+$`))
		Expect(helpers.GenerateRegex("19.x", false)).To(Equal(`^v?19.\d+$`))
		Expect(helpers.GenerateRegex("2.12.x", false)).To(Equal(`^v?2.12.\d+$`))
		Expect(helpers.GenerateRegex("4.56.x", false)).To(Equal(`^v?4.56.\d+$`))
		Expect(helpers.GenerateRegex("2.12.x", true)).To(Equal(`^v?2.12.\d+(-rc\d)?$`))
		Expect(helpers.GenerateRegex("3.34.x", true)).To(Equal(`^v?3.34.\d+(-rc\d)?$`))
	})
})

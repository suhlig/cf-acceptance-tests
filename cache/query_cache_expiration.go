package cache

import (
	"strings"
	"time"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
)

var _ = Describe("Query Cache Expires", func() {
	var appNameDora string
	var regularUserName string
	var orgName string
	var spaceName string

	var timeout time.Duration = 3 * time.Second
	var deleteTimeout time.Duration = 10 * time.Second
	var cacheRetentionTimeout time.Duration = 60 * time.Second

	var pollingInterval time.Duration = 1 * time.Second

	BeforeSuite(func() {
		Expect(cf.Cf("api", "api.bosh-lite.com", "--skip-ssl-validation").
			Wait(timeout)).To(Exit(0))
	})

	BeforeEach(func() {
		appNameDora = random_name.CATSRandomName("APP")
		regularUserName = random_name.CATSRandomName("USER")
		orgName = random_name.CATSRandomName("ORG")
		spaceName = random_name.CATSRandomName("SPACE")

		asUser("admin", "admin", timeout, func() {
			Expect(cf.Cf("create-org", orgName).
				Wait(timeout)).To(Exit(0))
			Expect(cf.Cf("create-space", "-o", orgName, spaceName).
				Wait(timeout)).To(Exit(0))
			Expect(cf.Cf("create-user", regularUserName, "meow").
				Wait(timeout)).To(Exit(0))

			for _, role := range []string{"SpaceManager", "SpaceDeveloper", "SpaceAuditor"} {
				Expect(cf.Cf("set-space-role", regularUserName, orgName, spaceName, role).
					Wait(timeout)).To(Exit(0))
			}
		})
	})

	AfterEach(func() {
		asUser("admin", "admin", timeout, func() {
			Expect(cf.Cf("delete-org", orgName, "-f").
				Wait(deleteTimeout)).To(Exit(0))
			Expect(cf.Cf("delete-user", regularUserName).
				Wait(deleteTimeout)).To(Exit(0))
		})
	})

	Describe("Removing all roles of a developer from a space", func() {
		It("prevents access", func() {

			asUser(regularUserName, "meow", timeout, func() {
				Expect(cf.Cf("target", "-o", orgName).
					Wait(timeout)).To(Exit(0))

				Eventually(getLastSpace(timeout), timeout, pollingInterval).
					Should(Equal(spaceName))
			})

			asUser("admin", "admin", timeout, func() {
				for _, role := range []string{"SpaceManager", "SpaceDeveloper", "SpaceAuditor"} {
					Expect(cf.Cf("unset-space-role", regularUserName, orgName, spaceName, role).
						Wait(timeout)).To(Exit(0))
				}
			})

			asUser(regularUserName, "meow", timeout, func() {
				Expect(cf.Cf("target", "-o", orgName).
					Wait(timeout)).To(Exit(0))

				Eventually(getLastSpace(timeout), cacheRetentionTimeout, pollingInterval).
					ShouldNot(Equal(spaceName))
			})
		})
	})
})

func asUser(username, password string, timeout time.Duration, action func()) {
	Expect(cf.Cf("auth", username, password).Wait(timeout)).To(Exit(0))
	defer func() {
		Expect(cf.Cf("logout").Wait(timeout)).To(Exit(0))
	}()

	action()
}

func getLastSpace(timeout time.Duration) func() string {
	return func() string {
		session := cf.Cf("spaces").Wait(timeout)
		str := string(session.Buffer().Contents())
		sl := strings.Split(str, "\n")
		if len(sl) < 2 {
			return ""
		}
		return sl[len(sl)-2]
	}
}

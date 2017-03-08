package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

/*
Before:
  * As admin:
    - Create a new space s1
    - Create a new user u1
    - Assign her to be developer in s1

Test:
  * As u1:
    - Push Dora into s1
    - List apps. Expect it to succeed.
  * As admin:
    - Remove u1 from s1 so that she is no longer a SpaceDeveloper
  * As u1:
    - List apps. Expect it to succeed.
  * After cache timeout has passed, as u1:
    - List apps. Expect to NOT succeed.

After:
  * Remove s1 and u1
*/
var _ = AppsDescribe("Query Cache Expires", func() {
	var appNameDora string
	var regularUserName string
	var orgName string
	var spaceName string

	BeforeEach(func() {
		appNameDora = random_name.CATSRandomName("APP")
		regularUserName = TestSetup.RegularUserContext().Username
		orgName = TestSetup.RegularUserContext().Org
		spaceName = TestSetup.RegularUserContext().Space
	})

	AfterEach(func() {
		app_helpers.AppReport(appNameDora, Config.DefaultTimeoutDuration())
		Expect(cf.Cf("delete", appNameDora, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Describe("Removing all roles of a developer from a space", func() {
		XIt("prevents access", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf(
					"push", appNameDora,
					"--no-start",
					"-b", Config.GetRubyBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Dora,
					"-d", Config.GetAppsDomain(),
				).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				app_helpers.SetBackend(appNameDora)
				Expect(cf.Cf("start", appNameDora).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				Eventually(func() string {
					return string(cf.Cf("apps").Out.Contents())
				}, Config.CfPushTimeoutDuration()).Should(ContainSubstring("dora"))
			})

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("unset-space-role", regularUserName, orgName, spaceName, "SpaceDeveloper").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				Eventually(func() string {
					return string(cf.Cf("apps").Out.Contents())
				}, Config.CfPushTimeoutDuration()).Should(Not(ContainSubstring("dora")))
			})
		})
	})
})
